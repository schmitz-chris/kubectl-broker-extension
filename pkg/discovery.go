package pkg

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DiscoverBrokers finds potential HiveMQ broker pods across all accessible namespaces
func (k *K8sClient) DiscoverBrokers(ctx context.Context) error {
	fmt.Println("Discovering broker pods across all accessible namespaces...")

	// Get all namespaces
	namespaces, err := k.coreClient.Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	fmt.Printf("Checking %d namespaces...\n\n", len(namespaces.Items))

	foundBrokers := false

	for _, ns := range namespaces.Items {
		// Skip system namespaces
		if strings.HasPrefix(ns.Name, "kube-") || strings.HasPrefix(ns.Name, "kubernetes-") {
			continue
		}

		// Look for pods with "broker" in the name or labels
		pods, err := k.coreClient.Pods(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Skip namespaces we can't access
			continue
		}

		brokerPods := []string{}
		for _, pod := range pods.Items {
			if isBrokerPod(pod.Name, pod.Labels) {
				brokerPods = append(brokerPods, pod.Name)
			}
		}

		if len(brokerPods) > 0 {
			foundBrokers = true
			fmt.Printf("Namespace: %s\n", ns.Name)
			for _, podName := range brokerPods {
				fmt.Printf("  - %s\n", podName)
			}
			fmt.Printf("  Single pod: kubectl broker --pod %s --namespace %s\n", brokerPods[0], ns.Name)
			if len(brokerPods) > 1 {
				fmt.Printf("  All pods:   kubectl broker --statefulset broker --namespace %s\n", ns.Name)
			}
			fmt.Println()
		}
	}

	if !foundBrokers {
		fmt.Println("No broker pods found. Looking for StatefulSets named 'broker'...")

		for _, ns := range namespaces.Items {
			if strings.HasPrefix(ns.Name, "kube-") || strings.HasPrefix(ns.Name, "kubernetes-") {
				continue
			}

			statefulSets, err := k.appsClient.StatefulSets(ns.Name).List(ctx, metav1.ListOptions{})
			if err != nil {
				continue
			}

			for _, sts := range statefulSets.Items {
				if strings.Contains(strings.ToLower(sts.Name), "broker") {
					foundBrokers = true
					fmt.Printf("StatefulSet: %s (namespace: %s)\n", sts.Name, ns.Name)
					if sts.Status.Replicas > 0 {
						fmt.Printf("  Example pod: %s-0\n", sts.Name)
						fmt.Printf("  Single pod: kubectl broker --pod %s-0 --namespace %s\n", sts.Name, ns.Name)
						fmt.Printf("  All pods:   kubectl broker --statefulset %s --namespace %s\n\n", sts.Name, ns.Name)
					}
				}
			}
		}
	}

	if !foundBrokers {
		fmt.Println("No broker resources found. You may need to:")
		fmt.Println("1. Check if you're connected to the right cluster")
		fmt.Println("2. Verify your kubeconfig has access to the namespaces containing brokers")
		fmt.Println("3. Use specific pod and namespace names if brokers don't follow naming conventions")
	}

	return nil
}

// isBrokerPod checks if a pod looks like a HiveMQ broker based on name and labels
func isBrokerPod(name string, labels map[string]string) bool {
	// Check name patterns
	lowerName := strings.ToLower(name)
	if strings.Contains(lowerName, "broker") || strings.Contains(lowerName, "hivemq") {
		return true
	}

	// Check labels
	for key, value := range labels {
		lowerKey := strings.ToLower(key)
		lowerValue := strings.ToLower(value)

		if strings.Contains(lowerKey, "broker") || strings.Contains(lowerKey, "hivemq") ||
			strings.Contains(lowerValue, "broker") || strings.Contains(lowerValue, "hivemq") {
			return true
		}
	}

	return false
}
