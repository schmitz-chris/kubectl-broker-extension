package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	"kubectl-broker/pkg"
	"kubectl-broker/pkg/volumes"
)

var (
	// Global volumes flags
	volumesNamespace     string
	volumesAllNamespaces bool
	volumesMinAge        string
	volumesMinSize       string
	volumesDryRun        bool
	volumesConfirm       bool
	volumesForce         bool
	volumesShowReleased  bool
	volumesShowOrphaned  bool
	volumesShowAll       bool
	volumesShowDetailed  bool
)

func newVolumesCommand() *cobra.Command {
	var volumesCmd = &cobra.Command{
		Use:   "volumes",
		Short: "Manage HiveMQ broker volumes and cleanup orphaned storage",
		Long: `Volumes command provides comprehensive volume management for HiveMQ broker clusters
running on Kubernetes. It can discover orphaned volumes, analyze storage usage, 
and safely clean up released persistent volumes to reclaim storage space.

The command operates in the current kubectl context namespace by default, but can
also perform cluster-wide operations with the --all-namespaces flag.

Examples:
  # List volumes in current namespace
  kubectl broker volumes list

  # List all orphaned volumes in current namespace  
  kubectl broker volumes list --orphaned

  # Preview cleanup in current namespace
  kubectl broker volumes cleanup --dry-run

  # Clean up orphaned volumes in current namespace
  kubectl broker volumes cleanup --confirm

  # Global cluster-wide operations
  kubectl broker volumes list --all-namespaces
  kubectl broker volumes cleanup --all-namespaces --dry-run
  kubectl broker volumes cleanup --all-namespaces --confirm

  # Safety features
  kubectl broker volumes cleanup --older-than 30d --dry-run
  kubectl broker volumes cleanup --min-size 1Gi --confirm`,
	}

	// Add persistent flags for all subcommands
	volumesCmd.PersistentFlags().StringVarP(&volumesNamespace, "namespace", "n", "", "Namespace to operate in (defaults to current kubectl context)")
	volumesCmd.PersistentFlags().BoolVar(&volumesAllNamespaces, "all-namespaces", false, "Operate across all namespaces in the cluster")
	volumesCmd.PersistentFlags().StringVar(&volumesMinAge, "older-than", "", "Only show/delete volumes older than specified duration (e.g., 7d, 30d)")
	volumesCmd.PersistentFlags().StringVar(&volumesMinSize, "min-size", "", "Only show/delete volumes larger than specified size (e.g., 1Gi, 100Mi)")

	// Add subcommands
	volumesCmd.AddCommand(newVolumesListCommand())
	volumesCmd.AddCommand(newVolumesCleanupCommand())
	volumesCmd.AddCommand(newVolumesDiscoverCommand())

	return volumesCmd
}

func newVolumesListCommand() *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List volumes and their status",
		Long: `List persistent volumes and persistent volume claims in the specified namespace
or across the entire cluster. Shows volume status, size, age, and associated resources.

By default, shows volumes in the current kubectl context namespace.`,
		RunE: runVolumesList,
	}

	listCmd.Flags().BoolVar(&volumesShowReleased, "released", false, "Show only released persistent volumes")
	listCmd.Flags().BoolVar(&volumesShowOrphaned, "orphaned", false, "Show only orphaned volumes (PVCs without pods)")
	listCmd.Flags().BoolVar(&volumesShowAll, "all", false, "Show all volumes including bound ones")
	listCmd.Flags().BoolVar(&volumesShowDetailed, "detailed", false, "Show detailed usage information (slower, queries Node Stats API)")

	return listCmd
}

func newVolumesCleanupCommand() *cobra.Command {
	var cleanupCmd = &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up orphaned and released volumes",
		Long: `Clean up orphaned persistent volume claims and released persistent volumes.
This operation helps reclaim storage space from deleted HiveMQ broker instances.

By default, operates in the current kubectl context namespace. Use --all-namespaces
for cluster-wide cleanup.

IMPORTANT: Always run with --dry-run first to preview what will be deleted!`,
		RunE: runVolumesCleanup,
	}

	cleanupCmd.Flags().BoolVar(&volumesDryRun, "dry-run", false, "Preview what would be deleted without actually deleting")
	cleanupCmd.Flags().BoolVar(&volumesConfirm, "confirm", false, "Confirm deletion (required for actual deletion)")
	cleanupCmd.Flags().BoolVar(&volumesForce, "force", false, "Skip confirmation prompts (dangerous!)")

	return cleanupCmd
}

