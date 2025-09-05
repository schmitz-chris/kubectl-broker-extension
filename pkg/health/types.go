package health

import (
	"fmt"
	"strings"
	"time"
)

// HealthStatus represents the overall health status values from HiveMQ
type HealthStatus string

const (
	StatusUP           HealthStatus = "UP"
	StatusDOWN         HealthStatus = "DOWN"
	StatusDEGRADED     HealthStatus = "DEGRADED"
	StatusUNKNOWN      HealthStatus = "UNKNOWN"
	StatusOUTOFSERVICE HealthStatus = "OUT_OF_SERVICE"
)

// IsValid checks if the HealthStatus is a valid value
func (hs HealthStatus) IsValid() bool {
	switch hs {
	case StatusUP, StatusDOWN, StatusDEGRADED, StatusUNKNOWN, StatusOUTOFSERVICE:
		return true
	default:
		return false
	}
}

// String returns the string representation of HealthStatus
func (hs HealthStatus) String() string {
	return string(hs)
}

// IsHealthy returns true if the status indicates a healthy state
func (hs HealthStatus) IsHealthy() bool {
	return hs == StatusUP
}

// IsDegraded returns true if the status indicates a degraded state
func (hs HealthStatus) IsDegraded() bool {
	return hs == StatusDEGRADED
}

// IsUnhealthy returns true if the status indicates an unhealthy state
func (hs HealthStatus) IsUnhealthy() bool {
	return hs == StatusDOWN || hs == StatusOUTOFSERVICE
}

// Validate checks if the health status is valid and returns an error if not
func (hs HealthStatus) Validate() error {
	if !hs.IsValid() {
		return fmt.Errorf("invalid health status: %q, must be one of: UP, DOWN, DEGRADED, UNKNOWN, OUT_OF_SERVICE", string(hs))
	}
	return nil
}

// HealthResponse represents the structure of HiveMQ health API responses
type HealthResponse struct {
	Status     HealthStatus               `json:"status"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
	Details    map[string]interface{}     `json:"details,omitempty"`
}

// ComponentHealth represents health information for individual components
type ComponentHealth struct {
	Status     HealthStatus               `json:"status"`
	Details    map[string]interface{}     `json:"details,omitempty"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
}

// ParsedHealthData represents analyzed health information for display
type ParsedHealthData struct {
	PodName             string        `validate:"required"` // Pod name for JSON output
	OverallStatus       HealthStatus  `validate:"required"` 
	ComponentCount      int           `validate:"min=0"`
	HealthyComponents   int           `validate:"min=0"`
	DegradedComponents  int           `validate:"min=0"`
	UnhealthyComponents int           `validate:"min=0"`
	ComponentDetails    []ComponentStatus
	RawJSON             []byte
}

// Validate validates the ParsedHealthData
func (phd *ParsedHealthData) Validate() error {
	if phd == nil {
		return fmt.Errorf("ParsedHealthData cannot be nil")
	}

	// Validate pod name
	if strings.TrimSpace(phd.PodName) == "" {
		return fmt.Errorf("PodName cannot be empty")
	}

	// Validate overall status
	if err := phd.OverallStatus.Validate(); err != nil {
		return fmt.Errorf("invalid OverallStatus: %w", err)
	}

	// Validate component counts are non-negative
	if phd.ComponentCount < 0 {
		return fmt.Errorf("ComponentCount cannot be negative: %d", phd.ComponentCount)
	}
	if phd.HealthyComponents < 0 {
		return fmt.Errorf("HealthyComponents cannot be negative: %d", phd.HealthyComponents)
	}
	if phd.DegradedComponents < 0 {
		return fmt.Errorf("DegradedComponents cannot be negative: %d", phd.DegradedComponents)
	}
	if phd.UnhealthyComponents < 0 {
		return fmt.Errorf("UnhealthyComponents cannot be negative: %d", phd.UnhealthyComponents)
	}

	// Validate that component counts add up correctly
	totalCounted := phd.HealthyComponents + phd.DegradedComponents + phd.UnhealthyComponents
	if totalCounted > phd.ComponentCount {
		return fmt.Errorf("sum of health component counts (%d) exceeds total component count (%d)", 
			totalCounted, phd.ComponentCount)
	}

	// Validate individual component details
	for i, comp := range phd.ComponentDetails {
		if err := comp.Validate(); err != nil {
			return fmt.Errorf("invalid ComponentStatus at index %d: %w", i, err)
		}
	}

	return nil
}

