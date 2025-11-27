// Smartap-cfg is a configuration utility for Smartap IoT devices.
//
// It provides device discovery, an interactive configuration wizard, and
// direct configuration commands for Smartap smart shower controllers.
// This tool communicates with devices over HTTP and does not require
// hardware modification.
//
// Usage:
//
//	smartap-cfg [command] [flags]
//
// Running without arguments launches the interactive wizard.
// See 'smartap-cfg --help' for available commands.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/muurk/smartap/internal/version"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "smartap-cfg",
	Short: "Smartap Device Configuration Utility",
	Long: `A standalone utility for configuring Smartap IoT devices.

Provides device discovery, interactive configuration wizard, and
direct configuration commands for Smartap smart shower controllers.

If no command is specified, the interactive wizard will launch automatically.`,
	Version: version.Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default behavior: run wizard when no subcommand provided
		return runWizard(cmd, args)
	},
}

func init() {
	// Disable automatic completion command generation
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("smartap-cfg %s (commit: %s)\n", version.Version, version.Commit)
	},
}