func newVolumesDiscoverCommand() *cobra.Command {
	var discoverCmd = &cobra.Command{
		Use:   "discover",
		Short: "Discover and analyze volume usage patterns",
		Long: `Discover persistent volumes and claims across the cluster and analyze 
storage usage patterns. Provides insights into total storage usage, 
reclaimable space, and volume distribution by namespace.`,
		RunE: runVolumesDiscover,
	}

	return discoverCmd
}

// Apply intelligent defaults similar to status and backup commands
func applyVolumesDefaults() error {
	if volumesNamespace == "" && !volumesAllNamespaces {
		// Get current namespace from kubeconfig context
		namespace, err := pkg.GetDefaultNamespace()
		if err != nil {
			return fmt.Errorf("failed to get current namespace: %w\n\nPlease either:\n- Set a kubectl context with namespace: kubectl config set-context --current --namespace=<namespace>\n- Specify namespace explicitly: --namespace <namespace>\n- Use --all-namespaces for cluster-wide operations", err)
		}
		volumesNamespace = namespace
		fmt.Printf("Using namespace: %s (from kubeconfig context)\n", volumesNamespace)
	}

	return nil
}

func runVolumesList(cmd *cobra.Command, args []string) error {
	if err := applyVolumesDefaults(); err != nil {
		return err
	}

	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(false)
	if err != nil {
		return pkg.EnhanceError(err, "failed to initialize Kubernetes client")
	}

	// Create volume analyzer
	analyzer := volumes.NewAnalyzer(k8sClient)

	// Set up analysis options
	options := volumes.AnalysisOptions{
		Namespace:     volumesNamespace,
		AllNamespaces: volumesAllNamespaces,
		MinAge:        parseMinAge(volumesMinAge),
		MinSize:       volumesMinSize,
		ShowReleased:  volumesShowReleased,
		ShowOrphaned:  volumesShowOrphaned,
		ShowAll:       volumesShowAll,
		ShowDetailed:  volumesShowDetailed,
		UseColors:     true,
	}

	// Perform analysis
	ctx := context.Background()
	result, err := analyzer.AnalyzeVolumes(ctx, options)
	if err != nil {
		return fmt.Errorf("volume analysis failed: %w", err)
	}

	// Display results
	displayVolumesList(result, options)

	return nil
}

func runVolumesCleanup(cmd *cobra.Command, args []string) error {
	if err := applyVolumesDefaults(); err != nil {
		return err
	}

	// Validate flags
	if !volumesDryRun && !volumesConfirm && !volumesForce {
		return fmt.Errorf("cleanup requires either --dry-run, --confirm, or --force flag")
	}

	if volumesConfirm && volumesForce {
		return fmt.Errorf("cannot use both --confirm and --force flags together")
	}

	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(false)
	if err != nil {
		return pkg.EnhanceError(err, "failed to initialize Kubernetes client")
	}

	// Create volume cleaner
	cleaner := volumes.NewCleaner(k8sClient)

	// Set up cleanup options
	options := volumes.CleanupOptions{
		Namespace:     volumesNamespace,
		AllNamespaces: volumesAllNamespaces,
		MinAge:        parseMinAge(volumesMinAge),
		MinSize:       volumesMinSize,
		DryRun:        volumesDryRun,
		Force:         volumesForce,
		UseColors:     true,
	}

	// Perform cleanup
	ctx := context.Background()
	result, err := cleaner.CleanupVolumes(ctx, options)
	if err != nil {
		return fmt.Errorf("volume cleanup failed: %w", err)
	}

	// Display results
	displayCleanupResults(result, options)

	return nil
}

func runVolumesDiscover(cmd *cobra.Command, args []string) error {
	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(false)
	if err != nil {
		return pkg.EnhanceError(err, "failed to initialize Kubernetes client")
	}

	// Create volume analyzer
	analyzer := volumes.NewAnalyzer(k8sClient)

	// Set up discovery options for cluster-wide analysis
	options := volumes.AnalysisOptions{
		AllNamespaces: true,
		ShowAll:       true,
		UseColors:     true,
	}

	fmt.Println("Discovering volumes across cluster...")

	// Perform cluster-wide analysis
	ctx := context.Background()
	result, err := analyzer.AnalyzeVolumes(ctx, options)
	if err != nil {
		return fmt.Errorf("volume discovery failed: %w", err)
	}

	// Display discovery summary
	displayDiscoverySummary(result)

	return nil
}

