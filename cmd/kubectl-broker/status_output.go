package main

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"kubectl-broker/pkg/health"
)

// displayHealthCheckResults formats and displays the health check results
func displayHealthCheckResults(pod *v1.Pod, parsedHealth *health.ParsedHealthData, rawJSON []byte, options health.HealthCheckOptions) error {
	if options.OutputRaw {
		fmt.Print(string(rawJSON))
		return nil
	}

	if options.OutputJSON {
		fmt.Println(string(rawJSON))
		return nil
	}

	if options.Detailed && parsedHealth != nil {
		return displayDetailedHealthResults(pod, parsedHealth, options)
	}

	return displayStandardHealthResults(parsedHealth, options)
}

// displayDetailedHealthResults shows detailed component breakdown
func displayDetailedHealthResults(pod *v1.Pod, parsedHealth *health.ParsedHealthData, options health.HealthCheckOptions) error {
	fmt.Printf("Pod: %s\n", pod.Name)
	fmt.Printf("Overall Health: %s\n", health.FormatHealthStatusWithColor(parsedHealth.OverallStatus, options.UseColors))

	if len(parsedHealth.ComponentDetails) > 0 {
		fmt.Println("Components:")
		for _, comp := range parsedHealth.ComponentDetails {
			displayComponentDetails(comp, options.UseColors)
		}
	}

	return nil
}

// displayComponentDetails shows details for a single component
func displayComponentDetails(comp health.ComponentStatus, useColors bool) {
	fmt.Printf("  - %s: %s", comp.Name, health.FormatHealthStatusWithColor(comp.Status, useColors))

	if comp.Details != "" {
		fmt.Printf(" (%s)", comp.Details)
	}

	if comp.Name == "extensions" && len(comp.SubComponents) > 0 {
		displayExtensionDetails(comp.SubComponents, useColors)
	} else {
		fmt.Println()
	}
}

// displayExtensionDetails shows individual extension details
func displayExtensionDetails(extensions []health.ComponentStatus, useColors bool) {
	fmt.Printf(" (%d extensions)", len(extensions))
	fmt.Println()

	for _, ext := range extensions {
		fmt.Printf("    - %s: %s", ext.Name, health.FormatHealthStatusWithColor(ext.Status, useColors))
		if ext.Details != "" {
			fmt.Printf(" (%s)", ext.Details)
		}
		fmt.Println()
	}
}

// displayStandardHealthResults shows standard output format
func displayStandardHealthResults(parsedHealth *health.ParsedHealthData, options health.HealthCheckOptions) error {
	if parsedHealth != nil {
		fmt.Printf("Health check successful: %s\n", health.FormatHealthStatusWithColor(parsedHealth.OverallStatus, options.UseColors))
		fmt.Printf("Summary: %s\n", health.GetHealthSummaryWithColor(parsedHealth, options.UseColors))
	} else {
		fmt.Println("Health check completed")
	}
	return nil
}

// shouldShowDebugInfo returns whether debug information should be displayed
func shouldShowDebugInfo() bool {
	return !outputJSON && !outputRaw && detailed
}
