package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"text/tabwriter"
	"time"

	"kubectl-broker/pkg/health"

	v1 "k8s.io/api/core/v1"
)

// HealthCheckResult represents the result of a health check for a single pod
type HealthCheckResult struct {
	PodName      string
	Status       string
	HealthPort   int32
	LocalPort    int
	ResponseTime time.Duration
	Details      string
	Error        error
	ParsedHealth *health.ParsedHealthData
	RawJSON      []byte
}

// WorkerPoolConfig configures the worker pool for concurrent operations
type WorkerPoolConfig struct {
	MaxWorkers      int
	QueueSize       int
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// DefaultWorkerPoolConfig returns sensible defaults for the worker pool
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		MaxWorkers:      min(runtime.NumCPU()*2, 10), // Limit to reasonable concurrency
		QueueSize:       100,
		RequestTimeout:  30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// HealthCheckJob represents a health check job for the worker pool
type HealthCheckJob struct {
	Index   int
	Pod     *v1.Pod
	Port    int32
	Options health.HealthCheckOptions
	Result  chan<- HealthCheckResult
}

// WorkerPool manages concurrent health check operations
type WorkerPool struct {
	workers   int
	jobs      chan HealthCheckJob
	results   chan HealthCheckResult
	k8sClient *K8sClient
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	config    WorkerPoolConfig
}

// NewWorkerPool creates a new worker pool for health checks
func NewWorkerPool(k8sClient *K8sClient, config WorkerPoolConfig) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		workers:   config.MaxWorkers,
		jobs:      make(chan HealthCheckJob, config.QueueSize),
		results:   make(chan HealthCheckResult, config.QueueSize),
		k8sClient: k8sClient,
		ctx:       ctx,
		cancel:    cancel,
		config:    config,
	}
}

// Start initializes and starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() error {
	close(wp.jobs)

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(wp.config.ShutdownTimeout):
		wp.cancel() // Force cancellation if timeout exceeded
		return NewConfigurationError("worker_pool_shutdown", "timeout exceeded during worker pool shutdown")
	}
}

// SubmitJob submits a health check job to the worker pool
func (wp *WorkerPool) SubmitJob(job HealthCheckJob) error {
	select {
	case wp.jobs <- job:
		return nil
	case <-wp.ctx.Done():
		return NewConfigurationError("worker_pool_submit", "worker pool is shutting down")
	default:
		return NewConfigurationError("worker_pool_submit", "job queue is full")
	}
}

// worker processes health check jobs
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for {
		select {
		case job, ok := <-wp.jobs:
			if !ok {
				return // Channel closed, worker should exit
			}

			// Create context with timeout for this specific job
			jobCtx, cancel := context.WithTimeout(wp.ctx, wp.config.RequestTimeout)
			result := wp.k8sClient.performSinglePodHealthCheckWithContext(jobCtx, job.Pod, job.Port, job.Options)
			cancel()

			// Send result back
			select {
			case job.Result <- result:
			case <-wp.ctx.Done():
				return
			}

		case <-wp.ctx.Done():
			return
		}
	}
}

