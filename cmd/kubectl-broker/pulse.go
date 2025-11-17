package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubectl-broker/pkg"
	"kubectl-broker/pkg/health"
)

var (
	pulseNamespace  string
	pulseOutputJSON bool
	pulseOutputRaw  bool
	pulseDetailed   bool
	pulseEndpoint   string
	pulseDiscover   bool
	pulsePort       int
)

func newPulseCommand() *cobra.Command {
	var pulseCmd = &cobra.Command{
		Use:   "pulse",
		Short: "HiveMQ Pulse server status and diagnostics",
		Long: `Pulse command performs health diagnostics for HiveMQ Pulse servers running 
on Kubernetes. It checks the liveness and readiness endpoints of Pulse server pods
using the app.kubernetes.io/name=hivemq-pulse-server label selector.`,
	}

	// Add status subcommand
	pulseCmd.AddCommand(newPulseStatusCommand())

	return pulseCmd
}

func newPulseStatusCommand() *cobra.Command {
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check HiveMQ Pulse server health status",
		Long: `Status command performs health diagnostics for HiveMQ Pulse servers. It discovers
Pulse server pods using the app.kubernetes.io/name=hivemq-pulse-server label and checks
their liveness and readiness endpoints on the internal-http port.

Examples:
  # Check status with current namespace context
  kubectl broker pulse status

  # Check specific namespace
  kubectl broker pulse status --namespace pulse

  # Discovery mode to find Pulse servers
  kubectl broker pulse status --discover

  # Check readiness endpoint instead of liveness
  kubectl broker pulse status --endpoint readiness

  # Get detailed output with debug information
  kubectl broker pulse status --detailed

  # Get JSON output for external tools
  kubectl broker pulse status --json`,
		RunE: runPulseStatus,
	}

	// Add flags
	statusCmd.Flags().StringVarP(&pulseNamespace, "namespace", "n", "", "Namespace (defaults to current kubectl context)")
	statusCmd.Flags().BoolVar(&pulseDiscover, "discover", false, "Discover available Pulse server pods and namespaces")
	statusCmd.Flags().IntVarP(&pulsePort, "port", "p", 0, "Port number to use for health check (overrides auto-discovery)")

	// Health output format flags
	statusCmd.Flags().BoolVar(&pulseOutputJSON, "json", false, "Output raw JSON response for external parsing")
	statusCmd.Flags().BoolVar(&pulseOutputRaw, "raw", false, "Output unprocessed health response")
	statusCmd.Flags().BoolVar(&pulseDetailed, "detailed", false, "Show detailed component breakdown")
	statusCmd.Flags().StringVar(&pulseEndpoint, "endpoint", "liveness", "Health endpoint to query (liveness, readiness)")

	// Apply intelligent defaults and validate flags
	statusCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if err := mutuallyExclusive(pulseOutputJSON, "--json", pulseOutputRaw, "--raw"); err != nil {
			return err
		}

		// Validate endpoint (pulse-specific)
		if pulseEndpoint != "liveness" && pulseEndpoint != "readiness" {
			return fmt.Errorf("endpoint must be either 'liveness' or 'readiness'")
		}

		if !pulseDiscover {
			resolvedNamespace, fromContext, err := resolveNamespace(pulseNamespace, false)
			if err != nil {
				return err
			}
			pulseNamespace = resolvedNamespace
			if fromContext && !pulseOutputJSON && !pulseOutputRaw && pulseDetailed {
				fmt.Printf("Using namespace from context: %s\n", pulseNamespace)
			}
		}
		return nil
	}

	return statusCmd
}

func runPulseStatus(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Initialize Kubernetes client
	k8sClient, err := pkg.NewK8sClient(pulseDetailed && !pulseOutputJSON && !pulseOutputRaw)
	if err != nil {
		return pkg.EnhanceError(err, "Kubernetes client initialization")
	}

	// Handle discovery mode
	if pulseDiscover {
		return discoverPulseServers(ctx, k8sClient)
	}

	return runPulseHealthCheck(ctx, k8sClient)
}

