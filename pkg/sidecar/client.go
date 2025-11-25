package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout    = 30 * time.Second
	restoreEndpoint   = "/v1/restore"
	localListPath     = "/v1/backup/list"
	remoteListPath    = "/v1/backup/list-remote"
	purgePath         = "/v1/backup/purge"
	forceUploadPath   = "/v1/backup/upload"
	metricsPath       = "/metrics"
	authHeader        = "Authorization"
	bearerTokenPrefix = "Bearer "
)

// ClientOptions configure the HTTP client.
type ClientOptions struct {
	Timeout  time.Duration
	APIToken string
}

// Client wraps HTTP operations against the sidecar API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiToken   string
}

// NewClient builds a Client for the provided base URL.
func NewClient(baseURL string, opts ClientOptions) *Client {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiToken: strings.TrimSpace(opts.APIToken),
	}
}

// ListInventory returns local backup and cluster inventory.
func (c *Client) ListInventory(ctx context.Context) (Inventory, error) {
	var out Inventory
	if err := c.getJSON(ctx, localListPath, &out, nil); err != nil {
		return Inventory{}, err
	}
	return out, nil
}

// ListRemoteBackups returns remote backups already uploaded to object storage.
func (c *Client) ListRemoteBackups(ctx context.Context, limit int) ([]RemoteBackupInfo, error) {
	var query url.Values
	if limit > 0 {
		query = url.Values{}
		query.Set("limit", strconv.Itoa(limit))
	}
	var resp struct {
		Backups []RemoteBackupInfo `json:"backups"`
	}
	if err := c.getJSON(ctx, remoteListPath, &resp, query); err != nil {
		return nil, err
	}
	return resp.Backups, nil
}

// Restore triggers POST /v1/restore.
func (c *Client) Restore(ctx context.Context, req RestoreRequest) (*RestoreResult, error) {
	var result RestoreResult
	if err := c.postJSON(ctx, restoreEndpoint, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// PurgeBackup deletes a backup directory once confirmed safe.
func (c *Client) PurgeBackup(ctx context.Context, name string) error {
	payload := PurgeRequest{Name: strings.TrimSpace(name)}
	if payload.Name == "" {
		return fmt.Errorf("purge name is required")
	}
	return c.postJSON(ctx, purgePath, payload, nil)
}

// TriggerUpload forces an upload outside of the watch cadence.
func (c *Client) TriggerUpload(ctx context.Context, req UploadRequest) error {
	req.Type = strings.TrimSpace(req.Type)
	req.Name = strings.TrimSpace(req.Name)
	if req.Type == "" {
		req.Type = "backup"
	}
	if req.Name == "" {
		return fmt.Errorf("upload name is required")
	}
	return c.postJSON(ctx, forceUploadPath, req, nil)
}

// FetchMetrics retrieves the Prometheus metrics payload.
func (c *Client) FetchMetrics(ctx context.Context) ([]byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, metricsPath, nil, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.errorFromResponse(resp)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) getJSON(ctx context.Context, p string, dest any, query url.Values) error {
	req, err := c.newRequest(ctx, http.MethodGet, p, nil, query)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.errorFromResponse(resp)
	}
	if dest == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) postJSON(ctx context.Context, p string, payload any, dest any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}
	req, err := c.newRequest(ctx, http.MethodPost, p, bytes.NewReader(body), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return c.errorFromResponse(resp)
	}

	if dest == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) newRequest(ctx context.Context, method, p string, body io.Reader, query url.Values) (*http.Request, error) {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url %q: %w", c.baseURL, err)
	}
	u.Path = path.Join(u.Path, p)

	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if c.apiToken != "" {
		req.Header.Set(authHeader, bearerTokenPrefix+c.apiToken)
	}
	return req, nil
}

func (c *Client) errorFromResponse(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if len(body) == 0 {
		return fmt.Errorf("sidecar API returned %s", resp.Status)
	}
	return fmt.Errorf("sidecar API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
}
