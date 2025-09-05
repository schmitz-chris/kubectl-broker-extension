package testutils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTimeout is the default timeout for tests
const TestTimeout = 5 * time.Second

// LoadTestData loads test data from the testdata directory
func LoadTestData(t *testing.T, filename string) []byte {
	t.Helper()

	path := filepath.Join("testdata", filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to load test data from %s: %v", path, err)
	}
	return data
}

// LoadHealthTestData loads health response test data
func LoadHealthTestData(t *testing.T, filename string) string {
	t.Helper()
	return string(LoadTestData(t, filename))
}

// LoadBackupTestData loads backup response test data
func LoadBackupTestData(t *testing.T) BackupTestResponses {
	t.Helper()

	data := LoadTestData(t, "backup_responses.json")

	var responses BackupTestResponses
	if err := json.Unmarshal(data, &responses); err != nil {
		t.Fatalf("Failed to unmarshal backup test data: %v", err)
	}

	return responses
}

// BackupTestResponses holds all backup test response data
type BackupTestResponses struct {
	BackupCreateResponse  json.RawMessage `json:"backup_create_response"`
	BackupStatusRunning   json.RawMessage `json:"backup_status_running"`
	BackupStatusCompleted json.RawMessage `json:"backup_status_completed"`
	BackupStatusFailed    json.RawMessage `json:"backup_status_failed"`
	BackupListResponse    json.RawMessage `json:"backup_list_response"`
	BackupListEmpty       json.RawMessage `json:"backup_list_empty"`
}

// AsString converts json.RawMessage to string
func (b BackupTestResponses) AsString(msg json.RawMessage) string {
	return string(msg)
}

// CompareJSON compares two JSON strings for equality, ignoring formatting
func CompareJSON(t *testing.T, expected, actual string) {
	t.Helper()

	var expectedObj, actualObj interface{}

	if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v", err)
	}

	if err := json.Unmarshal([]byte(actual), &actualObj); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v", err)
	}

	expectedBytes, _ := json.Marshal(expectedObj)
	actualBytes, _ := json.Marshal(actualObj)

	if string(expectedBytes) != string(actualBytes) {
		t.Errorf("JSON mismatch:\nExpected: %s\nActual: %s", string(expectedBytes), string(actualBytes))
	}
}

// AssertStringContains checks if a string contains a substring
func AssertStringContains(t *testing.T, str, substr, message string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Errorf("%s: string %q does not contain %q", message, str, substr)
	}
}

// AssertStringNotContains checks if a string does not contain a substring
func AssertStringNotContains(t *testing.T, str, substr, message string) {
	t.Helper()
	if strings.Contains(str, substr) {
		t.Errorf("%s: string %q should not contain %q", message, str, substr)
	}
}

// AssertNoError checks that no error occurred
func AssertNoError(t *testing.T, err error, message string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", message, err)
	}
}

// AssertError checks that an error occurred
func AssertError(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error but got none", message)
	}
}

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, message string) {
	t.Helper()
	if expected != actual {
		t.Errorf("%s: expected %v, got %v", message, expected, actual)
	}
}

// AssertNotEqual checks if two values are not equal
func AssertNotEqual(t *testing.T, expected, actual interface{}, message string) {
	t.Helper()
	if expected == actual {
		t.Errorf("%s: expected values to be different, but both are %v", message, expected)
	}
}

// AssertNil checks if a value is nil
func AssertNil(t *testing.T, value interface{}, message string) {
	t.Helper()
	if value != nil {
		t.Errorf("%s: expected nil, got %v", message, value)
	}
}

// AssertNotNil checks if a value is not nil
func AssertNotNil(t *testing.T, value interface{}, message string) {
	t.Helper()
	if value == nil {
		t.Errorf("%s: expected non-nil value", message)
	}
}

// AssertTrue checks if a condition is true
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Errorf("%s: expected true but got false", message)
	}
}

// AssertFalse checks if a condition is false
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Errorf("%s: expected false but got true", message)
	}
}

// WaitForCondition waits for a condition to become true within timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timeoutChan:
			t.Fatalf("%s: condition not met within %v", message, timeout)
		}
	}
}

// RunWithTimeout runs a function with a timeout
func RunWithTimeout(t *testing.T, timeout time.Duration, fn func(), message string) {
	t.Helper()

	done := make(chan struct{})

	go func() {
		defer close(done)
		fn()
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		t.Fatalf("%s: operation timed out after %v", message, timeout)
	}
}

// CreateTempFile creates a temporary file for testing
func CreateTempFile(t *testing.T, content string) string {
	t.Helper()

	file, err := os.CreateTemp("", "kubectl-broker-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Clean up after test
	t.Cleanup(func() {
		os.Remove(file.Name())
	})

	return file.Name()
}

// CreateTempDir creates a temporary directory for testing
func CreateTempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "kubectl-broker-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Clean up after test
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}
}

// SkipIfCI skips the test if running in CI environment
func SkipIfCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping in CI environment")
	}
}

// GetTestNamespaces returns test namespaces that look like UUIDs
func GetTestNamespaces() []string {
	return []string{
		"9141c41b-686e-42d8-8524-9876229d41ce",
		"9d3cb2fa-55f1-481c-b7b6-1bd4e5690056",
		"c7e3f207-0341-4e37-a951-e09665f35916",
	}
}

// GetTestPodNames returns typical HiveMQ broker pod names
func GetTestPodNames() []string {
	return []string{
		"broker-0",
		"broker-1",
	}
}

// GetTestStatefulSetName returns the standard HiveMQ StatefulSet name
func GetTestStatefulSetName() string {
	return "broker"
}

// MockTime provides predictable time values for testing
type MockTime struct {
	Current time.Time
}

// NewMockTime creates a new mock time helper
func NewMockTime(baseTime time.Time) *MockTime {
	return &MockTime{Current: baseTime}
}

// Now returns the current mocked time
func (m *MockTime) Now() time.Time {
	return m.Current
}

// Advance advances the mock time by the given duration
func (m *MockTime) Advance(d time.Duration) {
	m.Current = m.Current.Add(d)
}

// RFC3339 returns the current time in RFC3339 format
func (m *MockTime) RFC3339() string {
	return m.Current.Format(time.RFC3339)
}

// TestBackupID generates a test backup ID based on mock time
func (m *MockTime) TestBackupID() string {
	return m.Current.Format("20060102-150405")
}

// IsContextError checks if an error is a context-related error
func IsContextError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	contextErrors := []string{
		"context canceled",
		"context deadline exceeded",
		"operation was canceled",
	}

	for _, contextErr := range contextErrors {
		if strings.Contains(errMsg, contextErr) {
			return true
		}
	}

	return false
}
