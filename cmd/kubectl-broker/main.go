package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ProductMode represents the invocation mode
type ProductMode int

const (
	ModeBroker ProductMode = iota
	ModePulse
)

// ProductContext holds the product mode and related configuration
type ProductContext struct {
	Mode ProductMode
	Name string
}

// GlobalFlags holds global configuration flags
type GlobalFlags struct {
	NoColor bool
	Output  string
}

// GlobalConfig holds the global configuration after processing flags and environment
type GlobalConfig struct {
	ColorsEnabled bool
	OutputFormat  string
}

var globalFlags GlobalFlags

func main() {
	// Detect product mode based on invocation name
	productCtx := detectProductMode()

	// Create root command based on product mode
	rootCmd := createRootCommand(productCtx)

	// Add global flags
	addGlobalFlags(rootCmd)

	// Add appropriate subcommands based on mode
	addSubcommands(rootCmd, productCtx)

	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// addGlobalFlags adds global flags to the root command
func addGlobalFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().BoolVar(&globalFlags.NoColor, "no-color", false, "Disable ANSI color output")
	rootCmd.PersistentFlags().StringVar(&globalFlags.Output, "output", "table", "Output format: table, json, yaml")

	// Add validation for output format
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if globalFlags.Output != "table" && globalFlags.Output != "json" && globalFlags.Output != "yaml" {
			return fmt.Errorf("invalid output format '%s'. Must be one of: table, json, yaml", globalFlags.Output)
		}
		return nil
	}
}

// processGlobalConfig processes global flags and environment variables to determine final config
func processGlobalConfig() GlobalConfig {
	config := GlobalConfig{
		OutputFormat: globalFlags.Output,
	}

	// Determine color settings based on precedence:
	// --no-color > NO_COLOR > CLICOLOR_FORCE > CI > TTY detection

	// Start with TTY detection
	colorsEnabled := isTerminal(os.Stdout)

	// Check CI environment (disable colors in CI by default)
	if os.Getenv("CI") != "" {
		colorsEnabled = false
	}

	// Check CLICOLOR_FORCE (enable colors)
	if os.Getenv("CLICOLOR_FORCE") == "1" {
		colorsEnabled = true
	}

	// Check NO_COLOR (disable colors)
	if os.Getenv("NO_COLOR") != "" {
		colorsEnabled = false
	}

	// Final override with --no-color flag
	if globalFlags.NoColor {
		colorsEnabled = false
	}

	config.ColorsEnabled = colorsEnabled
	return config
}

// isTerminal checks if the given file is a terminal
func isTerminal(f *os.File) bool {
	fileInfo, err := f.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (terminal)
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// detectProductMode determines the product mode based on the invocation name
func detectProductMode() ProductContext {
	// Get the program name (last component of os.Args[0])
	progName := filepath.Base(os.Args[0])

	// Check if invoked as kubectl-pulse or contains "pulse"
	if strings.Contains(progName, "pulse") {
		return ProductContext{
			Mode: ModePulse,
			Name: "kubectl-pulse",
		}
	}

	// Default to broker mode
	return ProductContext{
		Mode: ModeBroker,
		Name: "kubectl-broker",
	}
}

// createRootCommand creates the root command with context-aware metadata
func createRootCommand(ctx ProductContext) *cobra.Command {
	switch ctx.Mode {
	case ModePulse:
		return &cobra.Command{
			Use:   ctx.Name,
			Short: "HiveMQ Pulse server diagnostics for Kubernetes",
			Long: `kubectl-pulse is a kubectl plugin that provides health diagnostics 
and monitoring tools for HiveMQ Pulse servers running on Kubernetes. It specializes
in checking liveness and readiness endpoints of Pulse server deployments.`,
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Help()
			},
		}
	default: // ModeBroker
		return &cobra.Command{
			Use:   ctx.Name,
			Short: "Comprehensive HiveMQ cluster management toolkit for Kubernetes",
			Long: `kubectl-broker is a kubectl plugin that provides comprehensive management 
tools for HiveMQ clusters running on Kubernetes. It includes health diagnostics, 
backup operations, and other cluster management features.`,
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Help()
			},
		}
	}
}

// addSubcommands adds the appropriate subcommands based on product mode
func addSubcommands(rootCmd *cobra.Command, ctx ProductContext) {
	switch ctx.Mode {
	case ModePulse:
		// Pulse mode: only add status command from pulse.go
		rootCmd.AddCommand(newPulseStatusCommand())
	default: // ModeBroker
		// Broker mode: add all broker commands
		rootCmd.AddCommand(newStatusCommand())
		rootCmd.AddCommand(newBackupCommand())
		rootCmd.AddCommand(newVolumesCommand())
		// Also add pulse as a subcommand for backward compatibility
		rootCmd.AddCommand(newPulseCommand())
	}
}