// Helper functions for parsing and display

func parseMinAge(ageStr string) time.Duration {
	if ageStr == "" {
		return 0
	}

	duration, err := time.ParseDuration(ageStr)
	if err != nil {
		// Try parsing with common suffixes
		if len(ageStr) > 0 {
			switch ageStr[len(ageStr)-1] {
			case 'd':
				if days, parseErr := time.ParseDuration(ageStr[:len(ageStr)-1] + "h"); parseErr == nil {
					return days * 24
				}
			case 'w':
				if weeks, parseErr := time.ParseDuration(ageStr[:len(ageStr)-1] + "h"); parseErr == nil {
					return weeks * 24 * 7
				}
			}
		}
		return 0
	}

	return duration
}

func displayVolumesList(result *volumes.AnalysisResult, options volumes.AnalysisOptions) {
	totalVolumes := len(result.ReleasedPVs) + len(result.OrphanedPVCs) + len(result.BoundVolumes)

	if totalVolumes == 0 {
		if options.AllNamespaces {
			fmt.Println("No volumes found across cluster.")
		} else {
			fmt.Printf("No volumes found in namespace: %s\n", options.Namespace)
		}
		return
	}

	// Display header - different format based on detailed mode
	if options.ShowDetailed {
		fmt.Printf("VOLUME NAME                               SIZE     USED     AVAIL    USAGE%%  AGE      STATUS       NAMESPACE\n")
		fmt.Printf("----------------------------------------  -------  -------  -------  ------  -------  -----------  ---------\n")
	} else {
		fmt.Printf("VOLUME NAME                               SIZE     AGE      STATUS       NAMESPACE\n")
		fmt.Printf("----------------------------------------  -------  -------  -----------  ---------\n")
	}

	// Display released PVs
	for _, pv := range result.ReleasedPVs {
		age := time.Since(pv.CreationTimestamp.Time).Round(24 * time.Hour)
		statusColor := getVolumeStatusColor("RELEASED", options.UseColors)
		namespace := ""
		if pv.Spec.ClaimRef != nil {
			namespace = pv.Spec.ClaimRef.Namespace
		}

		if options.ShowDetailed {
			// Released PVs don't have usage data
			used, available, usagePercent := "-", "-", "-"

			fmt.Printf("%-40s  %-7s  %-7s  %-7s  %-6s  %-7s  %s  %s\n",
				truncateString(pv.Name, 40),
				formatStorageSize(pv.Spec.Capacity["storage"]),
				used,
				available,
				usagePercent,
				formatDuration(age),
				statusColor.Sprint("RELEASED"),
				namespace)
		} else {
			fmt.Printf("%-40s  %-7s  %-7s  %s  %s\n",
				truncateString(pv.Name, 40),
				formatStorageSize(pv.Spec.Capacity["storage"]),
				formatDuration(age),
				statusColor.Sprint("RELEASED"),
				namespace)
		}
	}

	// Display orphaned PVCs
	for _, pvc := range result.OrphanedPVCs {
		age := time.Since(pvc.CreationTimestamp.Time).Round(24 * time.Hour)
		statusColor := getVolumeStatusColor("ORPHANED", options.UseColors)

		if options.ShowDetailed {
			// Orphaned PVCs don't have usage data
			used, available, usagePercent := "-", "-", "-"

			fmt.Printf("%-40s  %-7s  %-7s  %-7s  %-6s  %-7s  %s  %s\n",
				truncateString(pvc.Name, 40),
				formatStorageSize(pvc.Spec.Resources.Requests["storage"]),
				used,
				available,
				usagePercent,
				formatDuration(age),
				statusColor.Sprint("ORPHANED"),
				pvc.Namespace)
		} else {
			fmt.Printf("%-40s  %-7s  %-7s  %s  %s\n",
				truncateString(pvc.Name, 40),
				formatStorageSize(pvc.Spec.Resources.Requests["storage"]),
				formatDuration(age),
				statusColor.Sprint("ORPHANED"),
				pvc.Namespace)
		}
	}

	// Display bound volumes (only if --all flag is used or no specific filter)
	if options.ShowAll || (!options.ShowReleased && !options.ShowOrphaned) {
		for _, volume := range result.BoundVolumes {
			statusColor := getVolumeStatusColor("BOUND", options.UseColors)

			if options.ShowDetailed {
				// Get usage information for bound volumes
				used, available, usagePercent := formatUsageInfo(volume.Usage)

				fmt.Printf("%-40s  %-7s  %-7s  %-7s  %-6s  %-7s  %s  %s\n",
					truncateString(volume.PVC.Name, 40),
					formatStorageSize(volume.PVC.Spec.Resources.Requests["storage"]),
					used,
					available,
					usagePercent,
					formatDuration(volume.Age),
					statusColor.Sprint("BOUND"),
					volume.Namespace)
			} else {
				fmt.Printf("%-40s  %-7s  %-7s  %s  %s\n",
					truncateString(volume.PVC.Name, 40),
					formatStorageSize(volume.PVC.Spec.Resources.Requests["storage"]),
					formatDuration(volume.Age),
					statusColor.Sprint("BOUND"),
					volume.Namespace)
			}
		}
	}

	// Display summary
	releasedCount := len(result.ReleasedPVs)
	orphanedCount := len(result.OrphanedPVCs)
	boundCount := len(result.BoundVolumes)

	fmt.Printf("\nSummary: %d released PVs, %d orphaned PVCs", releasedCount, orphanedCount)
	if options.ShowAll || (!options.ShowReleased && !options.ShowOrphaned) {
		fmt.Printf(", %d bound volumes", boundCount)
	}
	fmt.Printf("\n")

	if result.TotalReclaimableStorage > 0 {
		fmt.Printf("Total reclaimable storage: %s\n",
			formatBytes(result.TotalReclaimableStorage))
	}
}

