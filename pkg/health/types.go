package health

import "time"

// HealthStatus represents the overall health status values from HiveMQ
type HealthStatus string

const (
	StatusUP           HealthStatus = "UP"
	StatusDOWN         HealthStatus = "DOWN"
	StatusDEGRADED     HealthStatus = "DEGRADED"
	StatusUNKNOWN      HealthStatus = "UNKNOWN"
	StatusOUTOFSERVICE HealthStatus = "OUT_OF_SERVICE"
)

// HealthResponse represents the structure of HiveMQ health API responses
type HealthResponse struct {
	Status     HealthStatus               `json:"status"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
	Details    map[string]interface{}     `json:"details,omitempty"`
}

// ComponentHealth represents health information for individual components
type ComponentHealth struct {
	Status  HealthStatus           `json:"status"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ParsedHealthData represents analyzed health information for display
type ParsedHealthData struct {
	OverallStatus       HealthStatus
	ComponentCount      int
	HealthyComponents   int
	DegradedComponents  int
	UnhealthyComponents int
	ComponentDetails    []ComponentStatus
	RawJSON             []byte
}

// ComponentStatus represents the status of an individual component
type ComponentStatus struct {
	Name    string
	Status  HealthStatus
	Details string
}

// HealthCheckOptions configures how health checks are performed and displayed
type HealthCheckOptions struct {
	Endpoint   string        // health endpoint to query (health, liveness, readiness)
	OutputJSON bool          // output raw JSON instead of parsed data
	OutputRaw  bool          // output unprocessed response
	Detailed   bool          // show detailed component breakdown
	Timeout    time.Duration // timeout for health check requests
	UseColors  bool          // enable colored output for health status
}

// Default health check options
var DefaultHealthCheckOptions = HealthCheckOptions{
	Endpoint:   "health",
	OutputJSON: false,
	OutputRaw:  false,
	Detailed:   false,
	Timeout:    10 * time.Second,
	UseColors:  true, // Enable colors by default
}
