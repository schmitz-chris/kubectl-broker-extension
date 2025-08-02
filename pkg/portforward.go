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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// PortForwarder manages port-forwarding to a Kubernetes pod
type PortForwarder struct {
	config    *rest.Config
	clientset *kubernetes.Clientset
}

// NewPortForwarder creates a new port forwarder
func NewPortForwarder(config *rest.Config, clientset *kubernetes.Clientset) *PortForwarder {
	return &PortForwarder{
		config:    config,
		clientset: clientset,
	}
}

// ForwardPort establishes a port-forward connection and performs a health check
func (pf *PortForwarder) ForwardPort(ctx context.Context, pod *v1.Pod, remotePort int32, localPort int) error {
	// Build the port-forward URL
	req := pf.clientset.CoreV1().RESTClient().Post().
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
func (pf *PortForwarder) PerformHealthCheckOnly(ctx context.Context, pod *v1.Pod, remotePort int32, localPort int) error {
	// Build the port-forward URL
	req := pf.clientset.CoreV1().RESTClient().Post().
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
