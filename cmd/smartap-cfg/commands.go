package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/muurk/smartap/internal/deviceconfig"
	"github.com/muurk/smartap/internal/discovery"
	"github.com/muurk/smartap/internal/wizard/tui"
)

// Configuration command flags
var (
	deviceIP     string
	devicePort   int
	scanTimeout  int
	outputFormat string
	noVerify     bool
	retries      int
)

func init() {
	// Common flags for device commands (persistent on root)
	rootCmd.PersistentFlags().StringVar(&deviceIP, "device", "", "Device IP address (skips discovery)")
	rootCmd.PersistentFlags().IntVar(&devicePort, "port", 80, "Device HTTP port")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "detailed", "Output format (detailed, compact, json)")

	// Add subcommands directly to root
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(showCmd)
	rootCmd.AddCommand(wizardCmd)
	rootCmd.AddCommand(setDiverterCmd)
	rootCmd.AddCommand(setServerCmd)
}

// scanCmd discovers devices on the network
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for Smartap devices on the network",
	Long: `Scan for Smartap devices using mDNS/DNS-SD discovery.

This command listens for mDNS broadcasts from Smartap devices and displays
all discovered devices with their IP addresses, serial numbers, and metadata.`,
	Example: `  # Scan for 10 seconds (default)
  smartap-cfg scan

  # Quick 3-second scan
  smartap-cfg scan --timeout 3

  # Longer scan for networks with many devices
  smartap-cfg scan --timeout 30`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().IntVar(&scanTimeout, "timeout", 10, "Scan timeout in seconds")
}

func runScan(cmd *cobra.Command, args []string) error {
	fmt.Printf("Scanning for Smartap devices (timeout: %ds)...\n\n", scanTimeout)

	devices, err := discovery.ScanForDevices(time.Duration(scanTimeout) * time.Second)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if len(devices) == 0 {
		fmt.Println("No devices found.")
		fmt.Println("\nTroubleshooting:")
		fmt.Println("  - Ensure device is powered on and in pairing mode")
		fmt.Println("  - Check that device WiFi hotspot is active")
		fmt.Println("  - Verify your computer is connected to the device's WiFi")
		fmt.Println("  - Try increasing --timeout for slower networks")
		fmt.Println("  - Use --device flag to specify IP manually if discovery fails")
		return nil
	}

	fmt.Printf("Found %d device(s):\n\n", len(devices))

	for i, device := range devices {
		fmt.Printf("%d. %s\n", i+1, device.Hostname)
		fmt.Printf("   Serial:  %s\n", device.Serial)
		fmt.Printf("   IP:      %s:%d\n", device.IP, device.Port)
		if len(device.Metadata) > 0 {
			fmt.Printf("   Metadata: %v\n", device.Metadata)
		}
		fmt.Println()
	}

	fmt.Println("Use 'smartap-cfg show --device <ip>' to view device configuration")
	fmt.Println("Use 'smartap-cfg wizard' for interactive configuration")

	return nil
}

// showCmd displays current device configuration
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show device configuration",
	Long: `Display the current configuration of a Smartap device.

This command connects to the device and retrieves its current configuration,
including outlet assignments, diverter button behavior, server settings, and
device information.`,
	Example: `  # Show config with auto-discovery
  smartap-cfg show

  # Show config for specific device
  smartap-cfg show --device 192.168.4.16

  # Compact output format
  smartap-cfg show --device 192.168.4.16 --format compact

  # JSON output for scripting
  smartap-cfg show --device 192.168.4.16 --format json`,
	RunE: runShow,
}

func runShow(cmd *cobra.Command, args []string) error {
	// Get device IP (via discovery or manual)
	ip, err := getDeviceIP()
	if err != nil {
		return err
	}

	// Create client and fetch configuration
	client := deviceconfig.NewClient(ip, devicePort)

	fmt.Printf("Fetching configuration from %s:%d...\n\n", ip, devicePort)

	config, err := client.GetConfiguration()
	if err != nil {
		return fmt.Errorf("failed to get configuration: %w", err)
	}

	// Display configuration based on format
	switch outputFormat {
	case "compact":
		fmt.Println(config.FormatCompact())
	case "json":
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
	case "detailed":
		fallthrough
	default:
		fmt.Println(config.FormatDetailed())
	}

	return nil
}

// wizardCmd launches the interactive TUI wizard
var wizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Launch interactive configuration wizard",
	Long: `Launch an interactive TUI wizard for device configuration.

The wizard provides a user-friendly interface for:
- Discovering devices on the network
- Viewing current configuration
- Editing outlet and diverter settings
- Previewing and applying changes

This is the recommended way to configure devices for most users.`,
	Example: `  # Launch wizard with auto-discovery
  smartap-cfg wizard
  # Or simply (wizard is default):
  smartap-cfg

  # Launch wizard for specific device
  smartap-cfg wizard --device 192.168.4.16
  smartap-cfg --device 192.168.4.16`,
	RunE: runWizard,
}

