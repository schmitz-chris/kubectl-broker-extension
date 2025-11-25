package sidecar

import (
	"context"
	"fmt"
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"

	"kubectl-broker/pkg"
)

// DefaultPort is the REST port exposed by the sidecar.
const DefaultPort int32 = 8085

// ConnectOptions control how the connector discovers a pod and establishes port-forwarding.
type ConnectOptions struct {
	Namespace      string
	StatefulSet    string
	Pod            string
	RemotePort     int32
	Timeout        time.Duration
	APIToken       string
	SkipValidation bool
}

// Connector wires Kubernetes port-forwarding with the HTTP client.
type Connector struct {
	k8sClient     *pkg.K8sClient
	portForwarder *pkg.PortForwarder
}

// NewConnector builds a Connector using the provided Kubernetes client.
func NewConnector(k8sClient *pkg.K8sClient) *Connector {
	return &Connector{
		k8sClient:     k8sClient,
		portForwarder: pkg.NewPortForwarder(k8sClient.GetConfig(), k8sClient.GetRESTClient()),
	}
}

// WithConnection establishes port-forwarding to the sidecar and invokes fn with a configured Client.
func (c *Connector) WithConnection(ctx context.Context, opts ConnectOptions, fn func(*Client) error) error {
	if c == nil || c.k8sClient == nil {
		return fmt.Errorf("sidecar connector is not initialized")
	}
	if fn == nil {
		return fmt.Errorf("callback is required")
	}
	if opts.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if opts.Pod == "" && opts.StatefulSet == "" {
		return fmt.Errorf("statefulset is required when pod is not specified")
	}
	remotePort := opts.RemotePort
	if remotePort == 0 {
		remotePort = DefaultPort
	}

	pod, err := ResolveSidecarPod(ctx, c.k8sClient, opts.Namespace, opts.StatefulSet, opts.Pod)
	if err != nil {
		return err
	}

	if !opts.SkipValidation {
		if err := pkg.ValidatePodStatus(pod); err != nil {
			return err
		}
	}

	localPort, err := pkg.GetRandomPort()
	if err != nil {
		return fmt.Errorf("allocate local port: %w", err)
	}

	return c.portForwarder.PerformWithPortForwarding(ctx, pod, remotePort, localPort, func(localPort int) error {
		baseURL := fmt.Sprintf("http://localhost:%d", localPort)
		client := NewClient(baseURL, ClientOptions{
			Timeout:  opts.Timeout,
			APIToken: opts.APIToken,
		})
		return fn(client)
	})
}

// ResolveSidecarPod picks the pod that hosts the sidecar.
func ResolveSidecarPod(ctx context.Context, k8sClient *pkg.K8sClient, namespace, statefulSetName, podName string) (*v1.Pod, error) {
	if k8sClient == nil {
		return nil, fmt.Errorf("kubernetes client is required")
	}
	if podName != "" {
		pod, err := k8sClient.GetPod(ctx, namespace, podName)
		if err != nil {
			return nil, pkg.EnhanceError(err, fmt.Sprintf("pod %s in namespace %s", podName, namespace))
		}
		return pod, nil
	}

	pods, err := k8sClient.GetPodsFromStatefulSet(ctx, namespace, statefulSetName)
	if err != nil {
		return nil, pkg.EnhanceError(err, fmt.Sprintf("StatefulSet %s in namespace %s", statefulSetName, namespace))
	}
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found for StatefulSet %s in namespace %s", statefulSetName, namespace)
	}

	sort.SliceStable(pods, func(i, j int) bool {
		return pods[i].Name < pods[j].Name
	})

	var firstErr error
	for _, pod := range pods {
		if err := pkg.ValidatePodStatus(pod); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		return pod, nil
	}

	if firstErr != nil {
		return nil, firstErr
	}
	return pods[0], nil
}