func displayCleanupResults(result *volumes.CleanupResult, options volumes.CleanupOptions) {
	if options.DryRun {
		fmt.Printf("DRY RUN - Would delete:\n")
	} else {
		fmt.Printf("Cleanup completed:\n")
	}

	fmt.Printf("- Released PVs: %d\n", len(result.DeletedPVs))
	fmt.Printf("- Orphaned PVCs: %d\n", len(result.DeletedPVCs))
	fmt.Printf("- Storage reclaimed: %s\n", formatBytes(result.TotalReclaimedStorage))

	if options.DryRun {
		fmt.Printf("\nUse --confirm to proceed with deletion.\n")
	}
}

func displayDiscoverySummary(result *volumes.AnalysisResult) {
	fmt.Printf("Volume Discovery Summary\n")
	fmt.Printf("========================\n\n")

	fmt.Printf("Total Persistent Volumes: %d\n", result.TotalPVs)
	fmt.Printf("Total Persistent Volume Claims: %d\n", result.TotalPVCs)
	fmt.Printf("Released PVs (reclaimable): %d\n", len(result.ReleasedPVs))
	fmt.Printf("Orphaned PVCs: %d\n", len(result.OrphanedPVCs))

	if result.TotalReclaimableStorage > 0 {
		fmt.Printf("Total reclaimable storage: %s\n", formatBytes(result.TotalReclaimableStorage))
	}

	fmt.Printf("\nNamespaces with orphaned volumes: %d\n", len(result.NamespaceStats))
}

// Utility functions

func getVolumeStatusColor(status string, useColors bool) *color.Color {
	if !useColors {
		return color.New()
	}

	switch status {
	case "RELEASED":
		return color.New(color.FgRed, color.Bold)
	case "ORPHANED":
		return color.New(color.FgYellow, color.Bold)
	case "BOUND":
		return color.New(color.FgGreen)
	default:
		return color.New(color.FgWhite)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

func formatStorageSize(quantity interface{}) string {
	// Handle Kubernetes resource.Quantity
	if q, ok := quantity.(resource.Quantity); ok {
		// Convert to bytes and format
		bytes := q.Value()
		return formatBytes(bytes)
	}

	// Handle string representation
	if qStr, ok := quantity.(string); ok {
		if parsed, err := resource.ParseQuantity(qStr); err == nil {
			return formatBytes(parsed.Value())
		}
		return qStr
	}

	// Fallback for unknown types
	return fmt.Sprintf("%v", quantity)
}

// formatUsageInfo formats volume usage information with fallback to "-" when not available
func formatUsageInfo(usage *volumes.VolumeUsage) (string, string, string) {
	if usage == nil {
		return "-", "-", "-"
	}

	used := formatBytes(usage.UsedBytes)
	available := formatBytes(usage.AvailableBytes)
	usagePercent := fmt.Sprintf("%.0f%%", usage.UsagePercent)

	return used, available, usagePercent
}
