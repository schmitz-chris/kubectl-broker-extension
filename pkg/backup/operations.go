package backup

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	v1 "k8s.io/api/core/v1"
	"kubectl-broker/pkg"
)

// CreateBackup performs a backup operation using the API service with progress feedback
func CreateBackup(ctx context.Context, k8sClient *pkg.K8sClient, service *v1.Service, options BackupOptions) (*BackupInfo, error) {
	// Skip verbose service details

	// Discover the API port for the service
	apiPort, err := k8sClient.DiscoverServiceAPIPort(service)
	if err != nil {
		return nil, fmt.Errorf("failed to discover API port: %w", err)
	}

	// Get a random local port for port-forwarding
	localPort, err := pkg.GetRandomPort()
	if err != nil {
		return nil, fmt.Errorf("failed to get random port: %w", err)
	}

	// Set up port forwarding
	pf := pkg.NewPortForwarder(k8sClient.GetConfig(), k8sClient.GetRESTClient())

	// Create base URL for the backup API
	baseURL := fmt.Sprintf("http://localhost:%d", localPort)
	client := NewClient(baseURL, options.Username, options.Password)
	client.SetTimeout(options.Timeout)

	var finalBackupInfo *BackupInfo

	// Use service port forwarding for backup operations
	err = pf.PerformWithServicePortForwarding(ctx, k8sClient, service, apiPort, localPort, func(localPort int) error {
		// Test connection first
		if err := client.TestConnection(); err != nil {
			return fmt.Errorf("management API connection failed: %w", err)
		}

		backupResp, err := client.CreateBackup()
		if err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}

		if options.ShowProgress {
			fmt.Printf("Backup created: %s\n", backupResp.Backup.ID)
			fmt.Printf("Waiting for completion...")
		}

		// Poll for completion
		if err := waitForBackupCompletion(client, backupResp.Backup.ID, options); err != nil {
			return err
		}

		// Get final backup info
		status, err := client.GetBackupStatus(backupResp.Backup.ID)
		if err != nil {
			return fmt.Errorf("failed to get final backup status: %w", err)
		}

		finalBackupInfo = &BackupInfo{
			ID:        status.ID,
			Status:    status.Status,
			CreatedAt: status.CreatedAt,
			Size:      status.Size,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return finalBackupInfo, nil
}

// ListBackups retrieves and formats all available backups using the API service
func ListBackups(ctx context.Context, k8sClient *pkg.K8sClient, service *v1.Service, options BackupOptions) ([]BackupInfo, error) {
	// Discover the API port for the service
	apiPort, err := k8sClient.DiscoverServiceAPIPort(service)
	if err != nil {
		return nil, fmt.Errorf("failed to discover API port: %w", err)
	}

	// Get a random local port for port-forwarding
	localPort, err := pkg.GetRandomPort()
	if err != nil {
		return nil, fmt.Errorf("failed to get random port: %w", err)
	}

	// Set up port forwarding
	pf := pkg.NewPortForwarder(k8sClient.GetConfig(), k8sClient.GetRESTClient())

	// Create base URL for the backup API
	baseURL := fmt.Sprintf("http://localhost:%d", localPort)
	client := NewClient(baseURL, options.Username, options.Password)

	var backups []BackupInfo

	// Use service port forwarding for backup operations
	err = pf.PerformWithServicePortForwarding(ctx, k8sClient, service, apiPort, localPort, func(localPort int) error {
		// List backups
		listResp, err := client.ListBackups()
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}

		// Sort backups by creation time (newest first)
		sort.Slice(listResp.Items, func(i, j int) bool {
			return listResp.Items[i].CreatedAt.After(listResp.Items[j].CreatedAt)
		})

		backups = listResp.Items
		return nil
	})

	if err != nil {
		return nil, err
	}

	return backups, nil
}

// DownloadBackup downloads a backup file to the specified location using the API service
func DownloadBackup(ctx context.Context, k8sClient *pkg.K8sClient, service *v1.Service, backupID string, options BackupOptions) (string, error) {
	if options.ShowProgress {
		fmt.Printf("Downloading backup %s using service %s...\n", backupID, service.Name)
	}

	// Discover the API port for the service
	apiPort, err := k8sClient.DiscoverServiceAPIPort(service)
	if err != nil {
		return "", fmt.Errorf("failed to discover API port: %w", err)
	}

	// Get a random local port for port-forwarding
	localPort, err := pkg.GetRandomPort()
	if err != nil {
		return "", fmt.Errorf("failed to get random port: %w", err)
	}

	// Set up port forwarding
	pf := pkg.NewPortForwarder(k8sClient.GetConfig(), k8sClient.GetRESTClient())

	// Create base URL for the backup API
	baseURL := fmt.Sprintf("http://localhost:%d", localPort)
	client := NewClient(baseURL, options.Username, options.Password)

	var savedPath string

	// Use service port forwarding for backup operations
	err = pf.PerformWithServicePortForwarding(ctx, k8sClient, service, apiPort, localPort, func(localPort int) error {
		// Download backup
		resp, err := client.DownloadBackup(backupID)
		if err != nil {
			return fmt.Errorf("failed to download backup: %w", err)
		}
		defer resp.Body.Close()

		// Determine filename
		filename := extractFilenameFromResponse(resp, backupID)
		if options.OutputFile != "" {
			filename = options.OutputFile
		}

		// Ensure output directory exists
		outputDir := options.OutputDir
		if outputDir == "" {
			outputDir = "./backups"
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		savedPath = filepath.Join(outputDir, filename)

		// Create output file
		file, err := os.Create(savedPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()

		// Stream response to file with progress indication
		if options.ShowProgress {
			contentLength := resp.ContentLength
			if contentLength > 0 {
				return copyWithProgress(file, resp.Body, contentLength, filename)
			}
		}

		_, err = io.Copy(file, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to save backup file: %w", err)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return savedPath, nil
}

// GetBackupStatus retrieves the current status of a backup operation using the API service
func GetBackupStatus(ctx context.Context, k8sClient *pkg.K8sClient, service *v1.Service, backupID string, options BackupOptions) (*BackupStatusResponse, error) {
	// Handle "latest" backup ID
	if backupID == "latest" {
		backups, err := ListBackups(ctx, k8sClient, service, options)
		if err != nil {
			return nil, fmt.Errorf("failed to list backups to find latest: %w", err)
		}
		if len(backups) == 0 {
			return nil, fmt.Errorf("no backups found")
		}
		backupID = backups[0].ID // Already sorted newest first
	}

	// Discover the API port for the service
	apiPort, err := k8sClient.DiscoverServiceAPIPort(service)
	if err != nil {
		return nil, fmt.Errorf("failed to discover API port: %w", err)
	}

	// Get a random local port for port-forwarding
	localPort, err := pkg.GetRandomPort()
	if err != nil {
		return nil, fmt.Errorf("failed to get random port: %w", err)
	}

	// Set up port forwarding
	pf := pkg.NewPortForwarder(k8sClient.GetConfig(), k8sClient.GetRESTClient())

	// Create base URL for the backup API
	baseURL := fmt.Sprintf("http://localhost:%d", localPort)
	client := NewClient(baseURL, options.Username, options.Password)

	var status *BackupStatusResponse

	// Use service port forwarding for backup operations
	err = pf.PerformWithServicePortForwarding(ctx, k8sClient, service, apiPort, localPort, func(localPort int) error {
		// Get backup status
		statusResp, err := client.GetBackupStatus(backupID)
		if err != nil {
			return fmt.Errorf("failed to get backup status: %w", err)
		}

		status = statusResp
		return nil
	})

	if err != nil {
		return nil, err
	}

	return status, nil
}

// waitForBackupCompletion polls the backup status until completion
func waitForBackupCompletion(client *Client, backupID string, options BackupOptions) error {
	for {
		status, err := client.GetBackupStatus(backupID)
		if err != nil {
			return fmt.Errorf("failed to check backup status: %w", err)
		}

		if options.ShowProgress {
			if status.Progress > 0 {
				fmt.Printf(" %d%%", status.Progress)
			} else {
				fmt.Printf(".")
			}
		}

		if status.Status.IsTerminal() {
			if status.Status.IsSuccess() {
				if options.ShowProgress {
					fmt.Printf(" done\n\n")
				}
				return nil
			} else {
				return fmt.Errorf("backup failed with status: %s", status.Status)
			}
		}

		time.Sleep(options.PollInterval)
	}
}

// extractFilenameFromResponse extracts filename from Content-Disposition header or generates one
func extractFilenameFromResponse(resp *http.Response, backupID string) string {
	contentDisp := resp.Header.Get("Content-Disposition")
	if contentDisp != "" {
		// Parse Content-Disposition header for filename
		parts := strings.Split(contentDisp, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "filename=") {
				filename := strings.Trim(part[9:], `"`)
				if filename != "" {
					return filename
				}
			}
		}
	}

	// Fallback to generated filename
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("backup-%s-%s.tar.gz", backupID[:8], timestamp)
}

// copyWithProgress copies data with progress indication
func copyWithProgress(dst io.Writer, src io.Reader, contentLength int64, filename string) error {
	buf := make([]byte, 32*1024)
	var written int64

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return ew
			}
			if nr != nw {
				return io.ErrShortWrite
			}

			// Show progress
			if contentLength > 0 {
				percent := float64(written) / float64(contentLength) * 100
				fmt.Printf("\rDownloading %s: %.1f%% (%s/%s)",
					filename,
					percent,
					formatBytes(written),
					formatBytes(contentLength))
			}
		}
		if er != nil {
			if er != io.EOF {
				return er
			}
			break
		}
	}
	fmt.Println() // New line after progress
	return nil
}

// getStatusColor returns a color function for the given backup status
func getStatusColor(status BackupStatus) *color.Color {
	switch status {
	case StatusCompleted, StatusRestoreCompleted:
		return color.New(color.FgGreen, color.Bold)
	case StatusFailed, StatusRestoreFailed:
		return color.New(color.FgRed, color.Bold)
	case StatusInProgress, StatusRestoreInProgress:
		return color.New(color.FgYellow, color.Bold)
	default:
		return color.New(color.FgWhite)
	}
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ValidateDestinationForMoveOnPod checks if the destination path is valid for a move operation on the pod
func ValidateDestinationForMoveOnPod(ctx context.Context, k8sClient *pkg.K8sClient, namespace, podName, destPath, backupID string) error {
	// For move operation, check that the final destination directory doesn't exist
	finalDestination := filepath.Join(destPath, backupID)
	cmd := []string{"test", "-d", finalDestination}
	_, err := k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	if err == nil {
		return fmt.Errorf("destination directory already exists on pod: %s", finalDestination)
	}

	// Check if parent directory exists, create if needed
	cmd = []string{"mkdir", "-p", destPath}
	_, err = k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	if err != nil {
		return fmt.Errorf("cannot create destination parent directory on pod: %w", err)
	}

	// Test if we can write to the directory
	testFile := filepath.Join(destPath, ".kubectl-broker-test")
	cmd = []string{"touch", testFile}
	_, err = k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	if err != nil {
		return fmt.Errorf("destination directory is not writable on pod: %w", err)
	}

	// Clean up test file
	cmd = []string{"rm", "-f", testFile}
	k8sClient.ExecCommand(ctx, namespace, podName, cmd) // Ignore error for cleanup

	return nil
}

// DetectBackupPod finds which pod in the StatefulSet contains the backup file
func DetectBackupPod(ctx context.Context, k8sClient *pkg.K8sClient, namespace, statefulSetName, backupID string) (string, error) {
	// Get pods from the StatefulSet
	pods, err := k8sClient.GetStatefulSetPods(ctx, namespace, statefulSetName)
	if err != nil {
		return "", fmt.Errorf("failed to get StatefulSet pods: %w", err)
	}

	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found in StatefulSet %s", statefulSetName)
	}

	// Get backup folder location from the first pod
	backupFolder, err := GetBackupFolder(ctx, k8sClient, namespace, pods[0].Name)
	if err != nil {
		return "", fmt.Errorf("failed to get backup folder location: %w", err)
	}

	// Check each pod for the backup file
	for _, pod := range pods {
		if pod.Status.Phase != v1.PodRunning {
			continue // Skip non-running pods
		}

		// Check if backup directory exists on this pod
		backupDir := filepath.Join(backupFolder, backupID)
		exists, err := directoryExistsOnPod(ctx, k8sClient, namespace, pod.Name, backupDir)
		if err != nil {
			continue // Try next pod if this one fails
		}

		if exists {
			return pod.Name, nil
		}
	}

	return "", fmt.Errorf("backup file %s not found on any pod in StatefulSet %s", backupID, statefulSetName)
}

// GetBackupFolder determines the backup folder path on a HiveMQ pod
func GetBackupFolder(ctx context.Context, k8sClient *pkg.K8sClient, namespace, podName string) (string, error) {
	// First, try to get the backup folder from environment variable
	cmd := []string{"printenv", "HIVEMQ_BACKUP_FOLDER"}
	output, err := k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	if err == nil && strings.TrimSpace(output) != "" {
		return strings.TrimSpace(output), nil
	}

	// Fallback to default HiveMQ backup folder
	return "/opt/hivemq/backup", nil
}

// fileExistsOnPod checks if a file exists on the specified pod
func fileExistsOnPod(ctx context.Context, k8sClient *pkg.K8sClient, namespace, podName, filePath string) (bool, error) {
	cmd := []string{"test", "-f", filePath}
	_, err := k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	return err == nil, nil
}

// directoryExistsOnPod checks if a directory exists on the specified pod
func directoryExistsOnPod(ctx context.Context, k8sClient *pkg.K8sClient, namespace, podName, dirPath string) (bool, error) {
	cmd := []string{"test", "-d", dirPath}
	_, err := k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	return err == nil, nil
}

// MoveBackupDirectoryWithinPod moves a backup directory to another location within the same pod
func MoveBackupDirectoryWithinPod(ctx context.Context, k8sClient *pkg.K8sClient, namespace, podName, backupID, backupFolder, destination string) error {
	backupDir := filepath.Join(backupFolder, backupID)
	destinationDir := filepath.Join(destination, backupID)

	fmt.Printf("Moving backup directory from %s to %s on pod %s\n", backupDir, destinationDir, podName)

	// Use kubectl exec to move the directory within the pod
	cmd := []string{"mv", backupDir, destinationDir}
	output, err := k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	if err != nil {
		return fmt.Errorf("failed to move backup directory within pod: %w", err)
	}

	// Verify the move was successful by checking if destination exists and source doesn't
	cmd = []string{"test", "-d", destinationDir}
	_, err = k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	if err != nil {
		return fmt.Errorf("failed to verify moved directory: destination not found")
	}

	// Verify source directory no longer exists
	cmd = []string{"test", "-d", backupDir}
	_, err = k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	if err == nil {
		return fmt.Errorf("move failed: source directory still exists at %s", backupDir)
	}

	// Get directory size for confirmation
	cmd = []string{"du", "-sh", destinationDir}
	sizeOutput, err := k8sClient.ExecCommand(ctx, namespace, podName, cmd)
	if err == nil {
		sizeOutput = strings.TrimSpace(sizeOutput)
		if parts := strings.Fields(sizeOutput); len(parts) > 0 {
			fmt.Printf("Successfully moved backup directory to %s (size: %s)\n", destinationDir, parts[0])
		}
	} else {
		fmt.Printf("Successfully moved backup directory to %s\n", destinationDir)
	}

	if output != "" {
		fmt.Printf("Move output: %s\n", output)
	}

	return nil
}

// MoveBackupToDestination orchestrates the complete backup move process within the pod
func MoveBackupToDestination(ctx context.Context, k8sClient *pkg.K8sClient, namespace, statefulSetName, backupID, destination string) error {
	// Find which pod has the backup
	podName, err := DetectBackupPod(ctx, k8sClient, namespace, statefulSetName, backupID)
	if err != nil {
		return fmt.Errorf("backup pod detection failed: %w", err)
	}

	// Validate destination on the pod for move operation
	if err := ValidateDestinationForMoveOnPod(ctx, k8sClient, namespace, podName, destination, backupID); err != nil {
		return fmt.Errorf("destination validation failed: %w", err)
	}

	// Get backup folder location
	backupFolder, err := GetBackupFolder(ctx, k8sClient, namespace, podName)
	if err != nil {
		return fmt.Errorf("failed to get backup folder: %w", err)
	}

	// Move the backup directory within the pod
	if err := MoveBackupDirectoryWithinPod(ctx, k8sClient, namespace, podName, backupID, backupFolder, destination); err != nil {
		return fmt.Errorf("directory move failed: %w", err)
	}

	return nil
}
