package volumes

import (
	"context"
	"fmt"
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubectl-broker/pkg"
)

// Analyzer provides volume analysis functionality
type Analyzer struct {
	k8sClient *pkg.K8sClient
}

// NewAnalyzer creates a new volume analyzer
func NewAnalyzer(k8sClient *pkg.K8sClient) *Analyzer {
	return &Analyzer{
		k8sClient: k8sClient,
	}
}

// AnalyzeVolumes performs comprehensive volume analysis
func (a *Analyzer) AnalyzeVolumes(ctx context.Context, options AnalysisOptions) (*AnalysisResult, error) {
	result := &AnalysisResult{
		NamespaceStats: make(map[string]*NamespaceVolumeStats),
	}

	// Initialize usage collector only if detailed mode is enabled
	var usageCollector *VolumeUsageCollector
	if options.ShowDetailed {
		usageCollector = NewVolumeUsageCollector(a.k8sClient)
	}

	if options.AllNamespaces {
		return a.analyzeClusterWide(ctx, options, result, usageCollector)
	} else {
		return a.analyzeNamespace(ctx, options.Namespace, options, result, usageCollector)
	}
}

// analyzeClusterWide performs cluster-wide volume analysis
func (a *Analyzer) analyzeClusterWide(ctx context.Context, options AnalysisOptions, result *AnalysisResult, usageCollector *VolumeUsageCollector) (*AnalysisResult, error) {
	// Get all PVs in cluster
	pvs, err := a.getAllPersistentVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get persistent volumes: %w", err)
	}
	result.TotalPVs = len(pvs)

	// Get all namespaces to check which ones exist
	namespaces, err := a.getAllNamespaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespaces: %w", err)
	}
	namespaceMap := make(map[string]bool)
	for _, ns := range namespaces {
		namespaceMap[ns.Name] = true
	}

	// Analyze each PV
	for _, pv := range pvs {
		if err := a.analyzePersistentVolume(ctx, pv, namespaceMap, options, result); err != nil {
			return nil, fmt.Errorf("failed to analyze PV %s: %w", pv.Name, err)
		}
	}

	// Get all PVCs across all namespaces
	allPVCs, err := a.getAllPersistentVolumeClaims(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get persistent volume claims: %w", err)
	}
	result.TotalPVCs = len(allPVCs)

	// Collect volume usage statistics for all namespaces
	var volumeUsageMap map[string]*VolumeUsage
	if usageCollector != nil {
		volumeUsageMap, err = usageCollector.GetAllVolumeUsage(ctx)
		if err != nil {
			// Log warning but continue without usage data
			fmt.Printf("Warning: failed to collect volume usage statistics: %v\n", err)
			volumeUsageMap = make(map[string]*VolumeUsage)
		}
	}

	// Analyze PVCs for orphaned ones
	for _, pvc := range allPVCs {
		if err := a.analyzePersistentVolumeClaim(ctx, pvc, options, result, volumeUsageMap); err != nil {
			return nil, fmt.Errorf("failed to analyze PVC %s: %w", pvc.Name, err)
		}
	}

	a.calculateTotalReclaimableStorage(result)
	a.generateRecommendations(result)

	return result, nil
}

// analyzeNamespace performs namespace-specific volume analysis
func (a *Analyzer) analyzeNamespace(ctx context.Context, namespace string, options AnalysisOptions, result *AnalysisResult, usageCollector *VolumeUsageCollector) (*AnalysisResult, error) {
	// Get PVCs in the specific namespace
	pvcs, err := a.getPersistentVolumeClaimsInNamespace(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get PVCs in namespace %s: %w", namespace, err)
	}
	result.TotalPVCs = len(pvcs)

	// Collect volume usage statistics for this namespace
	var volumeUsageMap map[string]*VolumeUsage
	if usageCollector != nil {
		volumeUsageMap, err = usageCollector.GetVolumeUsage(ctx, namespace)
		if err != nil {
			// Log warning but continue without usage data
			fmt.Printf("Warning: failed to collect volume usage statistics: %v\n", err)
			volumeUsageMap = make(map[string]*VolumeUsage)
		}
	}

	// Analyze each PVC for orphaned status
	for _, pvc := range pvcs {
		if err := a.analyzePersistentVolumeClaim(ctx, pvc, options, result, volumeUsageMap); err != nil {
			return nil, fmt.Errorf("failed to analyze PVC %s: %w", pvc.Name, err)
		}
	}

	// Get all PVs and check which ones belong to this namespace
	allPVs, err := a.getAllPersistentVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get persistent volumes: %w", err)
	}

	namespaceMap := map[string]bool{namespace: true}

	// Analyze PVs that were bound to PVCs in this namespace
	for _, pv := range allPVs {
		if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.Namespace == namespace {
			if err := a.analyzePersistentVolume(ctx, pv, namespaceMap, options, result); err != nil {
				return nil, fmt.Errorf("failed to analyze PV %s: %w", pv.Name, err)
			}
		}
	}

	result.TotalPVs = len(allPVs) // Total cluster PVs for context

	a.calculateTotalReclaimableStorage(result)
	a.generateRecommendations(result)

	return result, nil
}

