package main

import (
	"encoding/json"
	"fmt"
	"time"

	"kubectl-broker/pkg/volumes"

	"github.com/fatih/color"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
)

func displayVolumesList(result *volumes.AnalysisResult, options volumes.AnalysisOptions) error {
	switch currentOutputFormat() {
	case "json":
		return writeStructuredVolumesOutput(result, options, "json")
	case "yaml":
		return writeStructuredVolumesOutput(result, options, "yaml")
	default:
		displayVolumesListTable(result, options)
		return nil
	}
}

func displayVolumesListTable(result *volumes.AnalysisResult, options volumes.AnalysisOptions) {
	totalVolumes := len(result.ReleasedPVs) + len(result.OrphanedPVCs) + len(result.BoundVolumes)

	if totalVolumes == 0 {
		if options.AllNamespaces {
			fmt.Println("No volumes found across cluster.")
		} else {
			fmt.Printf("No volumes found in namespace: %s\n", options.Namespace)
		}
		return
	}

	if options.ShowDetailed {
		fmt.Printf("VOLUME NAME                               SIZE     USED     AVAIL    USAGE%%  AGE      STATUS       NAMESPACE\n")
		fmt.Printf("----------------------------------------  -------  -------  -------  ------  -------  -----------  ---------\n")
	} else {
		fmt.Printf("VOLUME NAME                               SIZE     AGE      STATUS       NAMESPACE\n")
		fmt.Printf("----------------------------------------  -------  -------  -----------  ---------\n")
	}

	for _, pv := range result.ReleasedPVs {
		age := time.Since(pv.CreationTimestamp.Time).Round(24 * time.Hour)
		statusColor := getVolumeStatusColor("RELEASED", options.UseColors)
		namespace := ""
		if pv.Spec.ClaimRef != nil {
			namespace = pv.Spec.ClaimRef.Namespace
		}

		if options.ShowDetailed {
			used, available, usagePercent := "-", "-", "-"

			fmt.Printf("%-40s  %-7s  %-7s  %-7s  %-6s  %-7s  %s  %s\n",
				truncateString(pv.Name, 40),
				formatStorageSize(pv.Spec.Capacity["storage"]),
				used,
				available,
				usagePercent,
				formatDuration(age),
				statusColor.Sprint("RELEASED"),
				namespace)
		} else {
			fmt.Printf("%-40s  %-7s  %-7s  %s  %s\n",
				truncateString(pv.Name, 40),
				formatStorageSize(pv.Spec.Capacity["storage"]),
				formatDuration(age),
				statusColor.Sprint("RELEASED"),
				namespace)
		}
	}

	for _, pvc := range result.OrphanedPVCs {
		age := time.Since(pvc.CreationTimestamp.Time).Round(24 * time.Hour)
		statusColor := getVolumeStatusColor("ORPHANED", options.UseColors)

		if options.ShowDetailed {
			used, available, usagePercent := "-", "-", "-"

			fmt.Printf("%-40s  %-7s  %-7s  %-7s  %-6s  %-7s  %s  %s\n",
				truncateString(pvc.Name, 40),
				formatStorageSize(pvc.Spec.Resources.Requests["storage"]),
				used,
				available,
				usagePercent,
				formatDuration(age),
				statusColor.Sprint("ORPHANED"),
				pvc.Namespace)
		} else {
			fmt.Printf("%-40s  %-7s  %-7s  %s  %s\n",
				truncateString(pvc.Name, 40),
				formatStorageSize(pvc.Spec.Resources.Requests["storage"]),
				formatDuration(age),
				statusColor.Sprint("ORPHANED"),
				pvc.Namespace)
		}
	}

	if options.ShowAll || (!options.ShowReleased && !options.ShowOrphaned) {
		for _, volume := range result.BoundVolumes {
			statusColor := getVolumeStatusColor("BOUND", options.UseColors)

			if options.ShowDetailed {
				used, available, usagePercent := formatUsageInfo(volume.Usage)

				fmt.Printf("%-40s  %-7s  %-7s  %-7s  %-6s  %-7s  %s  %s\n",
					truncateString(volume.PVC.Name, 40),
					formatStorageSize(volume.PVC.Spec.Resources.Requests["storage"]),
					used,
					available,
					usagePercent,
					formatDuration(volume.Age),
					statusColor.Sprint("BOUND"),
					volume.Namespace)
			} else {
				fmt.Printf("%-40s  %-7s  %-7s  %s  %s\n",
					truncateString(volume.PVC.Name, 40),
					formatStorageSize(volume.PVC.Spec.Resources.Requests["storage"]),
					formatDuration(volume.Age),
					statusColor.Sprint("BOUND"),
					volume.Namespace)
			}
		}
	}

	releasedCount := len(result.ReleasedPVs)
	orphanedCount := len(result.OrphanedPVCs)
	boundCount := len(result.BoundVolumes)

	fmt.Printf("\nSummary: %d released PVs, %d orphaned PVCs", releasedCount, orphanedCount)
	if options.ShowAll || (!options.ShowReleased && !options.ShowOrphaned) {
		fmt.Printf(", %d bound volumes", boundCount)
	}
	fmt.Printf("\n")

	if result.TotalReclaimableStorage > 0 {
		fmt.Printf("Total reclaimable storage: %s\n", formatBytes(result.TotalReclaimableStorage))
	}
}

