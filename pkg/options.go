package pkg

import (
	"time"

	"kubectl-broker/pkg/health"
)

// ClientOption defines a functional option for configuring clients
type ClientOption func(*ClientConfig) error

// ClientConfig holds configuration options for the K8s client
type ClientConfig struct {
	ShowDebug      bool
	RequestTimeout time.Duration
	RetryCount     int
	KubeConfigPath string
	Context        string
	UserAgent      string
	QPS            float32
	Burst          int
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ShowDebug:      false,
		RequestTimeout: 30 * time.Second,
		RetryCount:     3,
		UserAgent:      "kubectl-broker/1.0",
		QPS:            50.0,
		Burst:          100,
	}
}

// WithShowDebug enables or disables debug output
func WithShowDebug(debug bool) ClientOption {
	return func(config *ClientConfig) error {
		config.ShowDebug = debug
		return nil
	}
}

// WithRequestTimeout sets the request timeout
func WithRequestTimeout(timeout time.Duration) ClientOption {
	return func(config *ClientConfig) error {
		if timeout <= 0 {
			return NewValidationError("client_config", "timeout",
				"timeout must be positive")
		}
		if timeout > 5*time.Minute {
			return NewValidationError("client_config", "timeout",
				"timeout cannot exceed 5 minutes")
		}
		config.RequestTimeout = timeout
		return nil
	}
}

// WithRetryCount sets the number of retries for failed requests
func WithRetryCount(count int) ClientOption {
	return func(config *ClientConfig) error {
		if count < 0 {
			return NewValidationError("client_config", "retry_count",
				"retry count cannot be negative")
		}
		if count > 10 {
			return NewValidationError("client_config", "retry_count",
				"retry count cannot exceed 10")
		}
		config.RetryCount = count
		return nil
	}
}

// WithKubeConfigPath sets a custom kubeconfig path
func WithKubeConfigPath(path string) ClientOption {
	return func(config *ClientConfig) error {
		config.KubeConfigPath = path
		return nil
	}
}

// WithContext sets a specific Kubernetes context to use
func WithContext(context string) ClientOption {
	return func(config *ClientConfig) error {
		config.Context = context
		return nil
	}
}

// WithUserAgent sets a custom user agent string
func WithUserAgent(userAgent string) ClientOption {
	return func(config *ClientConfig) error {
		if userAgent == "" {
			return NewValidationError("client_config", "user_agent",
				"user agent cannot be empty")
		}
		config.UserAgent = userAgent
		return nil
	}
}

// WithQPS sets the queries per second rate limit
func WithQPS(qps float32) ClientOption {
	return func(config *ClientConfig) error {
		if qps <= 0 {
			return NewValidationError("client_config", "qps",
				"QPS must be positive")
		}
		if qps > 1000 {
			return NewValidationError("client_config", "qps",
				"QPS cannot exceed 1000")
		}
		config.QPS = qps
		return nil
	}
}

// WithBurst sets the burst limit for rate limiting
func WithBurst(burst int) ClientOption {
	return func(config *ClientConfig) error {
		if burst <= 0 {
			return NewValidationError("client_config", "burst",
				"burst must be positive")
		}
		if burst > 10000 {
			return NewValidationError("client_config", "burst",
				"burst cannot exceed 10000")
		}
		config.Burst = burst
		return nil
	}
}

// HealthCheckOption defines a functional option for health check configuration
type HealthCheckOption func(*health.HealthCheckOptions) error

// WithEndpoint sets the health check endpoint
func WithEndpoint(endpoint string) HealthCheckOption {
	return func(opts *health.HealthCheckOptions) error {
		validEndpoints := map[string]bool{
			"health":    true,
			"liveness":  true,
			"readiness": true,
		}

		if !validEndpoints[endpoint] {
			return NewValidationError("health_check_options", "endpoint",
				"invalid endpoint: must be health, liveness, or readiness")
		}

		opts.Endpoint = endpoint
		return nil
	}
}

// WithOutputFormat sets the output format for health checks
func WithOutputFormat(json, raw bool) HealthCheckOption {
	return func(opts *health.HealthCheckOptions) error {
		if json && raw {
			return NewValidationError("health_check_options", "output_format",
				"cannot specify both JSON and raw output formats")
		}

		opts.OutputJSON = json
		opts.OutputRaw = raw
		return nil
	}
}

