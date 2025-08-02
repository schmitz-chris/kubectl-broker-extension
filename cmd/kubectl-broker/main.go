package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"
	"kubectl-broker/pkg"
)

var (
	podName   string
	namespace string
	port      int
	discover  bool
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
	rootCmd.Flags().StringVar(&podName, "pod", "", "Name of the pod to check (required)")
	rootCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace of the pod (required)")
	rootCmd.Flags().IntVarP(&port, "port", "p", 0, "Port number to use for health check (overrides auto-discovery)")
	rootCmd.Flags().BoolVar(&discover, "discover", false, "Discover available broker pods and namespaces")

	// Mark required flags conditionally
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if !discover {
			if podName == "" {
				return fmt.Errorf("--pod is required when not using --discover")
			}
			if namespace == "" {
				return fmt.Errorf("--namespace is required when not using --discover")
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
	k8sClient, err := pkg.NewK8sClient()
	if err != nil {
		return pkg.EnhanceError(err, "Kubernetes client initialization")
	}
	
	// Handle discovery mode
	if discover {
		return k8sClient.DiscoverBrokers(ctx)
	}
	
	fmt.Printf("Checking health of pod %s in namespace %s\n", podName, namespace)
	
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
		fmt.Printf("Using specified port: %d\n", healthPort)
	} else {
		healthPort, err = k8sClient.DiscoverHealthPort(pod)
		if err != nil {
			return err
		}
		fmt.Printf("Discovered health port: %d\n", healthPort)
	}
	
	// 5. Get a random local port for port-forwarding
	localPort, err := getRandomPort()
	if err != nil {
		return fmt.Errorf("failed to get available local port: %w", err)
	}
	
	// 6. Establish port-forward connection and perform health check
	pf := pkg.NewPortForwarder(k8sClient.GetConfig(), k8sClient.GetClientset())
	if err := pf.ForwardPort(ctx, pod, healthPort, localPort); err != nil {
		return pkg.EnhanceError(err, "port forwarding")
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