package pkg

import (
	"errors"
	"testing"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubectl-broker/testutils"
)

func TestAppError(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		name            string
		err             *AppError
		expectedMessage string
	}{
		{
			name: "error with custom message",
			err: &AppError{
				Type:     ErrTypeKubernetes,
				Op:       "get_pod",
				Resource: "broker-0",
				Err:      originalErr,
				Message:  "custom error message",
			},
			expectedMessage: "custom error message",
		},
		{
			name: "error with resource",
			err: &AppError{
				Type:     ErrTypeHealthCheck,
				Op:       "health_check",
				Resource: "broker-0",
				Err:      originalErr,
			},
			expectedMessage: "health_check failed for broker-0: original error",
		},
		{
			name: "error without resource",
			err: &AppError{
				Type: ErrTypeNetwork,
				Op:   "port_forward",
				Err:  originalErr,
			},
			expectedMessage: "port_forward failed: original error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expectedMessage {
				t.Errorf("Error() = %q, expected %q", tt.err.Error(), tt.expectedMessage)
			}
		})
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	appErr := &AppError{
		Type: ErrTypeKubernetes,
		Op:   "test",
		Err:  originalErr,
	}

	if errors.Unwrap(appErr) != originalErr {
		t.Error("Unwrap should return the original error")
	}
}

func TestAppErrorIs(t *testing.T) {
	err1 := &AppError{Type: ErrTypeKubernetes, Op: "test1"}
	err2 := &AppError{Type: ErrTypeKubernetes, Op: "test2"}
	err3 := &AppError{Type: ErrTypeNetwork, Op: "test3"}

	if !errors.Is(err1, err2) {
		t.Error("Errors with same type should be considered equal")
	}

	if errors.Is(err1, err3) {
		t.Error("Errors with different types should not be considered equal")
	}

	regularErr := errors.New("regular error")
	if errors.Is(err1, regularErr) {
		t.Error("AppError should not be equal to regular error")
	}
}

func TestErrorConstructors(t *testing.T) {
	originalErr := errors.New("test error")

	tests := []struct {
		name         string
		constructor  func() *AppError
		expectedType ErrType
	}{
		{
			name:         "NewKubernetesError",
			constructor:  func() *AppError { return NewKubernetesError("get_pod", "broker-0", originalErr) },
			expectedType: ErrTypeKubernetes,
		},
		{
			name:         "NewNetworkError",
			constructor:  func() *AppError { return NewNetworkError("connect", "broker-0", originalErr) },
			expectedType: ErrTypeNetwork,
		},
		{
			name:         "NewValidationError",
			constructor:  func() *AppError { return NewValidationError("validate", "broker-0", "invalid input") },
			expectedType: ErrTypeValidation,
		},
		{
			name:         "NewHealthCheckError",
			constructor:  func() *AppError { return NewHealthCheckError("health_check", "broker-0", originalErr) },
			expectedType: ErrTypeHealthCheck,
		},
		{
			name:         "NewPortforwardError",
			constructor:  func() *AppError { return NewPortforwardError("port_forward", "broker-0", originalErr) },
			expectedType: ErrTypePortforward,
		},
		{
			name:         "NewConfigurationError",
			constructor:  func() *AppError { return NewConfigurationError("config", "invalid config") },
			expectedType: ErrTypeConfiguration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.constructor()
			if err.Type != tt.expectedType {
				t.Errorf("Expected error type %v, got %v", tt.expectedType, err.Type)
			}
		})
	}
}

func TestEnhanceError(t *testing.T) {
	tests := []struct {
		name            string
		originalError   error
		context         string
		expectedMessage string
		expectedType    ErrType
	}{
		{
			name:            "nil error",
			originalError:   nil,
			context:         "test context",
			expectedMessage: "",
			expectedType:    "",
		},
		{
			name:            "not found error for StatefulSet",
			originalError:   apierrors.NewNotFound(schema.GroupResource{Resource: "statefulsets"}, "broker"),
			context:         "StatefulSet broker",
			expectedMessage: "StatefulSet broker not found. Please check the StatefulSet name and namespace are correct. Use --discover to find available StatefulSets",
			expectedType:    ErrTypeKubernetes,
		},
		{
			name:            "not found error for Pod",
			originalError:   apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "broker-0"),
			context:         "Pod broker-0",
			expectedMessage: "Pod broker-0 not found. Please check the name and namespace are correct",
			expectedType:    ErrTypeKubernetes,
		},
		{
			name:            "forbidden error",
			originalError:   apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "broker-0", errors.New("forbidden")),
			context:         "Pod broker-0",
			expectedMessage: "access denied for Pod broker-0. Ensure your kubeconfig has permissions to list/portforward pods in the specified namespace",
			expectedType:    ErrTypeKubernetes,
		},
		{
			name:            "unauthorized error",
			originalError:   apierrors.NewUnauthorized("unauthorized"),
			context:         "Pod broker-0",
			expectedMessage: "authentication failed. Check your kubeconfig and cluster connection",
			expectedType:    ErrTypeKubernetes,
		},
		{
			name:            "timeout error",
			originalError:   apierrors.NewTimeoutError("timeout", 30),
			context:         "Pod broker-0",
			expectedMessage: "request timed out for Pod broker-0. Check your network connection to the cluster",
			expectedType:    ErrTypeKubernetes,
		},
		{
			name:            "connection refused error",
			originalError:   errors.New("connection refused"),
			context:         "Pod broker-0",
			expectedMessage: "connection refused to Pod broker-0. Check if the pod is running and network policies allow connection",
			expectedType:    ErrTypeNetwork,
		},
		{
			name:            "no such host error",
			originalError:   errors.New("no such host"),
			context:         "cluster connection",
			expectedMessage: "cannot resolve cluster hostname. Check your kubeconfig and network connection",
			expectedType:    ErrTypeNetwork,
		},
		{
			name:            "generic error",
			originalError:   errors.New("generic error"),
			context:         "test operation",
			expectedMessage: "test operation: generic error",
			expectedType:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enhancedErr := EnhanceError(tt.originalError, tt.context)

			if tt.originalError == nil {
				if enhancedErr != nil {
					t.Error("EnhanceError should return nil for nil input")
				}
				return
			}

			if enhancedErr == nil {
				t.Fatal("EnhanceError should not return nil for non-nil input")
			}

			if enhancedErr.Error() != tt.expectedMessage {
				t.Errorf("Expected message %q, got %q", tt.expectedMessage, enhancedErr.Error())
			}

			if tt.expectedType != "" {
				var appErr *AppError
				if errors.As(enhancedErr, &appErr) {
					if appErr.Type != tt.expectedType {
						t.Errorf("Expected error type %v, got %v", tt.expectedType, appErr.Type)
					}
				} else {
					t.Error("Expected AppError for domain-specific errors")
				}
			}
		})
	}
}

