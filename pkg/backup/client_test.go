package backup

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	baseURL := "http://localhost:8081"
	username := "admin"
	password := "secret"

	client := NewClient(baseURL, username, password)

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.baseURL != baseURL {
		t.Errorf("Expected baseURL %q, got %q", baseURL, client.baseURL)
	}

	if client.username != username {
		t.Errorf("Expected username %q, got %q", username, client.username)
	}

	if client.password != password {
		t.Errorf("Expected password %q, got %q", password, client.password)
	}

	// Test URL trimming
	clientWithSlash := NewClient("http://localhost:8081/", username, password)
	if clientWithSlash.baseURL != baseURL {
		t.Errorf("Expected trimmed baseURL %q, got %q", baseURL, clientWithSlash.baseURL)
	}
}

func TestSetTimeout(t *testing.T) {
	client := NewClient("http://localhost:8081", "", "")

	customTimeout := 45 * time.Second
	client.SetTimeout(customTimeout)

	if client.httpClient.Timeout != customTimeout {
		t.Errorf("Expected timeout %v, got %v", customTimeout, client.httpClient.Timeout)
	}
}

func TestCreateBackup(t *testing.T) {
	testData, err := loadBackupTestData()
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	tests := []struct {
		name          string
		responseCode  int
		responseBody  string
		expectedID    string
		expectedState string
		shouldError   bool
	}{
		{
			name:          "successful creation",
			responseCode:  200,
			responseBody:  testData.BackupCreateResponse,
			expectedID:    "20250905-130215",
			expectedState: "RUNNING",
			shouldError:   false,
		},
		{
			name:          "server error",
			responseCode:  500,
			responseBody:  `{"error": "Internal server error"}`,
			expectedID:    "",
			expectedState: "",
			shouldError:   true,
		},
		{
			name:          "invalid JSON response",
			responseCode:  200,
			responseBody:  `{invalid json}`,
			expectedID:    "",
			expectedState: "",
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check request method and path
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/management/backups" {
					t.Errorf("Expected path /api/v1/management/backups, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.responseCode)
				fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(server.URL, "", "")
			backup, err := client.CreateBackup()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if backup.Backup.ID != tt.expectedID {
				t.Errorf("Expected backup ID %q, got %q", tt.expectedID, backup.Backup.ID)
			}

			if string(backup.Backup.State) != tt.expectedState {
				t.Errorf("Expected backup state %q, got %q", tt.expectedState, backup.Backup.State)
			}
		})
	}
}

func TestGetBackupStatus(t *testing.T) {
	testData, err := loadBackupTestData()
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	tests := []struct {
		name          string
		backupID      string
		responseCode  int
		responseBody  string
		expectedState string
		expectedBytes int64
		shouldError   bool
	}{
		{
			name:          "running backup",
			backupID:      "20250905-130215",
			responseCode:  200,
			responseBody:  testData.BackupStatusRunning,
			expectedState: "RUNNING",
			expectedBytes: 512,
			shouldError:   false,
		},
		{
			name:          "completed backup",
			backupID:      "20250905-130215",
			responseCode:  200,
			responseBody:  testData.BackupStatusCompleted,
			expectedState: "COMPLETED",
			expectedBytes: 1024,
			shouldError:   false,
		},
		{
			name:          "failed backup",
			backupID:      "20250905-130215",
			responseCode:  200,
			responseBody:  testData.BackupStatusFailed,
			expectedState: "FAILED",
			expectedBytes: 0,
			shouldError:   false,
		},
		{
			name:          "backup not found",
			backupID:      "nonexistent",
			responseCode:  404,
			responseBody:  `{"error": "Backup not found"}`,
			expectedState: "",
			expectedBytes: 0,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check request method and path
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				expectedPath := fmt.Sprintf("/api/v1/management/backups/%s", tt.backupID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.responseCode)
				fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(server.URL, "", "")
			backup, err := client.GetBackupStatus(tt.backupID)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if string(backup.Status) != tt.expectedState {
				t.Errorf("Expected backup state %q, got %q", tt.expectedState, backup.Status)
			}

			if backup.Size != tt.expectedBytes {
				t.Errorf("Expected backup size %d, got %d", tt.expectedBytes, backup.Size)
			}
		})
	}
}

func TestListBackups(t *testing.T) {
	testData, err := loadBackupTestData()
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	tests := []struct {
		name          string
		responseCode  int
		responseBody  string
		expectedCount int
		shouldError   bool
	}{
		{
			name:          "list with backups",
			responseCode:  200,
			responseBody:  testData.BackupListResponse,
			expectedCount: 3,
			shouldError:   false,
		},
		{
			name:          "empty list",
			responseCode:  200,
			responseBody:  testData.BackupListEmpty,
			expectedCount: 0,
			shouldError:   false,
		},
		{
			name:          "server error",
			responseCode:  500,
			responseBody:  `{"error": "Internal server error"}`,
			expectedCount: 0,
			shouldError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/management/backups" {
					t.Errorf("Expected path /api/v1/management/backups, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.responseCode)
				fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(server.URL, "", "")
			backups, err := client.ListBackups()

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(backups.Items) != tt.expectedCount {
				t.Errorf("Expected %d backups, got %d", tt.expectedCount, len(backups.Items))
			}

			// Check first backup details if available
			if tt.expectedCount > 0 && len(backups.Items) > 0 {
				first := backups.Items[0]
				if first.ID == "" {
					t.Error("First backup should have an ID")
				}
				if first.Status == "" {
					t.Error("First backup should have a state")
				}
			}
		})
	}
}

