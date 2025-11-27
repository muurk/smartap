package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/muurk/smartap/internal/gdb"
	"github.com/muurk/smartap/internal/gdb/scripts"
	"github.com/muurk/smartap/internal/logging"
	"github.com/muurk/smartap/internal/ui"
	"github.com/muurk/smartap/internal/urls"
)

// Command flags
var (
	gdbPath         string
	openocdHost     string
	openocdPort     int
	gdbTimeout      string
	gdbVerbose      bool // Show GDB raw output
	certName        string
	certFile        string
	targetFile      string
	noDetect        bool
	firmwareVersion string
	logDuration     string
	logOutput       string
	memOutput       string // Only output path is configurable for dump-memory
	remoteFile      string
	readOutput      string
	maxFileSize     int
)

func init() {
	// Common flags for all commands (persistent on root)
	rootCmd.PersistentFlags().StringVar(&openocdHost, "openocd-host", "localhost", "OpenOCD hostname")
	rootCmd.PersistentFlags().IntVar(&openocdPort, "openocd-port", 3333, "OpenOCD port")
	rootCmd.PersistentFlags().StringVar(&gdbPath, "gdb-path", "arm-none-eabi-gdb", "Path to arm-none-eabi-gdb binary")
	rootCmd.PersistentFlags().StringVar(&gdbTimeout, "timeout", "5m", "GDB operation timeout (e.g., 30s, 5m, 1h)")
	rootCmd.PersistentFlags().BoolVarP(&gdbVerbose, "verbose", "v", false, "Show detailed GDB output")

	// Add subcommands
	rootCmd.AddCommand(injectCertsCmd)
	rootCmd.AddCommand(detectFirmwareCmd)
	rootCmd.AddCommand(verifySetupCmd)
	rootCmd.AddCommand(captureLogsCmd)
	rootCmd.AddCommand(dumpMemoryCmd)
	rootCmd.AddCommand(readFileCmd)
}

// createGDBExecutor creates a GDB executor with the configured settings
func createGDBExecutor() (*gdb.Executor, error) {
	timeout, err := time.ParseDuration(gdbTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout value: %w", err)
	}

	config := gdb.Config{
		GDBPath:     gdbPath,
		OpenOCDHost: openocdHost,
		OpenOCDPort: openocdPort,
		Timeout:     timeout,
		WorkDir:     "",
	}

	// Initialize logging from environment variable (silent by default)
	// Set SMARTAP_LOG_LEVEL=debug to see detailed logs
	if err := logging.InitializeFromEnv(); err != nil {
		// Ignore error, GetLogger will create fallback logger
		_ = err
	}

	logger := logging.GetLogger()
	executor := gdb.NewExecutor(config, logger)
	return executor, nil
}

// injectCertsCmd implements the 'inject-certs' command
var injectCertsCmd = &cobra.Command{
	Use:   "inject-certs",
	Short: "Inject CA certificate into device flash",
	Long: `Inject a CA certificate into the CC3200 device flash memory via JTAG.

This command will:
  1. Validate prerequisites (GDB, OpenOCD)
  2. Load certificate (embedded or custom)
  3. Detect firmware version (unless --no-detect is set)
  4. Inject certificate into device flash at /cert/129.der
  5. Verify injection was successful

The device will automatically use the injected certificate for TLS connections.

By default, the embedded Smartap Revival Root CA is injected, which allows
the device to connect to the smartap-server with no additional setup.
For security-conscious users, a custom certificate can be provided.`,
	Example: `  # Inject embedded root CA (default)
  smartap-jtag inject-certs

  # Inject custom certificate
  smartap-jtag inject-certs --cert-file /path/to/custom-ca.der

  # Inject to different target file
  smartap-jtag inject-certs --target-file /cert/130.der

  # Skip firmware detection (faster, but requires correct version)
  smartap-jtag inject-certs --no-detect --firmware-version 0x355`,
	RunE: runInjectCerts,
}

func init() {
	injectCertsCmd.Flags().StringVar(&certName, "cert", "root_ca", "Embedded certificate name to inject")
	injectCertsCmd.Flags().StringVar(&certFile, "cert-file", "", "Custom certificate file to inject (overrides --cert)")
	injectCertsCmd.Flags().StringVar(&targetFile, "target-file", "/cert/129.der", "Target file path on device")
	injectCertsCmd.Flags().BoolVar(&noDetect, "no-detect", false, "Skip firmware detection (use --firmware-version)")
	injectCertsCmd.Flags().StringVar(&firmwareVersion, "firmware-version", "", "Firmware version (required if --no-detect is set)")
}