func runWizard(cmd *cobra.Command, args []string) error {
	var model tea.Model

	if deviceIP != "" {
		// Direct to device configuration with manual IP
		// First verify we can connect
		client := deviceconfig.NewClient(deviceIP, devicePort)
		_, err := client.GetConfiguration()
		if err != nil {
			return fmt.Errorf("failed to connect to device at %s:%d: %w", deviceIP, devicePort, err)
		}

		// Create manual device entry
		device := &discovery.Device{
			IP:       deviceIP,
			Port:     devicePort,
			Hostname: deviceIP,
			Serial:   "manual",
		}

		// Start with dashboard screen
		model = tui.NewAppModel(tui.ScreenDashboard, device)
	} else {
		// Start with discovery screen (will auto-scan)
		model = tui.NewAppModel(tui.ScreenDiscovery, nil)
	}

	p := tea.NewProgram(model)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("wizard error: %w", err)
	}

	return nil
}

// setDiverterCmd directly sets diverter configuration
var setDiverterCmd = &cobra.Command{
	Use:   "set-diverter <button1> <button2> <button3> [k3mode]",
	Short: "Set diverter button configuration",
	Long: `Directly set the diverter button configuration without using the wizard.

Button values are 3-bit bitmasks (0-7) where:
  Bit 0 (1): Outlet 1 enabled
  Bit 1 (2): Outlet 2 enabled
  Bit 2 (4): Outlet 3 enabled

Examples:
  1 = Outlet 1 only
  3 = Outlets 1+2 simultaneously
  7 = All three outlets

K3 mode (third knob separation) can be enabled or disabled (true/false).`,
	Example: `  # Set sequential outlets (1→2→4), K3 mode disabled
  smartap-cfg set-diverter 1 2 4 false --device 192.168.4.16

  # Set first press to all outlets, second to outlets 1+2, third to outlet 3
  smartap-cfg set-diverter 7 3 4 true --device 192.168.4.16

  # Disable third knob separation
  smartap-cfg set-diverter 1 2 4 false --device 192.168.4.16`,
	Args: cobra.RangeArgs(3, 4),
	RunE: runSetDiverter,
}

func init() {
	setDiverterCmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip configuration verification after update")
	setDiverterCmd.Flags().IntVar(&retries, "retries", 3, "Number of verification retries")
}

func runSetDiverter(cmd *cobra.Command, args []string) error {
	// Get device IP
	ip, err := getDeviceIP()
	if err != nil {
		return err
	}

	// Parse button values
	button1, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid button1 value: %w", err)
	}
	button2, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid button2 value: %w", err)
	}
	button3, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("invalid button3 value: %w", err)
	}

	// Parse K3 mode (optional, defaults to false)
	k3Mode := false
	if len(args) >= 4 {
		k3Mode, err = strconv.ParseBool(args[3])
		if err != nil {
			return fmt.Errorf("invalid k3mode value (use true/false): %w", err)
		}
	}

	// Validate bitmasks
	if err := deviceconfig.ValidateDiverterBitmask(button1); err != nil {
		return fmt.Errorf("button1: %w", err)
	}
	if err := deviceconfig.ValidateDiverterBitmask(button2); err != nil {
		return fmt.Errorf("button2: %w", err)
	}
	if err := deviceconfig.ValidateDiverterBitmask(button3); err != nil {
		return fmt.Errorf("button3: %w", err)
	}

	// Create client
	client := deviceconfig.NewClient(ip, devicePort)

	// Create diverter configuration
	diverterConfig := &deviceconfig.DiverterConfig{
		FirstPress:  button1,
		SecondPress: button2,
		ThirdPress:  button3,
		K3Mode:      k3Mode,
	}

	fmt.Printf("Setting diverter configuration on %s:%d...\n", ip, devicePort)
	fmt.Printf("  First Press:  %d (%s)\n", button1, deviceconfig.FormatBitmask(button1))
	fmt.Printf("  Second Press: %d (%s)\n", button2, deviceconfig.FormatBitmask(button2))
	fmt.Printf("  Third Press:  %d (%s)\n", button3, deviceconfig.FormatBitmask(button3))
	fmt.Printf("  K3 Mode:      %v\n", k3Mode)
	fmt.Println()

	// Apply configuration
	update := &deviceconfig.ConfigUpdate{
		Diverter: diverterConfig,
	}

	if noVerify {
		// Just update without verification
		if err := client.UpdateConfiguration(update); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		fmt.Println("✓ Configuration updated successfully (not verified)")
	} else {
		// Update and verify
		opts := &deviceconfig.VerificationOptions{
			MaxRetries:            retries,
			InitialDelay:          500 * time.Millisecond,
			RetryDelay:            1 * time.Second,
			UseExponentialBackoff: true,
			MaxRetryDelay:         5 * time.Second,
		}

		result := client.UpdateAndVerify(update, opts)

		if !result.Success {
			fmt.Printf("✗ Configuration failed: %v\n", result.Error)
			if len(result.Mismatches) > 0 {
				fmt.Println("\nMismatches detected:")
				for _, mismatch := range result.Mismatches {
					fmt.Printf("  - %s\n", mismatch)
				}
			}
			return fmt.Errorf("configuration verification failed after %d attempts", result.Attempts)
		}

		fmt.Printf("✓ Configuration updated and verified successfully (%d attempt(s))\n", result.Attempts)

		if result.ActualConfig != nil {
			fmt.Println("\nVerified configuration:")
			fmt.Printf("  First Press:  %d (%s)\n", result.ActualConfig.Outlet1, deviceconfig.FormatBitmask(result.ActualConfig.Outlet1))
			fmt.Printf("  Second Press: %d (%s)\n", result.ActualConfig.Outlet2, deviceconfig.FormatBitmask(result.ActualConfig.Outlet2))
			fmt.Printf("  Third Press:  %d (%s)\n", result.ActualConfig.Outlet3, deviceconfig.FormatBitmask(result.ActualConfig.Outlet3))
			fmt.Printf("  K3 Mode:      %v\n", result.ActualConfig.K3Outlet)
		}
	}

	return nil
}

