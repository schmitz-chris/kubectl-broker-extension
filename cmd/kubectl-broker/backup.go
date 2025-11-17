package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"kubectl-broker/pkg"
	"kubectl-broker/pkg/backup"
)

var (
	// Global backup flags
	backupStatefulSetName string
	backupNamespace       string
	backupUsername        string
	backupPassword        string

	// Create command flags
	createDestination string

	// List command flags
	// (uses global flags)

	// Download command flags
	downloadBackupID  string
	downloadOutputDir string
	downloadOutput    string
	downloadLatest    bool

	// Status command flags
	statusBackupID string
	statusLatest   bool

	// Restore command flags
	restoreBackupID string
	restoreLatest   bool
)

var backupListColumns = []tableColumn{
	{Title: "BACKUP ID", Width: 16},
	{Title: "SIZE", Width: 10},
	{Title: "CREATED", Width: 20},
	{Title: "STATUS", Width: 10},
}

func newBackupCommand() *cobra.Command {
	var backupCmd = &cobra.Command{
		Use:   "backup",
		Short: "HiveMQ backup management operations",
		Long: `Backup command provides comprehensive backup management for HiveMQ broker clusters
running on Kubernetes. It supports creating backups, listing existing backups,
downloading backup files, and checking backup status.

Examples:
  # Create a backup
  kubectl broker backup create --statefulset broker --namespace production

  # List all backups
  kubectl broker backup list --pod broker-0 --namespace production

  # Download a specific backup
  kubectl broker backup download --id abc123 --output-dir ./backups

  # Download the latest backup
  kubectl broker backup download --latest

  # Check backup status
  kubectl broker backup status --id abc123
  
  # Restore from a specific backup
  kubectl broker backup restore --id abc123
  
  # Restore from the latest backup
  kubectl broker backup restore --latest`,
	}

	// Add persistent flags for all subcommands
	backupCmd.PersistentFlags().StringVar(&backupStatefulSetName, "statefulset", "", "Name of the StatefulSet to backup (defaults to 'broker')")
	backupCmd.PersistentFlags().StringVarP(&backupNamespace, "namespace", "n", "", "Namespace (defaults to current kubectl context)")
	backupCmd.PersistentFlags().StringVar(&backupUsername, "username", "", "Optional authentication username")
	backupCmd.PersistentFlags().StringVar(&backupPassword, "password", "", "Optional authentication password")

	// Add subcommands
	backupCmd.AddCommand(newBackupCreateCommand())
	backupCmd.AddCommand(newBackupListCommand())
	backupCmd.AddCommand(newBackupDownloadCommand())
	backupCmd.AddCommand(newBackupStatusCommand())
	backupCmd.AddCommand(newBackupRestoreCommand())
	backupCmd.AddCommand(newBackupTestCommand())

	return backupCmd
}

func newBackupCreateCommand() *cobra.Command {
	var createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new backup",
		Long: `Create a new backup of the HiveMQ broker cluster. This operation will:
1. Connect to the broker's management API
2. Initiate a backup operation
3. Monitor progress until completion
4. Display the final backup ID and size
5. Optionally move backup directory to another location within the pod`,
		RunE: runBackupCreate,
	}

	createCmd.Flags().StringVar(&createDestination, "destination", "", "Pod path to move backup directory to after creation (e.g., /opt/hivemq/data/backup)")

	return createCmd
}

func newBackupListCommand() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all available backups",
		Long: `List all available backups from the HiveMQ broker cluster. 
Shows backup ID, size, creation time, and status in a tabular format.`,
		RunE: runBackupList,
	}

	return listCmd
}

func newBackupDownloadCommand() *cobra.Command {
	var downloadCmd = &cobra.Command{
		Use:   "download",
		Short: "Download a backup file",
		Long: `Download a backup file to local storage. You can specify a backup ID
or use --latest to download the most recent backup. Files are saved to
the output directory with progress indication.`,
		RunE: runBackupDownload,
	}

	downloadCmd.Flags().StringVar(&downloadBackupID, "id", "", "Backup ID to download")
	downloadCmd.Flags().StringVar(&downloadOutputDir, "output-dir", "./backups", "Directory to save backup files")
	downloadCmd.Flags().StringVar(&downloadOutput, "output", "", "Specific output filename (overrides automatic naming)")
	downloadCmd.Flags().BoolVar(&downloadLatest, "latest", false, "Download the latest backup")

	return downloadCmd
}

