package pkg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// K8sClient wraps the Kubernetes clientset with helper methods
type K8sClient struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// NewK8sClient creates a new Kubernetes client using kubeconfig (supports kubie)
func NewK8sClient() (*K8sClient, error) {
	// Check for kubie environment variables first
	var kubeconfig string
	if kubieConfig := os.Getenv("KUBIE_KUBECONFIG"); kubieConfig != "" {
		kubeconfig = kubieConfig
		fmt.Printf("Using kubie kubeconfig: %s\n", kubeconfig)
	} else if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
		kubeconfig = envConfig
		fmt.Printf("Using KUBECONFIG env var: %s\n", kubeconfig)
	} else {
		// Fall back to default kubeconfig
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		fmt.Printf("Using default kubeconfig: %s\n", kubeconfig)
	}

	// Load kubeconfig with context information
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	// Get current context info for debugging
	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load raw kubeconfig: %w", err)
	}

	currentContext := rawConfig.CurrentContext
	if currentContext == "" {
		return nil, fmt.Errorf("no current context set in kubeconfig")
	}

	context, exists := rawConfig.Contexts[currentContext]
	if !exists {
		return nil, fmt.Errorf("current context '%s' not found in kubeconfig", currentContext)
	}

	cluster, exists := rawConfig.Clusters[context.Cluster]
	if !exists {
		return nil, fmt.Errorf("cluster '%s' not found in kubeconfig", context.Cluster)
	}

	fmt.Printf("Using cluster: %s\n", context.Cluster)
	fmt.Printf("Server: %s\n", cluster.Server)
	fmt.Printf("Current context: %s\n", currentContext)
	fmt.Printf("Namespace: %s\n", context.Namespace)
	fmt.Println()

	// Load config
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return &K8sClient{
		clientset: clientset,
		config:    config,
	}, nil
}

// GetPod retrieves a pod by name and namespace
func (k *K8sClient) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s in namespace %s: %w", name, namespace, err)
	}
	return pod, nil
}

// DiscoverHealthPort searches for a container port named "health" in the pod
func (k *K8sClient) DiscoverHealthPort(pod *v1.Pod) (int32, error) {
	var availablePorts []string

	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			portInfo := fmt.Sprintf("%s(%d)", port.Name, port.ContainerPort)
			availablePorts = append(availablePorts, portInfo)

			if port.Name == "health" {
				return port.ContainerPort, nil
			}
		}
	}

	if len(availablePorts) == 0 {
		return 0, fmt.Errorf("no container ports found in pod %s", pod.Name)
	}

	return 0, fmt.Errorf("health port not found. Available ports: %v. Use --port/-p to specify manually", availablePorts)
}

// GetConfig returns the Kubernetes config
func (k *K8sClient) GetConfig() *rest.Config {
	return k.config
}

// GetClientset returns the Kubernetes clientset
func (k *K8sClient) GetClientset() *kubernetes.Clientset {
	return k.clientset
}

// GetStatefulSet retrieves a StatefulSet by name and namespace
func (k *K8sClient) GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error) {
	sts, err := k.clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get StatefulSet %s in namespace %s: %w", name, namespace, err)
	}
	return sts, nil
}

// GetPodsFromStatefulSet retrieves all pods belonging to a StatefulSet using label selectors
func (k *K8sClient) GetPodsFromStatefulSet(ctx context.Context, namespace, statefulSetName string) ([]*v1.Pod, error) {
	// First, get the StatefulSet to understand its selector
	sts, err := k.GetStatefulSet(ctx, namespace, statefulSetName)
	if err != nil {
		return nil, err
	}
	
	// Create label selector from StatefulSet's selector
	labelSelector := metav1.FormatLabelSelector(sts.Spec.Selector)
	
	// Get pods matching the label selector
	podList, err := k.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods for StatefulSet %s: %w", statefulSetName, err)
	}
	
	// Convert to slice of pod pointers for easier handling
	pods := make([]*v1.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		pods = append(pods, &podList.Items[i])
	}
	
	return pods, nil
}
