package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "kubectl-broker",
		Short: "Comprehensive HiveMQ cluster management toolkit for Kubernetes",
		Long: `kubectl-broker is a kubectl plugin that provides comprehensive management 
tools for HiveMQ clusters running on Kubernetes. It includes health diagnostics, 
backup operations, and other cluster management features.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Add subcommands
	rootCmd.AddCommand(newStatusCommand())
	rootCmd.AddCommand(newBackupCommand())

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