func newBackupStatusCommand() *cobra.Command {
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check backup status",
		Long: `Check the status of a backup operation. Shows current status,
progress (if in progress), size, and creation time.`,
		RunE: runBackupStatus,
	}

	statusCmd.Flags().StringVar(&statusBackupID, "id", "", "Backup ID to check")
	statusCmd.Flags().BoolVar(&statusLatest, "latest", false, "Check status of the latest backup")

	return statusCmd
}

func newBackupRestoreCommand() *cobra.Command {
	var restoreCmd = &cobra.Command{
		Use:   "restore",
		Short: "Restore from a backup",
		Long: `Restore the HiveMQ broker cluster from a backup. This operation will:
1. Connect to the broker's management API
2. Initiate a restore operation from the specified backup
3. Monitor progress until completion
4. Display the final restore status`,
		RunE: runBackupRestore,
	}

	restoreCmd.Flags().StringVar(&restoreBackupID, "id", "", "Backup ID to restore from")
	restoreCmd.Flags().BoolVar(&restoreLatest, "latest", false, "Restore from the latest backup")

	return restoreCmd
}

func newBackupTestCommand() *cobra.Command {
	var testCmd = &cobra.Command{
		Use:   "test",
		Short: "Test HiveMQ management API connectivity",
		Long:  `Test if the HiveMQ management API is available and accessible for backup operations.`,
		RunE:  runBackupTest,
	}

	return testCmd
}

// Apply intelligent defaults similar to the status command
func applyBackupDefaults() error {
	resolvedNamespace, fromContext, err := resolveNamespace(backupNamespace, false)
	if err != nil {
		return err
	}
	backupNamespace = resolvedNamespace
	if fromContext {
		fmt.Printf("Using namespace from context: %s\n", backupNamespace)
	}

	var usedDefault bool
	backupStatefulSetName, usedDefault = applyDefaultStatefulSet(backupStatefulSetName)
	if usedDefault {
		fmt.Printf("Using default StatefulSet: %s\n", backupStatefulSetName)
	}

	return nil
}

func runBackupCreate(cmd *cobra.Command, args []string) error {
	if err := applyBackupDefaults(); err != nil {
		return err
	}

	fmt.Printf("Creating backup for StatefulSet %s in namespace %s\n", backupStatefulSetName, backupNamespace)

	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(false)
	if err != nil {
		return pkg.EnhanceError(err, "failed to initialize Kubernetes client")
	}

	// Get the API service from the StatefulSet
	service, err := k8sClient.GetAPIServiceFromStatefulSet(context.Background(), backupNamespace, backupStatefulSetName)
	if err != nil {
		return pkg.EnhanceError(err, fmt.Sprintf("StatefulSet %s in namespace %s", backupStatefulSetName, backupNamespace))
	}

	// Set up backup options
	options := backup.BackupOptions{
		Username:     backupUsername,
		Password:     backupPassword,
		Timeout:      5 * time.Minute,
		PollInterval: 2 * time.Second,
		ShowProgress: true,
		Destination:  createDestination,
	}

	// Create backup
	backupInfo, err := backup.CreateBackup(context.Background(), k8sClient, service, options)
	if err != nil {
		return fmt.Errorf("backup creation failed: %w", err)
	}

	// Display results
	fmt.Printf("Backup ID: %s\n", backupInfo.ID)
	fmt.Printf("Status: %s\n", getStatusColor(backupInfo.Status).Sprint(string(backupInfo.Status)))
	fmt.Printf("Size: %s | Created: %s\n", formatBytes(backupInfo.Size), backupInfo.CreatedAt.Format(time.RFC3339))

	// Move backup directory to destination if specified
	if createDestination != "" {
		fmt.Printf("\nMoving backup directory to destination...\n")
		err := backup.MoveBackupToDestination(
			context.Background(),
			k8sClient,
			backupNamespace,
			backupStatefulSetName,
			backupInfo.ID,
			createDestination,
		)
		if err != nil {
			return fmt.Errorf("backup move failed: %w", err)
		}
	}

	return nil
}

