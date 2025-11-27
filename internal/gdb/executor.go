package gdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"text/template"
	"time"

	"github.com/muurk/smartap/internal/gdb/scripts"
	"go.uber.org/zap"
)

// Config holds the configuration for GDB execution.
type Config struct {
	// GDBPath is the path to the arm-none-eabi-gdb binary.
	// Default: "arm-none-eabi-gdb" (searches PATH)
	GDBPath string

	// OpenOCDHost is the hostname/IP where OpenOCD is running.
	// Default: "localhost"
	OpenOCDHost string

	// OpenOCDPort is the port where OpenOCD is listening.
	// Default: 3333
	OpenOCDPort int

	// Timeout is the maximum time to wait for GDB to complete.
	// Default: 5 minutes
	Timeout time.Duration

	// WorkDir is the working directory for temporary files.
	// Default: os.TempDir()
	WorkDir string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		GDBPath:     "arm-none-eabi-gdb",
		OpenOCDHost: "localhost",
		OpenOCDPort: 3333,
		Timeout:     5 * time.Minute,
		WorkDir:     os.TempDir(),
	}
}

// Executor executes GDB scripts via os/exec.
type Executor struct {
	config Config
	logger *zap.Logger
}

// NewExecutor creates a new GDB executor with the given configuration.
func NewExecutor(config Config, logger *zap.Logger) *Executor {
	return &Executor{
		config: config,
		logger: logger,
	}
}

// Execute runs a GDB script and returns the parsed result.
// The script is rendered as a template, written to a temporary file,
// executed via arm-none-eabi-gdb, and then parsed for results.
//
// Steps:
//  1. Render script template with parameters
//  2. Write rendered script to temporary file
//  3. Execute GDB with script file
//  4. Capture stdout/stderr
//  5. Parse output using script.Parse()
//  6. Clean up temporary file
//  7. Return result or error
func (e *Executor) Execute(ctx context.Context, script scripts.Script) (*scripts.Result, error) {
	startTime := time.Now()

	e.logger.Info("executing GDB script",
		zap.String("script", script.Name()),
		zap.String("gdb_path", e.config.GDBPath),
		zap.String("openocd", fmt.Sprintf("%s:%d", e.config.OpenOCDHost, e.config.OpenOCDPort)),
		zap.Duration("timeout", e.config.Timeout),
	)

	// Render template
	rendered, err := e.renderTemplate(script)
	if err != nil {
		return nil, &TemplateError{
			Template: script.Name(),
			Err:      err,
		}
	}

	e.logger.Debug("rendered GDB script template",
		zap.String("script", script.Name()),
		zap.Int("size", len(rendered)),
		zap.String("content", rendered),
	)

	// Write to temporary file
	scriptFile, err := e.writeScriptFile(script.Name(), rendered)
	if err != nil {
		return nil, fmt.Errorf("failed to write script file: %w", err)
	}
	defer os.Remove(scriptFile) // Clean up temp file

	e.logger.Debug("wrote GDB script to temporary file",
		zap.String("script", script.Name()),
		zap.String("file", scriptFile),
	)

	// Execute GDB (with or without streaming based on script configuration)
	stdout, stderr, exitCode, err := e.executeGDB(ctx, scriptFile, script.Streaming())
	duration := time.Since(startTime)

	e.logger.Debug("GDB execution complete",
		zap.String("script", script.Name()),
		zap.Duration("duration", duration),
		zap.Int("exit_code", exitCode),
		zap.Int("stdout_size", len(stdout)),
		zap.Int("stderr_size", len(stderr)),
		zap.String("stdout", stdout),
		zap.String("stderr", stderr),
	)

	// Check for execution errors
	if err != nil {
		return nil, &GDBExecutionError{
			Script:   script.Name(),
			ExitCode: exitCode,
			Stderr:   stderr,
			Stdout:   stdout,
			Err:      err,
		}
	}

	// Check for non-zero exit code
	if exitCode != 0 {
		return nil, &GDBExecutionError{
			Script:   script.Name(),
			ExitCode: exitCode,
			Stderr:   stderr,
			Stdout:   stdout,
		}
	}

	// Parse output
	result, err := script.Parse(stdout)
	if err != nil {
		return nil, err // script.Parse should return GDBParseError
	}

	// Set metadata
	result.Duration = duration
	result.RawOutput = stdout
	result.RawStderr = stderr

	e.logger.Info("GDB script executed successfully",
		zap.String("script", script.Name()),
		zap.Duration("duration", duration),
		zap.Bool("success", result.Success),
		zap.Int("steps", result.TotalSteps()),
		zap.Int("bytes_written", result.BytesWritten),
		zap.Int("bytes_read", result.BytesRead),
	)

	return result, nil
}

