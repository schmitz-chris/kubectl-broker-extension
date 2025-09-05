package health

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/fatih/color"
)

// Parser pools for memory optimization
var (
	bytesBufferPool = sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}

	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			return &strings.Builder{}
		},
	}

	parsedDataPool = sync.Pool{
		New: func() interface{} {
			return &ParsedHealthData{}
		},
	}
)

// ParseHealthResponse parses a HiveMQ health API JSON response
func ParseHealthResponse(jsonData []byte) (*ParsedHealthData, error) {
	return ParseHealthResponseWithPodName(jsonData, "")
}

// ParseHealthResponseWithPodName parses a HiveMQ health API JSON response with pod name (optimized)
func ParseHealthResponseWithPodName(jsonData []byte, podName string) (*ParsedHealthData, error) {
	if len(jsonData) == 0 {
		return nil, fmt.Errorf("empty JSON data provided")
	}

	var healthResp HealthResponse
	if err := json.Unmarshal(jsonData, &healthResp); err != nil {
		return nil, fmt.Errorf("failed to parse health response JSON: %w", err)
	}

	// Get parsed data from pool and reset it
	parsed := parsedDataPool.Get().(*ParsedHealthData)
	resetParsedHealthData(parsed)

	// Set basic fields
	parsed.PodName = podName
	parsed.OverallStatus = healthResp.Status
	parsed.RawJSON = make([]byte, len(jsonData)) // Create a copy to avoid retaining the original slice
	copy(parsed.RawJSON, jsonData)

	// Parse components if available
	if healthResp.Components != nil {
		parsed.ComponentCount = len(healthResp.Components)

		// Pre-allocate slice with known capacity for better performance
		if parsed.ComponentDetails == nil {
			parsed.ComponentDetails = make([]ComponentStatus, 0, len(healthResp.Components))
		}

		for name, component := range healthResp.Components {
			componentStatus := ComponentStatus{
				Name:   name,
				Status: component.Status,
			}

			// Extract details as a formatted string using string builder for efficiency
			if component.Details != nil {
				componentStatus.Details = formatComponentDetails(component.Details)
			}

			// Special handling for extensions component - parse sub-components
			if name == "extensions" {
				componentStatus.SubComponents = parseExtensionsComponentsOptimized(component)
			}

			parsed.ComponentDetails = append(parsed.ComponentDetails, componentStatus)

			// Count component health status using method for consistency
			incrementHealthCounter(parsed, component.Status)
		}
	}

	return parsed, nil
}

// resetParsedHealthData resets a ParsedHealthData struct for reuse
func resetParsedHealthData(parsed *ParsedHealthData) {
	parsed.PodName = ""
	parsed.OverallStatus = ""
	parsed.ComponentCount = 0
	parsed.HealthyComponents = 0
	parsed.DegradedComponents = 0
	parsed.UnhealthyComponents = 0
	parsed.ComponentDetails = parsed.ComponentDetails[:0] // Reset slice but keep capacity
	parsed.RawJSON = nil
}

// formatComponentDetails formats component details efficiently using string builder
func formatComponentDetails(details map[string]interface{}) string {
	if len(details) == 0 {
		return ""
	}

	sb := stringBuilderPool.Get().(*strings.Builder)
	defer stringBuilderPool.Put(sb)
	sb.Reset()

	first := true
	for key, value := range details {
		if !first {
			sb.WriteString(", ")
		}
		sb.WriteString(key)
		sb.WriteString(": ")
		sb.WriteString(fmt.Sprintf("%v", value))
		first = false
	}

	return sb.String()
}

// incrementHealthCounter increments the appropriate health counter
func incrementHealthCounter(parsed *ParsedHealthData, status HealthStatus) {
	switch status {
	case StatusUP:
		parsed.HealthyComponents++
	case StatusDEGRADED:
		parsed.DegradedComponents++
	case StatusDOWN, StatusUNKNOWN, StatusOUTOFSERVICE:
		parsed.UnhealthyComponents++
	}
}

// ReleaseParsedHealthData returns a ParsedHealthData to the pool for reuse
func ReleaseParsedHealthData(parsed *ParsedHealthData) {
	if parsed != nil {
		resetParsedHealthData(parsed)
		parsedDataPool.Put(parsed)
	}
}

// parseExtensionsComponents parses the extensions component to extract individual extension details
func parseExtensionsComponents(extensionsComponent ComponentHealth) []ComponentStatus {
	return parseExtensionsComponentsOptimized(extensionsComponent)
}