func runInjectCerts(cmd *cobra.Command, args []string) error {
	// Suppress usage on execution errors (we're past argument parsing)
	cmd.SilenceUsage = true

	// Validate flags
	if noDetect && firmwareVersion == "" {
		ui.PrintFailure("Invalid arguments", fmt.Errorf("--firmware-version is required when --no-detect is set"), []string{
			"Provide firmware version: --firmware-version 0x355",
			"Or remove --no-detect to auto-detect firmware",
		})
		return fmt.Errorf("--firmware-version is required when --no-detect is set")
	}

	// Determine certificate source for header display
	certSource := "Embedded: " + certName
	if certFile != "" {
		certSource = "Custom: " + certFile
	}

	// Print styled header
	ui.PrintCommandHeader(
		"Certificate Injection",
		"smartap-jtag inject-certs",
		map[string]string{
			"Device":      fmt.Sprintf("%s:%d", openocdHost, openocdPort),
			"Certificate": certSource,
			"Target":      targetFile,
		},
	)

	// Create executor
	executor, err := createGDBExecutor()
	if err != nil {
		ui.PrintFailure("Certificate injection failed", err, []string{
			"Check GDB and OpenOCD setup: smartap-jtag verify-setup",
		})
		return fmt.Errorf("failed to create GDB executor: %w", err)
	}

	// Create certificate manager
	certManager, err := gdb.NewCertManager()
	if err != nil {
		ui.PrintFailure("Certificate injection failed", err, []string{
			"Certificate manager initialization failed",
			"Ensure embedded certificates are available",
		})
		return fmt.Errorf("failed to create certificate manager: %w", err)
	}

	// Track the detected/provided firmware version for output
	detectedFirmwareVersion := firmwareVersion
	ctx := context.Background()

	// If not skipping detection, detect firmware first and show it to the user
	if !noDetect && firmwareVersion == "" {
		// Load firmware catalog
		firmwareDB, err := gdb.LoadFirmwares()
		if err != nil {
			ui.PrintFailure("Certificate injection failed", err, []string{
				"Firmware catalog may be corrupted",
			})
			return fmt.Errorf("failed to load firmware catalog: %w", err)
		}

		// Detect firmware version
		detectScript := scripts.NewDetectFirmwareScript(openocdHost, openocdPort, firmwareDB.List())
		detectResult, err := executor.Execute(ctx, detectScript)
		if err != nil {
			ui.PrintFailure("Firmware detection failed", err, []string{
				"Certificate injection requires known firmware",
				"Try: smartap-jtag detect-firmware",
			})
			return fmt.Errorf("firmware detection failed: %w", err)
		}

		// Validate confidence
		confidence := detectResult.GetDataInt("confidence")
		if confidence < 100 {
			matches := detectResult.GetDataInt("matches")
			total := detectResult.GetDataInt("total")
			version := detectResult.GetDataString("version")

			ui.PrintFailure("Firmware unknown", fmt.Errorf("confidence too low: %d%% (need 100%%)", confidence), []string{
				fmt.Sprintf("Best match: %s (%d/%d signatures)", version, matches, total),
				"Certificate injection requires 100% firmware confidence",
				"Dump memory and submit for analysis: smartap-jtag dump-memory",
			})

			return &gdb.FirmwareConfidenceError{
				Version:    version,
				Confidence: confidence,
				Matches:    matches,
				Total:      total,
			}
		}

		detectedFirmwareVersion = detectResult.GetDataString("version")

		// Show firmware detection success
		fmt.Println()
		line := "  " + ui.StepCompleteStyle.Render("Firmware verified: "+detectedFirmwareVersion)
		line += "  " + ui.StepCompleteStyle.Render(ui.StepMarkerComplete)
		line += "  " + ui.StepNoteStyle.Render("(100% confidence)")
		fmt.Println(line)
		fmt.Println()
	}

	// Prompt user for confirmation before flash write
	if !ui.FlashWriteConfirmation() {
		return nil // User cancelled
	}

	// Create styled progress callback using ui package
	styledProgressCallback := func(step scripts.Step) {
		// Convert scripts.Step status to ui.StepStatus
		var status ui.StepStatus
		switch step.Status {
		case "success":
			status = ui.StepComplete
		case "failed":
			status = ui.StepFailed
		case "skipped":
			status = ui.StepSkipped
		case "in_progress":
			status = ui.StepRunning
		default:
			status = ui.StepPending
		}

		// Create and render a single step line
		s := ui.Step{
			Number:  0, // Will be parsed from name
			Name:    step.Name,
			Status:  status,
			Message: step.Message,
		}

		// Print the styled step
		marker := ui.StepMarkerPending
		style := ui.StepPendingStyle
		switch status {
		case ui.StepComplete:
			marker = ui.StepMarkerComplete
			style = ui.StepCompleteStyle
		case ui.StepFailed:
			marker = ui.FailureMarker
			style = ui.ErrorTitleStyle
		case ui.StepRunning:
			marker = ui.StepMarkerRunning
			style = ui.StepRunningStyle
		case ui.StepSkipped:
			marker = "⊘"
			style = ui.StepPendingStyle
		}

		line := "  " + style.Render(s.Name)
		line += "  " + style.Render(marker)
		if s.Message != "" {
			line += "  " + ui.StepNoteStyle.Render("("+s.Message+")")
		}
		fmt.Println(line)
	}

	// Prepare injection options - use detected version and skip internal detection
	opts := gdb.InjectOptions{
		Executor:        executor,
		CertManager:     certManager,
		CertName:        certName,
		CertPath:        certFile,
		TargetFile:      targetFile,
		FirmwareVersion: detectedFirmwareVersion,
		SkipDetection:   true, // We already detected above (or user provided)
		OnProgress:      styledProgressCallback,
	}

	// Execute injection
	ui.PrintPleaseWait("Injecting certificate", "this may take up to 60 seconds")
	result, err := gdb.InjectCertificate(ctx, opts)
	if err != nil {
		ui.PrintFailure("Certificate injection failed", err, []string{
			"Verify OpenOCD is still connected",
			"Check device hasn't reset unexpectedly",
			"Try: smartap-jtag verify-setup",
			"Run with --verbose for full GDB output",
		})
		return fmt.Errorf("certificate injection failed: %w", err)
	}

	// Print result
	if result.Success {
		ui.PrintSuccess("Certificate injection complete", map[string]string{
			"Target File":   targetFile,
			"Bytes Written": fmt.Sprintf("%d", result.BytesWritten),
			"Firmware":      fmt.Sprintf("%s (verified)", detectedFirmwareVersion),
			"Duration":      result.Duration.String(),
			"Device":        "Resumed and detached",
		})

		// Show GDB output in verbose mode
		if gdbVerbose && result.RawOutput != "" {
			ui.PrintGDBOutput(result.RawOutput)
		}
	} else {
		errMsg := "injection failed"
		if result.Error != nil {
			errMsg = result.Error.Error()
		}
		ui.PrintFailure("Certificate injection failed", fmt.Errorf("%s", errMsg), []string{
			"Check GDB output for specific error",
			"Verify firmware version is correct",
			"Try: smartap-jtag detect-firmware",
		})
		return fmt.Errorf("injection failed")
	}

	return nil
}