// renderTemplate renders the script template with parameters.
func (e *Executor) renderTemplate(script scripts.Script) (string, error) {
	tmpl, err := template.New(script.Name()).Parse(script.Template())
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, script.Params()); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// writeScriptFile writes the rendered script to a temporary file.
func (e *Executor) writeScriptFile(name, content string) (string, error) {
	// Create temporary file
	filename := fmt.Sprintf("smartap-gdb-%s-*.gdb", name)
	file, err := os.CreateTemp(e.config.WorkDir, filename)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	// Write content
	if _, err := file.WriteString(content); err != nil {
		os.Remove(file.Name())
		return "", fmt.Errorf("failed to write script content: %w", err)
	}

	return file.Name(), nil
}

// executeGDB executes arm-none-eabi-gdb with the given script file.
// If streaming is true, output is piped to os.Stdout/os.Stderr in real-time.
// If streaming is false, output is captured in buffers.
func (e *Executor) executeGDB(ctx context.Context, scriptFile string, streaming bool) (stdout, stderr string, exitCode int, err error) {
	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// Build command
	// -batch: Exit after processing script (not used for streaming - causes buffering)
	// -nx: Don't execute .gdbinit
	// -x: Execute commands from file
	var cmd *exec.Cmd
	if streaming {
		// For streaming, don't use -batch to avoid output buffering
		cmd = exec.CommandContext(timeoutCtx, e.config.GDBPath,
			"-nx",
			"-x", scriptFile,
		)
	} else {
		// For non-streaming, use -batch for clean exit
		cmd = exec.CommandContext(timeoutCtx, e.config.GDBPath,
			"-batch",
			"-nx",
			"-x", scriptFile,
		)
	}

	// Setup output capture based on streaming mode
	var stdoutBuf, stderrBuf bytes.Buffer

	if streaming {
		// Streaming mode: Use pipes for real-time unbuffered output
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return "", "", -1, fmt.Errorf("failed to create stdout pipe: %w", err)
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return "", "", -1, fmt.Errorf("failed to create stderr pipe: %w", err)
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			return "", "", -1, fmt.Errorf("failed to start GDB: %w", err)
		}

		// Copy output in real-time using goroutines
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			io.Copy(io.MultiWriter(&stdoutBuf, os.Stdout), stdoutPipe)
		}()

		go func() {
			defer wg.Done()
			io.Copy(io.MultiWriter(&stderrBuf, os.Stderr), stderrPipe)
		}()

		// Wait for command to finish and pipes to be fully read
		wg.Wait()
		err = cmd.Wait()
	} else {
		// Buffered mode: only capture to buffers
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		err = cmd.Run()
	}

	// Get output
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()

	// Get exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start or other error
			exitCode = -1
		}
	}

	// Check for timeout
	if timeoutCtx.Err() == context.DeadlineExceeded {
		err = &TimeoutError{
			Script:  filepath.Base(scriptFile),
			Timeout: e.config.Timeout.String(),
		}
	}

	return stdout, stderr, exitCode, err
}

// ValidateConfig validates the executor configuration.
func (e *Executor) ValidateConfig(ctx context.Context) error {
	// Validate GDB path
	if err := ValidateGDBPath(ctx, e.config.GDBPath); err != nil {
		return err
	}

	// Validate OpenOCD connection (warning only)
	if err := ValidateOpenOCDConnection(ctx, e.config.OpenOCDHost, e.config.OpenOCDPort); err != nil {
		e.logger.Warn("OpenOCD connection check failed (this is not fatal)",
			zap.String("host", e.config.OpenOCDHost),
			zap.Int("port", e.config.OpenOCDPort),
			zap.Error(err),
		)
	}

	return nil
}
