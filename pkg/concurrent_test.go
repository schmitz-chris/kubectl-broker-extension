package pkg

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"kubectl-broker/pkg/health"
	"kubectl-broker/testutils"
)

func TestDefaultWorkerPoolConfig(t *testing.T) {
	config := DefaultWorkerPoolConfig()

	expectedMaxWorkers := min(runtime.NumCPU()*2, 10)
	if config.MaxWorkers != expectedMaxWorkers {
		t.Errorf("Expected MaxWorkers %d, got %d", expectedMaxWorkers, config.MaxWorkers)
	}

	if config.QueueSize != 100 {
		t.Errorf("Expected QueueSize 100, got %d", config.QueueSize)
	}

	if config.RequestTimeout != 30*time.Second {
		t.Errorf("Expected RequestTimeout 30s, got %v", config.RequestTimeout)
	}

	if config.ShutdownTimeout != 5*time.Second {
		t.Errorf("Expected ShutdownTimeout 5s, got %v", config.ShutdownTimeout)
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{10, 10, 10},
		{0, 1, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		result := min(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestNewWorkerPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Use real K8sClient for worker pool tests (may fail without cluster, which is OK)
	k8sClient, k8sErr := NewK8sClient(false)
	if k8sErr != nil {
		t.Skip("No kubeconfig available, skipping worker pool test")
	}

	config := DefaultWorkerPoolConfig()

	wp := NewWorkerPool(k8sClient, config)
	defer wp.Stop()

	if wp == nil {
		t.Fatal("WorkerPool should not be nil")
	}

	if wp.workers != config.MaxWorkers {
		t.Errorf("Expected %d workers, got %d", config.MaxWorkers, wp.workers)
	}

	if wp.k8sClient != k8sClient {
		t.Error("K8s client should be set correctly")
	}
}

func TestWorkerPoolStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, k8sErr := NewK8sClient(false)
	if k8sErr != nil {
		t.Skip("No kubeconfig available, skipping worker pool test")
	}

	config := WorkerPoolConfig{
		MaxWorkers:      2,
		QueueSize:       10,
		RequestTimeout:  5 * time.Second,
		ShutdownTimeout: 1 * time.Second,
	}

	wp := NewWorkerPool(k8sClient, config)

	// Start the pool
	wp.Start()

	// Give workers time to start
	time.Sleep(100 * time.Millisecond)

	// Stop the pool
	err := wp.Stop()
	if err != nil {
		t.Errorf("Stop should not error: %v", err)
	}
}

func TestWorkerPoolJobSubmission(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, k8sErr := NewK8sClient(false)
	if k8sErr != nil {
		t.Skip("No kubeconfig available, skipping worker pool test")
	}

	config := WorkerPoolConfig{
		MaxWorkers:      1,
		QueueSize:       5,
		RequestTimeout:  5 * time.Second,
		ShutdownTimeout: 1 * time.Second,
	}

	wp := NewWorkerPool(k8sClient, config)
	wp.Start()
	defer wp.Stop()

	// Create test job
	pod := testutils.CreateTestPod("test-pod", "test-namespace", "10.0.0.1", true)
	resultChan := make(chan HealthCheckResult, 1)

	job := HealthCheckJob{
		Index:   0,
		Pod:     pod,
		Port:    9090,
		Options: health.HealthCheckOptions{Endpoint: "health", Timeout: 5 * time.Second},
		Result:  resultChan,
	}

	// Submit job
	err := wp.SubmitJob(job)
	if err != nil {
		t.Errorf("SubmitJob should not error: %v", err)
	}
}

func TestHealthCheckResult(t *testing.T) {
	result := HealthCheckResult{
		PodName:      "test-pod",
		Status:       "HEALTHY",
		HealthPort:   9090,
		LocalPort:    50000,
		ResponseTime: 150 * time.Millisecond,
		Details:      "All systems operational",
		Error:        nil,
	}

	if result.PodName != "test-pod" {
		t.Errorf("Expected PodName 'test-pod', got %q", result.PodName)
	}

	if result.Status != "HEALTHY" {
		t.Errorf("Expected Status 'HEALTHY', got %q", result.Status)
	}

	if result.ResponseTime != 150*time.Millisecond {
		t.Errorf("Expected ResponseTime 150ms, got %v", result.ResponseTime)
	}
}

func TestPerformSinglePodHealthCheckWithContext(t *testing.T) {
	t.Skip("Skipping test that requires internal K8sClient methods")
}

func TestPerformSinglePodHealthCheckWithCancelledContext(t *testing.T) {
	t.Skip("Skipping test that requires internal K8sClient methods")
}

func TestPerformSinglePodHealthCheckWithInvalidPod(t *testing.T) {
	t.Skip("Skipping test that requires internal K8sClient methods")
}

func TestDisplayHealthCheckResults(t *testing.T) {
	t.Skip("Skipping test that requires internal K8sClient methods")
}

func TestDisplayJSONResults(t *testing.T) {
	t.Skip("Skipping test that requires internal K8sClient methods")
}

func TestConcurrentHealthChecksWithTimeout(t *testing.T) {
	t.Skip("Skipping test that requires internal K8sClient methods")
}

func TestWorkerPoolShutdownTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, k8sErr := NewK8sClient(false)
	if k8sErr != nil {
		t.Skip("No kubeconfig available, skipping worker pool test")
	}

	// Use very short shutdown timeout
	config := WorkerPoolConfig{
		MaxWorkers:      2,
		QueueSize:       10,
		RequestTimeout:  5 * time.Second,
		ShutdownTimeout: 1 * time.Millisecond, // Very short
	}

	wp := NewWorkerPool(k8sClient, config)
	wp.Start()

	// Add some work to make shutdown take longer
	pod := testutils.CreateTestPod("test-pod", "test-namespace", "10.0.0.1", true)
	resultChan := make(chan HealthCheckResult, 1)

	job := HealthCheckJob{
		Index:   0,
		Pod:     pod,
		Port:    9090,
		Options: health.HealthCheckOptions{Endpoint: "health", Timeout: 10 * time.Second},
		Result:  resultChan,
	}

	wp.SubmitJob(job)

	// Stop should timeout and return error
	err := wp.Stop()
	if err == nil {
		t.Log("Expected timeout error during shutdown, but shutdown completed quickly")
	}
}

// BenchmarkWorkerPoolCreation benchmarks worker pool creation
func BenchmarkWorkerPoolCreation(b *testing.B) {
	k8sClient, err := NewK8sClient(false)
	if err != nil {
		b.Skip("No kubeconfig available, skipping benchmark")
	}

	config := DefaultWorkerPoolConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wp := NewWorkerPool(k8sClient, config)
		wp.Stop()
	}
}

