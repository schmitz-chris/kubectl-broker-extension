package pkg

import (
	"testing"
	"time"

	"kubectl-broker/pkg/health"
	"kubectl-broker/testutils"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	if config == nil {
		t.Fatal("DefaultClientConfig should not return nil")
	}

	if config.ShowDebug != false {
		t.Error("Default ShowDebug should be false")
	}

	if config.RequestTimeout != 30*time.Second {
		t.Errorf("Default RequestTimeout should be 30s, got %v", config.RequestTimeout)
	}

	if config.RetryCount != 3 {
		t.Errorf("Default RetryCount should be 3, got %d", config.RetryCount)
	}

	if config.UserAgent != "kubectl-broker/1.0" {
		t.Errorf("Default UserAgent should be 'kubectl-broker/1.0', got %q", config.UserAgent)
	}

	if config.QPS != 50.0 {
		t.Errorf("Default QPS should be 50.0, got %f", config.QPS)
	}

	if config.Burst != 100 {
		t.Errorf("Default Burst should be 100, got %d", config.Burst)
	}
}

func TestClientOptions(t *testing.T) {
	tests := []struct {
		name        string
		option      ClientOption
		shouldError bool
		validate    func(*testing.T, *ClientConfig)
	}{
		{
			name:   "WithShowDebug true",
			option: WithShowDebug(true),
			validate: func(t *testing.T, config *ClientConfig) {
				if !config.ShowDebug {
					t.Error("ShowDebug should be true")
				}
			},
		},
		{
			name:   "WithShowDebug false",
			option: WithShowDebug(false),
			validate: func(t *testing.T, config *ClientConfig) {
				if config.ShowDebug {
					t.Error("ShowDebug should be false")
				}
			},
		},
		{
			name:   "WithRequestTimeout valid",
			option: WithRequestTimeout(60 * time.Second),
			validate: func(t *testing.T, config *ClientConfig) {
				if config.RequestTimeout != 60*time.Second {
					t.Errorf("RequestTimeout should be 60s, got %v", config.RequestTimeout)
				}
			},
		},
		{
			name:        "WithRequestTimeout zero",
			option:      WithRequestTimeout(0),
			shouldError: true,
		},
		{
			name:        "WithRequestTimeout negative",
			option:      WithRequestTimeout(-1 * time.Second),
			shouldError: true,
		},
		{
			name:        "WithRequestTimeout too large",
			option:      WithRequestTimeout(10 * time.Minute),
			shouldError: true,
		},
		{
			name:   "WithRetryCount valid",
			option: WithRetryCount(5),
			validate: func(t *testing.T, config *ClientConfig) {
				if config.RetryCount != 5 {
					t.Errorf("RetryCount should be 5, got %d", config.RetryCount)
				}
			},
		},
		{
			name:        "WithRetryCount negative",
			option:      WithRetryCount(-1),
			shouldError: true,
		},
		{
			name:        "WithRetryCount too large",
			option:      WithRetryCount(15),
			shouldError: true,
		},
		{
			name:   "WithKubeConfigPath",
			option: WithKubeConfigPath("/path/to/kubeconfig"),
			validate: func(t *testing.T, config *ClientConfig) {
				if config.KubeConfigPath != "/path/to/kubeconfig" {
					t.Errorf("KubeConfigPath should be '/path/to/kubeconfig', got %q", config.KubeConfigPath)
				}
			},
		},
		{
			name:   "WithContext",
			option: WithContext("test-context"),
			validate: func(t *testing.T, config *ClientConfig) {
				if config.Context != "test-context" {
					t.Errorf("Context should be 'test-context', got %q", config.Context)
				}
			},
		},
		{
			name:   "WithUserAgent valid",
			option: WithUserAgent("test-agent/1.0"),
			validate: func(t *testing.T, config *ClientConfig) {
				if config.UserAgent != "test-agent/1.0" {
					t.Errorf("UserAgent should be 'test-agent/1.0', got %q", config.UserAgent)
				}
			},
		},
		{
			name:        "WithUserAgent empty",
			option:      WithUserAgent(""),
			shouldError: true,
		},
		{
			name:   "WithQPS valid",
			option: WithQPS(100.0),
			validate: func(t *testing.T, config *ClientConfig) {
				if config.QPS != 100.0 {
					t.Errorf("QPS should be 100.0, got %f", config.QPS)
				}
			},
		},
		{
			name:        "WithQPS zero",
			option:      WithQPS(0),
			shouldError: true,
		},
		{
			name:        "WithQPS too large",
			option:      WithQPS(2000),
			shouldError: true,
		},
		{
			name:   "WithBurst valid",
			option: WithBurst(200),
			validate: func(t *testing.T, config *ClientConfig) {
				if config.Burst != 200 {
					t.Errorf("Burst should be 200, got %d", config.Burst)
				}
			},
		},
		{
			name:        "WithBurst zero",
			option:      WithBurst(0),
			shouldError: true,
		},
		{
			name:        "WithBurst too large",
			option:      WithBurst(20000),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultClientConfig()
			err := tt.option(config)

			if tt.shouldError {
				testutils.AssertError(t, err, "Expected validation error")
			} else {
				testutils.AssertNoError(t, err, "Should not have validation error")
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestHealthCheckOptions(t *testing.T) {
	tests := []struct {
		name        string
		option      HealthCheckOption
		shouldError bool
		validate    func(*testing.T, *health.HealthCheckOptions)
	}{
		{
			name:   "WithEndpoint health",
			option: WithEndpoint("health"),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if opts.Endpoint != "health" {
					t.Errorf("Endpoint should be 'health', got %q", opts.Endpoint)
				}
			},
		},
		{
			name:   "WithEndpoint liveness",
			option: WithEndpoint("liveness"),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if opts.Endpoint != "liveness" {
					t.Errorf("Endpoint should be 'liveness', got %q", opts.Endpoint)
				}
			},
		},
		{
			name:   "WithEndpoint readiness",
			option: WithEndpoint("readiness"),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if opts.Endpoint != "readiness" {
					t.Errorf("Endpoint should be 'readiness', got %q", opts.Endpoint)
				}
			},
		},
		{
			name:        "WithEndpoint invalid",
			option:      WithEndpoint("invalid"),
			shouldError: true,
		},
		{
			name:   "WithOutputFormat JSON",
			option: WithOutputFormat(true, false),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if !opts.OutputJSON {
					t.Error("OutputJSON should be true")
				}
				if opts.OutputRaw {
					t.Error("OutputRaw should be false")
				}
			},
		},
		{
			name:   "WithOutputFormat Raw",
			option: WithOutputFormat(false, true),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if opts.OutputJSON {
					t.Error("OutputJSON should be false")
				}
				if !opts.OutputRaw {
					t.Error("OutputRaw should be true")
				}
			},
		},
		{
			name:        "WithOutputFormat both",
			option:      WithOutputFormat(true, true),
			shouldError: true,
		},
		{
			name:   "WithDetailedOutput true",
			option: WithDetailedOutput(true),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if !opts.Detailed {
					t.Error("Detailed should be true")
				}
			},
		},
		{
			name:   "WithDetailedOutput false",
			option: WithDetailedOutput(false),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if opts.Detailed {
					t.Error("Detailed should be false")
				}
			},
		},
		{
			name:   "WithHealthTimeout valid",
			option: WithHealthTimeout(30 * time.Second),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if opts.Timeout != 30*time.Second {
					t.Errorf("Timeout should be 30s, got %v", opts.Timeout)
				}
			},
		},
		{
			name:        "WithHealthTimeout too short",
			option:      WithHealthTimeout(500 * time.Millisecond),
			shouldError: true,
		},
		{
			name:        "WithHealthTimeout too long",
			option:      WithHealthTimeout(10 * time.Minute),
			shouldError: true,
		},
		{
			name:   "WithColors true",
			option: WithColors(true),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if !opts.UseColors {
					t.Error("UseColors should be true")
				}
			},
		},
		{
			name:   "WithColors false",
			option: WithColors(false),
			validate: func(t *testing.T, opts *health.HealthCheckOptions) {
				if opts.UseColors {
					t.Error("UseColors should be false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &health.HealthCheckOptions{
				Endpoint:   "health",
				OutputJSON: false,
				OutputRaw:  false,
				Detailed:   false,
				Timeout:    10 * time.Second,
				UseColors:  true,
			}

			err := tt.option(opts)

			if tt.shouldError {
				testutils.AssertError(t, err, "Expected validation error")
			} else {
				testutils.AssertNoError(t, err, "Should not have validation error")
				if tt.validate != nil {
					tt.validate(t, opts)
				}
			}
		})
	}
}

