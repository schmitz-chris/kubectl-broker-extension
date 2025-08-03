package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"
	"kubectl-broker/pkg"
	"kubectl-broker/pkg/health"
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

func main() {
	var rootCmd = &cobra.Command{
		Use:   "kubectl-broker",
		Short: "Health diagnostics for HiveMQ broker clusters on Kubernetes",
		Long: `kubectl-broker is a kubectl plugin that streamlines health diagnostics 
for HiveMQ clusters running on Kubernetes. It automates the process of checking 
the health status of broker nodes via port-forwarding.`,
		RunE: runHealthCheck,
	}

	// Add flags
	rootCmd.Flags().StringVar(&statefulSetName, "statefulset", "", "Name of the StatefulSet to check (defaults to 'broker')")
	rootCmd.Flags().StringVar(&podName, "pod", "", "Name of the pod to check (for single pod mode)")
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace (defaults to current kubectl context)")
	rootCmd.Flags().IntVarP(&port, "port", "p", 0, "Port number to use for health check (overrides auto-discovery)")
	rootCmd.Flags().BoolVar(&discover, "discover", false, "Discover available broker pods and namespaces")

	// Health output format flags
	rootCmd.Flags().BoolVar(&outputJSON, "json", false, "Output raw JSON response for external parsing")
	rootCmd.Flags().BoolVar(&outputRaw, "raw", false, "Output unprocessed health response")
	rootCmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed component breakdown")
	rootCmd.Flags().StringVar(&endpoint, "endpoint", "health", "Health endpoint to query (health, liveness, readiness)")

	// Apply intelligent defaults and validate flags
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Validate output format flags
		if outputJSON && outputRaw {
			return fmt.Errorf("cannot use both --json and --raw flags together")
		}

		if !discover {
			// Apply intelligent defaults
			if statefulSetName == "" && podName == "" {
				// Default to StatefulSet "broker"
				statefulSetName = "broker"
				if !outputJSON && !outputRaw && detailed {
					fmt.Printf("Using default StatefulSet: %s\n", statefulSetName)
				}
			}

			if statefulSetName != "" && podName != "" {
				return fmt.Errorf("cannot use both --statefulset and --pod flags together")
			}

			if namespace == "" {
				// Default to namespace from kubectl context
				defaultNamespace, err := pkg.GetDefaultNamespace()
				if err != nil {
					return fmt.Errorf("failed to determine default namespace: %w\n\nPlease either:\n- Set a kubectl context with namespace: kubectl config set-context --current --namespace=<namespace>\n- Specify namespace explicitly: --namespace <namespace>", err)
				}
				namespace = defaultNamespace
				if !outputJSON && !outputRaw && detailed {
					fmt.Printf("Using namespace from context: %s\n", namespace)
				}
			}
		}
		return nil
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runHealthCheck(cmd *cobra.Command, args []string) error {
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
	}

	// Perform concurrent health checks
	return k8sClient.PerformConcurrentHealthChecks(ctx, pods, int32(port), options)
}

func runSinglePodHealthCheck(ctx context.Context, k8sClient *pkg.K8sClient) error {
	if !outputJSON && !outputRaw && detailed {
		fmt.Printf("Checking health of pod %s in namespace %s\n", podName, namespace)
	}

	// 2. Get the pod
	pod, err := k8sClient.GetPod(ctx, namespace, podName)
	if err != nil {
		return pkg.EnhanceError(err, fmt.Sprintf("pod %s in namespace %s", podName, namespace))
	}

	// 3. Validate pod status
	if err := pkg.ValidatePodStatus(pod); err != nil {
		return err
	}

	// 4. Discover health port (if not specified with --port)
	var healthPort int32
	if port > 0 {
		healthPort = int32(port)
		if !outputJSON && !outputRaw && detailed {
			fmt.Printf("Using specified port: %d\n", healthPort)
		}
	} else {
		healthPort, err = k8sClient.DiscoverHealthPort(pod)
		if err != nil {
			return err
		}
		if !outputJSON && !outputRaw && detailed {
			fmt.Printf("Discovered health port: %d\n", healthPort)
		}
	}

	// 5. Get a random local port for port-forwarding
	localPort, err := getRandomPort()
	if err != nil {
		return fmt.Errorf("failed to get available local port: %w", err)
	}

	// Create health options
	options := health.HealthCheckOptions{
		Endpoint:   endpoint,
		OutputJSON: outputJSON,
		OutputRaw:  outputRaw,
		Detailed:   detailed,
		Timeout:    10 * time.Second,
	}

	// 6. Establish port-forward connection and perform health check
	pf := pkg.NewPortForwarder(k8sClient.GetConfig(), k8sClient.GetRESTClient())
	parsedHealth, rawJSON, err := pf.PerformHealthCheckWithOptions(ctx, pod, healthPort, localPort, options)
	if err != nil {
		return pkg.EnhanceError(err, "health check")
	}

	// Display results based on output format
	if options.OutputRaw {
		fmt.Print(string(rawJSON))
	} else if options.OutputJSON {
		fmt.Println(string(rawJSON))
	} else if options.Detailed && parsedHealth != nil {
		fmt.Printf("Pod: %s\n", pod.Name)
		fmt.Printf("Overall Health: %s\n", health.FormatHealthStatus(parsedHealth.OverallStatus))
		if len(parsedHealth.ComponentDetails) > 0 {
			fmt.Println("Components:")
			for _, comp := range parsedHealth.ComponentDetails {
				fmt.Printf("  - %s: %s", comp.Name, health.FormatHealthStatus(comp.Status))
				if comp.Details != "" {
					fmt.Printf(" (%s)", comp.Details)
				}
				fmt.Println()
			}
		}
	} else {
		// Standard output
		if parsedHealth != nil {
			fmt.Printf("Health check successful: %s\n", health.FormatHealthStatus(parsedHealth.OverallStatus))
			fmt.Printf("Summary: %s\n", health.GetHealthSummary(parsedHealth))
		} else {
			fmt.Println("Health check completed")
		}
	}

	return nil
}

// getRandomPort returns a random available port
func getRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
