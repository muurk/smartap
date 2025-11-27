// Smartap-jtag provides low-level device operations via JTAG/GDB.
//
// This utility uses arm-none-eabi-gdb to communicate with the CC3200 device
// through OpenOCD for operations that require direct hardware access:
//
//   - Certificate injection (device jailbreak)
//   - Firmware version detection
//   - Memory dumping and file operations
//   - Setup verification
//
// Prerequisites:
//
//   - arm-none-eabi-gdb installed and in PATH
//   - OpenOCD running and connected to device via JTAG
//   - Raspberry Pi with GPIO connections to device JTAG pins
//
// See 'smartap-jtag --help' for available commands.
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
	Use:   "smartap-jtag",
	Short: "Smartap JTAG/GDB Operations Utility",
	Long: `Low-level device operations using arm-none-eabi-gdb via OpenOCD.

This utility provides direct access to the CC3200 device via JTAG for:
  - Certificate injection
  - Firmware version detection
  - Log capture and debugging
  - Memory dumping and file operations
  - Setup verification

Prerequisites:
  - arm-none-eabi-gdb installed and in PATH
  - OpenOCD running and connected to device via JTAG
  - Device powered on and halted (OpenOCD handles this)

Use 'smartap-jtag verify-setup' to check prerequisites.`,
	Version: version.Version,
	Example: `  # Verify GDB and OpenOCD setup
  smartap-jtag verify-setup

  # Inject embedded root CA certificate
  smartap-jtag inject-certs

  # Inject custom certificate
  smartap-jtag inject-certs --cert-file /path/to/cert.der

  # Detect firmware version
  smartap-jtag detect-firmware`,
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
		fmt.Printf("smartap-jtag %s (commit: %s)\n", version.Version, version.Commit)
	},
}
