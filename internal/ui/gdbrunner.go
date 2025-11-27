package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// GDBRunnerConfig holds configuration for a GDB command execution
type GDBRunnerConfig struct {
	Title      string            // Command title (e.g., "Certificate Injection")
	Command    string            // Full command (e.g., "smartap-cfg gdb inject-certs")
	Params     map[string]string // Parameters to display in header
	TotalSteps int               // Total number of steps (for progress)
	StepNames  []string          // Names for each step
	Verbose    bool              // Whether to show GDB output
	Output     io.Writer         // Output writer (default: os.Stdout)
}

// GDBRunner orchestrates the UI for a GDB command execution.
// It manages the header → progress → result flow and provides
// callbacks for reporting progress.
type GDBRunner struct {
	config    GDBRunnerConfig
	header    *Header
	progress  *Progress
	output    io.Writer
	gdbOutput string
	startTime time.Time
	width     int
}

// NewGDBRunner creates a new runner for a GDB command
func NewGDBRunner(config GDBRunnerConfig) *GDBRunner {
	// Set defaults
	if config.Output == nil {
		config.Output = os.Stdout
	}

	width := GetTerminalWidth()

	// Create header
	header := NewHeader(config.Title, config.Command, config.Params)
	header.SetWidth(width)

	// Create progress tracker
	var progress *Progress
	if config.TotalSteps > 0 {
		progress = NewProgress("", config.TotalSteps)
		progress.SetWidth(width)
		if len(config.StepNames) > 0 {
			progress.SetStepNames(config.StepNames)
		}
	}

	return &GDBRunner{
		config:   config,
		header:   header,
		progress: progress,
		output:   config.Output,
		width:    width,
	}
}

// GDBOperation is the function signature for the actual GDB operation.
// The operation receives a StepCallback to report progress.
type GDBOperation func(onStep StepCallback) error

// Run executes the GDB operation with UI updates.
// It displays the header, tracks progress, and shows the result.
func (r *GDBRunner) Run(ctx context.Context, operation GDBOperation) error {
	r.startTime = time.Now()

	// Print header
	_, _ = fmt.Fprintln(r.output, r.header.Render())
	_, _ = fmt.Fprintln(r.output)

	// Create step callback
	stepCallback := r.createStepCallback()

	// Execute the operation
	err := operation(stepCallback)
	duration := time.Since(r.startTime)

	// Print final result
	if err != nil {
		r.printFailure(err, duration)
	} else {
		r.printSuccess(duration)
	}

	return err
}

// RunWithResult executes the GDB operation and allows custom result handling.
// Returns the result details that were displayed.
func (r *GDBRunner) RunWithResult(ctx context.Context, operation func(onStep StepCallback) (map[string]string, error)) (map[string]string, error) {
	r.startTime = time.Now()

	// Print header
	_, _ = fmt.Fprintln(r.output, r.header.Render())
	_, _ = fmt.Fprintln(r.output)

	// Create step callback
	stepCallback := r.createStepCallback()

	// Execute the operation
	details, err := operation(stepCallback)
	duration := time.Since(r.startTime)

	// Print final result
	if err != nil {
		r.printFailure(err, duration)
	} else {
		r.printSuccessWithDetails(details, duration)
	}

	return details, err
}

// SetGDBOutput stores GDB output for verbose display
func (r *GDBRunner) SetGDBOutput(output string) {
	r.gdbOutput = output
}

// createStepCallback creates the step callback function
func (r *GDBRunner) createStepCallback() StepCallback {
	return func(stepNumber int, name string, status StepStatus, message string) {
		if r.progress == nil {
			return
		}

		// Update step name if provided
		if name != "" && stepNumber > 0 && stepNumber <= len(r.progress.Steps) {
			r.progress.Steps[stepNumber-1].Name = name
		}

		// Update step status
		r.progress.UpdateStep(stepNumber, status, message)

		// Print progress line
		if status == StepComplete || status == StepFailed || status == StepSkipped {
			// Print completed step
			step := r.progress.Steps[stepNumber-1]
			_, _ = fmt.Fprintln(r.output, r.progress.renderStepLine(step))
		} else if status == StepRunning {
			// Print running step (will be overwritten when complete)
			step := r.progress.Steps[stepNumber-1]
			_, _ = fmt.Fprint(r.output, r.progress.renderStepLine(step)+"\r")
		}
	}
}