// setServerCmd directly sets server configuration
var setServerCmd = &cobra.Command{
	Use:   "set-server <dns> <port>",
	Short: "Set server configuration",
	Long: `Directly set the server DNS and port configuration.

This configures which server the device will connect to for WebSocket communication.`,
	Example: `  # Set server to custom domain
  smartap-cfg set-server smartap.local 443 --device 192.168.4.16

  # Set server to IP address
  smartap-cfg set-server 192.168.1.100 8443 --device 192.168.4.16`,
	Args: cobra.ExactArgs(2),
	RunE: runSetServer,
}

func init() {
	setServerCmd.Flags().BoolVar(&noVerify, "no-verify", false, "Skip configuration verification after update")
	setServerCmd.Flags().IntVar(&retries, "retries", 3, "Number of verification retries")
}

func runSetServer(cmd *cobra.Command, args []string) error {
	// Get device IP
	ip, err := getDeviceIP()
	if err != nil {
		return err
	}

	dns := args[0]
	port, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid port value: %w", err)
	}

	// Validate server configuration
	if err := deviceconfig.ValidateServerDNS(dns); err != nil {
		return fmt.Errorf("invalid DNS: %w", err)
	}
	if err := deviceconfig.ValidateServerPort(port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	// Create client
	client := deviceconfig.NewClient(ip, devicePort)

	// Create server configuration
	serverConfig := &deviceconfig.ServerConfig{
		DNS:  dns,
		Port: port,
	}

	fmt.Printf("Setting server configuration on %s:%d...\n", ip, devicePort)
	fmt.Printf("  DNS:  %s\n", dns)
	fmt.Printf("  Port: %d\n", port)
	fmt.Println()

	// Apply configuration
	update := &deviceconfig.ConfigUpdate{
		Server: serverConfig,
	}

	if noVerify {
		// Just update without verification
		if err := client.UpdateConfiguration(update); err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
		fmt.Println("✓ Configuration updated successfully (not verified)")
	} else {
		// Update and verify
		opts := &deviceconfig.VerificationOptions{
			MaxRetries:            retries,
			InitialDelay:          500 * time.Millisecond,
			RetryDelay:            1 * time.Second,
			UseExponentialBackoff: true,
			MaxRetryDelay:         5 * time.Second,
		}

		result := client.UpdateAndVerify(update, opts)

		if !result.Success {
			fmt.Printf("✗ Configuration failed: %v\n", result.Error)
			if len(result.Mismatches) > 0 {
				fmt.Println("\nMismatches detected:")
				for _, mismatch := range result.Mismatches {
					fmt.Printf("  - %s\n", mismatch)
				}
			}
			return fmt.Errorf("configuration verification failed after %d attempts", result.Attempts)
		}

		fmt.Printf("✓ Configuration updated and verified successfully (%d attempt(s))\n", result.Attempts)

		if result.ActualConfig != nil {
			fmt.Println("\nVerified configuration:")
			fmt.Printf("  DNS:  %s\n", result.ActualConfig.DNS)
			fmt.Printf("  Port: %d\n", result.ActualConfig.Port)
		}
	}

	return nil
}

// Helper function to get device IP (via discovery or manual flag)
func getDeviceIP() (string, error) {
	if deviceIP != "" {
		return deviceIP, nil
	}

	// Try discovery
	fmt.Println("No device IP specified, attempting auto-discovery...")
	devices, err := discovery.ScanForDevices(5 * time.Second)
	if err != nil {
		return "", fmt.Errorf("discovery failed: %w", err)
	}

	if len(devices) == 0 {
		return "", fmt.Errorf("no devices found. Use --device flag to specify IP manually")
	}

	if len(devices) > 1 {
		fmt.Printf("Found %d devices:\n", len(devices))
		for i, device := range devices {
			fmt.Printf("%d. %s (%s)\n", i+1, device.Serial, device.IP)
		}
		return "", fmt.Errorf("multiple devices found. Use --device flag to specify which one")
	}

	// Exactly one device found
	device := devices[0]
	fmt.Printf("Found device: %s (%s)\n\n", device.Serial, device.IP)
	return device.IP, nil
}
