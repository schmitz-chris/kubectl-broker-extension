package main

import (
	"context"
	"fmt"
	"time"

	"kubectl-broker/pkg"
	"kubectl-broker/pkg/volumes"

	"github.com/spf13/cobra"
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
		resolvedNamespace, fromContext, err := resolveNamespace(volumesNamespace, true)
		if err != nil {
			return err
		}
		volumesNamespace = resolvedNamespace
		if fromContext {
			fmt.Printf("Using namespace from context: %s\n", volumesNamespace)
		}
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
		return fmt.Errorf("cleanup requires either --dry-run, --confirm, or --force flag\n\nPlease either:\n- Preview changes: --dry-run\n- Confirm deletion: --confirm\n- Force deletion: --force")
	}

	if err := mutuallyExclusive(volumesConfirm, "--confirm", volumesForce, "--force"); err != nil {
		return err
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
