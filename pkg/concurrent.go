package pkg

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"text/tabwriter"
	"time"

	v1 "k8s.io/api/core/v1"
	"kubectl-broker/pkg/health"
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

// PerformConcurrentHealthChecks performs health checks on multiple pods concurrently
func (k *K8sClient) PerformConcurrentHealthChecks(ctx context.Context, pods []*v1.Pod, portOverride int32, options health.HealthCheckOptions) error {
	results := make([]HealthCheckResult, len(pods))
	var wg sync.WaitGroup

	fmt.Printf("Starting concurrent health checks for %d pods...\n\n", len(pods))

	// Launch a goroutine for each pod
	for i, pod := range pods {
		wg.Add(1)
		go func(index int, p *v1.Pod) {
			defer wg.Done()
			results[index] = k.performSinglePodHealthCheck(ctx, p, portOverride, options)
		}(i, pod)
	}

	// Wait for all health checks to complete
	wg.Wait()

	// Display results in tabular format
	return k.displayHealthCheckResults(results, options)
}

// performSinglePodHealthCheck performs a health check on a single pod
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
	localPort, err := getRandomPort()
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
		result.Details = health.GetHealthSummary(parsedHealth)
	} else if parsedHealth != nil {
		result.Status = string(parsedHealth.OverallStatus)
		result.Details = health.GetHealthSummary(parsedHealth)
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
		return k.displayDetailedResults(results)
	}

	// Default tabular output
	return k.displayTabularResults(results)
}

// displayJSONResults outputs results as JSON
func (k *K8sClient) displayJSONResults(results []HealthCheckResult) error {
	for _, result := range results {
		if result.RawJSON != nil {
			fmt.Printf("Pod: %s\n", result.PodName)
			fmt.Println(string(result.RawJSON))
			fmt.Println()
		}
	}
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
func (k *K8sClient) displayDetailedResults(results []HealthCheckResult) error {
	for _, result := range results {
		fmt.Printf("Pod: %s\n", result.PodName)
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Response Time: %v\n", result.ResponseTime.Round(time.Millisecond))

		if result.ParsedHealth != nil {
			fmt.Printf("Overall Health: %s\n", health.FormatHealthStatus(result.ParsedHealth.OverallStatus))
			if len(result.ParsedHealth.ComponentDetails) > 0 {
				fmt.Println("Components:")
				for _, comp := range result.ParsedHealth.ComponentDetails {
					fmt.Printf("  - %s: %s", comp.Name, health.FormatHealthStatus(comp.Status))
					if comp.Details != "" {
						fmt.Printf(" (%s)", comp.Details)
					}
					fmt.Println()
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
func (k *K8sClient) displayTabularResults(results []HealthCheckResult) error {
	// Create tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "POD NAME\tSTATUS\tHEALTH PORT\tLOCAL PORT\tRESPONSE TIME\tDETAILS")
	fmt.Fprintln(w, "--------\t------\t-----------\t----------\t-------------\t-------")

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
		if len(details) > 50 {
			details = details[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			result.PodName,
			result.Status,
			healthPortStr,
			localPortStr,
			responseTimeStr,
			details)

		if result.Status == "HEALTHY" {
			healthyCount++
		}
	}

	// Flush the tabwriter
	w.Flush()

	// Print summary
	fmt.Printf("\nSummary: %d/%d pods healthy\n", healthyCount, len(results))

	if healthyCount == len(results) {
		fmt.Println("All pods are healthy!")
	} else {
		fmt.Printf("%d pods have issues\n", len(results)-healthyCount)
	}

	return nil
}

// getRandomPort returns a random available port (helper function)
func getRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
