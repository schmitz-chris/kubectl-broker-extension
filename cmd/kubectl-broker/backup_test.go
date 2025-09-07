package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"kubectl-broker/testutils"
)

// TestBackupCommandIntegration tests the backup command with mock dependencies
func TestBackupCommandIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := testutils.NewMockHiveMQServer()
	defer mockServer.Close()

	backupTestData := testutils.LoadBackupTestData(t)

	tests := []struct {
		name           string
		args           []string
		setupServer    func()
		expectedOutput string
		shouldError    bool
	}{
		{
			name: "backup create success",
			args: []string{"backup", "create", "--namespace", "test-namespace"},
			setupServer: func() {
				// Mock successful backup creation
				mockServer.SetResponseCode(200)
				mockServer.SetBackupResponse("test-backup", backupTestData.AsString(backupTestData.BackupStatusCompleted))
			},
			expectedOutput: "Backup",
			shouldError:    false,
		},
		{
			name: "backup list success",
			args: []string{"backup", "list", "--namespace", "test-namespace"},
			setupServer: func() {
				mockServer.SetResponseCode(200)
			},
			expectedOutput: "ID",
			shouldError:    false,
		},
		{
			name: "backup status success",
			args: []string{"backup", "status", "--id", "test-backup", "--namespace", "test-namespace"},
			setupServer: func() {
				mockServer.SetResponseCode(200)
				mockServer.SetBackupResponse("test-backup", backupTestData.AsString(backupTestData.BackupStatusCompleted))
			},
			expectedOutput: "Status:",
			shouldError:    false,
		},
		{
			name:        "missing namespace",
			args:        []string{"backup", "create"},
			shouldError: true,
		},
		{
			name:        "missing backup ID for status",
			args:        []string{"backup", "status", "--namespace", "test"},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupServer != nil {
				tt.setupServer()
			}

			var stdout, stderr bytes.Buffer
			rootCmd := createTestRootCommand(&stdout, &stderr)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			if tt.shouldError {
				testutils.AssertError(t, err, "Expected error but got none")
				return
			}

			// Command might error due to missing Kubernetes cluster in test environment
			output := stdout.String() + stderr.String()
			if err != nil {
				if !isExpectedConnectionError(err) {
					t.Errorf("Unexpected error: %v", err)
				}
			} else if tt.expectedOutput != "" {
				testutils.AssertStringContains(t, output, tt.expectedOutput, "Output should contain expected content")
			}
		})
	}
}

// TestBackupSubcommands tests individual backup subcommands
func TestBackupSubcommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	subcommands := []struct {
		name string
		args []string
	}{
		{"create", []string{"backup", "create", "--help"}},
		{"list", []string{"backup", "list", "--help"}},
		{"status", []string{"backup", "status", "--help"}},
		{"download", []string{"backup", "download", "--help"}},
	}

	for _, sc := range subcommands {
		t.Run(sc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			rootCmd := createTestRootCommand(&stdout, &stderr)
			rootCmd.SetArgs(sc.args)

			err := rootCmd.Execute()
			testutils.AssertNoError(t, err, "Help command should not error")

			output := stdout.String()
			testutils.AssertStringContains(t, output, "Usage:", "Help should show usage information")
		})
	}
}

// TestBackupCreateFlags tests backup create flag combinations
func TestBackupCreateFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name:        "basic create",
			args:        []string{"backup", "create", "--namespace", "test"},
			shouldError: false, // May fail due to no cluster, but flags should be valid
		},
		{
			name:        "create with statefulset",
			args:        []string{"backup", "create", "--statefulset", "broker", "--namespace", "test"},
			shouldError: false,
		},
		{
			name:        "create with destination",
			args:        []string{"backup", "create", "--destination", "/tmp", "--namespace", "test"},
			shouldError: false,
		},
		{
			name:        "create with auth",
			args:        []string{"backup", "create", "--username", "admin", "--password", "secret", "--namespace", "test"},
			shouldError: false,
		},
		{
			name:        "missing required namespace",
			args:        []string{"backup", "create"},
			shouldError: true,
		},
		{
			name:        "empty namespace",
			args:        []string{"backup", "create", "--namespace", ""},
			shouldError: true,
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
			} else {
				// Command might fail due to missing cluster, but flag validation should pass
				if err != nil && !isExpectedConnectionError(err) {
					t.Errorf("Unexpected error (not connection-related): %v", err)
				}
			}
		})
	}
}

