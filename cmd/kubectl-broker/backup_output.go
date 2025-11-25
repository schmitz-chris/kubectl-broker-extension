package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"sigs.k8s.io/yaml"

	"kubectl-broker/pkg/sidecar"
)

var (
	remoteBackupColumns = []tableColumn{
		{Title: "OBJECT", Width: 48},
		{Title: "SIZE", Width: 12},
		{Title: "AGE", Width: 12},
	}
	sidecarBackupColumns = []tableColumn{
		{Title: "FOLDER", Width: 28},
		{Title: "SIZE", Width: 12},
		{Title: "UPDATED", Width: 20},
		{Title: "STATUS", Width: 12},
	}
)

func renderRemoteBackups(engine string, backups []sidecar.RemoteBackupInfo) {
	scope := backupScopeForEngine(engine)
	switch currentOutputFormat() {
	case "json":
		writeStructuredBackupOutput(remoteBackupsPayload{Scope: scope, Items: backups}, "json")
	case "yaml":
		writeStructuredBackupOutput(remoteBackupsPayload{Scope: scope, Items: backups}, "yaml")
	default:
		renderRemoteBackupTable(backups)
	}
}

func renderRemoteBackupTable(backups []sidecar.RemoteBackupInfo) {
	if len(backups) == 0 {
		fmt.Println("No remote backups found.")
		return
	}

	renderTableHeader(remoteBackupColumns, 2)
	now := time.Now()
	for _, item := range backups {
		age := formatRelativeAge(now.Sub(item.LastModified))
		fmt.Printf("%-48s  %-12s  %-12s\n",
			truncateString(item.Key, 48),
			formatBytes(item.SizeBytes),
			age)
	}
	fmt.Printf("\nSummary: %d remote backups\n", len(backups))
}

func renderSidecarInventory(engine string, inv sidecar.Inventory) {
	scope := backupScopeForEngine(engine)
	switch currentOutputFormat() {
	case "json":
		writeStructuredBackupOutput(struct {
			Scope     backupScope       `json:"scope"`
			Inventory sidecar.Inventory `json:"inventory"`
		}{Scope: scope, Inventory: inv}, "json")
	case "yaml":
		writeStructuredBackupOutput(struct {
			Scope     backupScope       `json:"scope"`
			Inventory sidecar.Inventory `json:"inventory"`
		}{Scope: scope, Inventory: inv}, "yaml")
	default:
		renderSidecarInventoryTables(inv)
	}
}

func renderSidecarInventoryTables(inv sidecar.Inventory) {
	fmt.Println("Backups (HiveMQ)")
	if len(inv.Backups) == 0 {
		fmt.Println("  No backups detected on disk.")
	} else {
		renderTableHeader(sidecarBackupColumns, 2)
		for _, backup := range inv.Backups {
			fmt.Printf("%-28s  %-12s  %-20s  %s\n",
				truncateString(backup.Name, 28),
				formatBytes(backup.SizeBytes),
				backup.LastModified.Format("2006-01-02 15:04:05"),
				formatSidecarStatus(backup.Status))
		}
	}

	fmt.Println()
	fmt.Println("Cluster Backups")
	if len(inv.ClusterBackups) == 0 {
		fmt.Println("  No cluster backups detected.")
	} else {
		renderTableHeader(sidecarBackupColumns, 2)
		for _, cluster := range inv.ClusterBackups {
			fmt.Printf("%-28s  %-12s  %-20s  %s\n",
				truncateString(cluster.Name, 28),
				formatBytes(cluster.SizeBytes),
				cluster.LastModified.Format("2006-01-02 15:04:05"),
				formatSidecarStatus(cluster.Status))
		}
	}
}

func renderRemoteRestoreResult(engine string, result *sidecar.RestoreResult, dryRun bool) {
	if result == nil {
		fmt.Println("Remote restore completed.")
		return
	}

	scope := backupScopeForEngine(engine)
	switch currentOutputFormat() {
	case "json":
		writeStructuredBackupOutput(struct {
			Scope  backupScope            `json:"scope"`
			Result *sidecar.RestoreResult `json:"result"`
			Mode   string                 `json:"mode"`
		}{Scope: scope, Result: result, Mode: restoreModeLabel(dryRun)}, "json")
	case "yaml":
		writeStructuredBackupOutput(struct {
			Scope  backupScope            `json:"scope"`
			Result *sidecar.RestoreResult `json:"result"`
			Mode   string                 `json:"mode"`
		}{Scope: scope, Result: result, Mode: restoreModeLabel(dryRun)}, "yaml")
	default:
		fmt.Printf("Remote restore completed (%s)\n", restoreModeLabel(dryRun))
		fmt.Printf("Object: %s\n", result.Key)
		fmt.Printf("Size: %s\n", formatBytes(result.Bytes))
		fmt.Printf("Target: %s\n", result.TargetPath)
		fmt.Printf("Last Checked: %s\n", result.LastChecked.Format(time.RFC3339))
	}
}

func restoreModeLabel(dryRun bool) string {
	if dryRun {
		return "dry-run"
	}
	return "full"
}

func formatSidecarStatus(state sidecar.BackupState) string {
	value := strings.ToUpper(string(state))
	if !colorOutputEnabled() {
		return value
	}

	switch state {
	case sidecar.BackupStateCompleted:
		return color.New(color.FgGreen, color.Bold).Sprint(value)
	case sidecar.BackupStateUploading:
		return color.New(color.FgYellow, color.Bold).Sprint(value)
	case sidecar.BackupStateFailed:
		return color.New(color.FgRed, color.Bold).Sprint(value)
	default:
		return color.New(color.FgWhite).Sprint(value)
	}
}

func formatRelativeAge(duration time.Duration) string {
	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	}
	days := int(duration.Hours()) / 24
	return fmt.Sprintf("%dd", days)
}

type backupScope struct {
	Namespace   string `json:"namespace" yaml:"namespace"`
	StatefulSet string `json:"statefulset" yaml:"statefulset"`
	Engine      string `json:"engine" yaml:"engine"`
}

type remoteBackupsPayload struct {
	Scope backupScope                `json:"scope" yaml:"scope"`
	Items []sidecar.RemoteBackupInfo `json:"items" yaml:"items"`
}

func backupScopeForEngine(engine string) backupScope {
	value := strings.ToLower(strings.TrimSpace(engine))
	if value == "" {
		value = backupScopeEngineManagement
	}
	return backupScope{
		Namespace:   backupNamespace,
		StatefulSet: backupStatefulSetName,
		Engine:      value,
	}
}

func writeStructuredBackupOutput(payload any, format string) {
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
		fmt.Printf("failed to render %s output: %v\n", format, err)
		return
	}
	fmt.Println(string(data))
}
