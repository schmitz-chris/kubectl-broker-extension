package sidecar

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListRemoteBackups(t *testing.T) {
	t.Parallel()

	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != remoteListPath {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"backups": []RemoteBackupInfo{
				{
					Key:          "snapshot.backup",
					LastModified: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
					SizeBytes:    42,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, ClientOptions{})
	items, err := client.ListRemoteBackups(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListRemoteBackups returned error: %v", err)
	}
	if capturedQuery != "limit=5" {
		t.Fatalf("expected query limit=5, got %q", capturedQuery)
	}
	if len(items) != 1 || items[0].Key != "snapshot.backup" {
		t.Fatalf("unexpected response: %+v", items)
	}
}

func TestRestoreAcceptsAcceptedStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"version":"latest","dry_run":true}` {
			t.Fatalf("unexpected body: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(RestoreResult{
			Key:        "ns/backup/latest.backup",
			Bytes:      1024,
			TargetPath: "/data/restore.zip",
			DryRun:     true,
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, ClientOptions{})
	result, err := client.Restore(context.Background(), RestoreRequest{
		Version: "latest",
		DryRun:  true,
	})
	if err != nil {
		t.Fatalf("Restore returned error: %v", err)
	}
	if result == nil || result.Key != "ns/backup/latest.backup" || !result.DryRun {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestErrorFromResponseIncludesBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL, ClientOptions{})
	err := client.PurgeBackup(context.Background(), "foo")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" || !containsAll(got, "400", "boom") {
		t.Fatalf("unexpected error text: %s", got)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

func TestListInventory(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != localListPath {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Inventory{
			Backups: []BackupInfo{
				{
					Name:         "backup-001",
					Path:         "/data/backup-001",
					SizeBytes:    1024,
					LastModified: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					Status:       BackupStateCompleted,
				},
			},
			ClusterBackups: []ClusterBackupInfo{
				{
					Name:         "cluster-001",
					Path:         "/data/cluster-001",
					SizeBytes:    2048,
					LastModified: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
					Status:       BackupStateUploading,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, ClientOptions{})
	inventory, err := client.ListInventory(context.Background())
	if err != nil {
		t.Fatalf("ListInventory returned error: %v", err)
	}
	if len(inventory.Backups) != 1 || inventory.Backups[0].Name != "backup-001" {
		t.Fatalf("unexpected backups: %+v", inventory.Backups)
	}
	if len(inventory.ClusterBackups) != 1 || inventory.ClusterBackups[0].Name != "cluster-001" {
		t.Fatalf("unexpected cluster backups: %+v", inventory.ClusterBackups)
	}
}

func TestTriggerUpload(t *testing.T) {
	t.Parallel()

	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != forceUploadPath {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, ClientOptions{})
	err := client.TriggerUpload(context.Background(), UploadRequest{
		Type: "backup",
		Name: "test-backup",
	})
	if err != nil {
		t.Fatalf("TriggerUpload returned error: %v", err)
	}
	if !strings.Contains(receivedBody, "test-backup") {
		t.Fatalf("expected body to contain test-backup, got: %s", receivedBody)
	}
}

func TestTriggerUploadRequiresName(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server")
	}))
	defer server.Close()

	client := NewClient(server.URL, ClientOptions{})
	err := client.TriggerUpload(context.Background(), UploadRequest{
		Type: "backup",
		Name: "",
	})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected name validation error, got: %v", err)
	}
}

func TestFetchMetrics(t *testing.T) {
	t.Parallel()

	expectedMetrics := "# HELP backup_count Total backups\nbackup_count 42"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != metricsPath {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(expectedMetrics))
	}))
	defer server.Close()

	client := NewClient(server.URL, ClientOptions{})
	metrics, err := client.FetchMetrics(context.Background())
	if err != nil {
		t.Fatalf("FetchMetrics returned error: %v", err)
	}
	if string(metrics) != expectedMetrics {
		t.Fatalf("unexpected metrics: %s", string(metrics))
	}
}

func TestBearerTokenAuthentication(t *testing.T) {
	t.Parallel()

	var capturedAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get(authHeader)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Inventory{})
	}))
	defer server.Close()

	client := NewClient(server.URL, ClientOptions{
		APIToken: "test-token-123",
	})
	_, err := client.ListInventory(context.Background())
	if err != nil {
		t.Fatalf("ListInventory returned error: %v", err)
	}
	expectedHeader := "Bearer test-token-123"
	if capturedAuthHeader != expectedHeader {
		t.Fatalf("expected Authorization header %q, got %q", expectedHeader, capturedAuthHeader)
	}
}

func TestPurgeBackupRequiresName(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach server")
	}))
	defer server.Close()

	client := NewClient(server.URL, ClientOptions{})
	err := client.PurgeBackup(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected name validation error, got: %v", err)
	}
}