// WithDetailedOutput enables or disables detailed output
func WithDetailedOutput(detailed bool) HealthCheckOption {
	return func(opts *health.HealthCheckOptions) error {
		opts.Detailed = detailed
		return nil
	}
}

// WithHealthTimeout sets the health check timeout
func WithHealthTimeout(timeout time.Duration) HealthCheckOption {
	return func(opts *health.HealthCheckOptions) error {
		if timeout < time.Second {
			return NewValidationError("health_check_options", "timeout",
				"timeout must be at least 1 second")
		}
		if timeout > 5*time.Minute {
			return NewValidationError("health_check_options", "timeout",
				"timeout cannot exceed 5 minutes")
		}

		opts.Timeout = timeout
		return nil
	}
}

// WithColors enables or disables colored output
func WithColors(colors bool) HealthCheckOption {
	return func(opts *health.HealthCheckOptions) error {
		opts.UseColors = colors
		return nil
	}
}

// NewK8sClientWithOptions creates a new K8s client with functional options
func NewK8sClientWithOptions(options ...ClientOption) (*K8sClient, error) {
	config := DefaultClientConfig()

	// Apply all options
	for _, option := range options {
		if err := option(config); err != nil {
			return nil, err
		}
	}

	// Use the original NewK8sClient function but with our config
	return NewK8sClient(config.ShowDebug)
}

// NewHealthCheckOptionsWithOptions creates health check options with functional options
func NewHealthCheckOptionsWithOptions(options ...HealthCheckOption) (*health.HealthCheckOptions, error) {
	opts := &health.HealthCheckOptions{
		Endpoint:   "health",
		OutputJSON: false,
		OutputRaw:  false,
		Detailed:   false,
		Timeout:    10 * time.Second,
		UseColors:  true,
	}

	// Apply all options
	for _, option := range options {
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	// Validate the final configuration
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	return opts, nil
}

// WorkerPoolOption defines a functional option for worker pool configuration
type WorkerPoolOption func(*WorkerPoolConfig) error

// WithMaxWorkers sets the maximum number of workers
func WithMaxWorkers(workers int) WorkerPoolOption {
	return func(config *WorkerPoolConfig) error {
		if workers <= 0 {
			return NewValidationError("worker_pool_config", "max_workers",
				"max workers must be positive")
		}
		if workers > 100 {
			return NewValidationError("worker_pool_config", "max_workers",
				"max workers cannot exceed 100")
		}

		config.MaxWorkers = workers
		return nil
	}
}

// WithQueueSize sets the job queue size
func WithQueueSize(size int) WorkerPoolOption {
	return func(config *WorkerPoolConfig) error {
		if size <= 0 {
			return NewValidationError("worker_pool_config", "queue_size",
				"queue size must be positive")
		}
		if size > 10000 {
			return NewValidationError("worker_pool_config", "queue_size",
				"queue size cannot exceed 10000")
		}

		config.QueueSize = size
		return nil
	}
}

// WithWorkerTimeout sets the request timeout for workers
func WithWorkerTimeout(timeout time.Duration) WorkerPoolOption {
	return func(config *WorkerPoolConfig) error {
		if timeout <= 0 {
			return NewValidationError("worker_pool_config", "request_timeout",
				"request timeout must be positive")
		}
		if timeout > 10*time.Minute {
			return NewValidationError("worker_pool_config", "request_timeout",
				"request timeout cannot exceed 10 minutes")
		}

		config.RequestTimeout = timeout
		return nil
	}
}

// WithShutdownTimeout sets the worker pool shutdown timeout
func WithShutdownTimeout(timeout time.Duration) WorkerPoolOption {
	return func(config *WorkerPoolConfig) error {
		if timeout <= 0 {
			return NewValidationError("worker_pool_config", "shutdown_timeout",
				"shutdown timeout must be positive")
		}
		if timeout > time.Minute {
			return NewValidationError("worker_pool_config", "shutdown_timeout",
				"shutdown timeout cannot exceed 1 minute")
		}

		config.ShutdownTimeout = timeout
		return nil
	}
}

// NewWorkerPoolWithOptions creates a worker pool with functional options
func NewWorkerPoolWithOptions(k8sClient *K8sClient, options ...WorkerPoolOption) (*WorkerPool, error) {
	config := DefaultWorkerPoolConfig()

	// Apply all options
	for _, option := range options {
		if err := option(&config); err != nil {
			return nil, err
		}
	}

	return NewWorkerPool(k8sClient, config), nil
}
