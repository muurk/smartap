package gdb

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// PrerequisiteCheck represents the result of checking a single prerequisite.
type PrerequisiteCheck struct {
	// Name is the human-readable name of the prerequisite
	Name string
	// Available indicates whether the prerequisite is available
	Available bool
	// Path is the resolved path (for binary checks)
	Path string
	// Version is the detected version (if applicable)
	Version string
	// Message provides additional context (error message or success info)
	Message string
	// Error contains the underlying error if check failed
	Error error
}

// PrerequisiteResult contains the results of all prerequisite checks.
type PrerequisiteResult struct {
	// Checks contains individual check results
	Checks []PrerequisiteCheck
	// AllAvailable is true if all prerequisites are available
	AllAvailable bool
}

// ValidatePrerequisites checks for all required prerequisites and returns a detailed report.
// This includes:
//   - arm-none-eabi-gdb binary
//   - OpenOCD connectivity (optional warning if not available)
func ValidatePrerequisites(ctx context.Context, openocdHost string, openocdPort int) (*PrerequisiteResult, error) {
	result := &PrerequisiteResult{
		Checks:       make([]PrerequisiteCheck, 0),
		AllAvailable: true,
	}

	// Check for arm-none-eabi-gdb
	gdbCheck := checkGDBBinary(ctx)
	result.Checks = append(result.Checks, gdbCheck)
	if !gdbCheck.Available {
		result.AllAvailable = false
	}

	// Check OpenOCD connectivity (warning only, not required for validation)
	openocdCheck := checkOpenOCDConnection(ctx, openocdHost, openocdPort)
	result.Checks = append(result.Checks, openocdCheck)
	// Note: OpenOCD not being available doesn't fail validation, it's just a warning

	return result, nil
}

// checkGDBBinary verifies that arm-none-eabi-gdb is available and executable.
func checkGDBBinary(ctx context.Context) PrerequisiteCheck {
	check := PrerequisiteCheck{
		Name: "arm-none-eabi-gdb",
	}

	// Try to find GDB binary
	path, err := exec.LookPath("arm-none-eabi-gdb")
	if err != nil {
		check.Available = false
		check.Error = err
		check.Message = "arm-none-eabi-gdb not found in PATH\n" +
			"Install on macOS: brew install --cask gcc-arm-embedded\n" +
			"Install on Linux: sudo apt-get install gdb-multiarch && ln -s /usr/bin/gdb-multiarch /usr/local/bin/arm-none-eabi-gdb"
		return check
	}

	check.Path = path

	// Try to get version
	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(versionCtx, "arm-none-eabi-gdb", "--version")
	output, err := cmd.Output()
	if err != nil {
		check.Available = false
		check.Error = err
		check.Message = fmt.Sprintf("arm-none-eabi-gdb found at %s but failed to execute: %v", path, err)
		return check
	}

	// Parse version from first line
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		check.Version = strings.TrimSpace(lines[0])
	}

	check.Available = true
	check.Message = fmt.Sprintf("Found at %s", path)
	return check
}

// checkOpenOCDConnection attempts to connect to OpenOCD to verify it's running.
// This is a non-fatal check - if OpenOCD is not available, we just warn the user.
func checkOpenOCDConnection(ctx context.Context, host string, port int) PrerequisiteCheck {
	check := PrerequisiteCheck{
		Name: "OpenOCD Connection",
	}

	// Try to connect with a short timeout
	address := fmt.Sprintf("%s:%d", host, port)
	dialer := net.Dialer{
		Timeout: 2 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		check.Available = false
		check.Error = err
		check.Message = fmt.Sprintf("Cannot connect to OpenOCD at %s\n"+
			"This is not fatal, but GDB operations will fail.\n"+
			"Ensure OpenOCD is running: openocd -f <your-config.cfg>", address)
		return check
	}
	defer conn.Close()

	check.Available = true
	check.Message = fmt.Sprintf("Connected successfully to %s", address)
	return check
}

// ValidateGDBPath checks if a specific GDB binary path is valid and executable.
func ValidateGDBPath(ctx context.Context, gdbPath string) error {
	// Check if the path exists
	if gdbPath == "" {
		return &PrerequisiteError{
			Prerequisite: "arm-none-eabi-gdb",
			Details:      "GDB path is empty",
		}
	}

	// Try to execute --version
	versionCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(versionCtx, gdbPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return &PrerequisiteError{
			Prerequisite: "arm-none-eabi-gdb",
			Details:      fmt.Sprintf("Failed to execute %s --version", gdbPath),
			Err:          err,
		}
	}

	// Verify it's actually GDB (check for "GNU gdb" in output)
	if !strings.Contains(string(output), "GNU gdb") {
		return &PrerequisiteError{
			Prerequisite: "arm-none-eabi-gdb",
			Details:      fmt.Sprintf("%s does not appear to be GNU GDB", gdbPath),
		}
	}

	return nil
}

// ValidateOpenOCDConnection checks if OpenOCD is accessible at the given host and port.
func ValidateOpenOCDConnection(ctx context.Context, host string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)
	dialer := net.Dialer{
		Timeout: 2 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return &GDBConnectionError{
			Host: host,
			Port: port,
			Err:  err,
		}
	}
	defer conn.Close()

	return nil
}

// FormatPrerequisiteReport formats a PrerequisiteResult into a human-readable string.
func FormatPrerequisiteReport(result *PrerequisiteResult) string {
	var sb strings.Builder

	sb.WriteString("GDB Prerequisites Check:\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	for _, check := range result.Checks {
		if check.Available {
			sb.WriteString(fmt.Sprintf("✓ %s\n", check.Name))
			if check.Version != "" {
				sb.WriteString(fmt.Sprintf("  Version: %s\n", check.Version))
			}
			if check.Path != "" {
				sb.WriteString(fmt.Sprintf("  Path: %s\n", check.Path))
			}
			if check.Message != "" {
				sb.WriteString(fmt.Sprintf("  %s\n", check.Message))
			}
		} else {
			sb.WriteString(fmt.Sprintf("✗ %s\n", check.Name))
			if check.Message != "" {
				sb.WriteString(fmt.Sprintf("  %s\n", check.Message))
			}
		}
		sb.WriteString("\n")
	}

	if result.AllAvailable {
		sb.WriteString("All required prerequisites are available.\n")
	} else {
		sb.WriteString("Some prerequisites are missing. Please install them before proceeding.\n")
	}

	return sb.String()
}
