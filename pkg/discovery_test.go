package pkg

import (
	"testing"
)

func TestDiscoverBrokers(t *testing.T) {
	t.Skip("Skipping test that requires real Kubernetes cluster access")
}

func TestIsBrokerPod(t *testing.T) {
	tests := []struct {
		name           string
		podName        string
		labels         map[string]string
		expectedResult bool
	}{
		{
			name:           "broker in name",
			podName:        "broker-0",
			labels:         map[string]string{},
			expectedResult: true,
		},
		{
			name:           "hivemq in name",
			podName:        "hivemq-cluster-0",
			labels:         map[string]string{},
			expectedResult: true,
		},
		{
			name:           "broker in labels key",
			podName:        "pod-0",
			labels:         map[string]string{"app.kubernetes.io/name": "hivemq-broker"},
			expectedResult: true,
		},
		{
			name:           "hivemq in labels value",
			podName:        "pod-0",
			labels:         map[string]string{"component": "hivemq"},
			expectedResult: true,
		},
		{
			name:           "broker case insensitive",
			podName:        "BROKER-0",
			labels:         map[string]string{},
			expectedResult: true,
		},
		{
			name:           "not a broker pod",
			podName:        "redis-0",
			labels:         map[string]string{"app": "redis"},
			expectedResult: false,
		},
		{
			name:           "empty name and labels",
			podName:        "",
			labels:         map[string]string{},
			expectedResult: false,
		},
		{
			name:           "partial match in name",
			podName:        "my-broker-service",
			labels:         map[string]string{},
			expectedResult: true,
		},
		{
			name:           "partial match in label key",
			podName:        "pod-123",
			labels:         map[string]string{"hivemq.component": "cluster"},
			expectedResult: true,
		},
		{
			name:           "partial match in label value",
			podName:        "pod-456",
			labels:         map[string]string{"type": "message-broker"},
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBrokerPod(tt.podName, tt.labels)
			if result != tt.expectedResult {
				t.Errorf("isBrokerPod(%q, %v) = %v, expected %v",
					tt.podName, tt.labels, result, tt.expectedResult)
			}
		})
	}
}

func TestDiscoverBrokersWithMockData(t *testing.T) {
	t.Skip("Skipping test that requires K8s integration")
}

func TestDiscoverBrokersContextCancellation(t *testing.T) {
	t.Skip("Skipping test that requires K8s integration")
}

// BenchmarkIsBrokerPod benchmarks the broker pod detection
func BenchmarkIsBrokerPod(b *testing.B) {
	testLabels := map[string]string{
		"app.kubernetes.io/name":      "hivemq",
		"app.kubernetes.io/component": "broker",
		"app":                         "hivemq-cluster",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isBrokerPod("broker-0", testLabels)
	}
}

// BenchmarkDiscoverBrokers benchmarks broker discovery
func BenchmarkDiscoverBrokers(b *testing.B) {
	b.Skip("Skipping benchmark that requires K8s integration")
}