func TestWorkerPoolOptions(t *testing.T) {
	tests := []struct {
		name        string
		option      WorkerPoolOption
		shouldError bool
		validate    func(*testing.T, *WorkerPoolConfig)
	}{
		{
			name:   "WithMaxWorkers valid",
			option: WithMaxWorkers(5),
			validate: func(t *testing.T, config *WorkerPoolConfig) {
				if config.MaxWorkers != 5 {
					t.Errorf("MaxWorkers should be 5, got %d", config.MaxWorkers)
				}
			},
		},
		{
			name:        "WithMaxWorkers zero",
			option:      WithMaxWorkers(0),
			shouldError: true,
		},
		{
			name:        "WithMaxWorkers too large",
			option:      WithMaxWorkers(150),
			shouldError: true,
		},
		{
			name:   "WithQueueSize valid",
			option: WithQueueSize(200),
			validate: func(t *testing.T, config *WorkerPoolConfig) {
				if config.QueueSize != 200 {
					t.Errorf("QueueSize should be 200, got %d", config.QueueSize)
				}
			},
		},
		{
			name:        "WithQueueSize zero",
			option:      WithQueueSize(0),
			shouldError: true,
		},
		{
			name:        "WithQueueSize too large",
			option:      WithQueueSize(20000),
			shouldError: true,
		},
		{
			name:   "WithWorkerTimeout valid",
			option: WithWorkerTimeout(2 * time.Minute),
			validate: func(t *testing.T, config *WorkerPoolConfig) {
				if config.RequestTimeout != 2*time.Minute {
					t.Errorf("RequestTimeout should be 2m, got %v", config.RequestTimeout)
				}
			},
		},
		{
			name:        "WithWorkerTimeout zero",
			option:      WithWorkerTimeout(0),
			shouldError: true,
		},
		{
			name:        "WithWorkerTimeout too large",
			option:      WithWorkerTimeout(15 * time.Minute),
			shouldError: true,
		},
		{
			name:   "WithShutdownTimeout valid",
			option: WithShutdownTimeout(30 * time.Second),
			validate: func(t *testing.T, config *WorkerPoolConfig) {
				if config.ShutdownTimeout != 30*time.Second {
					t.Errorf("ShutdownTimeout should be 30s, got %v", config.ShutdownTimeout)
				}
			},
		},
		{
			name:        "WithShutdownTimeout zero",
			option:      WithShutdownTimeout(0),
			shouldError: true,
		},
		{
			name:        "WithShutdownTimeout too large",
			option:      WithShutdownTimeout(2 * time.Minute),
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &WorkerPoolConfig{
				MaxWorkers:      2,
				QueueSize:       10,
				RequestTimeout:  30 * time.Second,
				ShutdownTimeout: 5 * time.Second,
			}

			err := tt.option(config)

			if tt.shouldError {
				testutils.AssertError(t, err, "Expected validation error")
			} else {
				testutils.AssertNoError(t, err, "Should not have validation error")
				if tt.validate != nil {
					tt.validate(t, config)
				}
			}
		})
	}
}