// detectFirmwareCmd implements the 'detect-firmware' command
var detectFirmwareCmd = &cobra.Command{
	Use:   "detect-firmware",
	Short: "Detect device firmware version",
	Long: `Detect the firmware version running on the CC3200 device.

This command connects to the device via GDB and reads the firmware version
from memory. The version is then looked up in the firmware catalog to
display detailed information about supported functions and memory addresses.

If the firmware version is not recognized, instructions will be provided
for dumping memory and submitting it for analysis to add support.`,
	Example: `  # Detect firmware version
  smartap-jtag detect-firmware

  # With custom OpenOCD settings
  smartap-jtag detect-firmware --openocd-host 192.168.1.100 --openocd-port 4444`,
	RunE: runDetectFirmware,
}

func runDetectFirmware(cmd *cobra.Command, args []string) error {
	// Suppress usage on execution errors (we're past argument parsing)
	cmd.SilenceUsage = true

	// Print styled header
	ui.PrintCommandHeader(
		"Firmware Detection",
		"smartap-jtag detect-firmware",
		map[string]string{
			"Device": fmt.Sprintf("%s:%d", openocdHost, openocdPort),
			"Method": "Signature matching",
		},
	)

	// Create executor
	executor, err := createGDBExecutor()
	if err != nil {
		ui.PrintFailure("Firmware detection failed", err, []string{
			"Check GDB and OpenOCD setup: smartap-jtag verify-setup",
		})
		return fmt.Errorf("failed to create GDB executor: %w", err)
	}

	// Load firmware catalog
	db, err := gdb.LoadFirmwares()
	if err != nil {
		ui.PrintFailure("Firmware detection failed", err, []string{
			"Firmware catalog may be corrupted",
			"Try reinstalling the application",
		})
		return fmt.Errorf("failed to load firmware catalog: %w", err)
	}

	// Create detection script
	detectScript := scripts.NewDetectFirmwareScript(openocdHost, openocdPort, db.List())

	// Execute detection
	ui.PrintPleaseWait("Detecting firmware", "this may take up to 30 seconds")
	ctx := context.Background()
	result, err := executor.Execute(ctx, detectScript)
	if err != nil {
		ui.PrintFailure("Firmware detection failed", err, []string{
			"Verify OpenOCD is still connected",
			"Check device hasn't reset unexpectedly",
			"Try: smartap-jtag verify-setup",
		})
		return fmt.Errorf("firmware detection failed: %w", err)
	}

	// Get detection results
	version := result.GetDataString("version")
	confidence := result.GetDataInt("confidence")
	matches := result.GetDataInt("matches")
	total := result.GetDataInt("total")

	// Handle 100% confidence - SUCCESS
	if confidence == 100 {
		// Look up firmware details
		firmware, ok := db.Get(version)
		if !ok {
			// This shouldn't happen if confidence is 100%, but handle it
			ui.PrintWarning("Firmware not in catalog", map[string]string{
				"Version":    version,
				"Confidence": fmt.Sprintf("%d%%", confidence),
			})
			return gdb.HandleUnknownFirmware(version)
		}

		// Build success details
		status := "Verified"
		if !firmware.Verified {
			status = "Unverified (community contribution)"
		}

		ui.PrintSuccess("Firmware detected", map[string]string{
			"Version":    version,
			"Name":       firmware.Name,
			"Confidence": fmt.Sprintf("%d%% (all %d signatures matched)", confidence, total),
			"Status":     status,
		})

		// Show function addresses in verbose mode or always (they're useful)
		if gdbVerbose {
			ui.PrintGDBOutput(fmt.Sprintf(
				"Function addresses:\n"+
					"  sl_FsOpen:    %s\n"+
					"  sl_FsRead:    %s\n"+
					"  sl_FsWrite:   %s\n"+
					"  sl_FsClose:   %s\n"+
					"  sl_FsDel:     %s\n"+
					"  sl_FsGetInfo: %s\n"+
					"  uart_log:     %s",
				formatHex(firmware.Functions.SlFsOpen),
				formatHex(firmware.Functions.SlFsRead),
				formatHex(firmware.Functions.SlFsWrite),
				formatHex(firmware.Functions.SlFsClose),
				formatHex(firmware.Functions.SlFsDel),
				formatHex(firmware.Functions.SlFsGetInfo),
				formatHex(firmware.Functions.UartLog),
			))
		}

		return nil
	}

	// Handle < 100% confidence - FAILURE with helpful guidance
	errorMsg := "Firmware detection incomplete"
	if version != "UNKNOWN" && version != "0" && confidence > 0 {
		errorMsg = fmt.Sprintf("Best match: %s (%d%% confidence, %d/%d signatures)", version, confidence, matches, total)
	} else {
		errorMsg = "No known firmware signatures matched"
	}

	ui.PrintFailure("Firmware unknown", fmt.Errorf("%s", errorMsg), []string{
		"Dump device memory: smartap-jtag dump-memory --output firmware.bin",
		fmt.Sprintf("Analysis guide: %s", urls.FindingFunctionsInMemory),
		fmt.Sprintf("Submit findings: %s", urls.ContributingFirmware),
	})

	// Additional warning about operations being blocked
	ui.PrintWarning("JTAG operations blocked", map[string]string{
		"Reason": "100% confidence required to ensure correct function addresses",
		"Risk":   "Without reliable addresses, operations may corrupt device memory",
	})

	return fmt.Errorf("firmware confidence too low: %d%% (need 100%%)", confidence)
}

