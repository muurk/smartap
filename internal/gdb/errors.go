package gdb

import (
	"fmt"

	"github.com/muurk/smartap/internal/urls"
)

// GDBExecutionError represents a failure during GDB script execution.
// This occurs when the GDB command itself fails (non-zero exit code, stderr output, etc.).
type GDBExecutionError struct {
	// Script is the name of the script that failed
	Script string
	// ExitCode is the GDB process exit code
	ExitCode int
	// Stderr is the GDB stderr output
	Stderr string
	// Stdout is the GDB stdout output (for context)
	Stdout string
	// Underlying error if any
	Err error
}

func (e *GDBExecutionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("gdb execution failed for script %q (exit code %d): %v\nstderr: %s",
			e.Script, e.ExitCode, e.Err, e.Stderr)
	}
	return fmt.Sprintf("gdb execution failed for script %q (exit code %d)\nstderr: %s",
		e.Script, e.ExitCode, e.Stderr)
}

func (e *GDBExecutionError) Unwrap() error {
	return e.Err
}

// GDBConnectionError represents a failure to connect to OpenOCD.
// This typically means OpenOCD is not running, the port is wrong, or the device is not connected.
type GDBConnectionError struct {
	// Host is the OpenOCD host that failed to connect
	Host string
	// Port is the OpenOCD port that failed to connect
	Port int
	// Underlying error
	Err error
}

func (e *GDBConnectionError) Error() string {
	return fmt.Sprintf("failed to connect to OpenOCD at %s:%d: %v\n"+
		"Hint: Ensure OpenOCD is running and the device is connected via JTAG.\n"+
		"Start OpenOCD with: openocd -f <your-config.cfg>",
		e.Host, e.Port, e.Err)
}

func (e *GDBConnectionError) Unwrap() error {
	return e.Err
}

// GDBParseError represents a failure to parse GDB output.
// This occurs when the output doesn't match expected format or patterns.
type GDBParseError struct {
	// Script is the name of the script whose output failed to parse
	Script string
	// Field is the specific field that failed to parse
	Field string
	// Output is the GDB output that failed to parse
	Output string
	// Underlying error
	Err error
}

func (e *GDBParseError) Error() string {
	return fmt.Sprintf("failed to parse GDB output for script %q, field %q: %v\n"+
		"Output: %s",
		e.Script, e.Field, e.Err, e.Output)
}

func (e *GDBParseError) Unwrap() error {
	return e.Err
}

// FirmwareUnsupportedError represents an unknown or unsupported firmware version.
// This occurs when the detected firmware version is not in the firmware catalog.
type FirmwareUnsupportedError struct {
	// Version is the detected firmware version
	Version string
	// Available lists known firmware versions
	Available []string
}

func (e *FirmwareUnsupportedError) Error() string {
	return fmt.Sprintf("unsupported firmware version: %s\n"+
		"\n"+
		"This firmware version is not in the catalog. To add support:\n"+
		"  1. Dump device memory: smartap-cfg gdb dump-memory --output firmware-%s.bin\n"+
		"  2. Submit for analysis: Open issue at https://github.com/yourrepo/smartap-revival/issues\n"+
		"  3. Include device details: model, manufacture date, any visible version numbers\n"+
		"\n"+
		"Known firmware versions:\n%s\n"+
		"\n"+
		"You can also specify the firmware version manually if you know the function addresses:\n"+
		"  smartap-cfg gdb inject-certs --firmware-version %s",
		e.Version, e.Version, formatVersionList(e.Available), e.Version)
}

func formatVersionList(versions []string) string {
	if len(versions) == 0 {
		return "  (none)"
	}
	result := ""
	for _, v := range versions {
		result += fmt.Sprintf("  - %s\n", v)
	}
	return result
}

// PrerequisiteError represents a missing prerequisite (GDB binary, OpenOCD, etc.).
type PrerequisiteError struct {
	// Prerequisite is the name of the missing prerequisite
	Prerequisite string
	// Details provides additional context
	Details string
	// Underlying error
	Err error
}

func (e *PrerequisiteError) Error() string {
	msg := fmt.Sprintf("missing prerequisite: %s", e.Prerequisite)
	if e.Details != "" {
		msg += "\n" + e.Details
	}
	if e.Err != nil {
		msg += fmt.Sprintf("\nError: %v", e.Err)
	}
	return msg
}

func (e *PrerequisiteError) Unwrap() error {
	return e.Err
}

// CertificateError represents a certificate-related error (loading, parsing, validation).
type CertificateError struct {
	// Operation describes what certificate operation failed
	Operation string
	// Path is the certificate file path (if applicable)
	Path string
	// Underlying error
	Err error
}

func (e *CertificateError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("certificate error during %s (file: %s): %v", e.Operation, e.Path, e.Err)
	}
	return fmt.Sprintf("certificate error during %s: %v", e.Operation, e.Err)
}

func (e *CertificateError) Unwrap() error {
	return e.Err
}

// TemplateError represents a template rendering error.
type TemplateError struct {
	// Template is the name of the template that failed to render
	Template string
	// Underlying error
	Err error
}

func (e *TemplateError) Error() string {
	return fmt.Sprintf("failed to render template %q: %v", e.Template, e.Err)
}

func (e *TemplateError) Unwrap() error {
	return e.Err
}

// TimeoutError represents a timeout during GDB operation.
type TimeoutError struct {
	// Script is the name of the script that timed out
	Script string
	// Timeout is the duration that was exceeded
	Timeout string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("gdb operation timed out for script %q after %s\n"+
		"Hint: Increase timeout with --timeout flag or check device connection",
		e.Script, e.Timeout)
}

// FirmwareConfidenceError represents insufficient confidence in firmware detection.
// This occurs when signature matching results in less than 100% confidence.
type FirmwareConfidenceError struct {
	// Version is the best-match firmware version
	Version string
	// Confidence is the confidence percentage (0-100)
	Confidence int
	// Matches is the number of signatures that matched
	Matches int
	// Total is the total number of signatures checked
	Total int
}

func (e *FirmwareConfidenceError) Error() string {
	return fmt.Sprintf(`firmware detection confidence too low: %d%% (%d/%d signatures matched)

GDB operations require 100%% confidence to ensure correct function addresses.
Without reliable addresses, operations may corrupt device memory.

Detected version: %s (unverified)

Recommended next steps:
1. Dump device memory for analysis:
   smartap-cfg gdb dump-memory --address 0x20000000 --size 262144 --output firmware-dump.bin

2. Analyze the dump to find function signatures:
   %s

3. Submit findings to add support for this firmware version:
   %s`,
		e.Confidence, e.Matches, e.Total, e.Version, urls.FindingFunctionsInMemory, urls.ContributingFirmware)
}