// parseExtensionsComponentsOptimized efficiently parses extensions components with memory optimization
func parseExtensionsComponentsOptimized(extensionsComponent ComponentHealth) []ComponentStatus {
	if extensionsComponent.Components == nil {
		return nil
	}

	// Pre-allocate slice with known capacity
	subComponents := make([]ComponentStatus, 0, len(extensionsComponent.Components))
	sb := stringBuilderPool.Get().(*strings.Builder)
	defer stringBuilderPool.Put(sb)

	for extName, extComponent := range extensionsComponent.Components {
		subComp := ComponentStatus{
			Name:   extName,
			Status: extComponent.Status,
		}

		// Extract extension details efficiently
		sb.Reset()
		first := true

		// Extract version
		if extComponent.Details != nil {
			if version, exists := extComponent.Details["version"]; exists {
				sb.WriteString(fmt.Sprintf("v%v", version))
				first = false
			}
		}

		// Look for license information in nested components
		licenseInfo := extractLicenseInfoOptimized(extComponent)
		if licenseInfo != "" {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(licenseInfo)
		}

		subComp.Details = sb.String()
		subComponents = append(subComponents, subComp)
	}

	return subComponents
}

// extractLicenseInfoOptimized efficiently extracts license information
func extractLicenseInfoOptimized(extComponent ComponentHealth) string {
	if extComponent.Components == nil {
		return ""
	}

	internals, exists := extComponent.Components["internals"]
	if !exists || internals.Components == nil {
		return ""
	}

	license, exists := internals.Components["license"]
	if !exists || license.Details == nil {
		return ""
	}

	return extractLicenseInfo(license.Details)
}

// extractLicenseInfo extracts license information from license details
func extractLicenseInfo(licenseDetails map[string]interface{}) string {
	var licenseInfo []string

	if isEnterprise, exists := licenseDetails["is-enterprise"]; exists {
		if enterprise, ok := isEnterprise.(bool); ok && enterprise {
			licenseInfo = append(licenseInfo, "Enterprise")

			// Check trial status for enterprise licenses
			if isTrial, exists := licenseDetails["is-trial"]; exists {
				if trial, ok := isTrial.(bool); ok && trial {
					if isTrialExpired, exists := licenseDetails["is-trial-expired"]; exists {
						if expired, ok := isTrialExpired.(bool); ok && expired {
							licenseInfo = append(licenseInfo, "Trial Expired")
						} else {
							licenseInfo = append(licenseInfo, "Trial")
						}
					} else {
						licenseInfo = append(licenseInfo, "Trial")
					}
				} else {
					licenseInfo = append(licenseInfo, "Licensed")
				}
			} else {
				licenseInfo = append(licenseInfo, "Licensed")
			}
		} else {
			licenseInfo = append(licenseInfo, "Community")
		}
	}

	return strings.Join(licenseInfo, ", ")
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
	return FormatHealthStatusWithColor(status, false)
}

// FormatHealthStatusWithColor returns a formatted string representation of health status with optional colors
func FormatHealthStatusWithColor(status HealthStatus, enableColors bool) string {
	var text string
	switch status {
	case StatusUP:
		text = "[UP]"
		if enableColors {
			return color.New(color.FgGreen, color.Bold).Sprint(text)
		}
	case StatusDOWN:
		text = "[DOWN]"
		if enableColors {
			return color.New(color.FgRed, color.Bold).Sprint(text)
		}
	case StatusDEGRADED:
		text = "[DEGRADED]"
		if enableColors {
			return color.New(color.FgYellow, color.Bold).Sprint(text)
		}
	case StatusUNKNOWN:
		text = "[UNKNOWN]"
		if enableColors {
			return color.New(color.FgWhite).Sprint(text)
		}
	case StatusOUTOFSERVICE:
		text = "[OUT_OF_SERVICE]"
		if enableColors {
			return color.New(color.FgMagenta).Sprint(text)
		}
	default:
		text = fmt.Sprintf("[%s]", string(status))
		if enableColors {
			return color.New(color.FgWhite).Sprint(text)
		}
	}
	return text
}

// IsHealthy returns true if the health status indicates a healthy state
func IsHealthy(status HealthStatus) bool {
	return status == StatusUP
}

// GetHealthSummary returns a human-readable summary of parsed health data
func GetHealthSummary(parsed *ParsedHealthData) string {
	return GetHealthSummaryWithColor(parsed, false)
}

// GetHealthSummaryWithColor returns a human-readable summary of parsed health data with optional colors
func GetHealthSummaryWithColor(parsed *ParsedHealthData, enableColors bool) string {
	if parsed.ComponentCount == 0 {
		return fmt.Sprintf("Overall: %s", FormatHealthStatusWithColor(parsed.OverallStatus, enableColors))
	}

	summary := fmt.Sprintf("Overall: %s, Components: %d total",
		FormatHealthStatusWithColor(parsed.OverallStatus, enableColors), parsed.ComponentCount)

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
