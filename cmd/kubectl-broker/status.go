package main

import (
	"context"
	"fmt"
	"time"

	"kubectl-broker/pkg"
	"kubectl-broker/pkg/health"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
)

var (
	statefulSetName string
	podName         string
	namespace       string
	port            int
	discover        bool
	outputJSON      bool
	outputRaw       bool
	detailed        bool
	endpoint        string
)

func newStatusCommand() *cobra.Command {
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Health diagnostics for HiveMQ broker clusters",
		Long: `Status command performs health diagnostics for HiveMQ clusters running 
on Kubernetes. It automates the process of checking the health status of 
broker nodes via port-forwarding with intelligent defaults and concurrent checks.`,
		RunE: runHealthCheck,
	}

	// Add flags
	statusCmd.Flags().StringVar(&statefulSetName, "statefulset", "", "Name of the StatefulSet to check (defaults to 'broker')")
	statusCmd.Flags().StringVar(&podName, "pod", "", "Name of the pod to check (for single pod mode)")
	statusCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace (defaults to current kubectl context)")
	statusCmd.Flags().IntVarP(&port, "port", "p", 0, "Port number to use for health check (overrides auto-discovery)")
	statusCmd.Flags().BoolVar(&discover, "discover", false, "Discover available broker pods and namespaces")

	// Health output format flags
	statusCmd.Flags().BoolVar(&outputJSON, "json", false, "Output raw JSON response for external parsing")
	statusCmd.Flags().BoolVar(&outputRaw, "raw", false, "Output unprocessed health response")
	statusCmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed component breakdown")
	statusCmd.Flags().StringVar(&endpoint, "endpoint", "health", "Health endpoint to query (health, liveness, readiness)")

	// Apply intelligent defaults and validate flags
	statusCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if err := mutuallyExclusive(outputJSON, "--json", outputRaw, "--raw"); err != nil {
			return err
		}

		if !discover {
			// Apply intelligent defaults
			if statefulSetName == "" && podName == "" {
				var usedDefault bool
				statefulSetName, usedDefault = applyDefaultStatefulSet(statefulSetName)
				if usedDefault && !outputJSON && !outputRaw && detailed {
					fmt.Printf("Using default StatefulSet: %s\n", statefulSetName)
				}
			}

			if statefulSetName != "" && podName != "" {
				return fmt.Errorf("cannot use both --statefulset and --pod flags together")
			}

			resolvedNamespace, fromContext, err := resolveNamespace(namespace, false)
			if err != nil {
				return err
			}
			namespace = resolvedNamespace
			if fromContext && !outputJSON && !outputRaw && detailed {
				fmt.Printf("Using namespace from context: %s\n", namespace)
			}
		}
		return nil
	}

	return statusCmd
}

func runHealthCheck(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// 1. Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(detailed && !outputJSON && !outputRaw)
	if err != nil {
		return pkg.EnhanceError(err, "Kubernetes client initialization")
	}

	// Handle discovery mode
	if discover {
		return k8sClient.DiscoverBrokers(ctx)
	}

	// Handle StatefulSet mode (Phase 2)
	if statefulSetName != "" {
		return runStatefulSetHealthCheck(ctx, k8sClient)
	}

	// Handle single pod mode (Phase 1)
	return runSinglePodHealthCheck(ctx, k8sClient)
}

func runStatefulSetHealthCheck(ctx context.Context, k8sClient *pkg.K8sClient) error {
	if !outputJSON && !outputRaw && detailed {
		fmt.Printf("Checking health of StatefulSet %s in namespace %s\n", statefulSetName, namespace)
	}

	// Get all pods from the StatefulSet
	pods, err := k8sClient.GetPodsFromStatefulSet(ctx, namespace, statefulSetName)
	if err != nil {
		return pkg.EnhanceError(err, fmt.Sprintf("StatefulSet %s in namespace %s", statefulSetName, namespace))
	}

	if len(pods) == 0 {
		return fmt.Errorf("no pods found for StatefulSet %s in namespace %s", statefulSetName, namespace)
	}

	if !outputJSON && !outputRaw && detailed {
		fmt.Printf("Found %d pods in StatefulSet\n\n", len(pods))
	}

	// Create health options
	options := health.HealthCheckOptions{
		Endpoint:   endpoint,
		OutputJSON: outputJSON,
		OutputRaw:  outputRaw,
		Detailed:   detailed,
		Timeout:    10 * time.Second,
		UseColors:  !outputJSON && !outputRaw, // Disable colors for JSON/raw output
	}

	// Perform concurrent health checks
	return k8sClient.PerformConcurrentHealthChecks(ctx, pods, int32(port), options)
}

func runSinglePodHealthCheck(ctx context.Context, k8sClient *pkg.K8sClient) error {
	// Get and validate the pod
	pod, err := getPodAndValidate(ctx, k8sClient)
	if err != nil {
		return err
	}

	// Discover or use specified health port
	healthPort, err := resolveHealthPort(k8sClient, pod)
	if err != nil {
		return err
	}

	// Get local port and create options
	localPort, options, err := prepareHealthCheckOptions()
	if err != nil {
		return err
	}

	// Perform the health check
	parsedHealth, rawJSON, err := performHealthCheck(ctx, k8sClient, pod, healthPort, localPort, options)
	if err != nil {
		return err
	}

	// Display results
	return displayHealthCheckResults(pod, parsedHealth, rawJSON, options)
}

