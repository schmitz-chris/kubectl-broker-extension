package sidecar

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	v1 "k8s.io/api/core/v1"

	"kubectl-broker/pkg"
)

// DefaultPort is the REST port exposed by the sidecar.
const DefaultPort int32 = 8085

// ErrUnavailable indicates the sidecar connector could not establish a connection.
var ErrUnavailable = errors.New("sidecar unavailable")

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
		return fmt.Errorf("%w: connector is not initialized", ErrUnavailable)
	}
	if fn == nil {
		return fmt.Errorf("%w: callback is required", ErrUnavailable)
	}
	if opts.Namespace == "" {
		return fmt.Errorf("%w: namespace is required", ErrUnavailable)
	}
	if opts.Pod == "" && opts.StatefulSet == "" {
		return fmt.Errorf("%w: statefulset is required when pod is not specified", ErrUnavailable)
	}
	remotePort := opts.RemotePort
	if remotePort == 0 {
		remotePort = DefaultPort
	}

	pod, err := ResolveSidecarPod(ctx, c.k8sClient, opts.Namespace, opts.StatefulSet, opts.Pod)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}

	if !opts.SkipValidation {
		if err := pkg.ValidatePodStatus(pod); err != nil {
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
	}

	localPort, err := pkg.GetRandomPort()
	if err != nil {
		return fmt.Errorf("allocate local port: %w", err)
	}

	err = c.portForwarder.PerformWithPortForwarding(ctx, pod, remotePort, localPort, func(localPort int) error {
		baseURL := fmt.Sprintf("http://localhost:%d", localPort)
		client := NewClient(baseURL, ClientOptions{
			Timeout:  opts.Timeout,
			APIToken: opts.APIToken,
		})
		if fnErr := fn(client); fnErr != nil {
			return clientFnError{err: fnErr}
		}
		return nil
	})
	if err != nil {
		var fnErr clientFnError
		if errors.As(err, &fnErr) {
			return fnErr.err
		}
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
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

type clientFnError struct {
	err error
}

func (e clientFnError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e clientFnError) Unwrap() error {
	return e.err
}
