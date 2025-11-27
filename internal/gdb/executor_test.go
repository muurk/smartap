package gdb

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/muurk/smartap/internal/gdb/scripts"
	"go.uber.org/zap"
)

// mockScript implements scripts.Script for testing
type mockScript struct {
	name         string
	template     string
	params       map[string]interface{}
	parseFunc    func(output string) (*scripts.Result, error)
	parseError   error
	parseSuccess bool
	streaming    bool
}

func (m *mockScript) Name() string {
	return m.name
}

func (m *mockScript) Template() string {
	return m.template
}

func (m *mockScript) Params() map[string]interface{} {
	return m.params
}

func (m *mockScript) Parse(output string) (*scripts.Result, error) {
	if m.parseFunc != nil {
		return m.parseFunc(output)
	}
	result := scripts.NewResult()
	result.Success = m.parseSuccess
	result.AddStep("[1/1] Test step", "success", "")
	return result, m.parseError
}

func (m *mockScript) Streaming() bool {
	return m.streaming
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.GDBPath != "arm-none-eabi-gdb" {
		t.Errorf("expected GDBPath to be 'arm-none-eabi-gdb', got %s", config.GDBPath)
	}

	if config.OpenOCDHost != "localhost" {
		t.Errorf("expected OpenOCDHost to be 'localhost', got %s", config.OpenOCDHost)
	}

	if config.OpenOCDPort != 3333 {
		t.Errorf("expected OpenOCDPort to be 3333, got %d", config.OpenOCDPort)
	}

	if config.Timeout != 5*time.Minute {
		t.Errorf("expected Timeout to be 5 minutes, got %s", config.Timeout)
	}

	if config.WorkDir != os.TempDir() {
		t.Errorf("expected WorkDir to be temp dir, got %s", config.WorkDir)
	}
}

func TestNewExecutor(t *testing.T) {
	config := Config{
		GDBPath:     "/usr/bin/gdb",
		OpenOCDHost: "192.168.1.1",
		OpenOCDPort: 4444,
		Timeout:     10 * time.Minute,
		WorkDir:     "/tmp",
	}

	logger := zap.NewNop()
	executor := NewExecutor(config, logger)

	if executor == nil {
		t.Fatal("expected non-nil executor")
	}

	if executor.config.GDBPath != config.GDBPath {
		t.Errorf("expected GDBPath %s, got %s", config.GDBPath, executor.config.GDBPath)
	}

	if executor.config.OpenOCDHost != config.OpenOCDHost {
		t.Errorf("expected OpenOCDHost %s, got %s", config.OpenOCDHost, executor.config.OpenOCDHost)
	}

	if executor.config.OpenOCDPort != config.OpenOCDPort {
		t.Errorf("expected OpenOCDPort %d, got %d", config.OpenOCDPort, executor.config.OpenOCDPort)
	}

	if executor.logger != logger {
		t.Error("expected logger to be set")
	}
}

func TestExecutor_RenderTemplate(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		params      map[string]interface{}
		expected    string
		expectError bool
	}{
		{
			name:     "simple template",
			template: "Hello {{.Name}}!",
			params: map[string]interface{}{
				"Name": "World",
			},
			expected: "Hello World!",
		},
		{
			name:     "nested fields",
			template: "Function: {{.Firmware.Functions.sl_FsOpen}}",
			params: map[string]interface{}{
				"Firmware": map[string]interface{}{
					"Functions": map[string]interface{}{
						"sl_FsOpen": "0x20015c64",
					},
				},
			},
			expected: "Function: 0x20015c64",
		},
		{
			name:     "multiple parameters",
			template: "Host: {{.Host}}, Port: {{.Port}}",
			params: map[string]interface{}{
				"Host": "localhost",
				"Port": 3333,
			},
			expected: "Host: localhost, Port: 3333",
		},
		{
			name:        "invalid template syntax",
			template:    "Hello {{.Name",
			params:      map[string]interface{}{},
			expectError: true,
		},
		{
			name:     "missing parameter (should use zero value)",
			template: "Value: {{.Missing}}",
			params:   map[string]interface{}{},
			expected: "Value: <no value>",
		},
	}

	executor := NewExecutor(DefaultConfig(), zap.NewNop())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := &mockScript{
				name:     "test",
				template: tt.template,
				params:   tt.params,
			}

			result, err := executor.renderTemplate(script)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestExecutor_WriteScriptFile(t *testing.T) {
	executor := NewExecutor(DefaultConfig(), zap.NewNop())

	content := "# GDB test script\nquit"
	filename, err := executor.writeScriptFile("test", content)
	if err != nil {
		t.Fatalf("writeScriptFile failed: %v", err)
	}
	defer func() { _ = os.Remove(filename) }()

	// Check file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Errorf("script file was not created: %s", filename)
	}

	// Check filename pattern
	if !strings.Contains(filepath.Base(filename), "smartap-gdb-test") {
		t.Errorf("unexpected filename: %s", filename)
	}

	// Check content
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read script file: %v", err)
	}

	if string(data) != content {
		t.Errorf("expected content %q, got %q", content, string(data))
	}
}