func runBackupList(cmd *cobra.Command, args []string) error {
	if err := applyBackupDefaults(); err != nil {
		return err
	}

	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(false)
	if err != nil {
		return pkg.EnhanceError(err, "failed to initialize Kubernetes client")
	}

	// Get the API service from the StatefulSet
	service, err := k8sClient.GetAPIServiceFromStatefulSet(context.Background(), backupNamespace, backupStatefulSetName)
	if err != nil {
		return pkg.EnhanceError(err, fmt.Sprintf("StatefulSet %s in namespace %s", backupStatefulSetName, backupNamespace))
	}

	// Set up backup options
	options := backup.BackupOptions{
		Username: backupUsername,
		Password: backupPassword,
	}

	// List backups
	backups, err := backup.ListBackups(context.Background(), k8sClient, service, options)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		fmt.Println("No backups found.")
		return nil
	}

	// Display results in tabular format
	renderTableHeader(backupListColumns, 2)

	var totalSize int64
	for _, b := range backups {
		backupID := b.ID
		if len(backupID) > 16 {
			backupID = backupID[:16]
		}

		statusColor := getStatusColor(b.Status)
		fmt.Printf("%-16s  %-10s  %-20s  %s\n",
			backupID,
			formatBytes(b.Size),
			b.CreatedAt.Format("2006-01-02 15:04:05"),
			statusColor.Sprint(string(b.Status)))

		totalSize += b.Size
	}

	fmt.Printf("\nSummary: %d backups, %s total\n", len(backups), formatBytes(totalSize))

	return nil
}

func runBackupDownload(cmd *cobra.Command, args []string) error {
	if err := applyBackupDefaults(); err != nil {
		return err
	}

	if downloadBackupID == "" && !downloadLatest {
		return fmt.Errorf("either --id or --latest must be specified\n\nPlease either:\n- Specify a backup ID: --id <backup-id>\n- Use latest backup: --latest")
	}

	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(false)
	if err != nil {
		return pkg.EnhanceError(err, "failed to initialize Kubernetes client")
	}

	// Get the API service from the StatefulSet
	service, err := k8sClient.GetAPIServiceFromStatefulSet(context.Background(), backupNamespace, backupStatefulSetName)
	if err != nil {
		return pkg.EnhanceError(err, fmt.Sprintf("StatefulSet %s in namespace %s", backupStatefulSetName, backupNamespace))
	}

	// Set up backup options
	options := backup.BackupOptions{
		Username:     backupUsername,
		Password:     backupPassword,
		OutputDir:    downloadOutputDir,
		OutputFile:   downloadOutput,
		ShowProgress: true,
	}

	// Handle the latest backup selection
	backupID := downloadBackupID
	if downloadLatest {
		backups, err := backup.ListBackups(context.Background(), k8sClient, service, options)
		if err != nil {
			return fmt.Errorf("failed to list backups to find latest: %w", err)
		}
		if len(backups) == 0 {
			return fmt.Errorf("no backups found")
		}
		backupID = backups[0].ID // Already sorted newest first
		fmt.Printf("Using latest backup: %s\n", backupID)
	}

	// Download backup
	savedPath, err := backup.DownloadBackup(context.Background(), k8sClient, service, backupID, options)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Get an absolute path for display
	absPath, err := filepath.Abs(savedPath)
	if err != nil {
		absPath = savedPath
	}

	fmt.Printf("\nDownload completed successfully!\n")
	fmt.Printf("Saved to: %s\n", absPath)

	return nil
}

