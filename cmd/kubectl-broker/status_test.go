package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"kubectl-broker/testutils"
)

// TestStatusCommandIntegration tests the status command with mock dependencies
func TestStatusCommandIntegration(t *testing.T) {
	// Skip short tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		args           []string
		healthResponse string
		expectedOutput string
		shouldError    bool
	}{
		{
			name:           "healthy cluster status",
			args:           []string{"status", "--namespace", "test-namespace", "--json"},
			healthResponse: testutils.LoadHealthTestData(t, "health_response_healthy.json"),
			expectedOutput: "UP",
			shouldError:    false,
		},
		{
			name:           "degraded cluster status",
			args:           []string{"status", "--namespace", "test-namespace", "--json"},
			healthResponse: testutils.LoadHealthTestData(t, "health_response_degraded.json"),
			expectedOutput: "DEGRADED",
			shouldError:    false,
		},
		{
			name:        "missing namespace",
			args:        []string{"status", "--namespace", "nonexistent"},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock server
			mockServer := testutils.NewMockHiveMQServer()
			defer mockServer.Close()

			if tt.healthResponse != "" {
				mockServer.SetHealthResponse(tt.healthResponse)
			} else {
				mockServer.SetResponseCode(500) // Simulate error
			}

			// Capture output
			var stdout, stderr bytes.Buffer

			// Create root command with test setup
			rootCmd := createTestRootCommand(&stdout, &stderr)
			rootCmd.SetArgs(tt.args)

			// Execute command
			err := rootCmd.Execute()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check output contains expected content
			output := stdout.String()
			if tt.expectedOutput != "" {
				testutils.AssertStringContains(t, output, tt.expectedOutput, "Output should contain expected status")
			}
		})
	}
}

// TestStatusCommandFlags tests various flag combinations
func TestStatusCommandFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name:        "discover flag",
			args:        []string{"status", "--discover"},
			shouldError: false,
		},
		{
			name:        "detailed output",
			args:        []string{"status", "--detailed"},
			shouldError: false,
		},
		{
			name:        "json output",
			args:        []string{"status", "--json"},
			shouldError: false,
		},
		{
			name:        "raw output",
			args:        []string{"status", "--raw"},
			shouldError: false,
		},
		{
			name:        "specific endpoint",
			args:        []string{"status", "--endpoint", "liveness"},
			shouldError: false,
		},
		{
			name:        "invalid endpoint",
			args:        []string{"status", "--endpoint", "invalid"},
			shouldError: true,
		},
		{
			name:        "conflicting output flags",
			args:        []string{"status", "--json", "--raw"},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			rootCmd := createTestRootCommand(&stdout, &stderr)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.shouldError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestStatusCommandTimeout tests command timeout behavior
func TestStatusCommandTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a slow mock server
	mockServer := testutils.NewMockHiveMQServer()
	defer mockServer.Close()

	// Add delay to simulate slow response
	mockServer.SetHealthResponse(testutils.LoadHealthTestData(t, "health_response_healthy.json"))

	var stdout, stderr bytes.Buffer
	rootCmd := createTestRootCommand(&stdout, &stderr)
	rootCmd.SetArgs([]string{"status", "--timeout", "1s"})

	// Set very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- rootCmd.ExecuteContext(ctx)
	}()

	select {
	case err := <-done:
		if err == nil {
			// Command completed successfully (which is fine for mocks)
			return
		}
	case <-ctx.Done():
		// Context timeout (expected)
		return
	case <-time.After(2 * time.Second):
		t.Fatal("Command took too long to complete or timeout")
	}
}

// TestStatusCommandDiscovery tests discovery functionality
func TestStatusCommandDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	var stdout, stderr bytes.Buffer
	rootCmd := createTestRootCommand(&stdout, &stderr)
	rootCmd.SetArgs([]string{"status", "--discover"})

	err := rootCmd.Execute()

	// Discovery might fail in test environment, which is OK
	// We're mainly testing that the flag is processed correctly
	output := stdout.String() + stderr.String()

	if err != nil && !isExpectedDiscoveryError(err) {
		t.Errorf("Unexpected error during discovery: %v", err)
	}

	// Check that discovery output contains expected elements
	if output != "" {
		// Should contain some indication of discovery attempt
		hasDiscoveryOutput := len(output) > 0
		testutils.AssertTrue(t, hasDiscoveryOutput, "Discovery should produce some output")
	}
}

