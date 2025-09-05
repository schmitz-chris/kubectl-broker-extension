package testutils

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
)

// MockK8sClient provides a simple mock Kubernetes client for testing
type MockK8sClient struct {
	Config *rest.Config

	// Test data storage
	pods         map[string]*v1.Pod
	statefulsets map[string]*appsv1.StatefulSet
	services     map[string]*v1.Service
}

// NewMockK8sClient creates a new mock Kubernetes client with test data
func NewMockK8sClient() *MockK8sClient {
	mock := &MockK8sClient{
		Config:       &rest.Config{Host: "https://mock-cluster"},
		pods:         make(map[string]*v1.Pod),
		statefulsets: make(map[string]*appsv1.StatefulSet),
		services:     make(map[string]*v1.Service),
	}

	// Add test data
	namespace := "9141c41b-686e-42d8-8524-9876229d41ce"

	pod0 := CreateTestPod("broker-0", namespace, "10.244.1.100", true)
	pod1 := CreateTestPod("broker-1", namespace, "10.244.1.101", true)
	sts := CreateTestStatefulSet("broker", namespace, 2)
	svc := CreateTestService("hivemq-broker-api", namespace)

	mock.pods[namespace+"/broker-0"] = pod0
	mock.pods[namespace+"/broker-1"] = pod1
	mock.statefulsets[namespace+"/broker"] = sts
	mock.services[namespace+"/hivemq-broker-api"] = svc

	return mock
}

// GetConfig returns the REST config
func (m *MockK8sClient) GetConfig() *rest.Config {
	return m.Config
}

// GetPod returns a mock pod
func (m *MockK8sClient) GetPod(namespace, name string) (*v1.Pod, error) {
	key := namespace + "/" + name
	if pod, exists := m.pods[key]; exists {
		return pod, nil
	}
	return nil, fmt.Errorf("pod %s not found in namespace %s", name, namespace)
}

// GetStatefulSet returns a mock statefulset
func (m *MockK8sClient) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	key := namespace + "/" + name
	if sts, exists := m.statefulsets[key]; exists {
		return sts, nil
	}
	return nil, fmt.Errorf("statefulset %s not found in namespace %s", name, namespace)
}

// GetService returns a mock service
func (m *MockK8sClient) GetService(namespace, name string) (*v1.Service, error) {
	key := namespace + "/" + name
	if svc, exists := m.services[key]; exists {
		return svc, nil
	}
	return nil, fmt.Errorf("service %s not found in namespace %s", name, namespace)
}

// ExecuteCommand executes a command in a pod (mock implementation)
func (m *MockK8sClient) ExecuteCommand(namespace, podName, containerName string, command []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// Mock implementation - just write success message
	if stdout != nil {
		fmt.Fprintf(stdout, "Mock command execution in pod %s: %v\n", podName, command)
	}
	return nil
}

// CreateTestStatefulSet creates a test StatefulSet for HiveMQ broker
func CreateTestStatefulSet(name, namespace string, replicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "hivemq-broker",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "hivemq-broker",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "hivemq-broker",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  "hivemq",
							Image: "hivemq/hivemq4:4.42.0",
							Ports: []v1.ContainerPort{
								{Name: "mqtt", ContainerPort: 1883},
								{Name: "mqtts", ContainerPort: 8883},
								{Name: "control-center", ContainerPort: 8080},
								{Name: "rest-api", ContainerPort: 8081},
								{Name: "health", ContainerPort: 9090},
							},
						},
					},
				},
			},
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas: replicas,
			Replicas:      replicas,
		},
	}
}

// CreateTestPod creates a test Pod for HiveMQ broker
func CreateTestPod(name, namespace, podIP string, ready bool) *v1.Pod {
	phase := v1.PodRunning
	if !ready {
		phase = v1.PodPending
	}

	conditions := []v1.PodCondition{
		{
			Type:   v1.PodReady,
			Status: v1.ConditionTrue,
		},
	}
	if !ready {
		conditions[0].Status = v1.ConditionFalse
	}

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app":                                "hivemq-broker",
				"statefulset.kubernetes.io/pod-name": name,
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "hivemq",
					Image: "hivemq/hivemq4:4.42.0",
					Ports: []v1.ContainerPort{
						{Name: "mqtt", ContainerPort: 1883},
						{Name: "mqtts", ContainerPort: 8883},
						{Name: "control-center", ContainerPort: 8080},
						{Name: "rest-api", ContainerPort: 8081},
						{Name: "health", ContainerPort: 9090},
					},
				},
			},
		},
		Status: v1.PodStatus{
			Phase:      phase,
			PodIP:      podIP,
			Conditions: conditions,
		},
	}
}

// CreateTestService creates a test Service for HiveMQ broker
func CreateTestService(name, namespace string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "hivemq-broker",
			},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: "rest-api", Port: 8081, TargetPort: intstr.FromInt(8081)},
				{Name: "health", Port: 9090, TargetPort: intstr.FromInt(9090)},
			},
			Selector: map[string]string{
				"app": "hivemq-broker",
			},
		},
	}
}

// MockRESTClient provides a mock REST client for testing
type MockRESTClient struct{}

func (m *MockRESTClient) APIVersion() schema.GroupVersion {
	return schema.GroupVersion{Group: "", Version: "v1"}
}

// MockHiveMQServer provides a mock HiveMQ HTTP server for testing
type MockHiveMQServer struct {
	Server       *httptest.Server
	HealthData   string
	BackupData   map[string]string
	ResponseCode int
}

