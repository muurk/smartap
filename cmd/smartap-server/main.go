// Smartap-server is a WebSocket server for communicating with Smartap IoT devices.
//
// It provides TLS termination and WebSocket handling for devices that have been
// configured to connect to a custom server via certificate injection. The server
// accepts connections from jailbroken devices and logs protocol messages for
// analysis.
//
// Usage:
//
//	smartap-server server [flags]
//
// See 'smartap-server server --help' for available options.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/muurk/smartap/internal/server"
	"github.com/muurk/smartap/internal/version"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "smartap-server",
	Short: "Smartap WebSocket Server",
	Long: `A standalone WebSocket server for communicating with Smartap IoT devices.

This server provides complete control over HTTP/WebSocket headers and TLS configuration,
solving compatibility issues with TI CC3200-based devices that require exact HTTP 101
response formatting.

Note: For device configuration, use the separate 'smartap-cfg' utility.
For JTAG/GDB operations, use the separate 'smartap-jtag' utility.`,
	Version: version.Version,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(versionCmd)
	// Configure command has been moved to the separate 'smartap-cfg' utility
}

// Server command and flags
var (
	certPath    string
	keyPath     string
	host        string
	port        int
	logLevel    string
	analysisDir string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the WebSocket server",
	Long: `Start the Smartap WebSocket server to accept connections from devices.

The server will auto-generate a TLS certificate signed by the embedded Root CA
if no certificate is provided. Alternatively, you can provide your own certificate
and key files using the --cert and --key flags.

To capture WebSocket messages for protocol analysis, use the --analysis-dir flag
to specify a directory where message logs will be written.`,
	Example: `  # Start server with auto-generated certificate (signed by embedded Root CA)
  smartap-server server

  # Start with auto-generated certificate on custom port
  smartap-server server --port 8443 --log-level debug

  # Start server with custom certificates
  smartap-server server --cert /path/to/fullchain.pem --key /path/to/privkey.pem

  # Start with message analysis logging enabled
  smartap-server server --analysis-dir ./captures

  # Start with custom hostname and certificates
  smartap-server server --cert cert.pem --key key.pem --host smartap-tech.com`,
	RunE: runServer,
}

func init() {
	serverCmd.Flags().StringVar(&certPath, "cert", "", "Path to TLS certificate file (optional, will auto-generate if not provided)")
	serverCmd.Flags().StringVar(&keyPath, "key", "", "Path to TLS private key file (optional, will auto-generate if not provided)")
	serverCmd.Flags().StringVar(&host, "host", "", "Server hostname (empty = listen on all interfaces)")
	serverCmd.Flags().IntVar(&port, "port", 443, "Server port")
	serverCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	serverCmd.Flags().StringVar(&analysisDir, "analysis-dir", "", "Directory to write message analysis logs (disabled if not specified)")

	// cert and key are now optional - will auto-generate if not provided
}

func runServer(cmd *cobra.Command, args []string) error {
	// Determine certificate mode
	certProvided := certPath != "" && keyPath != ""
	generateCert := !certProvided

	// Validate: Either both cert and key are provided, or neither
	if (certPath != "" && keyPath == "") || (certPath == "" && keyPath != "") {
		return fmt.Errorf("both --cert and --key must be provided together, or neither (will auto-generate)")
	}

	// If files are provided, validate they exist
	if certProvided {
		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			return fmt.Errorf("certificate file not found: %s", certPath)
		}
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			return fmt.Errorf("private key file not found: %s", keyPath)
		}
	}

	// Validate analysis directory if specified
	if analysisDir != "" {
		info, err := os.Stat(analysisDir)
		if os.IsNotExist(err) {
			return fmt.Errorf("analysis directory does not exist: %s", analysisDir)
		}
		if err != nil {
			return fmt.Errorf("cannot access analysis directory: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("analysis path is not a directory: %s", analysisDir)
		}
	}

	// Create server configuration
	config := &server.Config{
		Host:         host,
		Port:         port,
		CertPath:     certPath,
		KeyPath:      keyPath,
		GenerateCert: generateCert,
		LogLevel:     logLevel,
		AnalysisDir:  analysisDir,
	}

	// Create and start server
	srv, err := server.New(config)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	return srv.Start()
}

// Version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("smartap-server %s (commit: %s)\n", version.Version, version.Commit)
	},
}
