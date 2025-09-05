package pkg

import (
	"testing"

	"kubectl-broker/testutils"
)

func TestNewK8sClient(t *testing.T) {
	client, err := NewK8sClient(false)
	if err != nil {
		// Skip test if no kubeconfig available
		t.Skip("No kubeconfig available, skipping integration test")
	}

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.config == nil {
		t.Error("Config should not be nil")
	}

	if client.coreClient == nil {
		t.Error("Core client should not be nil")
	}

	if client.appsClient == nil {
		t.Error("Apps client should not be nil")
	}
}

func TestMockK8sClient(t *testing.T) {
	mockClient := testutils.NewMockK8sClient()

	if mockClient == nil {
		t.Fatal("Mock client should not be nil")
	}

	config := mockClient.GetConfig()
	if config == nil {
		t.Error("Config should not be nil")
	}

	// Test getting test pods
	namespace := "9141c41b-686e-42d8-8524-9876229d41ce"
	pod, err := mockClient.GetPod(namespace, "broker-0")
	if err != nil {
		t.Errorf("Expected to find test pod: %v", err)
	}

	if pod != nil && pod.Name != "broker-0" {
		t.Errorf("Expected pod name 'broker-0', got %q", pod.Name)
	}

	// Test pod not found
	_, err = mockClient.GetPod(namespace, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent pod")
	}
}

func TestMockStatefulSet(t *testing.T) {
	mockClient := testutils.NewMockK8sClient()

	namespace := "9141c41b-686e-42d8-8524-9876229d41ce"
	sts, err := mockClient.GetStatefulSet(namespace, "broker")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if sts == nil {
		t.Fatal("StatefulSet should not be nil")
	}

	if sts.Name != "broker" {
		t.Errorf("Expected StatefulSet name 'broker', got %q", sts.Name)
	}

	if *sts.Spec.Replicas != 2 {
		t.Errorf("Expected 2 replicas, got %d", *sts.Spec.Replicas)
	}

	// Test StatefulSet not found
	_, err = mockClient.GetStatefulSet(namespace, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent StatefulSet")
	}
}

func TestMockService(t *testing.T) {
	mockClient := testutils.NewMockK8sClient()

	namespace := "9141c41b-686e-42d8-8524-9876229d41ce"
	svc, err := mockClient.GetService(namespace, "hivemq-broker-api")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if svc == nil {
		t.Fatal("Service should not be nil")
	}

	if svc.Name != "hivemq-broker-api" {
		t.Errorf("Expected service name 'hivemq-broker-api', got %q", svc.Name)
	}

	// Test service not found
	_, err = mockClient.GetService(namespace, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent service")
	}
}

func TestCreateTestPod(t *testing.T) {
	pod := testutils.CreateTestPod("broker-0", "test-namespace", "10.244.1.100", true)

	if pod == nil {
		t.Fatal("Pod should not be nil")
	}

	if pod.Name != "broker-0" {
		t.Errorf("Expected pod name 'broker-0', got %q", pod.Name)
	}

	if pod.Namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got %q", pod.Namespace)
	}

	if pod.Status.PodIP != "10.244.1.100" {
		t.Errorf("Expected pod IP '10.244.1.100', got %q", pod.Status.PodIP)
	}
}

func TestCreateTestStatefulSet(t *testing.T) {
	sts := testutils.CreateTestStatefulSet("broker", "test-namespace", 2)

	if sts == nil {
		t.Fatal("StatefulSet should not be nil")
	}

	if sts.Name != "broker" {
		t.Errorf("Expected StatefulSet name 'broker', got %q", sts.Name)
	}

	if sts.Namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got %q", sts.Namespace)
	}

	if *sts.Spec.Replicas != 2 {
		t.Errorf("Expected 2 replicas, got %d", *sts.Spec.Replicas)
	}
}

func TestCreateTestService(t *testing.T) {
	svc := testutils.CreateTestService("hivemq-broker-api", "test-namespace")

	if svc == nil {
		t.Fatal("Service should not be nil")
	}

	if svc.Name != "hivemq-broker-api" {
		t.Errorf("Expected service name 'hivemq-broker-api', got %q", svc.Name)
	}

	if svc.Namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got %q", svc.Namespace)
	}

	// Check service has expected ports
	foundRestAPI := false
	foundHealth := false
	for _, port := range svc.Spec.Ports {
		if port.Name == "rest-api" && port.Port == 8081 {
			foundRestAPI = true
		}
		if port.Name == "health" && port.Port == 9090 {
			foundHealth = true
		}
	}

	if !foundRestAPI {
		t.Error("Service should have rest-api port 8081")
	}

	if !foundHealth {
		t.Error("Service should have health port 9090")
	}
}