func TestAuthenticationHeaders(t *testing.T) {
	username := "admin"
	password := "secret"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("Expected Authorization header but got none")
			w.WriteHeader(401)
			return
		}

		if !strings.HasPrefix(auth, "Basic ") {
			t.Errorf("Expected Basic auth, got %s", auth)
			w.WriteHeader(401)
			return
		}

		// Return successful response
		w.WriteHeader(200)
		fmt.Fprint(w, `{"backup": {"id": "test", "state": "RUNNING", "created": "2025-09-05T13:02:15Z", "bytes": 0}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, username, password)
	_, err := client.CreateBackup()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestNoAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that no Authorization header is present
		auth := r.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("Expected no Authorization header, got %s", auth)
		}

		w.WriteHeader(200)
		fmt.Fprint(w, `{"backup": {"id": "test", "state": "RUNNING", "created": "2025-09-05T13:02:15Z", "bytes": 0}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")
	_, err := client.CreateBackup()

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestDownloadBackup(t *testing.T) {
	testContent := "test backup content"

	tests := []struct {
		name         string
		backupID     string
		responseCode int
		responseBody string
		shouldError  bool
	}{
		{
			name:         "successful download",
			backupID:     "20250905-130215",
			responseCode: 200,
			responseBody: testContent,
			shouldError:  false,
		},
		{
			name:         "backup not found",
			backupID:     "nonexistent",
			responseCode: 404,
			responseBody: `{"error": "Backup not found"}`,
			shouldError:  true,
		},
		{
			name:         "download not supported",
			backupID:     "20250905-130215",
			responseCode: 404,
			responseBody: `{"errors":[{"title":"Resource not found"}]}`,
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				// The client tries multiple endpoints, accept any of them
				validPaths := []string{
					fmt.Sprintf("/api/v1/management/files/backups/%s", tt.backupID),
					fmt.Sprintf("/api/v1/management/backups/%s/file", tt.backupID),
					fmt.Sprintf("/api/v1/management/backups/%s/download", tt.backupID),
					fmt.Sprintf("/api/v1/management/backups/%s/data", tt.backupID),
				}

				pathValid := false
				for _, validPath := range validPaths {
					if r.URL.Path == validPath {
						pathValid = true
						break
					}
				}

				if !pathValid {
					t.Errorf("Expected one of %v, got %s", validPaths, r.URL.Path)
				}

				w.WriteHeader(tt.responseCode)
				fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(server.URL, "", "")
			reader, err := client.DownloadBackup(tt.backupID)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if reader == nil {
				t.Fatal("Expected reader but got nil")
			}

			// Read the content and verify
			content, err := io.ReadAll(reader.Body)
			if err != nil {
				t.Fatalf("Failed to read response: %v", err)
			}
			reader.Body.Close()

			if string(content) != testContent {
				t.Errorf("Expected content %q, got %q", testContent, string(content))
			}
		})
	}
}

func TestClientTimeout(t *testing.T) {
	// Create a server that responds slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(200)
		fmt.Fprint(w, `{"backup": {"id": "test", "state": "RUNNING", "created": "2025-09-05T13:02:15Z", "bytes": 0}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")
	client.SetTimeout(10 * time.Millisecond) // Very short timeout

	_, err := client.CreateBackup()

	if err == nil {
		t.Error("Expected timeout error but got none")
	}

	// Check that it's a timeout error
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected timeout error, got %v", err)
	}
}

func TestMakeRequestErrorHandling(t *testing.T) {
	client := NewClient("http://invalid-url-that-does-not-exist", "", "")

	_, err := client.CreateBackup()
	if err == nil {
		t.Error("Expected error for invalid URL but got none")
	}
}

// loadBackupTestData loads backup test data from testdata directory
func loadBackupTestData() (*BackupTestData, error) {
	path := filepath.Join("..", "..", "testdata", "backup_responses.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, err
	}

	// Convert RawMessage to string for each field
	result := &BackupTestData{}
	if val, ok := rawData["backup_create_response"]; ok {
		result.BackupCreateResponse = string(val)
	}
	if val, ok := rawData["backup_status_running"]; ok {
		result.BackupStatusRunning = string(val)
	}
	if val, ok := rawData["backup_status_completed"]; ok {
		result.BackupStatusCompleted = string(val)
	}
	if val, ok := rawData["backup_status_failed"]; ok {
		result.BackupStatusFailed = string(val)
	}
	if val, ok := rawData["backup_list_response"]; ok {
		result.BackupListResponse = string(val)
	}
	if val, ok := rawData["backup_list_empty"]; ok {
		result.BackupListEmpty = string(val)
	}

	return result, nil
}

// BackupTestData holds test response data for backup operations
type BackupTestData struct {
	BackupCreateResponse  string
	BackupStatusRunning   string
	BackupStatusCompleted string
	BackupStatusFailed    string
	BackupListResponse    string
	BackupListEmpty       string
}

// BenchmarkCreateBackup benchmarks backup creation
func BenchmarkCreateBackup(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"backup": {"id": "test", "state": "RUNNING", "created": "2025-09-05T13:02:15Z", "bytes": 0}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.CreateBackup()
		if err != nil {
			b.Fatalf("Create backup error: %v", err)
		}
	}
}

// BenchmarkListBackups benchmarks backup listing
func BenchmarkListBackups(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, `{"items": []}`)
	}))
	defer server.Close()

	client := NewClient(server.URL, "", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.ListBackups()
		if err != nil {
			b.Fatalf("List backups error: %v", err)
		}
	}
}