func TestValidatePodStatus(t *testing.T) {
	tests := []struct {
		name        string
		pod         *v1.Pod
		shouldError bool
		expectedMsg string
	}{
		{
			name:        "nil pod",
			pod:         nil,
			shouldError: true,
			expectedMsg: "pod cannot be nil",
		},
		{
			name:        "pending pod",
			pod:         createTestPodWithPhase("broker-0", v1.PodPending),
			shouldError: true,
			expectedMsg: "pod 'broker-0' is not ready (status: Pending)",
		},
		{
			name:        "failed pod",
			pod:         createTestPodWithPhase("broker-0", v1.PodFailed),
			shouldError: true,
			expectedMsg: "pod 'broker-0' has failed (status: Failed)",
		},
		{
			name:        "succeeded pod",
			pod:         createTestPodWithPhase("broker-0", v1.PodSucceeded),
			shouldError: true,
			expectedMsg: "pod 'broker-0' has completed (status: Succeeded)",
		},
		{
			name:        "running but not ready pod",
			pod:         createTestPodWithReadiness("broker-0", v1.PodRunning, false),
			shouldError: true,
			expectedMsg: "pod 'broker-0' is running but not ready",
		},
		{
			name:        "running and ready pod",
			pod:         createTestPodWithReadiness("broker-0", v1.PodRunning, true),
			shouldError: false,
			expectedMsg: "",
		},
		{
			name:        "unknown phase pod",
			pod:         createTestPodWithPhase("broker-0", v1.PodPhase("Unknown")),
			shouldError: true,
			expectedMsg: "pod 'broker-0' has unknown status: Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePodStatus(tt.pod)

			if tt.shouldError {
				testutils.AssertError(t, err, "Expected validation error")
				if tt.expectedMsg != "" {
					testutils.AssertStringContains(t, err.Error(), tt.expectedMsg, "Error should contain expected message")
				}

				// Check that it's a validation error
				var appErr *AppError
				if errors.As(err, &appErr) {
					testutils.AssertEqual(t, ErrTypeValidation, appErr.Type, "Should be validation error type")
				}
			} else {
				testutils.AssertNoError(t, err, "Should not have validation error")
			}
		})
	}
}

func TestWithMessage(t *testing.T) {
	originalErr := errors.New("original")
	appErr := NewKubernetesError("test", "resource", originalErr)

	customMessage := "custom message"
	enhanced := appErr.withMessage(customMessage)

	if enhanced.Message != customMessage {
		t.Errorf("Expected message %q, got %q", customMessage, enhanced.Message)
	}

	if enhanced.Error() != customMessage {
		t.Errorf("Error() should return custom message, got %q", enhanced.Error())
	}
}

// Helper functions for creating test pods

func createTestPodWithPhase(name string, phase v1.PodPhase) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Status: v1.PodStatus{
			Phase: phase,
		},
	}
}

func createTestPodWithReadiness(name string, phase v1.PodPhase, ready bool) *v1.Pod {
	conditionStatus := v1.ConditionFalse
	if ready {
		conditionStatus = v1.ConditionTrue
	}

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test",
		},
		Status: v1.PodStatus{
			Phase: phase,
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: conditionStatus,
				},
			},
		},
	}
}

// BenchmarkAppErrorCreation benchmarks error creation
func BenchmarkAppErrorCreation(b *testing.B) {
	originalErr := errors.New("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := NewKubernetesError("operation", "resource", originalErr)
		_ = err.Error()
	}
}

// BenchmarkEnhanceError benchmarks error enhancement
func BenchmarkEnhanceError(b *testing.B) {
	originalErr := apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "test-pod")
	context := "Pod test-pod"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enhanced := EnhanceError(originalErr, context)
		_ = enhanced.Error()
	}
}

// BenchmarkValidatePodStatus benchmarks pod validation
func BenchmarkValidatePodStatus(b *testing.B) {
	pod := createTestPodWithReadiness("broker-0", v1.PodRunning, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ValidatePodStatus(pod)
		_ = err
	}
}