// verifySetupCmd implements the 'verify-setup' command
var verifySetupCmd = &cobra.Command{
	Use:   "verify-setup",
	Short: "Verify GDB and OpenOCD setup",
	Long: `Verify that all prerequisites for JTAG operations are met.

This command checks:
  1. arm-none-eabi-gdb binary is installed and executable
  2. GDB version is compatible
  3. OpenOCD connection can be established
  4. Device is accessible via JTAG

Run this command first to troubleshoot any connection issues.`,
	Example: `  # Verify default setup
  smartap-jtag verify-setup

  # Verify with custom settings
  smartap-jtag verify-setup --openocd-host 192.168.1.100 --gdb-path /opt/gcc-arm/bin/arm-none-eabi-gdb`,
	RunE: runVerifySetup,
}

func runVerifySetup(cmd *cobra.Command, args []string) error {
	// Suppress usage on execution errors (we're past argument parsing)
	cmd.SilenceUsage = true

	// Print styled header
	ui.PrintCommandHeader(
		"Setup Verification",
		"smartap-jtag verify-setup",
		map[string]string{
			"GDB Path":     gdbPath,
			"OpenOCD Host": fmt.Sprintf("%s:%d", openocdHost, openocdPort),
		},
	)

	ctx := context.Background()
	var gdbErr, openocdErr error

	// Check GDB binary
	gdbErr = gdb.ValidateGDBPath(ctx, gdbPath)

	// Check OpenOCD connection
	openocdErr = gdb.ValidateOpenOCDConnection(ctx, openocdHost, openocdPort)

	// Determine overall result
	hasErrors := gdbErr != nil || openocdErr != nil

	if hasErrors {
		// Build error details
		var troubleshooting []string
		errorMsg := "Setup verification failed"

		if gdbErr != nil {
			troubleshooting = append(troubleshooting,
				"arm-none-eabi-gdb not found or not executable",
				"Install ARM toolchain: brew install arm-none-eabi-gcc (macOS)",
				"Or: apt install gcc-arm-none-eabi (Linux)",
			)
			errorMsg = fmt.Sprintf("GDB: %v", gdbErr)
		}

		if openocdErr != nil {
			troubleshooting = append(troubleshooting,
				"Ensure OpenOCD is running: openocd -f <your-config.cfg>",
				"Check OpenOCD is listening on the correct host/port",
				"Verify firewall settings allow connection",
				"Ensure JTAG debugger is connected to device",
			)
			if gdbErr == nil {
				errorMsg = fmt.Sprintf("OpenOCD: %v", openocdErr)
			}
		}

		ui.PrintFailure("Setup verification failed", fmt.Errorf("%s", errorMsg), troubleshooting)
		return fmt.Errorf("setup verification failed")
	}

	// Success - show details
	ui.PrintSuccess("Setup verification complete", map[string]string{
		"GDB":     gdbPath + " (found)",
		"OpenOCD": fmt.Sprintf("%s:%d (connected)", openocdHost, openocdPort),
		"Status":  "Ready for JTAG operations",
	})

	// Note about firmware detection
	ui.PrintWarning("Firmware not yet detected", map[string]string{
		"Next step": "Run 'smartap-jtag detect-firmware'",
	})

	return nil
}