// getPodAndValidate retrieves and validates a pod for health checking
func getPodAndValidate(ctx context.Context, k8sClient *pkg.K8sClient) (*v1.Pod, error) {
	if shouldShowDebugInfo() {
		fmt.Printf("Checking health of pod %s in namespace %s\n", podName, namespace)
	}

	pod, err := k8sClient.GetPod(ctx, namespace, podName)
	if err != nil {
		return nil, pkg.EnhanceError(err, fmt.Sprintf("pod %s in namespace %s", podName, namespace))
	}

	if err := pkg.ValidatePodStatus(pod); err != nil {
		return nil, err
	}

	return pod, nil
}

// resolveHealthPort determines the health port to use for the health check
func resolveHealthPort(k8sClient *pkg.K8sClient, pod *v1.Pod) (int32, error) {
	var healthPort int32
	var err error

	if port > 0 {
		healthPort = int32(port)
		if shouldShowDebugInfo() {
			fmt.Printf("Using specified port: %d\n", healthPort)
		}
	} else {
		healthPort, err = k8sClient.DiscoverHealthPort(pod)
		if err != nil {
			return 0, err
		}
		if shouldShowDebugInfo() {
			fmt.Printf("Discovered health port: %d\n", healthPort)
		}
	}

	return healthPort, nil
}

// prepareHealthCheckOptions creates local port and health check options
func prepareHealthCheckOptions() (int, health.HealthCheckOptions, error) {
	localPort, err := pkg.GetRandomPort()
	if err != nil {
		return 0, health.HealthCheckOptions{}, fmt.Errorf("failed to get available local port: %w", err)
	}

	options := health.HealthCheckOptions{
		Endpoint:   endpoint,
		OutputJSON: outputJSON,
		OutputRaw:  outputRaw,
		Detailed:   detailed,
		Timeout:    10 * time.Second,
		UseColors:  !outputJSON && !outputRaw,
	}

	return localPort, options, nil
}

// performHealthCheck executes the health check using port forwarding
func performHealthCheck(ctx context.Context, k8sClient *pkg.K8sClient, pod *v1.Pod, healthPort int32, localPort int, options health.HealthCheckOptions) (*health.ParsedHealthData, []byte, error) {
	pf := pkg.NewPortForwarder(k8sClient.GetConfig(), k8sClient.GetRESTClient())
	parsedHealth, rawJSON, err := pf.PerformHealthCheckWithOptions(ctx, pod, healthPort, localPort, options)
	if err != nil {
		return nil, nil, pkg.EnhanceError(err, "health check")
	}
	return parsedHealth, rawJSON, nil
}

// displayHealthCheckResults formats and displays the health check results
func displayHealthCheckResults(pod *v1.Pod, parsedHealth *health.ParsedHealthData, rawJSON []byte, options health.HealthCheckOptions) error {
	if options.OutputRaw {
		fmt.Print(string(rawJSON))
		return nil
	}

	if options.OutputJSON {
		fmt.Println(string(rawJSON))
		return nil
	}

	if options.Detailed && parsedHealth != nil {
		return displayDetailedHealthResults(pod, parsedHealth, options)
	}

	return displayStandardHealthResults(parsedHealth, options)
}

// displayDetailedHealthResults shows detailed component breakdown
func displayDetailedHealthResults(pod *v1.Pod, parsedHealth *health.ParsedHealthData, options health.HealthCheckOptions) error {
	fmt.Printf("Pod: %s\n", pod.Name)
	fmt.Printf("Overall Health: %s\n", health.FormatHealthStatusWithColor(parsedHealth.OverallStatus, options.UseColors))

	if len(parsedHealth.ComponentDetails) > 0 {
		fmt.Println("Components:")
		for _, comp := range parsedHealth.ComponentDetails {
			displayComponentDetails(comp, options.UseColors)
		}
	}

	return nil
}

// displayComponentDetails shows details for a single component
func displayComponentDetails(comp health.ComponentStatus, useColors bool) {
	fmt.Printf("  - %s: %s", comp.Name, health.FormatHealthStatusWithColor(comp.Status, useColors))

	if comp.Details != "" {
		fmt.Printf(" (%s)", comp.Details)
	}

	// Special handling for extensions
	if comp.Name == "extensions" && len(comp.SubComponents) > 0 {
		displayExtensionDetails(comp.SubComponents, useColors)
	} else {
		fmt.Println()
	}
}

// displayExtensionDetails shows individual extension details
func displayExtensionDetails(extensions []health.ComponentStatus, useColors bool) {
	fmt.Printf(" (%d extensions)", len(extensions))
	fmt.Println()

	for _, ext := range extensions {
		fmt.Printf("    - %s: %s", ext.Name, health.FormatHealthStatusWithColor(ext.Status, useColors))
		if ext.Details != "" {
			fmt.Printf(" (%s)", ext.Details)
		}
		fmt.Println()
	}
}

// displayStandardHealthResults shows standard output format
func displayStandardHealthResults(parsedHealth *health.ParsedHealthData, options health.HealthCheckOptions) error {
	if parsedHealth != nil {
		fmt.Printf("Health check successful: %s\n", health.FormatHealthStatusWithColor(parsedHealth.OverallStatus, options.UseColors))
		fmt.Printf("Summary: %s\n", health.GetHealthSummaryWithColor(parsedHealth, options.UseColors))
	} else {
		fmt.Println("Health check completed")
	}
	return nil
}

// shouldShowDebugInfo returns whether debug information should be displayed
func shouldShowDebugInfo() bool {
	return !outputJSON && !outputRaw && detailed
}
