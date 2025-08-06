package backup

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client represents an HTTP client for HiveMQ backup API operations
type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
}

// NewClient creates a new backup API client
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Default timeout for status checks
		},
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
	}
}

// SetTimeout configures the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// makeRequest performs an HTTP request with authentication if configured
func (c *Client) makeRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header if credentials are provided
	if c.username != "" && c.password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	// Set content type for POST requests
	if method == "POST" && body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// CreateBackup initiates a new backup operation
func (c *Client) CreateBackup() (*BackupResponse, error) {
	resp, err := c.makeRequest("POST", "/api/v1/management/backups", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	// Read the response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup response: %w", err)
	}

	var backupResp BackupResponse
	if err := json.Unmarshal(body, &backupResp); err != nil {
		return nil, fmt.Errorf("failed to decode backup response: %w. Response body: %s", err, string(body))
	}

	// Validate that we got a backup ID
	if backupResp.Backup.ID == "" {
		return nil, fmt.Errorf("backup created but no ID returned. Response body: %s", string(body))
	}

	return &backupResp, nil
}

// ListBackups retrieves all available backups
func (c *Client) ListBackups() (*BackupListResponse, error) {
	resp, err := c.makeRequest("GET", "/api/v1/management/backups", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var listResp BackupListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode backup list response: %w", err)
	}

	return &listResp, nil
}

// GetBackupStatus retrieves the status of a specific backup
func (c *Client) GetBackupStatus(backupID string) (*BackupStatusResponse, error) {
	path := fmt.Sprintf("/api/v1/management/backups/%s", backupID)
	resp, err := c.makeRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var wrapper struct {
		Backup BackupStatusResponse `json:"backup"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode backup status response: %w", err)
	}

	return &wrapper.Backup, nil
}

// DownloadBackup downloads a backup file and returns the response for streaming
func (c *Client) DownloadBackup(backupID string) (*http.Response, error) {
	// Set longer timeout for downloads
	originalTimeout := c.httpClient.Timeout
	c.httpClient.Timeout = 10 * time.Minute
	defer func() {
		c.httpClient.Timeout = originalTimeout
	}()

	// Try multiple potential download endpoints
	endpoints := []string{
		"/api/v1/management/backups/%s/file",
		"/api/v1/management/backups/%s/download", 
		"/api/v1/management/backups/%s/data",
	}

	var lastErr error
	for _, endpoint := range endpoints {
		path := fmt.Sprintf(endpoint, backupID)
		resp, err := c.makeRequest("GET", path, nil)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusOK {
			// Don't close the response body here - caller will handle it for streaming
			return resp, nil
		}

		// Close non-success response
		resp.Body.Close()
		lastErr = c.handleErrorResponse(resp)
		
		// If not 404, don't try other endpoints
		if resp.StatusCode != http.StatusNotFound {
			break
		}
	}

	// All endpoints failed
	return nil, fmt.Errorf("backup download not supported: all download endpoints returned 404. "+
		"This HiveMQ instance (version 4.x) may not have backup download functionality enabled or available. "+
		"You can create and list backups, but downloading them may not be supported in this configuration. "+
		"Last error: %w", lastErr)
}

// RestoreBackup initiates a restore operation
func (c *Client) RestoreBackup(backupID string) (*RestoreResponse, error) {
	reqBody := RestoreRequest{BackupID: backupID}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal restore request: %w", err)
	}

	resp, err := c.makeRequest("POST", "/api/v1/management/restores", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp)
	}

	var restoreResp RestoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&restoreResp); err != nil {
		return nil, fmt.Errorf("failed to decode restore response: %w", err)
	}

	return &restoreResp, nil
}

// handleErrorResponse processes error responses from the API
func (c *Client) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err != nil {
		// If we can't parse the error JSON, return the raw response
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if errorResp.Message != "" {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errorResp.Message)
	}

	if errorResp.Error != "" {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errorResp.Error)
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
}

// TestConnection tests if the HiveMQ management API is available
func (c *Client) TestConnection() error {
	// Test the backup endpoint specifically instead of the root management endpoint
	// Some HiveMQ instances don't expose /api/v1/management but do expose /api/v1/management/backups
	resp, err := c.makeRequest("GET", "/api/v1/management/backups", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to management API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("HiveMQ management API not found at %s/api/v1/management/backups. This HiveMQ instance may not have the management API enabled", c.baseURL)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("management API returned HTTP %d", resp.StatusCode)
	}

	return nil
}
