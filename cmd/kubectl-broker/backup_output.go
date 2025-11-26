package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/yaml"

	"kubectl-broker/pkg/sidecar"
)

var (
	remoteBackupColumns = []tableColumn{
		{Title: "OBJECT", Width: 48},
		{Title: "SIZE", Width: 12},
		{Title: "AGE", Width: 12},
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
