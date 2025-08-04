package pkg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"kubectl-broker/pkg/health"
)

// PortForwarder manages port-forwarding to a Kubernetes pod
type PortForwarder struct {
	config     *rest.Config
	restClient rest.Interface
}

// NewPortForwarder creates a new port forwarder
func NewPortForwarder(config *rest.Config, restClient rest.Interface) *PortForwarder {
	return &PortForwarder{
		config:     config,
		restClient: restClient,
	}
}

// ForwardPort establishes a port-forward connection and performs a health check
// Deprecated: Use PerformHealthCheckWithOptions instead for better flexibility
func (pf *PortForwarder) ForwardPort(ctx context.Context, pod *v1.Pod, remotePort int32, localPort int) error {
	// Build the port-forward URL
	req := pf.restClient.Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	// Create SPDY dialer
	transport, upgrader, err := spdy.RoundTripperFor(pf.config)
	if err != nil {
		return fmt.Errorf("failed to create SPDY round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	// Set up channels for port-forward lifecycle
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	errorChan := make(chan error, 1)

	// Handle interrupt signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		fmt.Println("\nReceived interrupt signal, closing port-forward...")
		close(stopChan)
	}()

	// Create port forwarder
	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}
	fw, err := portforward.New(dialer, ports, stopChan, readyChan, io.Discard, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %w", err)
	}

	// Start port forwarding in a goroutine
	go func() {
		if err := fw.ForwardPorts(); err != nil {
			errorChan <- fmt.Errorf("port forwarding failed: %w", err)
		}
	}()

	// Wait for port-forward to be ready or fail
	select {
	case <-readyChan:
		fmt.Printf("Port-forward established: localhost:%d -> %s:%d\n", localPort, pod.Name, remotePort)

		// Perform health check
		if err := pf.performHealthCheck(localPort); err != nil {
			close(stopChan)
			return fmt.Errorf("health check failed: %w", err)
		}

		close(stopChan)
		return nil

	case err := <-errorChan:
		close(stopChan)
		return err

	case <-ctx.Done():
		close(stopChan)
		return ctx.Err()
	}
}

// PerformHealthCheckOnly performs a health check without interactive mode
// Deprecated: Use PerformHealthCheckWithOptions instead for better flexibility
func (pf *PortForwarder) PerformHealthCheckOnly(ctx context.Context, pod *v1.Pod, remotePort int32, localPort int) error {
	// Build the port-forward URL
	req := pf.restClient.Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	// Create SPDY dialer
	transport, upgrader, err := spdy.RoundTripperFor(pf.config)
	if err != nil {
		return fmt.Errorf("failed to create SPDY round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	// Set up channels for port-forward lifecycle
	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})
	errorChan := make(chan error, 1)

	// Create port forwarder
	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}
	fw, err := portforward.New(dialer, ports, stopChan, readyChan, io.Discard, io.Discard)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %w", err)
	}

	// Start port forwarding in a goroutine
	go func() {
		if err := fw.ForwardPorts(); err != nil {
			errorChan <- fmt.Errorf("port forwarding failed: %w", err)
		}
	}()

	// Wait for port-forward to be ready or fail
	select {
	case <-readyChan:
		// Perform health check
		err := pf.performHealthCheckQuiet(localPort)
		close(stopChan)
		return err

	case err := <-errorChan:
		close(stopChan)
		return err

	case <-ctx.Done():
		close(stopChan)
		return ctx.Err()
	}
}

// performHealthCheck makes an HTTP request to the health endpoint
// Deprecated: Use performHealthCheckWithOptions instead
func (pf *PortForwarder) performHealthCheck(localPort int) error {
	healthURL := fmt.Sprintf("http://localhost:%d/api/v1/health", localPort)

	fmt.Printf("Performing health check: %s\n", healthURL)

	resp, err := http.Get(healthURL)
	if err != nil {
		return fmt.Errorf("failed to connect to health endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read health response: %w", err)
	}

	fmt.Printf("Health check response (Status: %s):\n", resp.Status)
	fmt.Println(string(body))

	return nil
}

// performHealthCheckQuiet makes an HTTP request to the health endpoint without verbose output
// Deprecated: Use performHealthCheckWithOptions instead
func (pf *PortForwarder) performHealthCheckQuiet(localPort int) error {
	healthURL := fmt.Sprintf("http://localhost:%d/api/v1/health", localPort)

	resp, err := http.Get(healthURL)
	if err != nil {
		return fmt.Errorf("failed to connect to health endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// PerformHealthCheckWithOptions performs a health check with configurable options
func (pf *PortForwarder) PerformHealthCheckWithOptions(ctx context.Context, pod *v1.Pod, remotePort int32, localPort int, options health.HealthCheckOptions) (*health.ParsedHealthData, []byte, error) {
	// Build the port-forward URL
	req := pf.restClient.Post().
		Resource("pods").
		Namespace(pod.Namespace).
		Name(pod.Name).
		SubResource("portforward")

	// Create SPDY dialer
	transport, upgrader, err := spdy.RoundTripperFor(pf.config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create SPDY round tripper: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	// Set up channels for port-forward lifecycle
	readyChan := make(chan struct{})
	stopChan := make(chan struct{})
	errorChan := make(chan error, 1)

	// Create port forwarder
	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}
	forwarder, err := portforward.New(dialer, ports, stopChan, readyChan, io.Discard, io.Discard)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create port forwarder: %w", err)
	}

	// Start port forwarding in a goroutine
	go func() {
		if err := forwarder.ForwardPorts(); err != nil {
			errorChan <- err
		}
	}()

	// Wait for port-forward to be ready or error
	select {
	case <-readyChan:
		// Perform health check with options
		parsedHealth, rawJSON, err := pf.performHealthCheckWithOptions(localPort, options)
		close(stopChan)
		return parsedHealth, rawJSON, err

	case err := <-errorChan:
		close(stopChan)
		return nil, nil, err

	case <-ctx.Done():
		close(stopChan)
		return nil, nil, ctx.Err()
	}
}

// performHealthCheckWithOptions makes an HTTP request to the specified health endpoint with options
func (pf *PortForwarder) performHealthCheckWithOptions(localPort int, options health.HealthCheckOptions) (*health.ParsedHealthData, []byte, error) {
	endpointPath := health.GetHealthEndpointPath(options.Endpoint)
	healthURL := fmt.Sprintf("http://localhost:%d%s", localPort, endpointPath)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: options.Timeout,
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to health endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read health response: %w", err)
	}

	// Always return raw JSON for potential use
	rawJSON := body

	// If raw output is requested, don't parse
	if options.OutputRaw {
		return nil, rawJSON, nil
	}

	// If JSON output is requested, parse but don't analyze
	if options.OutputJSON {
		parsed, err := health.ParseHealthResponse(body)
		if err != nil {
			// Still return raw JSON if parsing fails
			return nil, rawJSON, err
		}
		return parsed, rawJSON, nil
	}

	// Parse the response for analyzed output
	parsed, err := health.ParseHealthResponse(body)
	if err != nil {
		return nil, rawJSON, fmt.Errorf("failed to parse health response: %w", err)
	}

	return parsed, rawJSON, nil
}
