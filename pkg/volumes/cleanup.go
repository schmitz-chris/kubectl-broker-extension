package volumes

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"kubectl-broker/pkg"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cleaner provides volume cleanup functionality
type Cleaner struct {
	k8sClient *pkg.K8sClient
	analyzer  *Analyzer
}

// NewCleaner creates a new volume cleaner
func NewCleaner(k8sClient *pkg.K8sClient) *Cleaner {
	return &Cleaner{
		k8sClient: k8sClient,
		analyzer:  NewAnalyzer(k8sClient),
	}
}

// CleanupVolumes performs volume cleanup based on the provided options
func (c *Cleaner) CleanupVolumes(ctx context.Context, options CleanupOptions) (*CleanupResult, error) {
	// First, analyze volumes to find candidates for cleanup
	analysisOptions := AnalysisOptions{
		Namespace:     options.Namespace,
		AllNamespaces: options.AllNamespaces,
		MinAge:        options.MinAge,
		MinSize:       options.MinSize,
		ShowReleased:  true,
		ShowOrphaned:  true,
		UseColors:     options.UseColors,
	}

	analysisResult, err := c.analyzer.AnalyzeVolumes(ctx, analysisOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze volumes for cleanup: %w", err)
	}

	result := &CleanupResult{
		DeletedPVs:      []string{},
		DeletedPVCs:     []string{},
		FailedDeletions: []CleanupError{},
	}

	// Filter volumes by cleanup criteria
	pvCandidates := c.filterPVsForCleanup(analysisResult.ReleasedPVs, options)
	pvcCandidates := c.filterPVCsForCleanup(analysisResult.OrphanedPVCs, options)
	result.PlannedReleasedPVs = len(pvCandidates)
	result.PlannedOrphanedPVCs = len(pvcCandidates)

	if len(pvCandidates) == 0 && len(pvcCandidates) == 0 {
		if options.UseColors {
			fmt.Println("No volumes found matching cleanup criteria.")
		}
		return result, nil
	}

	// Create cleanup plan
	c.createCleanupPlan(result, pvCandidates, pvcCandidates)

	// If dry-run, just return the preview
	if options.DryRun {
		c.displayDryRunPreview(result, options)
		return result, nil
	}

	// Show cleanup plan and ask for confirmation unless forced
	if !options.Force {
		confirmed, err := c.confirmCleanup(result, options)
		if err != nil {
			return nil, fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("Cleanup cancelled by user.")
			return result, nil
		}
	}

	// Perform actual cleanup
	if err := c.performCleanup(ctx, result, pvCandidates, pvcCandidates, options); err != nil {
		return result, fmt.Errorf("cleanup failed: %w", err)
	}

	return result, nil
}

// filterPVsForCleanup filters persistent volumes based on cleanup criteria
func (c *Cleaner) filterPVsForCleanup(pvs []*v1.PersistentVolume, options CleanupOptions) []*v1.PersistentVolume {
	var candidates []*v1.PersistentVolume

	for _, pv := range pvs {
		if c.shouldCleanupPV(pv, options) {
			candidates = append(candidates, pv)
		}
	}

	return candidates
}

// filterPVCsForCleanup filters persistent volume claims based on cleanup criteria
func (c *Cleaner) filterPVCsForCleanup(pvcs []*v1.PersistentVolumeClaim, options CleanupOptions) []*v1.PersistentVolumeClaim {
	var candidates []*v1.PersistentVolumeClaim

	for _, pvc := range pvcs {
		if c.shouldCleanupPVC(pvc, options) {
			candidates = append(candidates, pvc)
		}
	}

	return candidates
}

// shouldCleanupPV determines if a persistent volume should be cleaned up
func (c *Cleaner) shouldCleanupPV(pv *v1.PersistentVolume, options CleanupOptions) bool {
	// Only cleanup released PVs
	if pv.Status.Phase != v1.VolumeReleased {
		return false
	}

	// Check age requirement
	if options.MinAge > 0 {
		age := time.Since(pv.CreationTimestamp.Time)
		if age < options.MinAge {
			return false
		}
	}

	// Check size requirement
	if options.MinSize != "" {
		minSize, err := resource.ParseQuantity(options.MinSize)
		if err != nil {
			return false // Skip if size parsing fails
		}
		if storage, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
			if storage.Cmp(minSize) < 0 {
				return false
			}
		}
	}

	return true
}

// shouldCleanupPVC determines if a persistent volume claim should be cleaned up
func (c *Cleaner) shouldCleanupPVC(pvc *v1.PersistentVolumeClaim, options CleanupOptions) bool {
	// Check age requirement
	if options.MinAge > 0 {
		age := time.Since(pvc.CreationTimestamp.Time)
		if age < options.MinAge {
			return false
		}
	}

	// Check size requirement
	if options.MinSize != "" {
		minSize, err := resource.ParseQuantity(options.MinSize)
		if err != nil {
			return false // Skip if size parsing fails
		}
		if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
			if storage.Cmp(minSize) < 0 {
				return false
			}
		}
	}

	return true
}

