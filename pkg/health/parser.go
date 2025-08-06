package health

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// ParseHealthResponse parses a HiveMQ health API JSON response
func ParseHealthResponse(jsonData []byte) (*ParsedHealthData, error) {
	return ParseHealthResponseWithPodName(jsonData, "")
}

// ParseHealthResponseWithPodName parses a HiveMQ health API JSON response with pod name
func ParseHealthResponseWithPodName(jsonData []byte, podName string) (*ParsedHealthData, error) {
	var healthResp HealthResponse
	if err := json.Unmarshal(jsonData, &healthResp); err != nil {
		return nil, fmt.Errorf("failed to parse health response JSON: %w", err)
	}

	parsed := &ParsedHealthData{
		PodName:       podName,
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

			// Special handling for extensions component - parse sub-components
			if name == "extensions" {
				componentStatus.SubComponents = parseExtensionsComponents(component)
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

// parseExtensionsComponents parses the extensions component to extract individual extension details
func parseExtensionsComponents(extensionsComponent ComponentHealth) []ComponentStatus {
	var subComponents []ComponentStatus

	// Parse the extensions from the Components field
	if extensionsComponent.Components != nil {
		for extName, extComponent := range extensionsComponent.Components {
			subComp := ComponentStatus{
				Name:   extName,
				Status: extComponent.Status,
			}

			// Extract extension details (version, license info)
			var details []string
			if extComponent.Details != nil {
				// Extract version
				if version, exists := extComponent.Details["version"]; exists {
					details = append(details, fmt.Sprintf("v%v", version))
				}

				// Look for license information in nested components
				if extComponent.Components != nil {
					if internals, exists := extComponent.Components["internals"]; exists {
						if internals.Components != nil {
							if license, exists := internals.Components["license"]; exists {
								if license.Details != nil {
									licenseInfo := extractLicenseInfo(license.Details)
									if licenseInfo != "" {
										details = append(details, licenseInfo)
									}
								}
							}
						}
					}
				}
			}

			subComp.Details = strings.Join(details, ", ")
			subComponents = append(subComponents, subComp)
		}
	}

	return subComponents
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