// PerformConcurrentHealthChecks performs health checks on multiple pods concurrently using a worker pool
func (k *K8sClient) PerformConcurrentHealthChecks(ctx context.Context, pods []*v1.Pod, portOverride int32, options health.HealthCheckOptions) error {
	if len(pods) == 0 {
		return NewValidationError("health_check", "", "no pods provided for health check")
	}

	// Use worker pool for better resource management
	config := DefaultWorkerPoolConfig()
	// Adjust worker count based on number of pods
	if len(pods) < config.MaxWorkers {
		config.MaxWorkers = len(pods)
	}

	wp := NewWorkerPool(k, config)
	wp.Start()
	defer func() {
		if err := wp.Stop(); err != nil {
			// Log error but don't fail the operation
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}
	}()

	// Create results channel and slice
	results := make([]HealthCheckResult, len(pods))
	resultsChan := make(chan HealthCheckResult, len(pods))

	// Submit jobs to worker pool
	for i, pod := range pods {
		job := HealthCheckJob{
			Index:   i,
			Pod:     pod,
			Port:    portOverride,
			Options: options,
			Result:  resultsChan,
		}

		if err := wp.SubmitJob(job); err != nil {
			// If we can't submit job, return error wrapped with context
			return fmt.Errorf("failed to submit health check job for pod %s: %w", pod.Name, err)
		}
	}

	// Collect results with timeout
	timeout := time.After(60 * time.Second) // Overall operation timeout
	completedCount := 0

	for completedCount < len(pods) {
		select {
		case result := <-resultsChan:
			// Find the correct index for this result
			for i, pod := range pods {
				if pod.Name == result.PodName {
					results[i] = result
					break
				}
			}
			completedCount++
		case <-timeout:
			return NewHealthCheckError("concurrent_health_check", fmt.Sprintf("%d pods", len(pods)),
				fmt.Errorf("operation timed out after 60 seconds, completed %d/%d checks", completedCount, len(pods)))
		case <-ctx.Done():
			return NewHealthCheckError("concurrent_health_check", fmt.Sprintf("%d pods", len(pods)), ctx.Err())
		}
	}

	// Display results in tabular format
	return k.displayHealthCheckResults(results, options)
}

// performSinglePodHealthCheckWithContext performs a health check on a single pod with better context handling
func (k *K8sClient) performSinglePodHealthCheckWithContext(ctx context.Context, pod *v1.Pod, portOverride int32, options health.HealthCheckOptions) HealthCheckResult {
	result := HealthCheckResult{
		PodName: pod.Name,
		Status:  "UNKNOWN",
	}

	// Check context cancellation early
	select {
	case <-ctx.Done():
		result.Status = "CANCELLED"
		result.Error = NewHealthCheckError("health_check", pod.Name, ctx.Err())
		result.Details = "Health check was cancelled"
		return result
	default:
	}

	// 1. Validate pod status
	if err := ValidatePodStatus(pod); err != nil {
		result.Status = "POD_NOT_READY"
		result.Error = err
		result.Details = err.Error()
		return result
	}

	// 2. Discover health port with error wrapping
	var healthPort int32
	var err error
	if portOverride > 0 {
		healthPort = portOverride
	} else {
		healthPort, err = k.DiscoverHealthPort(pod)
		if err != nil {
			result.Status = "PORT_DISCOVERY_FAILED"
			result.Error = NewKubernetesError("discover_health_port", pod.Name, err)
			result.Details = err.Error()
			return result
		}
	}
	result.HealthPort = healthPort

	// 3. Get random local port with retry logic
	localPort, err := GetRandomPortWithRetry(ctx, 3)
	if err != nil {
		result.Status = "LOCAL_PORT_FAILED"
		result.Error = NewNetworkError("get_random_port", pod.Name, err)
		result.Details = "Failed to get available local port"
		return result
	}
	result.LocalPort = localPort

	// 4. Perform health check with port-forwarding
	startTime := time.Now()
	pf := NewPortForwarder(k.config, k.restClient)

	parsedHealth, rawJSON, err := pf.PerformHealthCheckWithOptions(ctx, pod, healthPort, localPort, options)
	result.ResponseTime = time.Since(startTime)

	if err != nil {
		result.Status = "HEALTH_CHECK_FAILED"
		result.Error = NewHealthCheckError("perform_health_check", pod.Name, err)
		result.Details = err.Error()
		return result
	}

	// Store parsed health data and raw JSON
	result.ParsedHealth = parsedHealth
	result.RawJSON = rawJSON

	// Set status based on parsed health data with improved logic
	if parsedHealth != nil {
		if health.IsHealthy(parsedHealth.OverallStatus) {
			result.Status = "HEALTHY"
		} else {
			result.Status = string(parsedHealth.OverallStatus)
		}
		result.Details = health.GetHealthSummaryWithColor(parsedHealth, options.UseColors)
	} else {
		// Raw output mode
		result.Status = "RESPONSE_RECEIVED"
		result.Details = "Raw response received"
	}

	return result
}