// createCleanupPlan creates a detailed cleanup plan
func (c *Cleaner) createCleanupPlan(result *CleanupResult, pvs []*v1.PersistentVolume, pvcs []*v1.PersistentVolumeClaim) {
	result.DryRunPreview = []CleanupAction{}

	// Add PV cleanup actions
	for _, pv := range pvs {
		var size int64
		if storage, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
			size = storage.Value()
		}

		age := time.Since(pv.CreationTimestamp.Time)
		namespace := ""
		if pv.Spec.ClaimRef != nil {
			namespace = pv.Spec.ClaimRef.Namespace
		}

		action := CleanupAction{
			Type:      "PersistentVolume",
			Name:      pv.Name,
			Namespace: namespace,
			Size:      size,
			Age:       age,
			Reason:    "Volume is in Released state and can be reclaimed",
		}

		result.DryRunPreview = append(result.DryRunPreview, action)
		result.PlannedReclaimedStorage += size
	}

	// Add PVC cleanup actions
	for _, pvc := range pvcs {
		var size int64
		if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
			size = storage.Value()
		}

		age := time.Since(pvc.CreationTimestamp.Time)

		action := CleanupAction{
			Type:      "PersistentVolumeClaim",
			Name:      pvc.Name,
			Namespace: pvc.Namespace,
			Size:      size,
			Age:       age,
			Reason:    "PVC is not mounted by any running pods",
		}

		result.DryRunPreview = append(result.DryRunPreview, action)
		result.PlannedReclaimedStorage += size
	}
}