func TestNewK8sClientWithOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test successful client creation
	client, err := NewK8sClientWithOptions(
		WithShowDebug(false),
		WithRequestTimeout(45*time.Second),
		WithRetryCount(2),
	)

	// May fail if no kubeconfig available, which is OK for test
	if err != nil {
		t.Logf("Client creation failed (expected in test environment): %v", err)
		return
	}

	if client == nil {
		t.Error("Client should not be nil when creation succeeds")
	}
}

func TestNewK8sClientWithInvalidOptions(t *testing.T) {
	// Test client creation with invalid options
	_, err := NewK8sClientWithOptions(
		WithRequestTimeout(-1 * time.Second), // Invalid
	)

	testutils.AssertError(t, err, "Should error with invalid options")
}

func TestNewHealthCheckOptionsWithOptions(t *testing.T) {
	opts, err := NewHealthCheckOptionsWithOptions(
		WithEndpoint("liveness"),
		WithOutputFormat(true, false),
		WithDetailedOutput(true),
		WithHealthTimeout(20*time.Second),
		WithColors(false),
	)

	testutils.AssertNoError(t, err, "Should create options successfully")
	testutils.AssertNotNil(t, opts, "Options should not be nil")

	if opts.Endpoint != "liveness" {
		t.Errorf("Endpoint should be 'liveness', got %q", opts.Endpoint)
	}

	if !opts.OutputJSON {
		t.Error("OutputJSON should be true")
	}

	if !opts.Detailed {
		t.Error("Detailed should be true")
	}

	if opts.Timeout != 20*time.Second {
		t.Errorf("Timeout should be 20s, got %v", opts.Timeout)
	}

	if opts.UseColors {
		t.Error("UseColors should be false")
	}
}