// printSuccess prints a success result
func (r *GDBRunner) printSuccess(duration time.Duration) {
	_, _ = fmt.Fprintln(r.output)

	// Default success details
	details := map[string]string{
		"Duration": duration.Round(time.Millisecond).String(),
	}

	result := NewSuccessResult(r.config.Title+" complete", details)
	result.SetWidth(r.width)
	_, _ = fmt.Fprintln(r.output, result.Render())

	// Show GDB output in verbose mode
	if r.config.Verbose && r.gdbOutput != "" {
		_, _ = fmt.Fprintln(r.output)
		gdb := NewGDBOutput(r.gdbOutput)
		gdb.SetWidth(r.width)
		_, _ = fmt.Fprintln(r.output, gdb.Render())
	}
}

// printSuccessWithDetails prints a success result with custom details
func (r *GDBRunner) printSuccessWithDetails(details map[string]string, duration time.Duration) {
	_, _ = fmt.Fprintln(r.output)

	// Add duration to details
	if details == nil {
		details = make(map[string]string)
	}
	details["Duration"] = duration.Round(time.Millisecond).String()

	result := NewSuccessResult(r.config.Title+" complete", details)
	result.SetWidth(r.width)
	_, _ = fmt.Fprintln(r.output, result.Render())

	// Show GDB output in verbose mode
	if r.config.Verbose && r.gdbOutput != "" {
		_, _ = fmt.Fprintln(r.output)
		gdb := NewGDBOutput(r.gdbOutput)
		gdb.SetWidth(r.width)
		_, _ = fmt.Fprintln(r.output, gdb.Render())
	}
}

// printFailure prints a failure result with troubleshooting
func (r *GDBRunner) printFailure(err error, duration time.Duration) {
	_, _ = fmt.Fprintln(r.output)

	// Default troubleshooting tips
	troubleshooting := []string{
		"Verify OpenOCD is still connected",
		"Check device hasn't reset unexpectedly",
		"Try: smartap-cfg gdb verify-setup",
		"Run with --verbose for full GDB output",
	}

	result := NewFailureResult(r.config.Title+" failed", err, troubleshooting)
	result.SetWidth(r.width)
	_, _ = fmt.Fprintln(r.output, result.Render())

	// Always show GDB output on failure in verbose mode
	if r.config.Verbose && r.gdbOutput != "" {
		_, _ = fmt.Fprintln(r.output)
		gdb := NewGDBOutput(r.gdbOutput)
		gdb.SetWidth(r.width)
		_, _ = fmt.Fprintln(r.output, gdb.Render())
	}
}

// --- Simple helper functions for commands that don't need full GDBRunner ---

// PrintCommandHeader prints a styled command header
func PrintCommandHeader(title, command string, params map[string]string) {
	width := GetTerminalWidth()
	header := NewHeader(title, command, params)
	header.SetWidth(width)
	fmt.Println(header.Render())
	fmt.Println()
}

// PrintSuccess prints a styled success result
func PrintSuccess(title string, details map[string]string) {
	width := GetTerminalWidth()
	result := NewSuccessResult(title, details)
	result.SetWidth(width)
	fmt.Println()
	fmt.Println(result.Render())
}

// PrintFailure prints a styled failure result
func PrintFailure(title string, err error, troubleshooting []string) {
	width := GetTerminalWidth()
	result := NewFailureResult(title, err, troubleshooting)
	result.SetWidth(width)
	fmt.Println()
	fmt.Println(result.Render())
}

// PrintWarning prints a styled warning result
func PrintWarning(title string, details map[string]string) {
	width := GetTerminalWidth()
	result := NewWarningResult(title, details)
	result.SetWidth(width)
	fmt.Println()
	fmt.Println(result.Render())
}

// PrintGDBOutput prints a styled GDB output box (for verbose mode)
func PrintGDBOutput(output string) {
	width := GetTerminalWidth()
	gdb := NewGDBOutput(output)
	gdb.SetWidth(width)
	fmt.Println()
	fmt.Println(gdb.Render())
}

// PrintPleaseWait prints a styled "please wait" message for long-running operations.
// The message parameter should describe what's happening, e.g., "Injecting certificate".
// The duration hint helps set user expectations, e.g., "up to 60 seconds".
func PrintPleaseWait(message string, durationHint string) {
	// Use primary/purple color - stands out but doesn't cause alarm
	style := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true).
		PaddingLeft(2)

	hintStyle := lipgloss.NewStyle().
		Foreground(MutedColor).
		Italic(true)

	line := style.Render("⏳ " + message)
	if durationHint != "" {
		line += " " + hintStyle.Render("("+durationHint+")")
	}
	line += style.Render("...")

	fmt.Println()
	fmt.Println(line)
	fmt.Println()
}
