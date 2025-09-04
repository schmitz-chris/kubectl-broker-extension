package volumes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubectl-broker/pkg"
)

// VolumeUsage represents the usage statistics for a volume
type VolumeUsage struct {
	VolumeName     string
	PodName        string
	Namespace      string
	UsedBytes      int64
	AvailableBytes int64
	CapacityBytes  int64
	UsagePercent   float64
	Timestamp      time.Time
}

// NodeStatsResponse represents the response from kubelet stats/summary endpoint
type NodeStatsResponse struct {
	Node NodeStats  `json:"node"`
	Pods []PodStats `json:"pods"`
}

// NodeStats represents node-level statistics
type NodeStats struct {
	NodeName         string            `json:"nodeName"`
	SystemContainers []SystemContainer `json:"systemContainers"`
	StartTime        time.Time         `json:"startTime"`
	CPU              ResourceUsage     `json:"cpu"`
	Memory           ResourceUsage     `json:"memory"`
	Network          NetworkUsage      `json:"network"`
	Fs               FsUsage           `json:"fs"`
	Runtime          RuntimeUsage      `json:"runtime"`
}

// PodStats represents pod-level statistics
type PodStats struct {
	PodRef           PodReference    `json:"podRef"`
	StartTime        time.Time       `json:"startTime"`
	Containers       []ContainerStat `json:"containers"`
	CPU              ResourceUsage   `json:"cpu"`
	Memory           ResourceUsage   `json:"memory"`
	Network          NetworkUsage    `json:"network"`
	VolumeStats      []VolumeStat    `json:"volume,omitempty"`
	EphemeralStorage FsUsage         `json:"ephemeral-storage,omitempty"`
}

// PodReference represents a reference to a pod
type PodReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UID       string `json:"uid"`
}

// ContainerStat represents container-level statistics
type ContainerStat struct {
	Name      string        `json:"name"`
	StartTime time.Time     `json:"startTime"`
	CPU       ResourceUsage `json:"cpu"`
	Memory    ResourceUsage `json:"memory"`
	Rootfs    FsUsage       `json:"rootfs"`
	Logs      FsUsage       `json:"logs"`
}

// SystemContainer represents system container statistics
type SystemContainer struct {
	Name      string        `json:"name"`
	StartTime time.Time     `json:"startTime"`
	CPU       ResourceUsage `json:"cpu"`
	Memory    ResourceUsage `json:"memory"`
}

// VolumeStat represents volume-level statistics
type VolumeStat struct {
	Time           time.Time     `json:"time"`
	AvailableBytes *int64        `json:"availableBytes,omitempty"`
	CapacityBytes  *int64        `json:"capacityBytes,omitempty"`
	UsedBytes      *int64        `json:"usedBytes,omitempty"`
	InodesFree     *int64        `json:"inodesFree,omitempty"`
	Inodes         *int64        `json:"inodes,omitempty"`
	InodesUsed     *int64        `json:"inodesUsed,omitempty"`
	Name           string        `json:"name"`
	PVCRef         *PVCReference `json:"pvcRef,omitempty"`
}

// PVCReference represents a reference to a PVC
type PVCReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// ResourceUsage represents CPU/Memory usage statistics
type ResourceUsage struct {
	Time                 time.Time `json:"time"`
	UsageBytes           *int64    `json:"usageBytes,omitempty"`
	UsageNanoSeconds     *int64    `json:"usageNanoSeconds,omitempty"`
	UsageCoreNanoSeconds *int64    `json:"usageCoreNanoSeconds,omitempty"`
}

// NetworkUsage represents network usage statistics
type NetworkUsage struct {
	Time     time.Time `json:"time"`
	RxBytes  *int64    `json:"rxBytes,omitempty"`
	RxErrors *int64    `json:"rxErrors,omitempty"`
	TxBytes  *int64    `json:"txBytes,omitempty"`
	TxErrors *int64    `json:"txErrors,omitempty"`
}

// FsUsage represents filesystem usage statistics
type FsUsage struct {
	Time           time.Time `json:"time"`
	AvailableBytes *int64    `json:"availableBytes,omitempty"`
	CapacityBytes  *int64    `json:"capacityBytes,omitempty"`
	UsedBytes      *int64    `json:"usedBytes,omitempty"`
	InodesFree     *int64    `json:"inodesFree,omitempty"`
	Inodes         *int64    `json:"inodes,omitempty"`
	InodesUsed     *int64    `json:"inodesUsed,omitempty"`
}