// captureLogsCmd implements the 'capture-logs' command
// NOTE: This command is not yet implemented in this version
var captureLogsCmd = &cobra.Command{
	Use:   "capture-logs",
	Short: "Capture device UART logs in real-time (NOT IMPLEMENTED)",
	Long: `Capture device UART logs in real-time via GDB breakpoints.

⚠️  NOT IMPLEMENTED - This command is not available in the current version.

This command sets a breakpoint on the device's UART logging function and
captures all log messages as they are generated. This is extremely useful
for debugging device behavior, TLS handshake issues, and WiFi connectivity.

The logs are displayed to stdout by default, or can be saved to a file
with the --output flag.

Press Ctrl+C to stop log capture and detach from the device.`,
	Example: `  # Capture logs to stdout (Ctrl+C to stop)
  smartap-jtag capture-logs

  # Capture logs for 30 seconds
  smartap-jtag capture-logs --duration 30s

  # Save logs to file
  smartap-jtag capture-logs --output device-logs.txt

  # Capture for 1 minute and save to file
  smartap-jtag capture-logs --duration 1m --output startup-logs.txt`,
	RunE: runCaptureLogsNotImplemented,
}

func init() {
	captureLogsCmd.Flags().StringVar(&logDuration, "duration", "", "How long to capture (e.g., 30s, 5m). Default: infinite, Ctrl+C to stop")
	captureLogsCmd.Flags().StringVar(&logOutput, "output", "", "Write logs to file (default: stdout)")
}