// displayDryRunPreview shows what would be deleted in dry-run mode
func (c *Cleaner) displayDryRunPreview(result *CleanupResult, options CleanupOptions) {
	if len(result.DryRunPreview) == 0 {
		fmt.Println("No volumes would be deleted with current criteria.")
		return
	}

	fmt.Printf("DRY RUN - The following %d volumes would be deleted:\n\n", len(result.DryRunPreview))

	// Display header
	fmt.Printf("TYPE                    NAME                                  NAMESPACE                             AGE     SIZE    REASON\n")
	fmt.Printf("----------------------  ------------------------------------  ------------------------------------  ------  ------  -------------------------\n")

	// Display each action
	for _, action := range result.DryRunPreview {
		fmt.Printf("%-22s  %-36s  %-36s  %-6s  %-6s  %s\n",
			action.Type,
			truncateString(action.Name, 36),
			truncateString(action.Namespace, 36),
			formatDuration(action.Age),
			formatSize(action.Size),
			action.Reason)
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("- %d PersistentVolumes would be deleted\n", countActionsByType(result.DryRunPreview, "PersistentVolume"))
	fmt.Printf("- %d PersistentVolumeClaims would be deleted\n", countActionsByType(result.DryRunPreview, "PersistentVolumeClaim"))
	fmt.Printf("- Total storage to be reclaimed: %s\n", formatSize(result.PlannedReclaimedStorage))

	fmt.Printf("\nTo proceed with deletion, run the command again with --confirm flag.\n")
}

// confirmCleanup asks user for confirmation before proceeding with cleanup
func (c *Cleaner) confirmCleanup(result *CleanupResult, options CleanupOptions) (bool, error) {
	if len(result.DryRunPreview) == 0 {
		return false, nil
	}

	fmt.Printf("About to delete %d volumes reclaiming %s of storage.\n",
		len(result.DryRunPreview), formatSize(result.PlannedReclaimedStorage))

	// Show summary by type
	pvCount := countActionsByType(result.DryRunPreview, "PersistentVolume")
	pvcCount := countActionsByType(result.DryRunPreview, "PersistentVolumeClaim")

	if pvCount > 0 {
		fmt.Printf("- %d Released PersistentVolumes\n", pvCount)
	}
	if pvcCount > 0 {
		fmt.Printf("- %d Orphaned PersistentVolumeClaims\n", pvcCount)
	}

	// Warning for large operations
	if len(result.DryRunPreview) > 10 {
		fmt.Printf("\nWARNING: This will delete %d volumes. Consider using --older-than flag for additional safety.\n", len(result.DryRunPreview))
	}

	fmt.Printf("\nThis operation cannot be undone. Are you sure you want to proceed? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// performCleanup executes the actual volume deletion
func (c *Cleaner) performCleanup(ctx context.Context, result *CleanupResult, pvs []*v1.PersistentVolume, pvcs []*v1.PersistentVolumeClaim, options CleanupOptions) error {
	coreClient := c.k8sClient.GetCoreClient()
	result.TotalReclaimedStorage = 0
	result.DeletedReleasedPVs = 0
	result.DeletedOrphanedPVCs = 0
	result.AssociatedPVsDeleted = 0

	fmt.Printf("Starting cleanup of %d volumes...\n", len(pvs)+len(pvcs))

	// Delete PersistentVolumes
	for i, pv := range pvs {
		fmt.Printf("[%d/%d] Deleting PV %s...", i+1, len(pvs), pv.Name)

		err := coreClient.PersistentVolumes().Delete(ctx, pv.Name, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf(" FAILED\n")
			result.FailedDeletions = append(result.FailedDeletions, CleanupError{
				Type:      "PersistentVolume",
				Name:      pv.Name,
				Namespace: "",
				Error:     err,
			})
		} else {
			fmt.Printf(" OK\n")
			result.DeletedPVs = append(result.DeletedPVs, pv.Name)
			result.DeletedReleasedPVs++

			// Add to reclaimed storage
			if storage, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
				result.TotalReclaimedStorage += storage.Value()
			}
		}
	}

	// Delete PersistentVolumeClaims (with cascade PV deletion)
	for i, pvc := range pvcs {
		// Find associated PV before deleting PVC
		associatedPV, err := c.findAssociatedPV(ctx, pvc)
		if err != nil {
			fmt.Printf("[%d/%d] Warning: Could not find PV for PVC %s: %v\n", i+1, len(pvcs), pvc.Name, err)
		}

		// Delete PVC first
		fmt.Printf("[%d/%d] Deleting PVC %s in namespace %s...", i+1, len(pvcs), pvc.Name, pvc.Namespace)

		err = coreClient.PersistentVolumeClaims(pvc.Namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf(" FAILED\n")
			result.FailedDeletions = append(result.FailedDeletions, CleanupError{
				Type:      "PersistentVolumeClaim",
				Name:      pvc.Name,
				Namespace: pvc.Namespace,
				Error:     err,
			})
			continue // Skip PV deletion if PVC deletion failed
		} else {
			fmt.Printf(" OK")
			result.DeletedPVCs = append(result.DeletedPVCs, pvc.Name)
			result.DeletedOrphanedPVCs++

			// Add to reclaimed storage
			if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
				result.TotalReclaimedStorage += storage.Value()
			}
		}

		// Now delete associated PV if found
		if associatedPV != nil {
			fmt.Printf(" + Deleting associated PV %s...", associatedPV.Name)

			err = coreClient.PersistentVolumes().Delete(ctx, associatedPV.Name, metav1.DeleteOptions{})
			if err != nil {
				fmt.Printf(" FAILED\n")
				result.FailedDeletions = append(result.FailedDeletions, CleanupError{
					Type:      "PersistentVolume",
					Name:      associatedPV.Name,
					Namespace: "",
					Error:     fmt.Errorf("cascade deletion failed: %w", err),
				})
			} else {
				fmt.Printf(" OK\n")
				result.DeletedPVs = append(result.DeletedPVs, associatedPV.Name)
				result.AssociatedPVsDeleted++

				// Add PV storage to reclaimed total (avoid double counting)
				if pvcStorage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
					if pvStorage, ok := associatedPV.Spec.Capacity[v1.ResourceStorage]; ok {
						// Only add the difference if PV is larger than PVC
						if pvStorage.Cmp(pvcStorage) > 0 {
							diff := pvStorage.DeepCopy()
							diff.Sub(pvcStorage)
							result.TotalReclaimedStorage += diff.Value()
						}
					}
				}
			}
		} else {
			fmt.Printf("\n")
		}
	}

	return nil
}

// Utility functions

func countActionsByType(actions []CleanupAction, actionType string) int {
	count := 0
	for _, action := range actions {
		if action.Type == actionType {
			count++
		}
	}
	return count
}

func formatSize(bytes int64) string {
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

// findAssociatedPV finds the persistent volume associated with a PVC
func (c *Cleaner) findAssociatedPV(ctx context.Context, pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolume, error) {
	if pvc.Spec.VolumeName == "" {
		return nil, fmt.Errorf("PVC %s has no volume name", pvc.Name)
	}

	// Get the PV by name
	pv, err := c.k8sClient.GetCoreClient().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get PV %s: %w", pvc.Spec.VolumeName, err)
	}

	// Verify this PV is actually bound to our PVC
	if pv.Spec.ClaimRef != nil &&
		pv.Spec.ClaimRef.Name == pvc.Name &&
		pv.Spec.ClaimRef.Namespace == pvc.Namespace {
		return pv, nil
	}

	return nil, fmt.Errorf("PV %s is not bound to PVC %s/%s", pv.Name, pvc.Namespace, pvc.Name)
}