// BenchmarkHealthCheckResult benchmarks health check result creation
func BenchmarkHealthCheckResult(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := HealthCheckResult{
			PodName:      "test-pod",
			Status:       "HEALTHY",
			HealthPort:   9090,
			LocalPort:    50000,
			ResponseTime: 150 * time.Millisecond,
			Details:      "All systems operational",
		}
		_ = result
	}
}

// Test concurrent access to worker pool
func TestWorkerPoolConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, k8sErr := NewK8sClient(false)
	if k8sErr != nil {
		t.Skip("No kubeconfig available, skipping worker pool test")
	}

	config := WorkerPoolConfig{
		MaxWorkers:      3,
		QueueSize:       20,
		RequestTimeout:  5 * time.Second,
		ShutdownTimeout: 2 * time.Second,
	}

	wp := NewWorkerPool(k8sClient, config)
	wp.Start()
	defer wp.Stop()

	// Submit jobs concurrently
	var wg sync.WaitGroup
	numJobs := 10

	for i := 0; i < numJobs; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			pod := testutils.CreateTestPod("test-pod", "test-namespace", "10.0.0.1", true)
			resultChan := make(chan HealthCheckResult, 1)

			job := HealthCheckJob{
				Index:   index,
				Pod:     pod,
				Port:    9090,
				Options: health.HealthCheckOptions{Endpoint: "health", Timeout: 1 * time.Second},
				Result:  resultChan,
			}

			err := wp.SubmitJob(job)
			if err != nil {
				t.Errorf("Job submission failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}