// runCaptureLogsNotImplemented shows a not-implemented message
func runCaptureLogsNotImplemented(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	// Print styled header
	ui.PrintCommandHeader(
		"Log Capture",
		"smartap-jtag capture-logs",
		map[string]string{
			"Device": fmt.Sprintf("%s:%d", openocdHost, openocdPort),
			"Status": "Not Implemented",
		},
	)

	// Print styled warning
	ui.PrintWarning("Command not implemented", map[string]string{
		"Reason":      "Real-time log capture via GDB requires complex streaming support",
		"Status":      "Under development",
		"Alternative": "Use OpenOCD's native logging or connect via UART",
	})

	return nil
}

// dumpMemoryCmd implements the 'dump-memory' command
var dumpMemoryCmd = &cobra.Command{
	Use:   "dump-memory",
	Short: "Dump device memory to file",
	Long: `Dump the CC3200 device RAM to a binary file.

This command dumps the entire RAM region (0x20000000 - 0x20040000, 256KB)
which contains the running firmware code and data.

This is useful for:
  - Capturing firmware for analysis
  - Debugging memory corruption issues
  - Submitting memory dumps for unknown firmware support
  - Reverse engineering device behavior

The memory region is fixed to the CC3200's RAM layout and cannot be changed.`,
	Example: `  # Dump device RAM to file
  smartap-jtag dump-memory --output firmware.bin

  # With custom OpenOCD host
  smartap-jtag dump-memory --output firmware.bin --openocd-host 192.168.1.100`,
	RunE: runDumpMemory,
}

func init() {
	dumpMemoryCmd.Flags().StringVar(&memOutput, "output", "", "Output file (required)")
	dumpMemoryCmd.MarkFlagRequired("output")
}

