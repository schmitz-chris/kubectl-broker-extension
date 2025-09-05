package health

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHealthResponse(t *testing.T) {
	tests := []struct {
		name               string
		jsonFile           string
		expectedStatus     HealthStatus
		expectedComponents int
		expectedHealthy    int
		expectedDegraded   int
		expectedUnhealthy  int
		shouldError        bool
	}{
		{
			name:               "healthy response",
			jsonFile:           "health_response_healthy.json",
			expectedStatus:     StatusUP,
			expectedComponents: 8,
			expectedHealthy:    8,
			expectedDegraded:   0,
			expectedUnhealthy:  0,
			shouldError:        false,
		},
		{
			name:               "degraded response",
			jsonFile:           "health_response_degraded.json",
			expectedStatus:     StatusDEGRADED,
			expectedComponents: 8,
			expectedHealthy:    5,
			expectedDegraded:   2,
			expectedUnhealthy:  1,
			shouldError:        false,
		},
		{
			name:        "empty JSON",
			jsonFile:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var jsonData []byte
			var err error

			if tt.jsonFile == "" {
				jsonData = []byte("")
			} else {
				jsonData, err = loadTestData(tt.jsonFile)
				if err != nil {
					t.Fatalf("Failed to load test data: %v", err)
				}
			}

			parsed, err := ParseHealthResponseWithPodName(jsonData, "test-pod")

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if parsed == nil {
				t.Fatal("Parsed data is nil")
			}

			if parsed.OverallStatus != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, parsed.OverallStatus)
			}

			if parsed.ComponentCount != tt.expectedComponents {
				t.Errorf("Expected %d components, got %d", tt.expectedComponents, parsed.ComponentCount)
			}

			if parsed.HealthyComponents != tt.expectedHealthy {
				t.Errorf("Expected %d healthy components, got %d", tt.expectedHealthy, parsed.HealthyComponents)
			}

			if parsed.DegradedComponents != tt.expectedDegraded {
				t.Errorf("Expected %d degraded components, got %d", tt.expectedDegraded, parsed.DegradedComponents)
			}

			if parsed.UnhealthyComponents != tt.expectedUnhealthy {
				t.Errorf("Expected %d unhealthy components, got %d", tt.expectedUnhealthy, parsed.UnhealthyComponents)
			}

			// Test validation - should not fail with pod name set
			if err := parsed.Validate(); err != nil {
				t.Errorf("Parsed data validation failed: %v", err)
			}
		})
	}
}

func TestParseHealthResponseWithPodName(t *testing.T) {
	jsonData, err := loadTestData("health_response_healthy.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	podName := "broker-0"
	parsed, err := ParseHealthResponseWithPodName(jsonData, podName)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsed.PodName != podName {
		t.Errorf("Expected pod name %q, got %q", podName, parsed.PodName)
	}
}

func TestHealthStatusValidation(t *testing.T) {
	tests := []struct {
		status      HealthStatus
		shouldError bool
		isHealthy   bool
		isDegraded  bool
		isUnhealthy bool
	}{
		{StatusUP, false, true, false, false},
		{StatusDOWN, false, false, false, true},
		{StatusDEGRADED, false, false, true, false},
		{StatusUNKNOWN, false, false, false, false},
		{StatusOUTOFSERVICE, false, false, false, true},
		{HealthStatus("INVALID"), true, false, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			err := tt.status.Validate()
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}

			if tt.status.IsHealthy() != tt.isHealthy {
				t.Errorf("IsHealthy() = %v, expected %v", tt.status.IsHealthy(), tt.isHealthy)
			}

			if tt.status.IsDegraded() != tt.isDegraded {
				t.Errorf("IsDegraded() = %v, expected %v", tt.status.IsDegraded(), tt.isDegraded)
			}

			if tt.status.IsUnhealthy() != tt.isUnhealthy {
				t.Errorf("IsUnhealthy() = %v, expected %v", tt.status.IsUnhealthy(), tt.isUnhealthy)
			}
		})
	}
}

