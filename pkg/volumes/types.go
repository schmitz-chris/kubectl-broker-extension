package volumes

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// AnalysisOptions contains options for volume analysis
type AnalysisOptions struct {
	Namespace     string        // Target namespace (empty for current context)
	AllNamespaces bool          // Analyze across all namespaces
	MinAge        time.Duration // Only include volumes older than this
	MinSize       string        // Only include volumes larger than this
	ShowReleased  bool          // Show only released PVs
	ShowOrphaned  bool          // Show only orphaned PVCs
	ShowAll       bool          // Show all volumes including bound ones
	ShowDetailed  bool          // Show detailed usage information (enables Node Stats API)
	UseColors     bool          // Use color output
}

// CleanupOptions contains options for volume cleanup
type CleanupOptions struct {
	Namespace     string        // Target namespace (empty for current context)
	AllNamespaces bool          // Cleanup across all namespaces
	MinAge        time.Duration // Only delete volumes older than this
	MinSize       string        // Only delete volumes larger than this
	DryRun        bool          // Preview only, don't actually delete
	Force         bool          // Skip confirmation prompts
	UseColors     bool          // Use color output
}

// VolumeInfo represents a volume with analysis metadata
type VolumeInfo struct {
	PV             *v1.PersistentVolume
	PVC            *v1.PersistentVolumeClaim
	Type           VolumeType
	Status         VolumeStatus
	Size           resource.Quantity
	Age            time.Duration
	Namespace      string
	AssociatedPods []string
	IsHiveMQVolume bool
	ReclaimPolicy  v1.PersistentVolumeReclaimPolicy
	StorageClass   string
	// Volume usage information
	Usage *VolumeUsage
}

// VolumeType represents the type of volume issue
type VolumeType int

const (
	VolumeTypeUnknown     VolumeType = iota
	VolumeTypeReleasedPV             // PV with status=Released
	VolumeTypeOrphanedPVC            // PVC without associated pods
	VolumeTypeUnboundPVC             // PVC in pending state
	VolumeTypeBound                  // Normal bound volume
)

func (vt VolumeType) String() string {
	switch vt {
	case VolumeTypeReleasedPV:
		return "RELEASED_PV"
	case VolumeTypeOrphanedPVC:
		return "ORPHANED_PVC"
	case VolumeTypeUnboundPVC:
		return "UNBOUND_PVC"
	case VolumeTypeBound:
		return "BOUND"
	default:
		return "UNKNOWN"
	}
}

// VolumeStatus represents the current status of a volume
type VolumeStatus int

const (
	VolumeStatusUnknown  VolumeStatus = iota
	VolumeStatusReleased              // PV is released and can be deleted
	VolumeStatusOrphaned              // PVC exists but no pods are using it
	VolumeStatusUnbound               // PVC is pending binding
	VolumeStatusBound                 // PVC is bound and in use
	VolumeStatusPending               // PVC is waiting for provisioning
)

func (vs VolumeStatus) String() string {
	switch vs {
	case VolumeStatusReleased:
		return "RELEASED"
	case VolumeStatusOrphaned:
		return "ORPHANED"
	case VolumeStatusUnbound:
		return "UNBOUND"
	case VolumeStatusBound:
		return "BOUND"
	case VolumeStatusPending:
		return "PENDING"
	default:
		return "UNKNOWN"
	}
}

// AnalysisResult contains the results of volume analysis
type AnalysisResult struct {
	ReleasedPVs             []*v1.PersistentVolume
	OrphanedPVCs            []*v1.PersistentVolumeClaim
	UnboundPVCs             []*v1.PersistentVolumeClaim
	BoundVolumes            []VolumeInfo
	TotalPVs                int
	TotalPVCs               int
	TotalReclaimableStorage int64
	NamespaceStats          map[string]*NamespaceVolumeStats
	HiveMQVolumeCount       int
	Recommendations         []string
}

// NamespaceVolumeStats contains volume statistics for a namespace
type NamespaceVolumeStats struct {
	Namespace         string
	ReleasedPVs       int
	OrphanedPVCs      int
	TotalReclaimable  int64
	HiveMQVolumes     int
	IsHiveMQNamespace bool
	NamespaceExists   bool
}

// CleanupResult contains the results of volume cleanup operation
type CleanupResult struct {
	DeletedPVs              []string
	DeletedPVCs             []string
	FailedDeletions         []CleanupError
	TotalReclaimedStorage   int64
	PlannedReclaimedStorage int64
	DryRunPreview           []CleanupAction
	PlannedReleasedPVs      int
	PlannedOrphanedPVCs     int
	DeletedReleasedPVs      int
	DeletedOrphanedPVCs     int
	AssociatedPVsDeleted    int
}

// CleanupAction represents an action that would be taken during cleanup
type CleanupAction struct {
	Type      string
	Name      string
	Namespace string
	Size      int64
	Age       time.Duration
	Reason    string
}

// CleanupError represents an error that occurred during cleanup
type CleanupError struct {
	Type      string
	Name      string
	Namespace string
	Error     error
}

// HiveMQ-specific constants for pattern recognition
const (
	HiveMQPVCPattern       = "data-broker-"
	HiveMQStatefulSetName  = "broker"
	HiveMQStorageClassName = "broker-standard-1"
	MinHiveMQVolumeSize    = "5Gi"
)

// IsHiveMQVolume checks if a volume follows HiveMQ naming patterns
func IsHiveMQVolume(name, namespace string) bool {
	// Check for HiveMQ PVC pattern
	if len(name) >= len(HiveMQPVCPattern) && name[:len(HiveMQPVCPattern)] == HiveMQPVCPattern {
		return true
	}

	// Check for UUID-style namespace (HiveMQ Cloud pattern)
	if isUUIDNamespace(namespace) {
		return true
	}

	return false
}

// isUUIDNamespace checks if namespace follows UUID pattern (HiveMQ Cloud)
func isUUIDNamespace(namespace string) bool {
	// HiveMQ Cloud uses UUID namespaces like: 07379b05-4e05-46bf-b5d3-b4441252a8d1
	if len(namespace) == 36 && namespace[8] == '-' && namespace[13] == '-' &&
		namespace[18] == '-' && namespace[23] == '-' {
		return true
	}
	return false
}

// ShouldDeleteVolume determines if a volume should be deleted based on criteria
func ShouldDeleteVolume(info VolumeInfo, options CleanupOptions) bool {
	// Check volume type - only delete released/orphaned volumes
	if info.Type != VolumeTypeReleasedPV && info.Type != VolumeTypeOrphanedPVC {
		return false
	}

	// Check minimum age requirement
	if options.MinAge > 0 && info.Age < options.MinAge {
		return false
	}

	// Check minimum size requirement
	if options.MinSize != "" {
		minSize, err := resource.ParseQuantity(options.MinSize)
		if err == nil && info.Size.Cmp(minSize) < 0 {
			return false
		}
	}

	return true
}