func runDumpMemory(cmd *cobra.Command, args []string) error {
	// Suppress usage on execution errors (we're past argument parsing)
	cmd.SilenceUsage = true

	// CC3200 RAM region: fixed values based on device architecture
	const (
		startAddr = 0x20000000 // RAM start address
		memSize   = 262144     // 256KB (0x20000000 to 0x20040000)
	)

	// Print styled header
	ui.PrintCommandHeader(
		"Memory Dump",
		"smartap-jtag dump-memory",
		map[string]string{
			"Device":  fmt.Sprintf("%s:%d", openocdHost, openocdPort),
			"Address": fmt.Sprintf("0x%08x - 0x%08x", startAddr, startAddr+memSize),
			"Size":    fmt.Sprintf("%d KB (256 KB)", memSize/1024),
			"Output":  memOutput,
		},
	)

	// Safety check: Warn if output file already exists
	if _, err := os.Stat(memOutput); err == nil {
		ui.PrintWarning("Output file exists", map[string]string{
			"File":   memOutput,
			"Action": "Will be overwritten if you proceed",
		})
		// For now, we'll proceed without interactive confirmation
		// In future, could add a --force flag
	}

	// Create executor
	executor, err := createGDBExecutor()
	if err != nil {
		ui.PrintFailure("Memory dump failed", err, []string{
			"Check GDB and OpenOCD setup: smartap-jtag verify-setup",
		})
		return fmt.Errorf("failed to create GDB executor: %w", err)
	}

	// Create dump memory script
	dumpScript := scripts.NewDumpMemoryScript(openocdHost, openocdPort, startAddr, memSize, memOutput)

	// Execute dump
	ui.PrintPleaseWait("Dumping device memory", "this may take up to 2 minutes")
	ctx := context.Background()
	result, err := executor.Execute(ctx, dumpScript)
	if err != nil {
		ui.PrintFailure("Memory dump failed", err, []string{
			"Verify OpenOCD is still connected",
			"Check device hasn't reset unexpectedly",
			"Try: smartap-jtag verify-setup",
		})
		return fmt.Errorf("memory dump failed: %w", err)
	}

	if !result.Success {
		ui.PrintFailure("Memory dump failed", result.Error, []string{
			"Check GDB output for specific error",
			"Verify device is halted and accessible",
		})
		return fmt.Errorf("memory dump failed: %v", result.Error)
	}

	// Safety check: Verify the file was created
	fileInfo, err := os.Stat(memOutput)
	if err != nil {
		if os.IsNotExist(err) {
			ui.PrintFailure("Memory dump verification failed",
				fmt.Errorf("output file was not created: %s", memOutput),
				[]string{"GDB reported success but file doesn't exist", "Check disk space and permissions"})
			return fmt.Errorf("memory dump reported success but output file was not created: %s", memOutput)
		}
		ui.PrintFailure("Memory dump verification failed", err, []string{
			"Could not verify output file",
		})
		return fmt.Errorf("failed to verify output file: %w", err)
	}

	// Safety check: Verify the file size is correct
	actualSize := fileInfo.Size()
	expectedSize := int64(memSize)
	if actualSize != expectedSize {
		ui.PrintWarning("File size mismatch", map[string]string{
			"Expected": fmt.Sprintf("%d bytes (%d KB)", expectedSize, expectedSize/1024),
			"Actual":   fmt.Sprintf("%d bytes (%d KB)", actualSize, actualSize/1024),
		})
		return fmt.Errorf("memory dump incomplete: expected %d bytes, got %d bytes", expectedSize, actualSize)
	}

	// Success
	ui.PrintSuccess("Memory dump complete", map[string]string{
		"Output File":   memOutput,
		"File Size":     fmt.Sprintf("%d KB (verified)", actualSize/1024),
		"Start Address": fmt.Sprintf("0x%08x", startAddr),
		"End Address":   fmt.Sprintf("0x%08x", startAddr+memSize),
		"Duration":      result.Duration.String(),
	})

	// Show next steps
	ui.PrintWarning("Next steps for firmware analysis", map[string]string{
		"Step 1": "Create issue: " + urls.ContributingFirmware,
		"Step 2": "Attach the memory dump file",
		"Step 3": "Include device model information",
	})

	// Show GDB output in verbose mode
	if gdbVerbose && result.RawOutput != "" {
		ui.PrintGDBOutput(result.RawOutput)
	}

	return nil
}

// readFileCmd implements the 'read-file' command
var readFileCmd = &cobra.Command{
	Use:   "read-file",
	Short: "Read a file from device filesystem",
	Long: `Read a file from the CC3200 device filesystem via JTAG.

This command uses the TI SimpleLink sl_FsRead function to read files
from the device's flash filesystem. This is useful for:
  - Extracting firmware: /sys/mcuimg0.bin
  - Reading certificates: /cert/129.der
  - Reading configuration files
  - Debugging file operations

The file is read in chunks and saved to the specified output file.`,
	Example: `  # Read firmware image
  smartap-jtag read-file --remote-file /sys/mcuimg0.bin --output mcuimg0.bin

  # Read certificate
  smartap-jtag read-file --remote-file /cert/129.der --output device-cert.der

  # Read with size limit (safety)
  smartap-jtag read-file --remote-file /sys/config.txt --output config.txt --max-size 8192`,
	RunE: runReadFile,
}

func init() {
	readFileCmd.Flags().StringVar(&remoteFile, "remote-file", "", "File path on device (required)")
	readFileCmd.Flags().StringVar(&readOutput, "output", "", "Output file path (required)")
	readFileCmd.Flags().IntVar(&maxFileSize, "max-size", 262144, "Max file size to read (default: 256KB, safety limit)")
	readFileCmd.MarkFlagRequired("remote-file")
	readFileCmd.MarkFlagRequired("output")
}

