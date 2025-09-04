package pkg

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
)

// K8sClient wraps specific Kubernetes client interfaces with helper methods
type K8sClient struct {
	coreClient *corev1client.CoreV1Client
	appsClient *appsv1client.AppsV1Client
	restClient rest.Interface
	config     *rest.Config
}

// NewK8sClient creates a new Kubernetes client using kubeconfig (supports kubie)
func NewK8sClient(showDebug bool) (*K8sClient, error) {
	// Check for kubie environment variables first
	var kubeconfig string
	if kubieConfig := os.Getenv("KUBIE_KUBECONFIG"); kubieConfig != "" {
		kubeconfig = kubieConfig
		if showDebug {
			fmt.Printf("Using kubie kubeconfig: %s\n", kubeconfig)
		}
	} else if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
		kubeconfig = envConfig
		if showDebug {
			fmt.Printf("Using KUBECONFIG env var: %s\n", kubeconfig)
		}
	} else {
		// Fall back to default kubeconfig
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
		if showDebug {
			fmt.Printf("Using default kubeconfig: %s\n", kubeconfig)
		}
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

	if showDebug {
		fmt.Printf("Using cluster: %s\n", context.Cluster)
		fmt.Printf("Server: %s\n", cluster.Server)
		fmt.Printf("Current context: %s\n", currentContext)
		fmt.Printf("Namespace: %s\n", context.Namespace)
		fmt.Println()
	}

	// Load config
	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create specific typed clients instead of full clientset
	coreClient, err := corev1client.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create CoreV1 client: %w", err)
	}

	appsClient, err := appsv1client.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create AppsV1 client: %w", err)
	}

	// Create REST client for port-forwarding using CoreV1 configuration
	coreConfig := *config
	coreConfig.APIPath = "/api"
	coreConfig.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	coreConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	restClient, err := rest.RESTClientFor(&coreConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	return &K8sClient{
		coreClient: coreClient,
		appsClient: appsClient,
		restClient: restClient,
		config:     config,
	}, nil
}

// GetPod retrieves a pod by name and namespace
func (k *K8sClient) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	pod, err := k.coreClient.Pods(namespace).Get(ctx, name, metav1.GetOptions{})
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

// DiscoverAPIPort searches for a container port named "api" in the pod, with fallback to port 8081
func (k *K8sClient) DiscoverAPIPort(pod *v1.Pod) (int32, error) {
	var availablePorts []string

	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			portInfo := fmt.Sprintf("%s(%d)", port.Name, port.ContainerPort)
			availablePorts = append(availablePorts, portInfo)

			// First preference: named port "api"
			if port.Name == "api" {
				return port.ContainerPort, nil
			}

			// Second preference: port 8081 (common HiveMQ API port)
			if port.ContainerPort == 8081 {
				return port.ContainerPort, nil
			}
		}
	}

	if len(availablePorts) == 0 {
		return 0, fmt.Errorf("no container ports found in pod %s", pod.Name)
	}

	// If no named "api" port or port 8081 found, return error with available ports
	return 0, fmt.Errorf("API port not found (expected port named 'api' or port 8081). Available ports: %v", availablePorts)
}

// GetConfig returns the Kubernetes config
func (k *K8sClient) GetConfig() *rest.Config {
	return k.config
}

// GetRESTClient returns the REST client for port-forwarding
func (k *K8sClient) GetRESTClient() rest.Interface {
	return k.restClient
}

// GetCoreClient returns the CoreV1 client for direct API access
func (k *K8sClient) GetCoreClient() *corev1client.CoreV1Client {
	return k.coreClient
}

// GetAppsClient returns the AppsV1 client for direct API access
func (k *K8sClient) GetAppsClient() *appsv1client.AppsV1Client {
	return k.appsClient
}