func writeStructuredVolumesOutput(result *volumes.AnalysisResult, options volumes.AnalysisOptions, format string) error {
	payload := buildVolumeListStructuredOutput(result, options)

	var (
		data []byte
		err  error
	)

	switch format {
	case "yaml":
		data, err = yaml.Marshal(payload)
	default:
		data, err = json.MarshalIndent(payload, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("failed to render %s output: %w", format, err)
	}

	fmt.Println(string(data))
	return nil
}

func buildVolumeListStructuredOutput(result *volumes.AnalysisResult, options volumes.AnalysisOptions) volumeListStructuredOutput {
	output := volumeListStructuredOutput{
		Scope: volumeScope{
			Namespace:     options.Namespace,
			AllNamespaces: options.AllNamespaces,
		},
		Released:               make([]volumeEntry, 0, len(result.ReleasedPVs)),
		Orphaned:               make([]volumeEntry, 0, len(result.OrphanedPVCs)),
		TotalReclaimableBytes:  result.TotalReclaimableStorage,
		TotalReclaimableString: formatBytes(result.TotalReclaimableStorage),
		Summary: volumeSummary{
			Released: len(result.ReleasedPVs),
			Orphaned: len(result.OrphanedPVCs),
			Bound:    len(result.BoundVolumes),
		},
	}

	if options.AllNamespaces {
		output.Scope.Namespace = ""
	}

	for _, pv := range result.ReleasedPVs {
		sizeQuantity := pv.Spec.Capacity["storage"]
		entry := volumeEntry{
			Name:      pv.Name,
			Status:    "RELEASED",
			Age:       formatDuration(time.Since(pv.CreationTimestamp.Time).Round(24 * time.Hour)),
			Size:      formatStorageSize(sizeQuantity),
			SizeBytes: quantityToBytes(sizeQuantity),
		}
		if pv.Spec.ClaimRef != nil {
			entry.Namespace = pv.Spec.ClaimRef.Namespace
		}
		output.Released = append(output.Released, entry)
	}

	for _, pvc := range result.OrphanedPVCs {
		sizeQuantity := pvc.Spec.Resources.Requests["storage"]
		entry := volumeEntry{
			Name:       pvc.Name,
			Namespace:  pvc.Namespace,
			Status:     "ORPHANED",
			Age:        formatDuration(time.Since(pvc.CreationTimestamp.Time).Round(24 * time.Hour)),
			Size:       formatStorageSize(sizeQuantity),
			SizeBytes:  quantityToBytes(sizeQuantity),
			UsageStats: nil,
		}
		output.Orphaned = append(output.Orphaned, entry)
	}

	if options.ShowAll || (!options.ShowReleased && !options.ShowOrphaned) {
		output.Bound = make([]volumeEntry, 0, len(result.BoundVolumes))
		for _, volume := range result.BoundVolumes {
			sizeQuantity := volume.PVC.Spec.Resources.Requests["storage"]
			entry := volumeEntry{
				Name:      volume.PVC.Name,
				Namespace: volume.Namespace,
				Status:    "BOUND",
				Age:       formatDuration(volume.Age),
				Size:      formatStorageSize(sizeQuantity),
				SizeBytes: quantityToBytes(sizeQuantity),
			}

			if volume.Usage != nil {
				entry.UsageStats = &volumeUsageEntry{
					UsedBytes:      volume.Usage.UsedBytes,
					UsedHuman:      formatBytes(volume.Usage.UsedBytes),
					AvailableBytes: volume.Usage.AvailableBytes,
					AvailableHuman: formatBytes(volume.Usage.AvailableBytes),
					UsagePercent:   volume.Usage.UsagePercent,
				}
			}

			output.Bound = append(output.Bound, entry)
		}
	}

	return output
}

func displayCleanupResults(result *volumes.CleanupResult, options volumes.CleanupOptions) {
	if options.DryRun {
		fmt.Printf("DRY RUN - Cleanup summary:\n")
		fmt.Printf("- Released PVs eligible: %d\n", result.PlannedReleasedPVs)
		fmt.Printf("- Orphaned PVCs eligible: %d\n", result.PlannedOrphanedPVCs)
		fmt.Printf("- Total storage reclaimable: %s\n", formatBytes(result.PlannedReclaimedStorage))
		fmt.Printf("\nUse --confirm to proceed with deletion.\n")
		return
	}

	fmt.Printf("Cleanup completed:\n")

	totalPlanned := result.PlannedReleasedPVs + result.PlannedOrphanedPVCs
	totalDeleted := result.DeletedReleasedPVs + result.DeletedOrphanedPVCs
	fmt.Printf("- Planned volumes deleted: %d/%d\n", totalDeleted, totalPlanned)
	fmt.Printf("- Released PVs deleted: %d/%d\n", result.DeletedReleasedPVs, result.PlannedReleasedPVs)
	if result.AssociatedPVsDeleted > 0 {
		fmt.Printf("- Associated PVs deleted during PVC cleanup: %d\n", result.AssociatedPVsDeleted)
	}
	fmt.Printf("- Orphaned PVCs deleted: %d/%d\n", result.DeletedOrphanedPVCs, result.PlannedOrphanedPVCs)
	fmt.Printf("- Storage reclaimed: %s\n", formatBytes(result.TotalReclaimedStorage))

	if len(result.FailedDeletions) > 0 {
		fmt.Printf("- Failed deletions: %d\n", len(result.FailedDeletions))
		fmt.Printf("\nFailed deletions:\n")
		for _, failure := range result.FailedDeletions {
			if failure.Namespace != "" {
				fmt.Printf("- %s %s/%s: %v\n", failure.Type, failure.Namespace, failure.Name, failure.Error)
			} else {
				fmt.Printf("- %s %s: %v\n", failure.Type, failure.Name, failure.Error)
			}
		}
	}
}

func displayDiscoverySummary(result *volumes.AnalysisResult) {
	fmt.Printf("Volume Discovery Summary\n")
	fmt.Printf("========================\n\n")

	fmt.Printf("Total Persistent Volumes: %d\n", result.TotalPVs)
	fmt.Printf("Total Persistent Volume Claims: %d\n", result.TotalPVCs)
	fmt.Printf("Released PVs (reclaimable): %d\n", len(result.ReleasedPVs))
	fmt.Printf("Orphaned PVCs: %d\n", len(result.OrphanedPVCs))

	if result.TotalReclaimableStorage > 0 {
		fmt.Printf("Total reclaimable storage: %s\n", formatBytes(result.TotalReclaimableStorage))
	}

	fmt.Printf("\nNamespaces with orphaned volumes: %d\n", len(result.NamespaceStats))
}

func getVolumeStatusColor(status string, useColors bool) *color.Color {
	if !useColors {
		return color.New()
	}

	switch status {
	case "RELEASED":
		return color.New(color.FgRed, color.Bold)
	case "ORPHANED":
		return color.New(color.FgYellow, color.Bold)
	case "BOUND":
		return color.New(color.FgGreen)
	default:
		return color.New(color.FgWhite)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

func formatStorageSize(quantity interface{}) string {
	if q, ok := quantity.(resource.Quantity); ok {
		bytes := q.Value()
		return formatBytes(bytes)
	}

	if qStr, ok := quantity.(string); ok {
		if parsed, err := resource.ParseQuantity(qStr); err == nil {
			return formatBytes(parsed.Value())
		}
		return qStr
	}

	return fmt.Sprintf("%v", quantity)
}

func formatUsageInfo(usage *volumes.VolumeUsage) (string, string, string) {
	if usage == nil {
		return "-", "-", "-"
	}

	used := formatBytes(usage.UsedBytes)
	available := formatBytes(usage.AvailableBytes)
	usagePercent := fmt.Sprintf("%.0f%%", usage.UsagePercent)

	return used, available, usagePercent
}

type volumeListStructuredOutput struct {
	Scope                  volumeScope   `json:"scope"`
	Released               []volumeEntry `json:"released"`
	Orphaned               []volumeEntry `json:"orphaned"`
	Bound                  []volumeEntry `json:"bound,omitempty"`
	Summary                volumeSummary `json:"summary"`
	TotalReclaimableBytes  int64         `json:"totalReclaimableBytes"`
	TotalReclaimableString string        `json:"totalReclaimable"`
}

type volumeScope struct {
	Namespace     string `json:"namespace,omitempty"`
	AllNamespaces bool   `json:"allNamespaces,omitempty"`
}

type volumeSummary struct {
	Released int `json:"released"`
	Orphaned int `json:"orphaned"`
	Bound    int `json:"bound"`
}

type volumeEntry struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace,omitempty"`
	Status     string            `json:"status"`
	Age        string            `json:"age"`
	Size       string            `json:"size"`
	SizeBytes  int64             `json:"sizeBytes"`
	UsageStats *volumeUsageEntry `json:"usage,omitempty"`
}

type volumeUsageEntry struct {
	UsedBytes      int64   `json:"usedBytes"`
	UsedHuman      string  `json:"used"`
	AvailableBytes int64   `json:"availableBytes"`
	AvailableHuman string  `json:"available"`
	UsagePercent   float64 `json:"usagePercent"`
}

func quantityToBytes(quantity resource.Quantity) int64 {
	q := quantity
	return q.Value()
}