func runReadFile(cmd *cobra.Command, args []string) error {
	// Suppress usage on execution errors (we're past argument parsing)
	cmd.SilenceUsage = true

	// Validate max size
	if maxFileSize <= 0 {
		ui.PrintFailure("Invalid arguments", fmt.Errorf("max-size must be greater than 0"), []string{
			"Provide a positive value for --max-size",
		})
		return fmt.Errorf("max-size must be greater than 0")
	}

	if maxFileSize > 1024*1024 { // 1MB limit for safety
		ui.PrintFailure("Invalid arguments", fmt.Errorf("max-size too large: %d bytes (max 1MB)", maxFileSize), []string{
			"Maximum allowed size is 1MB (1048576 bytes)",
		})
		return fmt.Errorf("max-size too large: %d bytes (max 1MB)", maxFileSize)
	}

	// Print styled header
	ui.PrintCommandHeader(
		"File Read",
		"smartap-jtag read-file",
		map[string]string{
			"Device":      fmt.Sprintf("%s:%d", openocdHost, openocdPort),
			"Remote File": remoteFile,
			"Output":      readOutput,
			"Max Size":    fmt.Sprintf("%d KB", maxFileSize/1024),
		},
	)

	// Create executor
	executor, err := createGDBExecutor()
	if err != nil {
		ui.PrintFailure("File read failed", err, []string{
			"Check GDB and OpenOCD setup: smartap-jtag verify-setup",
		})
		return fmt.Errorf("failed to create GDB executor: %w", err)
	}

	// Load firmware catalog first
	firmwareDB, err := gdb.LoadFirmwares()
	if err != nil {
		ui.PrintFailure("File read failed", err, []string{
			"Firmware catalog may be corrupted",
		})
		return fmt.Errorf("failed to load firmware catalog: %w", err)
	}

	// Detect firmware version (required for sl_FsRead function address)
	detectScript := scripts.NewDetectFirmwareScript(openocdHost, openocdPort, firmwareDB.List())

	ctx := context.Background()
	detectResult, err := executor.Execute(ctx, detectScript)
	if err != nil {
		ui.PrintFailure("Firmware detection failed", err, []string{
			"File read requires known firmware to locate sl_FsRead function",
			"Try: smartap-jtag detect-firmware",
		})
		return fmt.Errorf("firmware detection failed: %w", err)
	}

	// Validate confidence
	confidence := detectResult.GetDataInt("confidence")
	if confidence < 100 {
		matches := detectResult.GetDataInt("matches")
		total := detectResult.GetDataInt("total")
		version := detectResult.GetDataString("version")

		ui.PrintFailure("Firmware unknown", fmt.Errorf("confidence too low: %d%% (need 100%%)", confidence), []string{
			fmt.Sprintf("Best match: %s (%d/%d signatures)", version, matches, total),
			"File read requires 100% firmware confidence",
			"Dump memory and submit for analysis: smartap-jtag dump-memory",
		})

		return &gdb.FirmwareConfidenceError{
			Version:    version,
			Confidence: confidence,
			Matches:    matches,
			Total:      total,
		}
	}

	version := detectResult.GetDataString("version")
	firmware, ok := firmwareDB.Get(version)
	if !ok {
		ui.PrintFailure("Firmware not in catalog", fmt.Errorf("version %s not found", version), []string{
			"This shouldn't happen with 100% confidence",
			"Please report this issue",
		})
		return gdb.HandleUnknownFirmware(version)
	}

	// Create read file script
	readScript := scripts.NewReadFileScript(openocdHost, openocdPort, firmware, remoteFile, readOutput, maxFileSize)

	// Execute read
	ui.PrintPleaseWait("Reading file from device", "this may take up to 60 seconds")
	result, err := executor.Execute(ctx, readScript)
	if err != nil {
		ui.PrintFailure("File read failed", err, []string{
			"Verify the remote file path exists on device",
			"Check file permissions on device",
			"Try: smartap-jtag verify-setup",
		})
		return fmt.Errorf("file read failed: %w", err)
	}

	if !result.Success {
		ui.PrintFailure("File read failed", result.Error, []string{
			"Check GDB output for specific error",
			"File may not exist or be too large",
		})
		return fmt.Errorf("file read failed: %v", result.Error)
	}

	// Success
	ui.PrintSuccess("File read complete", map[string]string{
		"Remote File": remoteFile,
		"Output File": readOutput,
		"Bytes Read":  fmt.Sprintf("%d (%d KB)", result.BytesRead, result.BytesRead/1024),
		"Firmware":    fmt.Sprintf("%s (100%% confidence)", version),
		"Duration":    result.Duration.String(),
	})

	// Show GDB output in verbose mode
	if gdbVerbose && result.RawOutput != "" {
		ui.PrintGDBOutput(result.RawOutput)
	}

	return nil
}

// Helper function to format hex addresses
func formatHex(addr int64) string {
	if addr == 0 {
		return "not set"
	}
	return fmt.Sprintf("0x%08x", addr)
}