// analyzePersistentVolume analyzes a single persistent volume
func (a *Analyzer) analyzePersistentVolume(ctx context.Context, pv *v1.PersistentVolume, namespaceMap map[string]bool, options AnalysisOptions, result *AnalysisResult) error {
	// Check if PV is released (can be reclaimed)
	if pv.Status.Phase == v1.VolumeReleased {
		// Check age requirements
		age := time.Since(pv.CreationTimestamp.Time)
		if options.MinAge > 0 && age < options.MinAge {
			return nil
		}

		// Check if this PV was from a deleted namespace
		var isFromDeletedNamespace bool
		if pv.Spec.ClaimRef != nil {
			claimNamespace := pv.Spec.ClaimRef.Namespace
			if !namespaceMap[claimNamespace] {
				isFromDeletedNamespace = true
			}

			// Update namespace statistics
			a.updateNamespaceStats(result, claimNamespace, pv, isFromDeletedNamespace)
		}

		result.ReleasedPVs = append(result.ReleasedPVs, pv)

		// Add storage to reclaimable total
		if storage, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
			result.TotalReclaimableStorage += storage.Value()
		}
	}

	return nil
}

// analyzePersistentVolumeClaim analyzes a single persistent volume claim
func (a *Analyzer) analyzePersistentVolumeClaim(ctx context.Context, pvc *v1.PersistentVolumeClaim, options AnalysisOptions, result *AnalysisResult, volumeUsageMap map[string]*VolumeUsage) error {
	age := time.Since(pvc.CreationTimestamp.Time)

	// Check if PVC is bound - if not bound, it might be orphaned
	if pvc.Status.Phase != v1.ClaimBound {
		// Check age requirements
		if options.MinAge > 0 && age < options.MinAge {
			return nil
		}

		result.OrphanedPVCs = append(result.OrphanedPVCs, pvc)

		// Add storage to reclaimable total
		if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
			result.TotalReclaimableStorage += storage.Value()
		}

		return nil
	}

	// For bound PVCs, check if they're actually being used by any pods
	isOrphaned, err := a.isPVCOrphaned(ctx, pvc)
	if err != nil {
		return fmt.Errorf("failed to check if PVC %s is orphaned: %w", pvc.Name, err)
	}

	if isOrphaned {
		// Check age requirements
		if options.MinAge > 0 && age < options.MinAge {
			return nil
		}

		result.OrphanedPVCs = append(result.OrphanedPVCs, pvc)

		// Add storage to reclaimable total
		if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
			result.TotalReclaimableStorage += storage.Value()
		}
	} else {
		// PVC is bound and being used - add to bound volumes list
		var size resource.Quantity
		if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
			size = storage
		}

		volumeInfo := VolumeInfo{
			PVC:            pvc,
			Type:           VolumeTypeBound,
			Status:         VolumeStatusBound,
			Size:           size,
			Age:            age,
			Namespace:      pvc.Namespace,
			IsHiveMQVolume: IsHiveMQVolume(pvc.Name, pvc.Namespace),
		}

		// Add volume usage information if available
		if volumeUsageMap != nil {
			if usage, exists := volumeUsageMap[pvc.Name]; exists {
				volumeInfo.Usage = usage
			}
		}

		// Try to find associated pods
		volumeInfo.AssociatedPods, _ = a.findPodsUsingPVC(ctx, pvc)

		result.BoundVolumes = append(result.BoundVolumes, volumeInfo)
	}

	return nil
}

// isPVCOrphaned checks if a PVC is bound but not used by any running pods
func (a *Analyzer) isPVCOrphaned(ctx context.Context, pvc *v1.PersistentVolumeClaim) (bool, error) {
	// Get all pods in the PVC's namespace
	podList, err := a.k8sClient.GetCoreClient().Pods(pvc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list pods in namespace %s: %w", pvc.Namespace, err)
	}

	// Check if any pod references this PVC
	for _, pod := range podList.Items {
		// Check all volumes in the pod
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil &&
				volume.PersistentVolumeClaim.ClaimName == pvc.Name {
				return false, nil // PVC is being used
			}
		}
	}

	return true, nil // No pods are using this PVC
}

// updateNamespaceStats updates namespace statistics for volume analysis
func (a *Analyzer) updateNamespaceStats(result *AnalysisResult, namespace string, pv *v1.PersistentVolume, namespaceExists bool) {
	if result.NamespaceStats[namespace] == nil {
		result.NamespaceStats[namespace] = &NamespaceVolumeStats{
			Namespace:         namespace,
			NamespaceExists:   namespaceExists,
			IsHiveMQNamespace: IsHiveMQVolume("", namespace),
		}
	}

	stats := result.NamespaceStats[namespace]
	stats.ReleasedPVs++

	if storage, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
		stats.TotalReclaimable += storage.Value()
	}

	if IsHiveMQVolume("", namespace) {
		stats.HiveMQVolumes++
		result.HiveMQVolumeCount++
	}
}