func runBackupStatus(cmd *cobra.Command, args []string) error {
	if err := applyBackupDefaults(); err != nil {
		return err
	}

	if statusBackupID == "" && !statusLatest {
		return fmt.Errorf("either --id or --latest must be specified\n\nPlease either:\n- Specify a backup ID: --id <backup-id>\n- Use latest backup: --latest")
	}

	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(false)
	if err != nil {
		return pkg.EnhanceError(err, "failed to initialize Kubernetes client")
	}

	// Get the API service from the StatefulSet
	service, err := k8sClient.GetAPIServiceFromStatefulSet(context.Background(), backupNamespace, backupStatefulSetName)
	if err != nil {
		return pkg.EnhanceError(err, fmt.Sprintf("StatefulSet %s in namespace %s", backupStatefulSetName, backupNamespace))
	}

	// Set up backup options
	options := backup.BackupOptions{
		Username: backupUsername,
		Password: backupPassword,
	}

	// Handle the latest backup selection
	backupID := statusBackupID
	if statusLatest {
		backupID = "latest" // Special ID handled by GetBackupStatus
		fmt.Printf("Checking status of latest backup\n")
	}

	// Get backup status
	status, err := backup.GetBackupStatus(context.Background(), k8sClient, service, backupID, options)
	if err != nil {
		return fmt.Errorf("failed to get backup status: %w", err)
	}

	// Display status
	statusColor := getStatusColor(status.Status)
	fmt.Printf("Backup ID: %s\n", status.ID)
	fmt.Printf("Status: %s\n", statusColor.Sprint(string(status.Status)))
	fmt.Printf("Created: %s\n", status.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Size: %s\n", formatBytes(status.Size))

	if status.Progress > 0 && !status.Status.IsTerminal() {
		fmt.Printf("Progress: %d%%\n", status.Progress)
	}

	if status.Message != "" {
		fmt.Printf("Message: %s\n", status.Message)
	}

	return nil
}

func runBackupRestore(cmd *cobra.Command, args []string) error {
	if err := applyBackupDefaults(); err != nil {
		return err
	}

	if restoreBackupID == "" && !restoreLatest {
		return fmt.Errorf("either --id or --latest must be specified\n\nPlease either:\n- Specify a backup ID: --id <backup-id>\n- Use latest backup: --latest")
	}

	fmt.Printf("Restoring backup for StatefulSet %s in namespace %s\n", backupStatefulSetName, backupNamespace)

	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(false)
	if err != nil {
		return pkg.EnhanceError(err, "failed to initialize Kubernetes client")
	}

	// Get the API service from the StatefulSet
	service, err := k8sClient.GetAPIServiceFromStatefulSet(context.Background(), backupNamespace, backupStatefulSetName)
	if err != nil {
		return pkg.EnhanceError(err, fmt.Sprintf("StatefulSet %s in namespace %s", backupStatefulSetName, backupNamespace))
	}

	// Set up backup options
	options := backup.BackupOptions{
		Username:     backupUsername,
		Password:     backupPassword,
		Timeout:      5 * time.Minute,
		PollInterval: 2 * time.Second,
		ShowProgress: true,
	}

	// Handle the latest backup selection
	backupID := restoreBackupID
	if restoreLatest {
		backupID = "latest" // Special ID handled by RestoreBackup
		fmt.Printf("Restoring from latest backup\n")
	}

	// Restore backup
	err = backup.RestoreBackup(context.Background(), k8sClient, service, backupID, options)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	return nil
}

func runBackupTest(cmd *cobra.Command, args []string) error {
	if err := applyBackupDefaults(); err != nil {
		return err
	}

	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(false)
	if err != nil {
		return pkg.EnhanceError(err, "failed to initialize Kubernetes client")
	}

	// Get the API service from the StatefulSet
	service, err := k8sClient.GetAPIServiceFromStatefulSet(context.Background(), backupNamespace, backupStatefulSetName)
	if err != nil {
		return pkg.EnhanceError(err, fmt.Sprintf("StatefulSet %s in namespace %s", backupStatefulSetName, backupNamespace))
	}

	fmt.Printf("Testing HiveMQ management API for StatefulSet %s in namespace %s\n\n", backupStatefulSetName, backupNamespace)
	fmt.Printf("Testing against service: %s\n", service.Name)

	// Discover the API port for the service
	apiPort, err := k8sClient.DiscoverServiceAPIPort(service)
	if err != nil {
		return fmt.Errorf("failed to discover API port: %w", err)
	}

	fmt.Printf("API port discovered: %d\n", apiPort)

	// Get a random local port for port-forwarding
	localPort, err := pkg.GetRandomPort()
	if err != nil {
		return fmt.Errorf("failed to get random port: %w", err)
	}

	// Set up port forwarding
	pf := pkg.NewPortForwarder(k8sClient.GetConfig(), k8sClient.GetRESTClient())

	// Use service port forwarding to test API
	err = pf.PerformWithServicePortForwarding(context.Background(), k8sClient, service, apiPort, localPort, func(localPort int) error {
		// Create base URL for the backup API
		baseURL := fmt.Sprintf("http://localhost:%d", localPort)
		client := backup.NewClient(baseURL, backupUsername, backupPassword)

		fmt.Printf("Testing management API at: %s\n", baseURL)

		// Test basic connection
		if err := client.TestConnection(); err != nil {
			return fmt.Errorf("management API test failed: %w", err)
		}

		fmt.Printf("Management API is accessible!\n")

		// Try to list backups to test backup endpoint specifically
		fmt.Printf("Testing backup endpoint...\n")
		_, err := client.ListBackups()
		if err != nil {
			fmt.Printf("Backup endpoint test failed: %v\n", err)
			fmt.Printf("This might mean backup functionality is not enabled on this HiveMQ instance.\n")
			return nil // Don't fail completely, just warn
		}

		fmt.Printf("Backup API is available!\n")
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("All tests passed! This HiveMQ instance supports backup operations.\n")
	return nil
}

// getStatusColor returns a color function for the given backup status
func getStatusColor(status backup.BackupStatus) *color.Color {
	switch status {
	case backup.StatusCompleted, backup.StatusRestoreCompleted:
		return color.New(color.FgGreen, color.Bold)
	case backup.StatusFailed, backup.StatusRestoreFailed:
		return color.New(color.FgRed, color.Bold)
	case backup.StatusInProgress, backup.StatusRestoreInProgress:
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