func TestNewHealthCheckOptionsWithInvalidOptions(t *testing.T) {
	// Test with conflicting options
	_, err := NewHealthCheckOptionsWithOptions(
		WithOutputFormat(true, true), // Invalid: both JSON and raw
	)

	testutils.AssertError(t, err, "Should error with conflicting options")
}

func TestNewWorkerPoolWithOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, err := NewK8sClient(false)
	if err != nil {
		t.Skip("No kubeconfig available, skipping worker pool test")
	}

	wp, err := NewWorkerPoolWithOptions(
		k8sClient,
		WithMaxWorkers(3),
		WithQueueSize(50),
		WithWorkerTimeout(45*time.Second),
		WithShutdownTimeout(10*time.Second),
	)

	testutils.AssertNoError(t, err, "Should create worker pool successfully")
	testutils.AssertNotNil(t, wp, "Worker pool should not be nil")

	if wp.workers != 3 {
		t.Errorf("Workers should be 3, got %d", wp.workers)
	}

	defer wp.Stop()
}

func TestNewWorkerPoolWithInvalidOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	k8sClient, err := NewK8sClient(false)
	if err != nil {
		t.Skip("No kubeconfig available, skipping worker pool test")
	}

	_, err = NewWorkerPoolWithOptions(
		k8sClient,
		WithMaxWorkers(-1), // Invalid
	)

	testutils.AssertError(t, err, "Should error with invalid options")
}

// BenchmarkClientOptionApplication benchmarks option application
func BenchmarkClientOptionApplication(b *testing.B) {
	options := []ClientOption{
		WithShowDebug(false),
		WithRequestTimeout(45 * time.Second),
		WithRetryCount(2),
		WithUserAgent("test-agent/1.0"),
		WithQPS(75.0),
		WithBurst(150),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := DefaultClientConfig()
		for _, option := range options {
			option(config)
		}
	}
}

// BenchmarkHealthCheckOptionApplication benchmarks health check option application
func BenchmarkHealthCheckOptionApplication(b *testing.B) {
	options := []HealthCheckOption{
		WithEndpoint("health"),
		WithOutputFormat(false, false),
		WithDetailedOutput(true),
		WithHealthTimeout(30 * time.Second),
		WithColors(true),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opts := &health.HealthCheckOptions{}
		for _, option := range options {
			option(opts)
		}
	}
}