// TestBackupListFlags tests backup list flag combinations
func TestBackupListFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name:        "basic list",
			args:        []string{"backup", "list", "--namespace", "test"},
			shouldError: false,
		},
		{
			name:        "list with statefulset",
			args:        []string{"backup", "list", "--statefulset", "broker", "--namespace", "test"},
			shouldError: false,
		},
		{
			name:        "missing namespace",
			args:        []string{"backup", "list"},
			shouldError: true,
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
			} else if err != nil && !isExpectedConnectionError(err) {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestBackupStatusFlags tests backup status flag combinations
func TestBackupStatusFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name:        "status with ID",
			args:        []string{"backup", "status", "--id", "test-backup", "--namespace", "test"},
			shouldError: false,
		},
		{
			name:        "status latest",
			args:        []string{"backup", "status", "--latest", "--namespace", "test"},
			shouldError: false,
		},
		{
			name:        "missing ID and latest",
			args:        []string{"backup", "status", "--namespace", "test"},
			shouldError: true,
		},
		{
			name:        "both ID and latest",
			args:        []string{"backup", "status", "--id", "test", "--latest", "--namespace", "test"},
			shouldError: true,
		},
		{
			name:        "missing namespace",
			args:        []string{"backup", "status", "--id", "test"},
			shouldError: true,
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
			} else if err != nil && !isExpectedConnectionError(err) {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestBackupDownloadFlags tests backup download flag combinations
func TestBackupDownloadFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tempDir := testutils.CreateTempDir(t)

	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name:        "download with ID",
			args:        []string{"backup", "download", "--id", "test-backup", "--output-dir", tempDir, "--namespace", "test"},
			shouldError: false,
		},
		{
			name:        "download latest",
			args:        []string{"backup", "download", "--latest", "--output-dir", tempDir, "--namespace", "test"},
			shouldError: false,
		},
		{
			name:        "missing ID and latest",
			args:        []string{"backup", "download", "--output-dir", tempDir, "--namespace", "test"},
			shouldError: true,
		},
		{
			name:        "missing output directory",
			args:        []string{"backup", "download", "--id", "test", "--namespace", "test"},
			shouldError: true,
		},
		{
			name:        "both ID and latest",
			args:        []string{"backup", "download", "--id", "test", "--latest", "--output-dir", tempDir, "--namespace", "test"},
			shouldError: true,
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
			} else if err != nil && !isExpectedConnectionError(err) {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestBackupCommandTimeout tests command timeout behavior
func TestBackupCommandTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	mockServer := testutils.NewMockHiveMQServer()
	defer mockServer.Close()

	var stdout, stderr bytes.Buffer
	rootCmd := createTestRootCommand(&stdout, &stderr)
	rootCmd.SetArgs([]string{"backup", "create", "--namespace", "test"})

	// Set short context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- rootCmd.ExecuteContext(ctx)
	}()

	select {
	case err := <-done:
		// Command completed (could succeed or fail, both are OK for test)
		if err != nil && !isExpectedConnectionError(err) && !isContextTimeoutError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Command took too long to complete")
	}
}

// TestBackupHelpCommand tests backup help functionality
func TestBackupHelpCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rootCmd := createTestRootCommand(&stdout, &stderr)
	rootCmd.SetArgs([]string{"backup", "--help"})

	err := rootCmd.Execute()
	testutils.AssertNoError(t, err, "Help command should not error")

	output := stdout.String()
	testutils.AssertStringContains(t, output, "backup operations", "Help should contain command description")
	testutils.AssertStringContains(t, output, "create", "Help should show create subcommand")
	testutils.AssertStringContains(t, output, "list", "Help should show list subcommand")
	testutils.AssertStringContains(t, output, "status", "Help should show status subcommand")
	testutils.AssertStringContains(t, output, "download", "Help should show download subcommand")
}

// TestBackupCommandValidation tests input validation for backup commands
func TestBackupCommandValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "invalid statefulset name",
			args:        []string{"backup", "create", "--statefulset", "", "--namespace", "test"},
			shouldError: true,
			errorMsg:    "statefulset",
		},
		{
			name:        "invalid destination path",
			args:        []string{"backup", "create", "--destination", "", "--namespace", "test"},
			shouldError: true,
			errorMsg:    "destination",
		},
		{
			name:        "invalid backup ID",
			args:        []string{"backup", "status", "--id", "", "--namespace", "test"},
			shouldError: true,
			errorMsg:    "id",
		},
		{
			name:        "invalid output directory",
			args:        []string{"backup", "download", "--id", "test", "--output-dir", "", "--namespace", "test"},
			shouldError: true,
			errorMsg:    "output-dir",
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

// isContextTimeoutError checks if error is due to context timeout
func isContextTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "context deadline exceeded") ||
		strings.Contains(errMsg, "timeout")
}

// BenchmarkBackupCommand benchmarks backup command execution
func BenchmarkBackupCommand(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var stdout, stderr bytes.Buffer
		rootCmd := createTestRootCommand(&stdout, &stderr)
		rootCmd.SetArgs([]string{"backup", "--help"}) // Use help to avoid actual connections

		if err := rootCmd.Execute(); err != nil {
			b.Fatalf("Command execution failed: %v", err)
		}
	}
}

// TestBackupCommandsExist tests that all backup subcommands are properly registered
func TestBackupCommandsExist(t *testing.T) {
	rootCmd := createTestRootCommand(&bytes.Buffer{}, &bytes.Buffer{})

	// Find backup command
	var backupCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "backup" {
			backupCmd = cmd
			break
		}
	}

	testutils.AssertNotNil(t, backupCmd, "Backup command should exist")

	if backupCmd != nil {
		expectedSubcommands := []string{"create", "list", "status", "download"}
		actualSubcommands := make(map[string]bool)

		for _, subcmd := range backupCmd.Commands() {
			actualSubcommands[subcmd.Use] = true
		}

		for _, expected := range expectedSubcommands {
			testutils.AssertTrue(t, actualSubcommands[expected],
				"Subcommand "+expected+" should exist")
		}
	}
}

// TestBackupErrorHandling tests error handling in backup operations
func TestBackupErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with server returning errors
	mockServer := testutils.NewMockHiveMQServer()
	defer mockServer.Close()

	mockServer.SetResponseCode(500) // Server error

	tests := []struct {
		name string
		args []string
	}{
		{"create error", []string{"backup", "create", "--namespace", "test"}},
		{"list error", []string{"backup", "list", "--namespace", "test"}},
		{"status error", []string{"backup", "status", "--id", "test", "--namespace", "test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			rootCmd := createTestRootCommand(&stdout, &stderr)
			rootCmd.SetArgs(tt.args)

			err := rootCmd.Execute()

			// Error is expected due to server error or missing cluster
			if err == nil {
				t.Error("Expected error due to server error, but got none")
			}
		})
	}
}
