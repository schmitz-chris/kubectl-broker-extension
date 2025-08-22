package backup

import "time"

// BackupStatus represents the status of backup operations from HiveMQ
type BackupStatus string

const (
	StatusInProgress        BackupStatus = "IN_PROGRESS"
	StatusCompleted         BackupStatus = "COMPLETED"
	StatusFailed            BackupStatus = "FAILED"
	StatusRestoreInProgress BackupStatus = "RESTORE_IN_PROGRESS"
	StatusRestoreCompleted  BackupStatus = "RESTORE_COMPLETED"
	StatusRestoreFailed     BackupStatus = "RESTORE_FAILED"
)

// IsTerminal returns true if the backup status represents a final state
func (s BackupStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusRestoreCompleted || s == StatusRestoreFailed
}

// IsSuccess returns true if the backup operation completed successfully
func (s BackupStatus) IsSuccess() bool {
	return s == StatusCompleted || s == StatusRestoreCompleted
}

// BackupResponse represents the response when creating a backup
type BackupResponse struct {
	Backup BackupData `json:"backup"`
}

// BackupData represents the backup data within the response
type BackupData struct {
	ID        string       `json:"id"`
	CreatedAt time.Time    `json:"createdAt"`
	State     BackupStatus `json:"state"`
}

// BackupInfo represents full backup details from the HiveMQ API
type BackupInfo struct {
	ID        string       `json:"id"`
	Status    BackupStatus `json:"state"` // HiveMQ API uses "state" not "status"
	CreatedAt time.Time    `json:"createdAt"`
	Size      int64        `json:"bytes"` // HiveMQ API uses "bytes" not "size"
	Filename  string       `json:"filename,omitempty"`
}

// BackupListResponse represents the response when listing backups
type BackupListResponse struct {
	Items []BackupInfo `json:"items"`
}

// BackupStatusResponse represents the response when checking backup status
type BackupStatusResponse struct {
	ID        string       `json:"id"`
	Status    BackupStatus `json:"state"` // HiveMQ API uses "state" not "status"
	CreatedAt time.Time    `json:"createdAt"`
	Size      int64        `json:"bytes"` // HiveMQ API uses "bytes" not "size"
	Progress  int          `json:"progress,omitempty"`
	Message   string       `json:"message,omitempty"`
}

// RestoreRequest represents the request to restore a backup
type RestoreRequest struct {
	BackupID string `json:"backupId"`
}

// RestoreResponse represents the response when initiating a restore
type RestoreResponse struct {
	ID       string       `json:"id"`
	Status   BackupStatus `json:"status"`
	BackupID string       `json:"backupId"`
}

// ErrorResponse represents error responses from the HiveMQ API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// BackupOptions configures how backup operations are performed
type BackupOptions struct {
	Username     string        // optional authentication username
	Password     string        // optional authentication password
	OutputDir    string        // directory to save backup files
	OutputFile   string        // specific output filename override
	Timeout      time.Duration // timeout for backup operations
	PollInterval time.Duration // interval for status polling
	ShowProgress bool          // show progress indicators
	Destination  string        // local destination path for copying backup files from pods
}

// DefaultBackupOptions provides sensible defaults for backup operations
var DefaultBackupOptions = BackupOptions{
	Username:     "",
	Password:     "",
	OutputDir:    "./backups",
	OutputFile:   "",
	Timeout:      5 * time.Minute,
	PollInterval: 2 * time.Second,
	ShowProgress: true,
}