func TestExecutor_WriteScriptFile_InvalidWorkDir(t *testing.T) {
	config := DefaultConfig()
	config.WorkDir = "/nonexistent/directory/that/does/not/exist"
	executor := NewExecutor(config, zap.NewNop())

	_, err := executor.writeScriptFile("test", "content")
	if err == nil {
		t.Error("expected error for invalid work directory, got nil")
	}
}

func TestExecutor_Execute_TemplateError(t *testing.T) {
	executor := NewExecutor(DefaultConfig(), zap.NewNop())

	script := &mockScript{
		name:     "test",
		template: "Invalid {{.Template", // Invalid syntax
		params:   map[string]interface{}{},
	}

	_, err := executor.Execute(context.Background(), script)
	if err == nil {
		t.Fatal("expected template error, got nil")
	}

	// Check error type
	var templateErr *TemplateError
	if !errors.As(err, &templateErr) {
		t.Errorf("expected TemplateError, got %T: %v", err, err)
	}
}

func TestExecutor_Execute_ParseError(t *testing.T) {
	// This test requires a mock GDB command that succeeds
	// We'll create a shell script that acts as GDB
	tempDir := t.TempDir()
	mockGDB := filepath.Join(tempDir, "mock-gdb")

	// Create mock GDB script
	mockGDBScript := `#!/bin/sh
echo "GDB output"
exit 0
`
	if err := os.WriteFile(mockGDB, []byte(mockGDBScript), 0755); err != nil {
		t.Fatalf("failed to create mock GDB: %v", err)
	}

	config := DefaultConfig()
	config.GDBPath = mockGDB
	config.WorkDir = tempDir
	executor := NewExecutor(config, zap.NewNop())

	parseError := errors.New("parse failed")
	script := &mockScript{
		name:       "test",
		template:   "quit",
		params:     map[string]interface{}{},
		parseError: parseError,
	}

	_, err := executor.Execute(context.Background(), script)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}

	if !errors.Is(err, parseError) {
		t.Errorf("expected parse error, got %v", err)
	}
}

func TestExecutor_Execute_Success(t *testing.T) {
	// Create mock GDB command
	tempDir := t.TempDir()
	mockGDB := filepath.Join(tempDir, "mock-gdb")

	mockGDBScript := `#!/bin/sh
echo "GDB mock output"
echo "[1/1] Test step complete"
exit 0
`
	if err := os.WriteFile(mockGDB, []byte(mockGDBScript), 0755); err != nil {
		t.Fatalf("failed to create mock GDB: %v", err)
	}

	config := DefaultConfig()
	config.GDBPath = mockGDB
	config.WorkDir = tempDir
	executor := NewExecutor(config, zap.NewNop())

	script := &mockScript{
		name:         "test",
		template:     "quit",
		params:       map[string]interface{}{},
		parseSuccess: true,
	}

	result, err := executor.Execute(context.Background(), script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if !result.Success {
		t.Error("expected success=true")
	}

	if result.Duration == 0 {
		t.Error("expected duration to be set")
	}

	if result.RawOutput == "" {
		t.Error("expected RawOutput to be set")
	}

	if !strings.Contains(result.RawOutput, "GDB mock output") {
		t.Errorf("expected output to contain 'GDB mock output', got: %s", result.RawOutput)
	}
}

func TestExecutor_Execute_NonZeroExitCode(t *testing.T) {
	// Create mock GDB command that exits with error
	tempDir := t.TempDir()
	mockGDB := filepath.Join(tempDir, "mock-gdb")

	mockGDBScript := `#!/bin/sh
echo "Error occurred" >&2
exit 1
`
	if err := os.WriteFile(mockGDB, []byte(mockGDBScript), 0755); err != nil {
		t.Fatalf("failed to create mock GDB: %v", err)
	}

	config := DefaultConfig()
	config.GDBPath = mockGDB
	config.WorkDir = tempDir
	executor := NewExecutor(config, zap.NewNop())

	script := &mockScript{
		name:     "test",
		template: "quit",
		params:   map[string]interface{}{},
	}

	_, err := executor.Execute(context.Background(), script)
	if err == nil {
		t.Fatal("expected error for non-zero exit code, got nil")
	}

	var execErr *GDBExecutionError
	if !errors.As(err, &execErr) {
		t.Errorf("expected GDBExecutionError, got %T: %v", err, err)
	} else {
		if execErr.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", execErr.ExitCode)
		}
		if !strings.Contains(execErr.Stderr, "Error occurred") {
			t.Errorf("expected stderr to contain 'Error occurred', got: %s", execErr.Stderr)
		}
	}
}

