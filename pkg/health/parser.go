package health

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseHealthResponse parses a HiveMQ health API JSON response
func ParseHealthResponse(jsonData []byte) (*ParsedHealthData, error) {
	var healthResp HealthResponse
	if err := json.Unmarshal(jsonData, &healthResp); err != nil {
		return nil, fmt.Errorf("failed to parse health response JSON: %w", err)
	}

	parsed := &ParsedHealthData{
		OverallStatus: healthResp.Status,
		RawJSON:       jsonData,
	}

	// Parse components if available
	if healthResp.Components != nil {
		parsed.ComponentCount = len(healthResp.Components)

		for name, component := range healthResp.Components {
			componentStatus := ComponentStatus{
				Name:   name,
				Status: component.Status,
			}

			// Extract details as a formatted string
			if component.Details != nil {
				details := make([]string, 0)
				for key, value := range component.Details {
					details = append(details, fmt.Sprintf("%s: %v", key, value))
				}
				componentStatus.Details = strings.Join(details, ", ")
			}

			parsed.ComponentDetails = append(parsed.ComponentDetails, componentStatus)

			// Count component health status
			switch component.Status {
			case StatusUP:
				parsed.HealthyComponents++
			case StatusDEGRADED:
				parsed.DegradedComponents++
			case StatusDOWN, StatusUNKNOWN, StatusOUTOFSERVICE:
				parsed.UnhealthyComponents++
			}
		}
	}

	return parsed, nil
}

// GetHealthEndpointPath returns the full API path for a given health endpoint
func GetHealthEndpointPath(endpoint string) string {
	switch endpoint {
	case "liveness":
		return "/api/v1/health/liveness"
	case "readiness":
		return "/api/v1/health/readiness"
	case "health", "":
		return "/api/v1/health"
	default:
		// Allow custom endpoints to be passed through
		if strings.HasPrefix(endpoint, "/") {
			return endpoint
		}
		return "/api/v1/health/" + endpoint
	}
}

// FormatHealthStatus returns a formatted string representation of health status
func FormatHealthStatus(status HealthStatus) string {
	switch status {
	case StatusUP:
		return "[UP]"
	case StatusDOWN:
		return "[DOWN]"
	case StatusDEGRADED:
		return "[DEGRADED]"
	case StatusUNKNOWN:
		return "[UNKNOWN]"
	case StatusOUTOFSERVICE:
		return "[OUT_OF_SERVICE]"
	default:
		return fmt.Sprintf("[%s]", string(status))
	}
}

// IsHealthy returns true if the health status indicates a healthy state
func IsHealthy(status HealthStatus) bool {
	return status == StatusUP
}

// GetHealthSummary returns a human-readable summary of parsed health data
func GetHealthSummary(parsed *ParsedHealthData) string {
	if parsed.ComponentCount == 0 {
		return fmt.Sprintf("Overall: %s", FormatHealthStatus(parsed.OverallStatus))
	}

	summary := fmt.Sprintf("Overall: %s, Components: %d total",
		FormatHealthStatus(parsed.OverallStatus), parsed.ComponentCount)

	if parsed.HealthyComponents > 0 {
		summary += fmt.Sprintf(", %d healthy", parsed.HealthyComponents)
	}
	if parsed.DegradedComponents > 0 {
		summary += fmt.Sprintf(", %d degraded", parsed.DegradedComponents)
	}
	if parsed.UnhealthyComponents > 0 {
		summary += fmt.Sprintf(", %d unhealthy", parsed.UnhealthyComponents)
	}

	return summary
}
