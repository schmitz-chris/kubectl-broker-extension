package pkg

import (
	"context"
	"fmt"
	"net"
	"sync"
	"text/tabwriter"
	"time"
	"os"

	v1 "k8s.io/api/core/v1"
)

// HealthCheckResult represents the result of a health check for a single pod
type HealthCheckResult struct {
	PodName     string
	Status      string
	HealthPort  int32
	LocalPort   int
	ResponseTime time.Duration
	Details     string
	Error       error
}

// PerformConcurrentHealthChecks performs health checks on multiple pods concurrently
func (k *K8sClient) PerformConcurrentHealthChecks(ctx context.Context, pods []*v1.Pod, portOverride int32) error {
	results := make([]HealthCheckResult, len(pods))
	var wg sync.WaitGroup
	
	fmt.Printf("Starting concurrent health checks for %d pods...\n\n", len(pods))
	
	// Launch a goroutine for each pod
	for i, pod := range pods {
		wg.Add(1)
		go func(index int, p *v1.Pod) {
			defer wg.Done()
			results[index] = k.performSinglePodHealthCheck(ctx, p, portOverride)
		}(i, pod)
	}
	
	// Wait for all health checks to complete
	wg.Wait()
	
	// Display results in tabular format
	return k.displayHealthCheckResults(results)
}

// performSinglePodHealthCheck performs a health check on a single pod
func (k *K8sClient) performSinglePodHealthCheck(ctx context.Context, pod *v1.Pod, portOverride int32) HealthCheckResult {
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
	pf := NewPortForwarder(k.config, k.clientset)
	
	// Use a separate context with timeout for each health check
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	if err := pf.PerformHealthCheckOnly(checkCtx, pod, healthPort, localPort); err != nil {
		result.Status = "HEALTH_CHECK_FAILED"
		result.Error = err
		result.Details = err.Error()
		result.ResponseTime = time.Since(startTime)
		return result
	}
	
	result.Status = "HEALTHY"
	result.Details = "Health check successful"
	result.ResponseTime = time.Since(startTime)
	
	return result
}

// displayHealthCheckResults displays the results in a formatted table
func (k *K8sClient) displayHealthCheckResults(results []HealthCheckResult) error {
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
		fmt.Println("✅ All pods are healthy!")
	} else {
		fmt.Printf("⚠️  %d pods have issues\n", len(results)-healthyCount)
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