func TestExecutor_Execute_CommandNotFound(t *testing.T) {
	config := DefaultConfig()
	config.GDBPath = "/nonexistent/gdb/binary"
	executor := NewExecutor(config, zap.NewNop())

	script := &mockScript{
		name:     "test",
		template: "quit",
		params:   map[string]interface{}{},
	}

	_, err := executor.Execute(context.Background(), script)
	if err == nil {
		t.Fatal("expected error for nonexistent GDB binary, got nil")
	}

	var execErr *GDBExecutionError
	if !errors.As(err, &execErr) {
		t.Errorf("expected GDBExecutionError, got %T: %v", err, err)
	}
}

func TestExecutor_Execute_Timeout(t *testing.T) {
	// Create mock GDB command that sleeps
	tempDir := t.TempDir()
	mockGDB := filepath.Join(tempDir, "mock-gdb")

	mockGDBScript := `#!/bin/sh
sleep 10
exit 0
`
	if err := os.WriteFile(mockGDB, []byte(mockGDBScript), 0755); err != nil {
		t.Fatalf("failed to create mock GDB: %v", err)
	}

	config := DefaultConfig()
	config.GDBPath = mockGDB
	config.WorkDir = tempDir
	config.Timeout = 100 * time.Millisecond // Short timeout
	executor := NewExecutor(config, zap.NewNop())

	script := &mockScript{
		name:     "test",
		template: "quit",
		params:   map[string]interface{}{},
	}

	_, err := executor.Execute(context.Background(), script)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	var timeoutErr *TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestExecutor_Execute_ContextCancellation(t *testing.T) {
	// Create mock GDB command that sleeps
	tempDir := t.TempDir()
	mockGDB := filepath.Join(tempDir, "mock-gdb")

	mockGDBScript := `#!/bin/sh
sleep 10
exit 0
`
	if err := os.WriteFile(mockGDB, []byte(mockGDBScript), 0755); err != nil {
		t.Fatalf("failed to create mock GDB: %v", err)
	}

	config := DefaultConfig()
	config.GDBPath = mockGDB
	config.WorkDir = tempDir
	executor := NewExecutor(config, zap.NewNop())

	script := &mockScript{
		name:     "test",
		template: "quit",
		params:   map[string]interface{}{},
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	_, err := executor.Execute(ctx, script)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

func TestExecutor_Execute_WithParameters(t *testing.T) {
	// Create mock GDB command
	tempDir := t.TempDir()
	mockGDB := filepath.Join(tempDir, "mock-gdb")

	mockGDBScript := `#!/bin/sh
# Arguments: gdb -batch -nx -x <file>
# So the script file is argument $4
cat "$4"
exit 0
`
	if err := os.WriteFile(mockGDB, []byte(mockGDBScript), 0755); err != nil {
		t.Fatalf("failed to create mock GDB: %v", err)
	}

	config := DefaultConfig()
	config.GDBPath = mockGDB
	config.WorkDir = tempDir
	executor := NewExecutor(config, zap.NewNop())

	script := &mockScript{
		name:     "test",
		template: "target remote {{.Host}}:{{.Port}}",
		params: map[string]interface{}{
			"Host": "localhost",
			"Port": 3333,
		},
		parseSuccess: true,
	}

	result, err := executor.Execute(context.Background(), script)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "target remote localhost:3333"
	if !strings.Contains(result.RawOutput, expected) {
		t.Errorf("expected output to contain %q, got: %s", expected, result.RawOutput)
	}
}

func TestExecutor_ValidateConfig_Success(t *testing.T) {
	// This test will fail if arm-none-eabi-gdb is not installed
	// We'll skip it in that case
	config := DefaultConfig()
	executor := NewExecutor(config, zap.NewNop())

	ctx := context.Background()
	err := executor.ValidateConfig(ctx)

	// We expect this to pass or give a warning about OpenOCD
	// but not fail completely if GDB exists
	if err != nil {
		// Check if it's a GDB not found error
		var prereqErr *PrerequisiteError
		if errors.As(err, &prereqErr) {
			t.Skip("arm-none-eabi-gdb not found, skipping validation test")
		} else {
			t.Errorf("unexpected validation error: %v", err)
		}
	}
}

func TestExecutor_ValidateConfig_InvalidGDBPath(t *testing.T) {
	config := DefaultConfig()
	config.GDBPath = "/nonexistent/gdb"
	executor := NewExecutor(config, zap.NewNop())

	ctx := context.Background()
	err := executor.ValidateConfig(ctx)

	if err == nil {
		t.Fatal("expected error for invalid GDB path, got nil")
	}

	var prereqErr *PrerequisiteError
	if !errors.As(err, &prereqErr) {
		t.Errorf("expected PrerequisiteError, got %T: %v", err, err)
	}
}
