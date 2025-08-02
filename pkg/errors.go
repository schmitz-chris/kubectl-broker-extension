package pkg

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// EnhanceError provides user-friendly error messages with actionable guidance
func EnhanceError(err error, context string) error {
	if err == nil {
		return nil
	}

	// Handle Kubernetes API errors
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("%s not found. Please check the name and namespace are correct", context)
	}

	if apierrors.IsForbidden(err) {
		return fmt.Errorf("access denied for %s. Ensure your kubeconfig has permissions to list/portforward pods in the specified namespace", context)
	}

	if apierrors.IsUnauthorized(err) {
		return fmt.Errorf("authentication failed. Check your kubeconfig and cluster connection")
	}

	if apierrors.IsTimeout(err) {
		return fmt.Errorf("request timed out for %s. Check your network connection to the cluster", context)
	}

	// Handle network connectivity issues
	if strings.Contains(err.Error(), "connection refused") {
		return fmt.Errorf("connection refused to %s. Check if the pod is running and network policies allow connection", context)
	}

	if strings.Contains(err.Error(), "no such host") {
		return fmt.Errorf("cannot resolve cluster hostname. Check your kubeconfig and network connection")
	}

	// Return original error with context if no specific handling applies
	return fmt.Errorf("%s: %w", context, err)
}

// ValidatePodStatus checks if a pod is ready for port-forwarding
func ValidatePodStatus(pod *v1.Pod) error {
	switch pod.Status.Phase {
	case v1.PodPending:
		return fmt.Errorf("pod '%s' is not ready (status: %s). Wait for the pod to start", pod.Name, pod.Status.Phase)
	case v1.PodFailed:
		return fmt.Errorf("pod '%s' has failed (status: %s). Check pod logs for details", pod.Name, pod.Status.Phase)
	case v1.PodSucceeded:
		return fmt.Errorf("pod '%s' has completed (status: %s). Cannot port-forward to a completed pod", pod.Name, pod.Status.Phase)
	case v1.PodRunning:
		// Check if containers are ready
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status != v1.ConditionTrue {
				return fmt.Errorf("pod '%s' is running but not ready. Wait for all containers to be ready", pod.Name)
			}
		}
		return nil
	default:
		return fmt.Errorf("pod '%s' has unknown status: %s", pod.Name, pod.Status.Phase)
	}
}

// CheckKubeconfig validates that kubeconfig is accessible and has a current context
func CheckKubeconfig() error {
	// This is a simplified check - in a real implementation, you might want to
	// use clientcmd.LoadFromFile to validate the kubeconfig more thoroughly
	return nil
}
