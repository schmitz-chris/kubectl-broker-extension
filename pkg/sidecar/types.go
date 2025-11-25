package sidecar

import "time"

// BackupState mirrors the sidecar backup status strings.
type BackupState string

const (
	BackupStatePending   BackupState = "pending"
	BackupStateUploading BackupState = "uploading"
	BackupStateCompleted BackupState = "uploaded"
	BackupStateFailed    BackupState = "failed"
)

// Inventory represents the payload returned by GET /v1/backup/list.
type Inventory struct {
	ClusterBackups []ClusterBackupInfo `json:"cluster_backups"`
	Backups        []BackupInfo        `json:"backups"`
}

// ClusterBackupInfo describes a directory under the cluster backup path.
type ClusterBackupInfo struct {
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	SizeBytes    int64       `json:"size_bytes"`
	LastModified time.Time   `json:"last_modified"`
	Status       BackupState `json:"status"`
	Error        string      `json:"error,omitempty"`
}

// BackupInfo describes a HiveMQ backup folder in BACKUP_DIR.
type BackupInfo struct {
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	SizeBytes    int64       `json:"size_bytes"`
	LastModified time.Time   `json:"last_modified"`
	Status       BackupState `json:"status"`
	Error        string      `json:"error,omitempty"`
}

// RemoteBackupInfo represents a backup object stored in S3.
type RemoteBackupInfo struct {
	Key          string    `json:"key"`
	LastModified time.Time `json:"last_modified"`
	SizeBytes    int64     `json:"size_bytes"`
}

// RestoreRequest mirrors the /v1/restore payload.
type RestoreRequest struct {
	Version string `json:"version,omitempty"`
	DryRun  bool   `json:"dry_run,omitempty"`
}

// RestoreResult captures the restore response.
type RestoreResult struct {
	Key         string    `json:"key"`
	Bytes       int64     `json:"bytes"`
	TargetPath  string    `json:"target_path"`
	DryRun      bool      `json:"dry_run"`
	LastChecked time.Time `json:"last_checked"`
}

// UploadRequest maps to POST /v1/backup/upload.
type UploadRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// PurgeRequest maps to POST /v1/backup/purge.
type PurgeRequest struct {
	Name string `json:"name"`
}
