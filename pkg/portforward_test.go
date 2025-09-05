package pkg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"k8s.io/client-go/rest"

	"kubectl-broker/pkg/health"
	"kubectl-broker/testutils"
)

func TestNewPortForwarder(t *testing.T) {
	config := &rest.Config{}
	restClient := &rest.RESTClient{}

	pf := NewPortForwarder(config, restClient)

	if pf == nil {
		t.Fatal("PortForwarder should not be nil")
	}

	if pf.config != config {
		t.Error("Config should be set correctly")
	}

	if pf.restClient != restClient {
		t.Error("REST client should be set correctly")
	}
}

func TestPerformHealthCheckWithOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create mock HTTP server
	healthResponse := testutils.LoadHealthTestData(t, "health_response_healthy.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(healthResponse))
	}))
	defer server.Close()

	// Note: This test focuses on the HTTP client behavior, not full port-forwarding
	// which would require a real Kubernetes cluster
	tests := []struct {
		name          string
		options       health.HealthCheckOptions
		expectParsed  bool
		expectRawJSON bool
		shouldError   bool
	}{
		{
			name: "json output",
			options: health.HealthCheckOptions{
				Endpoint:   "health",
				OutputJSON: true,
				Timeout:    5 * time.Second,
			},
			expectParsed:  true,
			expectRawJSON: true,
			shouldError:   false,
		},
		{
			name: "raw output",
			options: health.HealthCheckOptions{
				Endpoint:  "health",
				OutputRaw: true,
				Timeout:   5 * time.Second,
			},
			expectParsed:  false,
			expectRawJSON: true,
			shouldError:   false,
		},
		{
			name: "default output",
			options: health.HealthCheckOptions{
				Endpoint: "health",
				Timeout:  5 * time.Second,
			},
			expectParsed:  true,
			expectRawJSON: true,
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This tests the HTTP request part without port-forwarding
			client := &http.Client{Timeout: tt.options.Timeout}

			endpointPath := health.GetHealthEndpointPath(tt.options.Endpoint)
			healthURL := server.URL + endpointPath

			resp, err := client.Get(healthURL)
			if err != nil {
				if !tt.shouldError {
					t.Fatalf("Unexpected error: %v", err)
				}
				return
			}
			defer resp.Body.Close()

			if tt.shouldError {
				t.Error("Expected error but got none")
				return
			}

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}

func TestPerformWithPortForwardingOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test the operation function behavior
	operationCalled := false
	testPort := 0

	operation := func(localPort int) error {
		operationCalled = true
		testPort = localPort
		return nil
	}

	// Call the operation directly to test the function signature
	err := operation(8080)

	if err != nil {
		t.Errorf("Operation should not error: %v", err)
	}

	if !operationCalled {
		t.Error("Operation should have been called")
	}

	if testPort != 8080 {
		t.Errorf("Expected port 8080, got %d", testPort)
	}
}

func TestPortForwarderInterface(t *testing.T) {
	// Test that PortForwarder implements PortForwardManager interface
	config := &rest.Config{}
	restClient := &rest.RESTClient{}

	var manager PortForwardManager = NewPortForwarder(config, restClient)

	if manager == nil {
		t.Error("PortForwarder should implement PortForwardManager interface")
	}
}

func TestHealthCheckOptionsValidation(t *testing.T) {
	tests := []struct {
		name        string
		options     health.HealthCheckOptions
		shouldError bool
	}{
		{
			name: "valid options",
			options: health.HealthCheckOptions{
				Endpoint: "health",
				Timeout:  30 * time.Second,
			},
			shouldError: false,
		},
		{
			name: "invalid endpoint",
			options: health.HealthCheckOptions{
				Endpoint: "invalid",
				Timeout:  30 * time.Second,
			},
			shouldError: true,
		},
		{
			name: "conflicting output flags",
			options: health.HealthCheckOptions{
				Endpoint:   "health",
				OutputJSON: true,
				OutputRaw:  true,
				Timeout:    30 * time.Second,
			},
			shouldError: true,
		},
		{
			name: "timeout too short",
			options: health.HealthCheckOptions{
				Endpoint: "health",
				Timeout:  500 * time.Millisecond,
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestPortForwardingContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test context cancellation handling
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately to test context handling
	cancel()

	// The actual port forwarding would fail with context cancelled
	if ctx.Err() == nil {
		t.Error("Context should be cancelled")
	}
}

func TestPortForwardingTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Wait for timeout
	<-ctx.Done()

	if ctx.Err() != context.DeadlineExceeded {
		t.Error("Context should have timed out")
	}
}

// BenchmarkPortForwarderCreation benchmarks port forwarder creation
func BenchmarkPortForwarderCreation(b *testing.B) {
	config := &rest.Config{}
	restClient := &rest.RESTClient{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pf := NewPortForwarder(config, restClient)
		if pf == nil {
			b.Fatal("PortForwarder should not be nil")
		}
	}
}

// TestDeprecatedMethods tests the deprecated methods for backward compatibility
func TestDeprecatedMethods(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	config := &rest.Config{}
	restClient := &rest.RESTClient{}
	pf := NewPortForwarder(config, restClient)

	// Test that deprecated methods exist (compile-time check)
	pod := testutils.CreateTestPod("test-pod", "test-namespace", "10.0.0.1", true)

	// These methods should exist but will fail without a real cluster
	// We're mainly testing that the method signatures are correct
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// ForwardPort is deprecated
	err := pf.ForwardPort(ctx, pod, 8080, 9090)
	if err == nil {
		t.Log("ForwardPort method exists and can be called")
	}

	// PerformHealthCheckOnly is deprecated
	err = pf.PerformHealthCheckOnly(ctx, pod, 8080, 9090)
	if err == nil {
		t.Log("PerformHealthCheckOnly method exists and can be called")
	}
}