// GetStatefulSet retrieves a StatefulSet by name and namespace
func (k *K8sClient) GetStatefulSet(ctx context.Context, namespace, name string) (*appsv1.StatefulSet, error) {
	sts, err := k.appsClient.StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
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
	podList, err := k.coreClient.Pods(namespace).List(ctx, metav1.ListOptions{
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

// GetAPIServiceFromStatefulSet finds the API service for a StatefulSet
func (k *K8sClient) GetAPIServiceFromStatefulSet(ctx context.Context, namespace, statefulSetName string) (*v1.Service, error) {
	// First, try to find service with standard HiveMQ naming pattern: hivemq-broker-api
	serviceName := "hivemq-broker-api"
	service, err := k.coreClient.Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err == nil {
		// Validate it has an API port
		if hasAPIPort(service) {
			return service, nil
		}
	}

	// Fallback: look for services with labels matching the StatefulSet
	sts, err := k.GetStatefulSet(ctx, namespace, statefulSetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get StatefulSet %s: %w", statefulSetName, err)
	}

	// Create label selector from StatefulSet's selector
	labelSelector := metav1.FormatLabelSelector(sts.Spec.Selector)

	// List services matching the label selector
	services, err := k.coreClient.Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list services for StatefulSet %s: %w", statefulSetName, err)
	}

	// Find a service with API port
	for i := range services.Items {
		service := &services.Items[i]
		if hasAPIPort(service) {
			return service, nil
		}
	}

	return nil, fmt.Errorf("no API service found for StatefulSet %s in namespace %s. Expected service named 'hivemq-broker-api' or service with port named 'api' or port 8081", statefulSetName, namespace)
}

// DiscoverServiceAPIPort searches for API port in a service
func (k *K8sClient) DiscoverServiceAPIPort(service *v1.Service) (int32, error) {
	var availablePorts []string

	for _, port := range service.Spec.Ports {
		portInfo := fmt.Sprintf("%s(%d)", port.Name, port.Port)
		availablePorts = append(availablePorts, portInfo)

		// First preference: named port "api"
		if port.Name == "api" {
			return port.Port, nil
		}

		// Second preference: port 8081 (common HiveMQ API port)
		if port.Port == 8081 {
			return port.Port, nil
		}
	}

	if len(availablePorts) == 0 {
		return 0, fmt.Errorf("no ports found in service %s", service.Name)
	}

	// If no named "api" port or port 8081 found, return error with available ports
	return 0, fmt.Errorf("API port not found in service %s (expected port named 'api' or port 8081). Available ports: %v", service.Name, availablePorts)
}

// hasAPIPort checks if a service has an API port (named "api" or port 8081)
func hasAPIPort(service *v1.Service) bool {
	for _, port := range service.Spec.Ports {
		if port.Name == "api" || port.Port == 8081 {
			return true
		}
	}
	return false
}

// GetDefaultNamespace extracts the default namespace from the current kubectl context
func GetDefaultNamespace() (string, error) {
	// Check for kubie environment variables first
	var kubeconfig string
	if kubieConfig := os.Getenv("KUBIE_KUBECONFIG"); kubieConfig != "" {
		kubeconfig = kubieConfig
	} else if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
		kubeconfig = envConfig
	} else {
		// Fall back to default kubeconfig
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Load kubeconfig with context information
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	// Get current context info
	rawConfig, err := kubeConfig.RawConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	currentContext := rawConfig.CurrentContext
	if currentContext == "" {
		return "", fmt.Errorf("no current context set in kubeconfig. Use 'kubectl config use-context' to set a context")
	}

	context, exists := rawConfig.Contexts[currentContext]
	if !exists {
		return "", fmt.Errorf("current context '%s' not found in kubeconfig", currentContext)
	}

	// Return namespace from context, fallback to "default" if not set
	if context.Namespace != "" {
		return context.Namespace, nil
	}

	return "default", nil
}

// GetRandomPort returns a random available port
func GetRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// GetStatefulSetPods retrieves all pods belonging to a StatefulSet (returns slice of Pod values, not pointers)
func (k *K8sClient) GetStatefulSetPods(ctx context.Context, namespace, statefulSetName string) ([]v1.Pod, error) {
	pods, err := k.GetPodsFromStatefulSet(ctx, namespace, statefulSetName)
	if err != nil {
		return nil, err
	}

	// Convert pointer slice to value slice
	result := make([]v1.Pod, len(pods))
	for i, pod := range pods {
		result[i] = *pod
	}

	return result, nil
}

// ExecCommand executes a command in a pod and returns the output
func (k *K8sClient) ExecCommand(ctx context.Context, namespace, podName string, command []string) (string, error) {
	req := k.coreClient.RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command: command,
			Stdout:  true,
			Stderr:  true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(k.config, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create SPDY executor: %w", err)
	}

	// Capture output
	var stdout, stderr strings.Builder
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("command failed: %s", stderr.String())
		}
		return "", fmt.Errorf("exec failed: %w", err)
	}

	return stdout.String(), nil
}

// ExecCommandStream executes a command in a pod and returns a stream reader for the output
func (k *K8sClient) ExecCommandStream(ctx context.Context, namespace, podName string, command []string) (io.ReadCloser, error) {
	req := k.coreClient.RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command: command,
			Stdout:  true,
			Stderr:  true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(k.config, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to create SPDY executor: %w", err)
	}

	// Create a pipe to stream the output
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()
		err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
			Stdout: writer,
			Stderr: writer,
		})
		if err != nil {
			writer.CloseWithError(fmt.Errorf("exec stream failed: %w", err))
		}
	}()

	return reader, nil
}