func TestExtensionParsing(t *testing.T) {
	jsonData, err := loadTestData("health_response_healthy.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	parsed, err := ParseHealthResponseWithPodName(jsonData, "test-pod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that extensions are parsed correctly
	extensionFound := false
	for _, comp := range parsed.ComponentDetails {
		if comp.Name == "extensions" {
			extensionFound = true
			// Should have sub-components for individual extensions
			if len(comp.SubComponents) == 0 {
				t.Error("Extensions component should have sub-components")
			}

			// Check for specific extensions from test data
			foundExtensions := make(map[string]bool)
			for _, subComp := range comp.SubComponents {
				foundExtensions[subComp.Name] = true
			}

			expectedExtensions := []string{
				"hivemq-cloud-metering-extension",
				"hivemq-dns-cluster-discovery",
				"hivemq-enterprise-security-extension",
				"hivemq-prometheus-extension",
			}

			for _, expected := range expectedExtensions {
				if !foundExtensions[expected] {
					t.Errorf("Expected extension %q not found", expected)
				}
			}
			break
		}
	}

	if !extensionFound {
		t.Error("Extensions component not found in parsed data")
	}
}

func TestDegradedHealthParsing(t *testing.T) {
	jsonData, err := loadTestData("health_response_degraded.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	parsed, err := ParseHealthResponseWithPodName(jsonData, "test-pod")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should identify degraded/failed components
	if parsed.UnhealthyComponents == 0 {
		t.Error("Expected to find unhealthy components in degraded response")
	}

	if parsed.DegradedComponents == 0 {
		t.Error("Expected to find degraded components in degraded response")
	}

	// Overall status should be DEGRADED
	if parsed.OverallStatus != StatusDEGRADED {
		t.Errorf("Expected overall status DEGRADED, got %v", parsed.OverallStatus)
	}

	// Should not be considered healthy
	if parsed.IsHealthy() {
		t.Error("Degraded response should not be considered healthy")
	}
}

func TestMalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"invalid JSON", `{invalid json`},
		{"array instead of object", `[]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseHealthResponseWithPodName([]byte(tt.json), "test-pod")
			if err == nil {
				t.Error("Expected error for malformed JSON")
			}
		})
	}
}

func TestComponentStatusValidation(t *testing.T) {
	tests := []struct {
		name        string
		component   ComponentStatus
		shouldError bool
	}{
		{
			name: "valid component",
			component: ComponentStatus{
				Name:    "test-component",
				Status:  StatusUP,
				Details: "All systems operational",
			},
			shouldError: false,
		},
		{
			name: "empty name",
			component: ComponentStatus{
				Name:   "",
				Status: StatusUP,
			},
			shouldError: true,
		},
		{
			name: "invalid status",
			component: ComponentStatus{
				Name:   "test-component",
				Status: HealthStatus("INVALID"),
			},
			shouldError: true,
		},
		{
			name: "valid with sub-components",
			component: ComponentStatus{
				Name:   "extensions",
				Status: StatusUP,
				SubComponents: []ComponentStatus{
					{
						Name:   "prometheus-extension",
						Status: StatusUP,
					},
				},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.component.Validate()
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestHealthCheckOptionsValidation(t *testing.T) {
	tests := []struct {
		name        string
		options     HealthCheckOptions
		shouldError bool
	}{
		{
			name: "valid options",
			options: HealthCheckOptions{
				Endpoint:   "health",
				OutputJSON: false,
				OutputRaw:  false,
				Detailed:   true,
				Timeout:    30 * 1000 * 1000 * 1000, // 30 seconds in nanoseconds
				UseColors:  true,
			},
			shouldError: false,
		},
		{
			name: "invalid endpoint",
			options: HealthCheckOptions{
				Endpoint: "invalid",
				Timeout:  30 * 1000 * 1000 * 1000,
			},
			shouldError: true,
		},
		{
			name: "conflicting output options",
			options: HealthCheckOptions{
				Endpoint:   "health",
				OutputJSON: true,
				OutputRaw:  true,
				Timeout:    30 * 1000 * 1000 * 1000,
			},
			shouldError: true,
		},
		{
			name: "timeout too short",
			options: HealthCheckOptions{
				Endpoint: "health",
				Timeout:  500 * 1000 * 1000, // 500ms
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

func TestWithDefaults(t *testing.T) {
	options := HealthCheckOptions{}
	optionsWithDefaults := options.WithDefaults()

	if optionsWithDefaults.Endpoint == "" {
		t.Error("Expected default endpoint to be set")
	}

	if optionsWithDefaults.Timeout == 0 {
		t.Error("Expected default timeout to be set")
	}
}

// loadTestData loads JSON test data from testdata directory
func loadTestData(filename string) ([]byte, error) {
	path := filepath.Join("..", "..", "testdata", filename)
	return os.ReadFile(path)
}

// BenchmarkParseHealthResponse benchmarks the health response parsing
func BenchmarkParseHealthResponse(b *testing.B) {
	jsonData, err := loadTestData("health_response_healthy.json")
	if err != nil {
		b.Fatalf("Failed to load test data: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseHealthResponseWithPodName(jsonData, "test-pod")
		if err != nil {
			b.Fatalf("Parse error: %v", err)
		}
	}
}

// BenchmarkParseHealthResponseWithPodName benchmarks parsing with pod name
func BenchmarkParseHealthResponseWithPodName(b *testing.B) {
	jsonData, err := loadTestData("health_response_healthy.json")
	if err != nil {
		b.Fatalf("Failed to load test data: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseHealthResponseWithPodName(jsonData, "broker-0")
		if err != nil {
			b.Fatalf("Parse error: %v", err)
		}
	}
}

// Test to ensure pool efficiency for memory optimization
func TestMemoryPoolUsage(t *testing.T) {
	jsonData, err := loadTestData("health_response_healthy.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	// Parse multiple times to test pool reuse
	for i := 0; i < 100; i++ {
		parsed, err := ParseHealthResponseWithPodName(jsonData, "broker-0")
		if err != nil {
			t.Fatalf("Parse error on iteration %d: %v", i, err)
		}
		if parsed == nil {
			t.Fatalf("Nil parsed data on iteration %d", i)
		}
	}
}