// performSinglePodHealthCheck performs a health check on a single pod (legacy method)
func (k *K8sClient) performSinglePodHealthCheck(ctx context.Context, pod *v1.Pod, portOverride int32, options health.HealthCheckOptions) HealthCheckResult {
	result := HealthCheckResult{
		PodName: pod.Name,
		Status:  "UNKNOWN",
	}

	// 1. Validate pod status
	if err := ValidatePodStatus(pod); err != nil {
		result.Status = "POD_NOT_READY"
		result.Error = err
		result.Details = err.Error()
		return result
	}

	// 2. Discover health port
	var healthPort int32
	var err error
	if portOverride > 0 {
		healthPort = portOverride
	} else {
		healthPort, err = k.DiscoverHealthPort(pod)
		if err != nil {
			result.Status = "PORT_DISCOVERY_FAILED"
			result.Error = err
			result.Details = err.Error()
			return result
		}
	}
	result.HealthPort = healthPort

	// 3. Get random local port
	localPort, err := GetRandomPort()
	if err != nil {
		result.Status = "LOCAL_PORT_FAILED"
		result.Error = err
		result.Details = "Failed to get available local port"
		return result
	}
	result.LocalPort = localPort

	// 4. Perform health check with port-forwarding
	startTime := time.Now()
	pf := NewPortForwarder(k.config, k.restClient)

	// Use a separate context with timeout for each health check
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parsedHealth, rawJSON, err := pf.PerformHealthCheckWithOptions(checkCtx, pod, healthPort, localPort, options)
	result.ResponseTime = time.Since(startTime)

	if err != nil {
		result.Status = "HEALTH_CHECK_FAILED"
		result.Error = err
		result.Details = err.Error()
		return result
	}

	// Store parsed health data and raw JSON
	result.ParsedHealth = parsedHealth
	result.RawJSON = rawJSON

	// Set status based on parsed health data
	if parsedHealth != nil && health.IsHealthy(parsedHealth.OverallStatus) {
		result.Status = "HEALTHY"
		result.Details = health.GetHealthSummaryWithColor(parsedHealth, options.UseColors)
	} else if parsedHealth != nil {
		result.Status = string(parsedHealth.OverallStatus)
		result.Details = health.GetHealthSummaryWithColor(parsedHealth, options.UseColors)
	} else {
		// Raw output mode
		result.Status = "RESPONSE_RECEIVED"
		result.Details = "Raw response received"
	}

	return result
}

// displayHealthCheckResults displays the results in a formatted table
func (k *K8sClient) displayHealthCheckResults(results []HealthCheckResult, options health.HealthCheckOptions) error {
	// Handle JSON output mode
	if options.OutputJSON {
		return k.displayJSONResults(results)
	}

	// Handle raw output mode
	if options.OutputRaw {
		return k.displayRawResults(results)
	}

	// Handle detailed mode
	if options.Detailed {
		return k.displayDetailedResults(results, options)
	}

	// Default tabular output
	return k.displayTabularResults(results, options)
}

// displayJSONResults outputs results as JSON
func (k *K8sClient) displayJSONResults(results []HealthCheckResult) error {
	jsonResults := make([]map[string]interface{}, 0)

	for _, result := range results {
		if result.ParsedHealth != nil {
			// Create JSON object with pod name and health data
			jsonResult := map[string]interface{}{
				"podName": result.PodName,
				"status":  string(result.ParsedHealth.OverallStatus),
			}

			// Add raw health response components
			var rawHealthResp map[string]interface{}
			if err := json.Unmarshal(result.RawJSON, &rawHealthResp); err == nil {
				if components, exists := rawHealthResp["components"]; exists {
					jsonResult["components"] = components
				}
				if details, exists := rawHealthResp["details"]; exists {
					jsonResult["details"] = details
				}
			}

			jsonResults = append(jsonResults, jsonResult)
		}
	}

	// Output as proper JSON array
	jsonBytes, err := json.MarshalIndent(jsonResults, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON results: %w", err)
	}

	fmt.Println(string(jsonBytes))
	return nil
}