func discoverPulseServers(ctx context.Context, k8sClient *pkg.K8sClient) error {
	labelSelector := "app.kubernetes.io/name=hivemq-pulse-server"

	fmt.Printf("Discovering HiveMQ Pulse servers with label: %s\n\n", labelSelector)

	// Get all namespaces using the core client
	coreClient := k8sClient.GetCoreClient()
	namespaceList, err := coreClient.Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespaces: %w", err)
	}

	found := false
	for _, ns := range namespaceList.Items {
		// Skip system namespaces
		if strings.HasPrefix(ns.Name, "kube-") || strings.HasPrefix(ns.Name, "kubernetes-") {
			continue
		}

		// Get pods with label selector using the core client
		podList, err := coreClient.Pods(ns.Name).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			continue // Skip namespaces we can't access
		}

		if len(podList.Items) > 0 {
			found = true
			fmt.Printf("Namespace: %s\n", ns.Name)
			for _, pod := range podList.Items {
				status := "Unknown"
				if pod.Status.Phase != "" {
					status = string(pod.Status.Phase)
				}
				fmt.Printf("  Pod: %s (Status: %s)\n", pod.Name, status)
			}
			fmt.Printf("  Usage: kubectl broker pulse status --namespace %s\n", ns.Name)
			fmt.Println()
		}
	}

	if !found {
		fmt.Printf("No HiveMQ Pulse server pods found with label: %s\n", labelSelector)
		fmt.Printf("Searched across all accessible namespaces.\n\n")
		fmt.Printf("If your Pulse servers use different labels, you may need to use the broker status command instead:\n")
		fmt.Printf("  kubectl broker status --discover\n")
	}

	return nil
}

func runPulseHealthCheck(ctx context.Context, k8sClient *pkg.K8sClient) error {
	labelSelector := "app.kubernetes.io/name=hivemq-pulse-server"

	if !pulseOutputJSON && !pulseOutputRaw && pulseDetailed {
		fmt.Printf("Checking health of HiveMQ Pulse servers in namespace %s\n", pulseNamespace)
		fmt.Printf("Using label selector: %s\n", labelSelector)
	}

	// Get all Pulse server pods using label selector via core client
	coreClient := k8sClient.GetCoreClient()
	podList, err := coreClient.Pods(pulseNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return pkg.EnhanceError(err, fmt.Sprintf("failed to get Pulse server pods in namespace %s", pulseNamespace))
	}

	if len(podList.Items) == 0 {
		return fmt.Errorf("no HiveMQ Pulse server pods found with label %s in namespace %s\n\nTry:\n- Using discovery mode: kubectl broker pulse status --discover\n- Checking a different namespace: kubectl broker pulse status --namespace <namespace>\n- Using broker status instead: kubectl broker status --discover", labelSelector, pulseNamespace)
	}

	if !pulseOutputJSON && !pulseOutputRaw && pulseDetailed {
		fmt.Printf("Found %d Pulse server pods\n\n", len(podList.Items))
	}

	// Convert to slice of pod pointers for compatibility with PerformConcurrentHealthChecks
	pods := make([]*v1.Pod, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &podList.Items[i]
	}

	// Create health options
	options := health.HealthCheckOptions{
		Endpoint:   pulseEndpoint,
		OutputJSON: pulseOutputJSON,
		OutputRaw:  pulseOutputRaw,
		Detailed:   pulseDetailed,
		Timeout:    10 * time.Second,
		UseColors:  !pulseOutputJSON && !pulseOutputRaw, // Disable colors for JSON/raw output
	}

	// Use the internal-http port for Pulse servers
	portName := "internal-http"

	// Discover the port by name instead of hardcoded port number
	var healthPort int32
	if pulsePort > 0 {
		healthPort = int32(pulsePort)
		if !pulseOutputJSON && !pulseOutputRaw && pulseDetailed {
			fmt.Printf("Using specified port: %d\n", healthPort)
		}
	} else {
		// Find the internal-http port from the first pod (assuming all pods have the same port config)
		found := false
		for _, container := range pods[0].Spec.Containers {
			for _, port := range container.Ports {
				if port.Name == portName {
					healthPort = port.ContainerPort
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			var availablePorts []string
			for _, container := range pods[0].Spec.Containers {
				for _, port := range container.Ports {
					portInfo := fmt.Sprintf("%s(%d)", port.Name, port.ContainerPort)
					availablePorts = append(availablePorts, portInfo)
				}
			}

			if len(availablePorts) == 0 {
				return fmt.Errorf("could not find port named '%s' in Pulse server pod %s\n\nNo container ports found. Use --port/-p to specify manually", portName, pods[0].Name)
			}

			return fmt.Errorf("could not find port named '%s' in Pulse server pod %s\n\nAvailable ports: %v\nUse --port/-p to specify manually", portName, pods[0].Name, availablePorts)
		}

		if !pulseOutputJSON && !pulseOutputRaw && pulseDetailed {
			fmt.Printf("Discovered port '%s': %d\n", portName, healthPort)
		}
	}

	// Perform concurrent health checks
	return k8sClient.PerformConcurrentHealthChecks(ctx, pods, healthPort, options)
}