// RuntimeUsage represents container runtime usage statistics
type RuntimeUsage struct {
	Time           time.Time `json:"time"`
	AvailableBytes *int64    `json:"availableBytes,omitempty"`
	CapacityBytes  *int64    `json:"capacityBytes,omitempty"`
	UsedBytes      *int64    `json:"usedBytes,omitempty"`
	InodesFree     *int64    `json:"inodesFree,omitempty"`
	Inodes         *int64    `json:"inodes,omitempty"`
	InodesUsed     *int64    `json:"inodesUsed,omitempty"`
}

// VolumeUsageCollector collects volume usage statistics using Node Stats API
type VolumeUsageCollector struct {
	k8sClient *pkg.K8sClient
}

// NewVolumeUsageCollector creates a new volume usage collector
func NewVolumeUsageCollector(k8sClient *pkg.K8sClient) *VolumeUsageCollector {
	return &VolumeUsageCollector{
		k8sClient: k8sClient,
	}
}

// GetVolumeUsage retrieves volume usage statistics for PVCs in a namespace
func (c *VolumeUsageCollector) GetVolumeUsage(ctx context.Context, namespace string) (map[string]*VolumeUsage, error) {
	usage := make(map[string]*VolumeUsage)

	// Get all nodes in the cluster
	nodeList, err := c.k8sClient.GetCoreClient().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Get stats from each node
	for _, node := range nodeList.Items {
		nodeUsage, err := c.getNodeVolumeStats(ctx, node.Name, namespace)
		if err != nil {
			// Log error but continue with other nodes
			fmt.Printf("Warning: failed to get stats from node %s: %v\n", node.Name, err)
			continue
		}

		// Merge node usage data
		for pvcName, volumeUsage := range nodeUsage {
			usage[pvcName] = volumeUsage
		}
	}

	return usage, nil
}

// getNodeVolumeStats retrieves volume statistics from a specific node
func (c *VolumeUsageCollector) getNodeVolumeStats(ctx context.Context, nodeName, namespace string) (map[string]*VolumeUsage, error) {
	usage := make(map[string]*VolumeUsage)

	// Create request to kubelet stats endpoint
	statsPath := fmt.Sprintf("/api/v1/nodes/%s/proxy/stats/summary", nodeName)

	// Get stats from kubelet via proxy
	data, err := c.k8sClient.GetCoreClient().RESTClient().Get().
		AbsPath(statsPath).
		DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats from node %s: %w", nodeName, err)
	}

	// Parse the response
	var stats NodeStatsResponse
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats response from node %s: %w", nodeName, err)
	}

	// Process pod volume statistics
	for _, podStat := range stats.Pods {
		// Skip pods not in the target namespace
		if namespace != "" && podStat.PodRef.Namespace != namespace {
			continue
		}

		// Process volume statistics for this pod
		for _, volumeStat := range podStat.VolumeStats {
			// Skip volumes without PVC reference
			if volumeStat.PVCRef == nil {
				continue
			}

			// Create usage entry
			pvcName := volumeStat.PVCRef.Name
			volumeUsage := &VolumeUsage{
				VolumeName: volumeStat.Name,
				PodName:    podStat.PodRef.Name,
				Namespace:  podStat.PodRef.Namespace,
				Timestamp:  volumeStat.Time,
			}

			// Set usage statistics
			if volumeStat.UsedBytes != nil {
				volumeUsage.UsedBytes = *volumeStat.UsedBytes
			}
			if volumeStat.AvailableBytes != nil {
				volumeUsage.AvailableBytes = *volumeStat.AvailableBytes
			}
			if volumeStat.CapacityBytes != nil {
				volumeUsage.CapacityBytes = *volumeStat.CapacityBytes
			}

			// Calculate usage percentage
			if volumeUsage.CapacityBytes > 0 {
				volumeUsage.UsagePercent = float64(volumeUsage.UsedBytes) / float64(volumeUsage.CapacityBytes) * 100.0
			}

			usage[pvcName] = volumeUsage
		}
	}

	return usage, nil
}

// GetAllVolumeUsage retrieves volume usage statistics for all PVCs across all namespaces
func (c *VolumeUsageCollector) GetAllVolumeUsage(ctx context.Context) (map[string]*VolumeUsage, error) {
	return c.GetVolumeUsage(ctx, "")
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
