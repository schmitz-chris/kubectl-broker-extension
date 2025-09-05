package pkg

import (
	"errors"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// Custom error types for domain-specific errors

// ErrType represents different categories of errors
type ErrType string

const (
	ErrTypeKubernetes   ErrType = "kubernetes"
	ErrTypeNetwork      ErrType = "network"
	ErrTypeValidation   ErrType = "validation"
	ErrTypeHealthCheck  ErrType = "health_check"
	ErrTypePortforward  ErrType = "portforward"
	ErrTypeConfiguration ErrType = "configuration"
)

// AppError represents a domain-specific error with additional context
type AppError struct {
	Type     ErrType
	Op       string // operation that failed
	Resource string // resource involved (pod name, statefulset, etc)
	Err      error  // underlying error
	Message  string // user-friendly message
}

func (e *AppError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Resource != "" {
		return fmt.Sprintf("%s failed for %s: %v", e.Op, e.Resource, e.Err)
	}
	return fmt.Sprintf("%s failed: %v", e.Op, e.Err)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) Is(target error) bool {
	var appErr *AppError
	if errors.As(target, &appErr) {
		return e.Type == appErr.Type
	}
	return false
}

// Error constructors for different domains

// NewKubernetesError creates a new Kubernetes-related error
func NewKubernetesError(op, resource string, err error) *AppError {
	return &AppError{
		Type:     ErrTypeKubernetes,
		Op:       op,
		Resource: resource,
		Err:      err,
	}
}

// NewNetworkError creates a new network-related error
func NewNetworkError(op, resource string, err error) *AppError {
	return &AppError{
		Type:     ErrTypeNetwork,
		Op:       op,
		Resource: resource,
		Err:      err,
	}
}

// NewValidationError creates a new validation error
func NewValidationError(op, resource, message string) *AppError {
	return &AppError{
		Type:     ErrTypeValidation,
		Op:       op,
		Resource: resource,
		Message:  message,
	}
}

// NewHealthCheckError creates a new health check error
func NewHealthCheckError(op, resource string, err error) *AppError {
	return &AppError{
		Type:     ErrTypeHealthCheck,
		Op:       op,
		Resource: resource,
		Err:      err,
	}
}

// NewPortforwardError creates a new port forwarding error
func NewPortforwardError(op, resource string, err error) *AppError {
	return &AppError{
		Type:     ErrTypePortforward,
		Op:       op,
		Resource: resource,
		Err:      err,
	}
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(op, message string) *AppError {
	return &AppError{
		Type:    ErrTypeConfiguration,
		Op:      op,
		Message: message,
	}
}

// EnhanceError provides user-friendly error messages with actionable guidance
func EnhanceError(err error, context string) error {
	if err == nil {
		return nil
	}

	// Extract operation and resource from context
	parts := strings.SplitN(context, " ", 2)
	op := parts[0]
	resource := ""
	if len(parts) > 1 {
		resource = parts[1]
	}

	// Handle Kubernetes API errors
	if apierrors.IsNotFound(err) {
		var message string
		if strings.Contains(context, "StatefulSet") {
			message = fmt.Sprintf("%s not found. Please check the StatefulSet name and namespace are correct. Use --discover to find available StatefulSets", context)
		} else {
			message = fmt.Sprintf("%s not found. Please check the name and namespace are correct", context)
		}
		return NewKubernetesError(op, resource, err).withMessage(message)
	}

	if apierrors.IsForbidden(err) {
		message := fmt.Sprintf("access denied for %s. Ensure your kubeconfig has permissions to list/portforward pods in the specified namespace", context)
		return NewKubernetesError(op, resource, err).withMessage(message)
	}

	if apierrors.IsUnauthorized(err) {
		message := "authentication failed. Check your kubeconfig and cluster connection"
		return NewKubernetesError(op, resource, err).withMessage(message)
	}

	if apierrors.IsTimeout(err) {
		message := fmt.Sprintf("request timed out for %s. Check your network connection to the cluster", context)
		return NewKubernetesError(op, resource, err).withMessage(message)
	}

	// Handle network connectivity issues
	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") {
		message := fmt.Sprintf("connection refused to %s. Check if the pod is running and network policies allow connection", context)
		return NewNetworkError(op, resource, err).withMessage(message)
	}

	if strings.Contains(errMsg, "no such host") {
		message := "cannot resolve cluster hostname. Check your kubeconfig and network connection"
		return NewNetworkError(op, resource, err).withMessage(message)
	}

	// Return original error with context if no specific handling applies
	return fmt.Errorf("%s: %w", context, err)
}

// withMessage is a helper method to set custom message on AppError
func (e *AppError) withMessage(msg string) *AppError {
	e.Message = msg
	return e
}

// ValidatePodStatus checks if a pod is ready for port-forwarding
func ValidatePodStatus(pod *v1.Pod) error {
	if pod == nil {
		return NewValidationError("validate_pod_status", "", "pod cannot be nil")
	}

	switch pod.Status.Phase {
	case v1.PodPending:
		message := fmt.Sprintf("pod '%s' is not ready (status: %s). Wait for the pod to start", pod.Name, pod.Status.Phase)
		return NewValidationError("validate_pod_status", pod.Name, message)
	case v1.PodFailed:
		message := fmt.Sprintf("pod '%s' has failed (status: %s). Check pod logs for details", pod.Name, pod.Status.Phase)
		return NewValidationError("validate_pod_status", pod.Name, message)
	case v1.PodSucceeded:
		message := fmt.Sprintf("pod '%s' has completed (status: %s). Cannot port-forward to a completed pod", pod.Name, pod.Status.Phase)
		return NewValidationError("validate_pod_status", pod.Name, message)
	case v1.PodRunning:
		// Check if containers are ready
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady && condition.Status != v1.ConditionTrue {
				message := fmt.Sprintf("pod '%s' is running but not ready. Wait for all containers to be ready", pod.Name)
				return NewValidationError("validate_pod_status", pod.Name, message)
			}
		}
		return nil
	default:
		message := fmt.Sprintf("pod '%s' has unknown status: %s", pod.Name, pod.Status.Phase)
		return NewValidationError("validate_pod_status", pod.Name, message)
	}
}