// NewMockHiveMQServer creates a new mock HiveMQ server
func NewMockHiveMQServer() *MockHiveMQServer {
	mock := &MockHiveMQServer{
		BackupData:   make(map[string]string),
		ResponseCode: 200,
	}

	mux := http.NewServeMux()

	// Health endpoints
	mux.HandleFunc("/api/v1/health/", mock.handleHealth)
	mux.HandleFunc("/api/v1/health/liveness", mock.handleHealth)
	mux.HandleFunc("/api/v1/health/readiness", mock.handleHealth)

	// Backup endpoints
	mux.HandleFunc("/api/v1/backups", mock.handleBackups)
	mux.HandleFunc("/api/v1/backups/", mock.handleBackupByID)

	mock.Server = httptest.NewServer(mux)
	return mock
}

// Close closes the mock server
func (m *MockHiveMQServer) Close() {
	m.Server.Close()
}

// URL returns the server URL
func (m *MockHiveMQServer) URL() string {
	return m.Server.URL
}

// SetHealthResponse sets the health response data
func (m *MockHiveMQServer) SetHealthResponse(data string) {
	m.HealthData = data
}

// SetBackupResponse sets backup response data for a specific ID
func (m *MockHiveMQServer) SetBackupResponse(backupID, data string) {
	m.BackupData[backupID] = data
}

// SetResponseCode sets the HTTP response code for all endpoints
func (m *MockHiveMQServer) SetResponseCode(code int) {
	m.ResponseCode = code
}

func (m *MockHiveMQServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(m.ResponseCode)
	if m.HealthData != "" {
		w.Write([]byte(m.HealthData))
	} else {
		w.Write([]byte(`{"status": "UP", "components": {}}`))
	}
}

func (m *MockHiveMQServer) handleBackups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(m.ResponseCode)

	switch r.Method {
	case http.MethodGet:
		// List backups
		w.Write([]byte(`{"items": []}`))
	case http.MethodPost:
		// Create backup
		backupID := fmt.Sprintf("mock-%d", time.Now().Unix())
		response := fmt.Sprintf(`{"backup": {"id": "%s", "state": "RUNNING", "created": "%s", "bytes": 0}}`,
			backupID, time.Now().Format(time.RFC3339))
		w.Write([]byte(response))
	default:
		w.WriteHeader(405)
	}
}

func (m *MockHiveMQServer) handleBackupByID(w http.ResponseWriter, r *http.Request) {
	// Extract backup ID from URL path
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		w.WriteHeader(400)
		return
	}

	backupID := parts[4]

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(m.ResponseCode)

	if data, exists := m.BackupData[backupID]; exists {
		w.Write([]byte(data))
	} else {
		// Default response
		response := fmt.Sprintf(`{"backup": {"id": "%s", "state": "COMPLETED", "created": "%s", "bytes": 1024}}`,
			backupID, time.Now().Format(time.RFC3339))
		w.Write([]byte(response))
	}
}

// MockPortForwarder provides a mock port forwarder for testing
type MockPortForwarder struct {
	LocalPort    int
	RemotePort   int
	StopChannel  chan struct{}
	ReadyChannel chan struct{}
	ErrorChannel chan error
	ForwardFunc  func() error
}

// NewMockPortForwarder creates a new mock port forwarder
func NewMockPortForwarder(localPort, remotePort int) *MockPortForwarder {
	return &MockPortForwarder{
		LocalPort:    localPort,
		RemotePort:   remotePort,
		StopChannel:  make(chan struct{}),
		ReadyChannel: make(chan struct{}),
		ErrorChannel: make(chan error),
		ForwardFunc: func() error {
			return nil
		},
	}
}

// ForwardPorts starts the mock port forwarding
func (m *MockPortForwarder) ForwardPorts() error {
	go func() {
		close(m.ReadyChannel)
		if m.ForwardFunc != nil {
			if err := m.ForwardFunc(); err != nil {
				m.ErrorChannel <- err
			}
		}
		<-m.StopChannel
	}()
	return nil
}

// GetPorts returns the local and remote ports
func (m *MockPortForwarder) GetPorts() ([]ForwardedPort, error) {
	return []ForwardedPort{{Local: uint16(m.LocalPort), Remote: uint16(m.RemotePort)}}, nil
}

// Stop stops the port forwarding
func (m *MockPortForwarder) Stop() {
	close(m.StopChannel)
}

// ForwardedPort represents a forwarded port pair
type ForwardedPort struct {
	Local  uint16
	Remote uint16
}

// MockExecutor provides a mock command executor for testing
type MockExecutor struct {
	Commands [][]string
	Results  map[string]ExecuteResult
}

// ExecuteResult represents the result of command execution
type ExecuteResult struct {
	Stdout []byte
	Stderr []byte
	Error  error
}

// NewMockExecutor creates a new mock executor
func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Commands: make([][]string, 0),
		Results:  make(map[string]ExecuteResult),
	}
}

// SetResult sets the result for a specific command
func (m *MockExecutor) SetResult(command string, result ExecuteResult) {
	m.Results[command] = result
}

// Execute executes a command and returns the mocked result
func (m *MockExecutor) Execute(command []string, stdin io.Reader, stdout, stderr io.Writer) error {
	m.Commands = append(m.Commands, command)

	cmdStr := strings.Join(command, " ")
	if result, exists := m.Results[cmdStr]; exists {
		if stdout != nil && len(result.Stdout) > 0 {
			stdout.Write(result.Stdout)
		}
		if stderr != nil && len(result.Stderr) > 0 {
			stderr.Write(result.Stderr)
		}
		return result.Error
	}

	// Default success
	if stdout != nil {
		fmt.Fprintf(stdout, "Mock execution of: %s\n", cmdStr)
	}
	return nil
}

// GetExecutedCommands returns all executed commands
func (m *MockExecutor) GetExecutedCommands() [][]string {
	return m.Commands
}