// displayRawResults outputs raw responses
func (k *K8sClient) displayRawResults(results []HealthCheckResult) error {
	for _, result := range results {
		if result.RawJSON != nil {
			fmt.Print(string(result.RawJSON))
		}
	}
	return nil
}

// displayDetailedResults shows detailed component breakdown
func (k *K8sClient) displayDetailedResults(results []HealthCheckResult, options health.HealthCheckOptions) error {
	for _, result := range results {
		fmt.Printf("Pod: %s\n", result.PodName)
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Response Time: %v\n", result.ResponseTime.Round(time.Millisecond))

		if result.ParsedHealth != nil {
			fmt.Printf("Overall Health: %s\n", health.FormatHealthStatusWithColor(result.ParsedHealth.OverallStatus, options.UseColors))
			if len(result.ParsedHealth.ComponentDetails) > 0 {
				fmt.Println("Components:")
				for _, comp := range result.ParsedHealth.ComponentDetails {
					fmt.Printf("  - %s: %s", comp.Name, health.FormatHealthStatusWithColor(comp.Status, options.UseColors))
					if comp.Details != "" {
						fmt.Printf(" (%s)", comp.Details)
					}

					// Special handling for extensions - show individual extensions
					if comp.Name == "extensions" && len(comp.SubComponents) > 0 {
						fmt.Printf(" (%d extensions)", len(comp.SubComponents))
						fmt.Println()
						for _, ext := range comp.SubComponents {
							fmt.Printf("    - %s: %s", ext.Name, health.FormatHealthStatusWithColor(ext.Status, options.UseColors))
							if ext.Details != "" {
								fmt.Printf(" (%s)", ext.Details)
							}
							fmt.Println()
						}
					} else {
						fmt.Println()
					}
				}
			}
		} else if result.Error != nil {
			fmt.Printf("Error: %v\n", result.Error)
		}
		fmt.Println()
	}
	return nil
}

// displayTabularResults shows results in tabular format
func (k *K8sClient) displayTabularResults(results []HealthCheckResult, options health.HealthCheckOptions) error {
	// Create tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header based on detailed mode
	if options.Detailed {
		fmt.Fprintln(w, "POD NAME\tSTATUS\tHEALTH PORT\tLOCAL PORT\tRESPONSE TIME\tDETAILS")
		fmt.Fprintln(w, "--------\t------\t-----------\t----------\t-------------\t-------")
	} else {
		fmt.Fprintln(w, "POD NAME\tSTATUS\tDETAILS")
		fmt.Fprintln(w, "--------\t------\t-------")
	}

	// Print results
	healthyCount := 0
	for _, result := range results {
		responseTimeStr := "-"
		if result.ResponseTime > 0 {
			responseTimeStr = result.ResponseTime.Round(time.Millisecond).String()
		}

		localPortStr := "-"
		if result.LocalPort > 0 {
			localPortStr = fmt.Sprintf("%d", result.LocalPort)
		}

		healthPortStr := "-"
		if result.HealthPort > 0 {
			healthPortStr = fmt.Sprintf("%d", result.HealthPort)
		}

		details := result.Details
		if len(details) > 80 {
			details = details[:77] + "..."
		}

		if options.Detailed {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				result.PodName,
				result.Status,
				healthPortStr,
				localPortStr,
				responseTimeStr,
				details)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				result.PodName,
				result.Status,
				details)
		}

		if result.Status == "HEALTHY" {
			healthyCount++
		}
	}

	// Flush the tabwriter
	w.Flush()

	// Print summary
	fmt.Printf("\nSummary: %d/%d pods healthy\n", healthyCount, len(results))

	if healthyCount < len(results) {
		fmt.Printf("%d pods have issues\n", len(results)-healthyCount)
	}

	return nil
}