// calculateTotalReclaimableStorage calculates total storage that can be reclaimed
func (a *Analyzer) calculateTotalReclaimableStorage(result *AnalysisResult) {
	result.TotalReclaimableStorage = 0

	// Add storage from released PVs
	for _, pv := range result.ReleasedPVs {
		if storage, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
			result.TotalReclaimableStorage += storage.Value()
		}
	}

	// Add storage from orphaned PVCs
	for _, pvc := range result.OrphanedPVCs {
		if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
			result.TotalReclaimableStorage += storage.Value()
		}
	}
}

// generateRecommendations generates cleanup recommendations based on analysis
func (a *Analyzer) generateRecommendations(result *AnalysisResult) {
	result.Recommendations = []string{}

	if len(result.ReleasedPVs) > 0 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Found %d released persistent volumes that can be safely deleted", len(result.ReleasedPVs)))
	}

	if len(result.OrphanedPVCs) > 0 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Found %d orphaned persistent volume claims not used by any pods", len(result.OrphanedPVCs)))
	}

	if result.TotalReclaimableStorage > 0 {
		storageGB := float64(result.TotalReclaimableStorage) / (1024 * 1024 * 1024)
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Total reclaimable storage: %.1f GB", storageGB))
	}

	if result.HiveMQVolumeCount > 0 {
		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Detected %d HiveMQ-related volumes (UUID namespaces)", result.HiveMQVolumeCount))
	}

	// Add safety recommendations
	if len(result.ReleasedPVs) > 10 || len(result.OrphanedPVCs) > 10 {
		result.Recommendations = append(result.Recommendations,
			"Recommendation: Use --older-than flag to only delete volumes older than a safe threshold (e.g., 30d)")
	}
}

// Kubernetes client wrapper methods

func (a *Analyzer) getAllPersistentVolumes(ctx context.Context) ([]*v1.PersistentVolume, error) {
	pvList, err := a.k8sClient.GetCoreClient().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pvs := make([]*v1.PersistentVolume, len(pvList.Items))
	for i := range pvList.Items {
		pvs[i] = &pvList.Items[i]
	}

	return pvs, nil
}

func (a *Analyzer) getAllNamespaces(ctx context.Context) ([]*v1.Namespace, error) {
	nsList, err := a.k8sClient.GetCoreClient().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	namespaces := make([]*v1.Namespace, len(nsList.Items))
	for i := range nsList.Items {
		namespaces[i] = &nsList.Items[i]
	}

	return namespaces, nil
}

func (a *Analyzer) getAllPersistentVolumeClaims(ctx context.Context) ([]*v1.PersistentVolumeClaim, error) {
	pvcList, err := a.k8sClient.GetCoreClient().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pvcs := make([]*v1.PersistentVolumeClaim, len(pvcList.Items))
	for i := range pvcList.Items {
		pvcs[i] = &pvcList.Items[i]
	}

	return pvcs, nil
}

func (a *Analyzer) getPersistentVolumeClaimsInNamespace(ctx context.Context, namespace string) ([]*v1.PersistentVolumeClaim, error) {
	pvcList, err := a.k8sClient.GetCoreClient().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pvcs := make([]*v1.PersistentVolumeClaim, len(pvcList.Items))
	for i := range pvcList.Items {
		pvcs[i] = &pvcList.Items[i]
	}

	return pvcs, nil
}

// SortVolumesByAge sorts volumes by creation time (oldest first)
func SortVolumesByAge(pvs []*v1.PersistentVolume) {
	sort.Slice(pvs, func(i, j int) bool {
		return pvs[i].CreationTimestamp.Time.Before(pvs[j].CreationTimestamp.Time)
	})
}

// SortPVCsByAge sorts PVCs by creation time (oldest first)
func SortPVCsByAge(pvcs []*v1.PersistentVolumeClaim) {
	sort.Slice(pvcs, func(i, j int) bool {
		return pvcs[i].CreationTimestamp.Time.Before(pvcs[j].CreationTimestamp.Time)
	})
}

// FilterVolumesBySize filters volumes by minimum size requirement
func FilterVolumesBySize(pvs []*v1.PersistentVolume, minSize string) ([]*v1.PersistentVolume, error) {
	if minSize == "" {
		return pvs, nil
	}

	minQuantity, err := resource.ParseQuantity(minSize)
	if err != nil {
		return nil, fmt.Errorf("invalid size format %s: %w", minSize, err)
	}

	var filtered []*v1.PersistentVolume
	for _, pv := range pvs {
		if storage, ok := pv.Spec.Capacity[v1.ResourceStorage]; ok {
			if storage.Cmp(minQuantity) >= 0 {
				filtered = append(filtered, pv)
			}
		}
	}

	return filtered, nil
}

// findPodsUsingPVC finds all pods that are using a specific PVC
func (a *Analyzer) findPodsUsingPVC(ctx context.Context, pvc *v1.PersistentVolumeClaim) ([]string, error) {
	podList, err := a.k8sClient.GetCoreClient().Pods(pvc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", pvc.Namespace, err)
	}

	var podNames []string
	for _, pod := range podList.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil &&
				volume.PersistentVolumeClaim.ClaimName == pvc.Name {
				podNames = append(podNames, pod.Name)
				break
			}
		}
	}

	return podNames, nil
}