// IsHealthy returns true if all components are healthy
func (phd *ParsedHealthData) IsHealthy() bool {
	return phd.OverallStatus.IsHealthy() && phd.UnhealthyComponents == 0 && phd.DegradedComponents == 0
}

// ComponentStatus represents the status of an individual component
type ComponentStatus struct {
	Name          string        `validate:"required"`
	Status        HealthStatus  `validate:"required"`
	Details       string
	SubComponents []ComponentStatus // For nested components like individual extensions
}

// Validate validates the ComponentStatus
func (cs *ComponentStatus) Validate() error {
	if cs == nil {
		return fmt.Errorf("ComponentStatus cannot be nil")
	}

	// Validate name
	if strings.TrimSpace(cs.Name) == "" {
		return fmt.Errorf("Name cannot be empty")
	}

	// Validate status
	if err := cs.Status.Validate(); err != nil {
		return fmt.Errorf("invalid Status for component %q: %w", cs.Name, err)
	}

	// Validate sub-components recursively
	for i, subComp := range cs.SubComponents {
		if err := subComp.Validate(); err != nil {
			return fmt.Errorf("invalid SubComponent at index %d for component %q: %w", i, cs.Name, err)
		}
	}

	return nil
}

// HealthCheckOptions configures how health checks are performed and displayed
type HealthCheckOptions struct {
	Endpoint   string        `validate:"required,oneof=health liveness readiness"` // health endpoint to query (health, liveness, readiness)
	OutputJSON bool          // output raw JSON instead of parsed data
	OutputRaw  bool          // output unprocessed response
	Detailed   bool          // show detailed component breakdown
	Timeout    time.Duration `validate:"min=1s,max=300s"` // timeout for health check requests
	UseColors  bool          // enable colored output for health status
}

// Validate validates the HealthCheckOptions
func (opts *HealthCheckOptions) Validate() error {
	if opts == nil {
		return fmt.Errorf("HealthCheckOptions cannot be nil")
	}

	// Validate endpoint
	validEndpoints := []string{"health", "liveness", "readiness"}
	if opts.Endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty, must be one of: %s", strings.Join(validEndpoints, ", "))
	}
	
	found := false
	for _, valid := range validEndpoints {
		if opts.Endpoint == valid {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("invalid endpoint %q, must be one of: %s", opts.Endpoint, strings.Join(validEndpoints, ", "))
	}

	// Validate timeout
	if opts.Timeout < time.Second {
		return fmt.Errorf("timeout too short: %v, minimum is 1 second", opts.Timeout)
	}
	if opts.Timeout > 5*time.Minute {
		return fmt.Errorf("timeout too long: %v, maximum is 5 minutes", opts.Timeout)
	}

	// Validate output options are mutually exclusive
	if opts.OutputJSON && opts.OutputRaw {
		return fmt.Errorf("OutputJSON and OutputRaw cannot both be true")
	}

	return nil
}

// WithDefaults returns a copy of HealthCheckOptions with default values applied
func (opts HealthCheckOptions) WithDefaults() HealthCheckOptions {
	if opts.Endpoint == "" {
		opts.Endpoint = "health"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Second
	}
	return opts
}

// DefaultHealthCheckOptions Default health check options
var DefaultHealthCheckOptions = HealthCheckOptions{
	Endpoint:   "health",
	OutputJSON: false,
	OutputRaw:  false,
	Detailed:   false,
	Timeout:    10 * time.Second,
	UseColors:  true, // Enable colors by default
}