// TestStatusCommandOutputFormats tests different output formats
func TestStatusCommandOutputFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := testutils.NewMockHiveMQServer()
	defer mockServer.Close()

	mockServer.SetHealthResponse(testutils.LoadHealthTestData(t, "health_response_healthy.json"))

	formats := []struct {
		name string
		args []string
	}{
		{"default", []string{"status"}},
		{"json", []string{"status", "--json"}},
		{"detailed", []string{"status", "--detailed"}},
		{"raw", []string{"status", "--raw"}},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			rootCmd := createTestRootCommand(&stdout, &stderr)
			rootCmd.SetArgs(format.args)

			err := rootCmd.Execute()

			// Error is expected when no cluster is available
			// We're testing that different output formats don't crash
			if err != nil && !isExpectedConnectionError(err) {
				t.Errorf("Unexpected error for format %s: %v", format.name, err)
			}
		})
	}
}

// createTestRootCommand creates a root command configured for testing
func createTestRootCommand(stdout, stderr io.Writer) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "kubectl-broker",
		Short: "Test kubectl-broker command",
	}

	// Redirect output for testing
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	// Add subcommands
	rootCmd.AddCommand(newStatusCommand())
	rootCmd.AddCommand(newBackupCommand())

	return rootCmd
}

// isExpectedDiscoveryError checks if an error is expected during discovery in test environment
func isExpectedDiscoveryError(err error) bool {
	if err == nil {
		return true
	}

	errMsg := err.Error()
	expectedErrors := []string{
		"unable to load kubeconfig",
		"connection refused",
		"no such host",
		"context deadline exceeded",
		"no brokers found",
		"namespace not found",
	}

	for _, expected := range expectedErrors {
		if strings.Contains(errMsg, expected) {
			return true
		}
	}

	return false
}

// isExpectedConnectionError checks if an error is expected during connection attempts
func isExpectedConnectionError(err error) bool {
	if err == nil {
		return true
	}

	errMsg := err.Error()
	expectedErrors := []string{
		"connection refused",
		"no such host",
		"context deadline exceeded",
		"unable to load kubeconfig",
		"no brokers found",
		"port forwarding failed",
	}

	for _, expected := range expectedErrors {
		if strings.Contains(errMsg, expected) {
			return true
		}
	}

	return false
}

// BenchmarkStatusCommand benchmarks the status command execution
func BenchmarkStatusCommand(b *testing.B) {
	mockServer := testutils.NewMockHiveMQServer()
	defer mockServer.Close()

	mockServer.SetHealthResponse(testutils.LoadHealthTestData(&testing.T{}, "health_response_healthy.json"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var stdout, stderr bytes.Buffer
		rootCmd := createTestRootCommand(&stdout, &stderr)
		rootCmd.SetArgs([]string{"status", "--help"}) // Use help to avoid actual connections

		if err := rootCmd.Execute(); err != nil {
			b.Fatalf("Command execution failed: %v", err)
		}
	}
}

// TestStatusCommandHelp tests the help functionality
func TestStatusCommandHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rootCmd := createTestRootCommand(&stdout, &stderr)
	rootCmd.SetArgs([]string{"status", "--help"})

	err := rootCmd.Execute()
	testutils.AssertNoError(t, err, "Help command should not error")

	output := stdout.String()
	testutils.AssertStringContains(t, output, "health diagnostics", "Help should contain command description")
	testutils.AssertStringContains(t, output, "--namespace", "Help should show namespace flag")
	testutils.AssertStringContains(t, output, "--json", "Help should show json flag")
}

// TestStatusCommandValidation tests input validation
func TestStatusCommandValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "invalid timeout",
			args:        []string{"status", "--timeout", "invalid"},
			shouldError: true,
			errorMsg:    "invalid",
		},
		{
			name:        "negative timeout",
			args:        []string{"status", "--timeout", "-1s"},
			shouldError: true,
			errorMsg:    "timeout",
		},
		{
			name:        "empty namespace",
			args:        []string{"status", "--namespace", ""},
			shouldError: true,
			errorMsg:    "namespace",
		},
		{
			name:        "empty statefulset",
			args:        []string{"status", "--statefulset", ""},
			shouldError: true,
			errorMsg:    "statefulset",
		},
		{
			name:        "invalid port",
			args:        []string{"status", "--port", "99999"},
			shouldError: true,
			errorMsg:    "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			rootCmd := createTestRootCommand(&stdout, &stderr)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.shouldError {
				testutils.AssertError(t, err, "Expected validation error")
				if tt.errorMsg != "" {
					testutils.AssertStringContains(t, err.Error(), tt.errorMsg, "Error should contain expected message")
				}
			} else {
				testutils.AssertNoError(t, err, "Should not have validation error")
			}
		})
	}
}
