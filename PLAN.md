# kubectl-broker Phase 7 Implementation Plan: HiveMQ Backup Management

## Rationale

The kubectl-broker tool currently provides comprehensive health monitoring for HiveMQ clusters running in Kubernetes. Users need backup management capabilities to protect their HiveMQ data and configurations. This implementation extends the existing tool with backup operations using the HiveMQ REST API.

### Why This Matters
- **Data Protection**: HiveMQ clusters contain critical MQTT broker state, retained messages, and configurations that need regular backups
- **Operational Safety**: Administrators need simple, reliable backup/restore capabilities integrated with their existing Kubernetes workflows
- **Consistency**: Backup operations should follow the same patterns as the existing health monitoring features

### Design Principles
- **Reuse Existing Infrastructure**: Leverage the established port-forwarding, HTTP client, and error handling patterns
- **Simple Defaults**: Follow the intelligent defaults pattern from the status command
- **Clear Feedback**: Provide progress indicators and clear status messages during long-running operations
- **Minimal Configuration**: No authentication by default, automatic discovery of API endpoints

## Implementation Plan

### Step 1: Package Structure Creation

Create a new package directory `pkg/backup/` with three main files following the established pattern from `pkg/health/`:

**1.1 Create `pkg/backup/types.go`**
- Define Go structs for all HiveMQ backup API request/response types
- Include structs for: BackupResponse (containing ID), BackupInfo (full backup details), BackupListResponse, BackupStatusResponse, RestoreRequest, RestoreResponse, ErrorResponse
- Define BackupStatus type as string with constants for all possible states: IN_PROGRESS, COMPLETED, FAILED, RESTORE_IN_PROGRESS, RESTORE_COMPLETED, RESTORE_FAILED
- Add helper methods on BackupStatus: IsTerminal() to check if status is final, IsSuccess() to check if operation succeeded
- Include proper JSON tags matching the HiveMQ API specification

**1.2 Create `pkg/backup/client.go`**
- Implement Client struct containing HTTP client, base URL, and optional authentication credentials
- Add NewClient constructor that accepts base URL and optional username/password
- Implement HTTP request helper method that adds authentication headers when credentials are provided
- Include proper error wrapping for all HTTP operations
- Set appropriate timeouts for long-running operations (5 minutes for backup creation/restore, 30 seconds for status checks)

**1.3 Create `pkg/backup/operations.go`**
- Implement high-level backup operations that will be called from the CLI commands
- Each operation should handle its own port-forwarding setup and teardown
- Include proper progress feedback mechanisms for long-running operations

### Step 2: Port Discovery Enhancement

**2.1 Extend `pkg/k8s/discovery.go`**
- Add DiscoverAPIPort function following the pattern of existing DiscoverHealthPort
- Target port 8081 with port name "api" as primary strategy
- Include fallback to port number 8081 if named port not found
- Return error with helpful message if API port cannot be discovered

### Step 3: Backup Operations Implementation

**3.1 Create Backup Operation**
- Implement CreateBackup function in operations.go
- Function should accept: k8s client config, pod name, namespace, optional auth credentials
- Establish port-forward to target pod's API port
- Send POST request to /api/v1/management/backups endpoint
- Extract backup ID from response
- Implement polling loop to check status until terminal state reached
- Use exponential backoff for polling: start at 2 seconds, max 30 seconds
- Return backup ID and final status

**3.2 List Backups Operation**
- Implement ListBackups function
- Establish port-forward and query GET /api/v1/management/backups
- Parse response into BackupListResponse struct
- Sort backups by timestamp (newest first)
- Calculate human-readable sizes (bytes to KB/MB/GB)
- Return structured list ready for display

**3.3 Download Backup Operation**
- Implement DownloadBackup function accepting backup ID and output options
- Query backup list if ID not provided to enable interactive selection
- Send GET request to /api/v1/management/backups/{id}/file
- Extract filename from Content-Disposition header
- Fall back to generated filename if header missing: backup-{id}-{timestamp}.tar.gz
- Handle --output flag for custom filename override
- Handle --output-dir flag for custom directory
- Stream response body to file with progress indication
- Return saved file path

**3.4 Check Status Operation**
- Implement GetBackupStatus function
- Query GET /api/v1/management/backups/{id}/status
- Support special "latest" ID by first listing backups and selecting most recent
- Return status information for display

### Step 4: CLI Command Enhancement

**4.1 Update `cmd/kubectl-broker/backup.go`**
- Transform existing placeholder into full subcommand with multiple operations
- Add persistent flags for authentication: --username and --password (both optional)
- Add persistent flags for targeting: --statefulset, --namespace (with intelligent defaults)
- Create subcommands: create, list, download, status

**4.2 Implement 'backup create' Subcommand**
- Display disk space warning before operation starts
- Show spinner with elapsed time during backup creation
- Poll status until completion
- Display final results: backup ID, size, status
- Handle errors with actionable guidance

**4.3 Implement 'backup list' Subcommand**
- Retrieve and display all backups in tabular format
- Columns: Backup ID (truncated to 16 chars), Size (human-readable), Created (formatted timestamp), Status
- Include summary line showing total backups and combined size
- Use fatih/color for status coloring matching health command patterns

**4.4 Implement 'backup download' Subcommand**
- Add flags: --id, --output, --output-dir, --latest
- If no ID provided and multiple backups exist, show numbered interactive menu
- Display backup size before starting download
- Show progress bar during download using existing progress indicator patterns
- Display success message with saved file location

**4.5 Implement 'backup status' Subcommand**
- Add flags: --id, --latest
- Query and display current status of specified backup
- Support checking restore operation status as well
- Use color coding for different states

### Step 5: Integration Points

**5.1 Intelligent Defaults Integration**
- Reuse namespace detection from kubeconfig context
- Default StatefulSet name to "broker"
- Show visual indicators (🎯) when defaults are applied
- Follow same pattern as status command

**5.2 Error Handling Integration**
- Wrap all errors with context following existing patterns
- Provide actionable error messages for common failures
- Include suggestions for resolution (e.g., checking permissions, connectivity)
- Special handling for authentication errors with helpful guidance

**5.3 Progress Indication**
- Use existing spinner implementation for indeterminate progress
- Implement progress bar for downloads showing percentage and size
- Include elapsed time display for long operations
- Ensure clean terminal output when operations complete

### Step 6: Testing and Documentation

**6.1 Update Test Infrastructure**
- Add test cases for backup client HTTP operations
- Include mock HiveMQ API responses for testing
- Test error conditions: auth failures, network issues, invalid backup IDs
- Verify Content-Disposition header parsing

**6.2 Update Documentation**
- Extend README.md with backup command examples
- Include common use cases and workflows
- Document authentication setup if needed
- Add troubleshooting section for backup operations

**6.3 Update CLAUDE.md**
- Document the new backup package structure
- Include details about Phase 7 completion
- Add examples of new commands
- Note the design decisions about disk space warnings and file naming

### Step 7: Future Considerations (Not in Initial Implementation)

Document but do not implement:
- Restore operations using POST /api/v1/management/restores
- Delete operations using DELETE /api/v1/management/backups/{id}
- Backup retention policies and automated cleanup
- Scheduled backup creation
- Backup verification and integrity checking

This incremental approach delivers core backup functionality while maintaining consistency with the existing codebase patterns and user experience principles.