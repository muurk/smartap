// Package ui provides terminal UI components for the smartap-cfg CLI.
//
// This package uses Bubble Tea and Lipgloss to render polished terminal output
// for GDB commands. Unlike the interactive TUI wizard, these components follow
// a "run once and exit" pattern - they render output compellingly but don't
// require user interaction.
//
// # Architecture
//
// The UI package provides four main component types:
//
//   - Header: Command banner showing operation name and parameters
//   - Progress: Progress bar with step list showing real-time status
//   - Result: Success/failure boxes with styled information
//   - GDBOutput: Raw GDB output box for verbose mode
//
// These components are orchestrated by the GDBRunner, which manages the
// header → progress → result flow for GDB command execution.
//
// # Usage Pattern
//
// GDB commands use this package by:
//
//  1. Creating a GDBRunner with command metadata
//  2. Calling Run() with their operation function
//  3. The operation reports progress via a step callback
//  4. GDBRunner handles all UI rendering automatically
//
// Example:
//
//	runner := ui.NewGDBRunner(ui.GDBRunnerConfig{
//	    Title:      "Certificate Injection",
//	    Command:    "smartap-cfg gdb inject-certs",
//	    Params:     map[string]string{"Device": "localhost:3333"},
//	    TotalSteps: 9,
//	    Verbose:    verbose,
//	})
//
//	err := runner.Run(ctx, func(onStep ui.StepCallback) error {
//	    onStep(1, "Halting device", ui.StepRunning, "")
//	    // ... do work ...
//	    onStep(1, "Halting device", ui.StepComplete, "")
//	    return nil
//	})
//
// # Logging Integration
//
// This package expects logging to be controlled via the SMARTAP_LOG_LEVEL
// environment variable. When unset or empty, zap logging is silent, allowing
// the curated UI output to be displayed cleanly. Set SMARTAP_LOG_LEVEL to
// "debug", "info", "warn", or "error" to enable logging output.
//
// # Verbose Mode
//
// When --verbose is passed to GDB commands, the GDBOutput component displays
// raw GDB output in a styled box after the result. This is useful for
// debugging and seeing exactly what GDB operations were performed.
package ui
