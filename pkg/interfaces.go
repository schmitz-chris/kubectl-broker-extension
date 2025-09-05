package pkg

import (
	"context"
	"io"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"kubectl-broker/pkg/health"
)

// KubernetesClient defines the interface for Kubernetes operations
type KubernetesClient interface {
	// Pod operations
	GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error)
	GetPodsFromStatefulSet(ctx context.Context, namespace, statefulSetName string) ([]*v1.Pod, error)
	GetStatefulSetPods(ctx context.Context, namespace, statefulSetName string) ([]v1.Pod, error)

	// Port discovery
	DiscoverHealthPort(pod *v1.Pod) (int32, error)
	DiscoverAPIPort(pod *v1.Pod) (int32, error)

	// StatefulSet operations
	GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error)

	// Service operations
	GetAPIServiceFromStatefulSet(ctx context.Context, namespace, statefulSetName string) (*v1.Service, error)
	DiscoverServiceAPIPort(service *v1.Service) (int32, error)

	// Health check operations
	PerformConcurrentHealthChecks(ctx context.Context, pods []*v1.Pod, port int32, options health.HealthCheckOptions) error
	DiscoverBrokers(ctx context.Context) error

	// Command execution
	ExecCommand(ctx context.Context, namespace, podName string, command []string) (string, error)
	ExecCommandStream(ctx context.Context, namespace, podName string, command []string) (io.ReadCloser, error)

	// Client access
	GetConfig() *rest.Config
	GetRESTClient() rest.Interface
	GetCoreClient() *corev1client.CoreV1Client
	GetAppsClient() *appsv1client.AppsV1Client
}

// PortForwardManager defines the interface for port forwarding operations
type PortForwardManager interface {
	PerformHealthCheckWithOptions(ctx context.Context, pod *v1.Pod, remotePort int32, localPort int, options health.HealthCheckOptions) (*health.ParsedHealthData, []byte, error)
	PerformHealthCheckOnly(ctx context.Context, pod *v1.Pod, remotePort int32, localPort int) error
	ForwardPort(ctx context.Context, pod *v1.Pod, remotePort int32, localPort int) error
	PerformWithPortForwarding(ctx context.Context, pod *v1.Pod, remotePort int32, localPort int, operation func(localPort int) error) error
}

// HealthChecker defines the interface for health checking operations
type HealthChecker interface {
	CheckHealth(ctx context.Context, endpoint string, timeout time.Duration) (*health.ParsedHealthData, []byte, error)
	ParseHealthResponse(rawJSON []byte) (*health.ParsedHealthData, error)
	ValidateEndpoint(endpoint string) error
}

// BackupManager defines the interface for backup operations
type BackupManager interface {
	CreateBackup(ctx context.Context, options BackupOptions) (*BackupResult, error)
	ListBackups(ctx context.Context, namespace, statefulSetName string) ([]BackupInfo, error)
	DownloadBackup(ctx context.Context, options DownloadOptions) error
	GetBackupStatus(ctx context.Context, namespace, statefulSetName, backupID string) (*BackupStatus, error)
}

// VolumeManager defines the interface for volume operations
type VolumeManager interface {
	ListVolumes(ctx context.Context, namespace string, options VolumeListOptions) ([]VolumeInfo, error)
	GetVolumeUsage(ctx context.Context, namespace, statefulSetName string, detailed bool) (*VolumeUsageResult, error)
	CleanupVolumes(ctx context.Context, options VolumeCleanupOptions) (*CleanupResult, error)
}

// Config interfaces for better dependency injection

// ConfigLoader defines the interface for loading configuration
type ConfigLoader interface {
	LoadKubeConfig() (*rest.Config, error)
	GetDefaultNamespace() (string, error)
	GetCurrentContext() (string, error)
}

// OutputFormatter defines the interface for formatting output
type OutputFormatter interface {
	FormatHealthStatus(status health.HealthStatus, useColors bool) string
	FormatTable(headers []string, rows [][]string) string
	FormatJSON(data interface{}) ([]byte, error)
}

// Supporting types for interfaces

// BackupOptions holds options for backup creation
type BackupOptions struct {
	Namespace       string
	StatefulSetName string
	Destination     string
	Username        string
	Password        string
	Timeout         time.Duration
}

// BackupResult holds the result of a backup operation
type BackupResult struct {
	ID        string
	Status    string
	Size      int64
	CreatedAt time.Time
	Message   string
}

// BackupInfo holds information about a backup
type BackupInfo struct {
	ID        string
	Status    string
	Size      int64
	CreatedAt time.Time
}

// DownloadOptions holds options for backup download
type DownloadOptions struct {
	Namespace       string
	StatefulSetName string
	BackupID        string
	OutputDir       string
	Latest          bool
	Username        string
	Password        string
}

// BackupStatus holds the status of a backup
type BackupStatus struct {
	ID        string
	Status    string
	Size      int64
	Progress  int
	CreatedAt time.Time
	Message   string
}

// VolumeListOptions holds options for listing volumes
type VolumeListOptions struct {
	AllNamespaces bool
	StatefulSet   string
	Detailed      bool
}

// VolumeInfo holds information about a volume
type VolumeInfo struct {
	Name       string
	Namespace  string
	Size       string
	Used       string
	Available  string
	UsePercent float64
	MountPath  string
	PodName    string
}

// VolumeUsageResult holds the result of volume usage analysis
type VolumeUsageResult struct {
	TotalSize     int64
	UsedSize      int64
	AvailableSize int64
	UsePercent    float64
	Volumes       []VolumeInfo
	Summary       string
}

// VolumeCleanupOptions holds options for volume cleanup
type VolumeCleanupOptions struct {
	Namespace       string
	StatefulSetName string
	DryRun          bool
	Force           bool
	OlderThan       time.Duration
}

// CleanupResult holds the result of volume cleanup
type CleanupResult struct {
	DeletedVolumes []string
	FreedSpace     int64
	Summary        string
}
