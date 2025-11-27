package scripts

import (
	"strings"
	"testing"
)

func TestParseInjectionSteps(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []Step
	}{
		{
			name:     "no steps",
			output:   "Some random output",
			expected: []Step{},
		},
		{
			name: "single step",
			output: `[1/6] Halting device...
GDB output`,
			expected: []Step{
				{Name: "[1/6] Halting device", Status: "success", Message: ""},
			},
		},
		{
			name: "multiple steps",
			output: `[1/6] Halting device...
[2/6] Setting up filename...
[3/6] Loading certificate to memory...
[4/6] Deleting old certificate...
[5/6] Creating new certificate file...
[6/6] Writing certificate data...`,
			expected: []Step{
				{Name: "[1/6] Halting device", Status: "success", Message: ""},
				{Name: "[2/6] Setting up filename", Status: "success", Message: ""},
				{Name: "[3/6] Loading certificate to memory", Status: "success", Message: ""},
				{Name: "[4/6] Deleting old certificate", Status: "success", Message: ""},
				{Name: "[5/6] Creating new certificate file", Status: "success", Message: ""},
				{Name: "[6/6] Writing certificate data", Status: "success", Message: ""},
			},
		},
		{
			name: "steps without trailing dots",
			output: `[1/2] First step
[2/2] Second step`,
			expected: []Step{
				{Name: "[1/2] First step", Status: "success", Message: ""},
				{Name: "[2/2] Second step", Status: "success", Message: ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := parseInjectionSteps(tt.output)

			if len(steps) != len(tt.expected) {
				t.Fatalf("expected %d steps, got %d", len(tt.expected), len(steps))
			}

			for i, step := range steps {
				expected := tt.expected[i]
				if step.Name != expected.Name {
					t.Errorf("step %d: expected Name %q, got %q", i, expected.Name, step.Name)
				}
				if step.Status != expected.Status {
					t.Errorf("step %d: expected Status %q, got %q", i, expected.Status, step.Status)
				}
			}
		})
	}
}

func TestExtractResult(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		field    string
		expected int
	}{
		{
			name:     "delete result success",
			output:   "delete_result: 0",
			field:    "delete_result",
			expected: 0,
		},
		{
			name:     "delete result not found",
			output:   "delete_result: -11",
			field:    "delete_result",
			expected: -11,
		},
		{
			name:     "bytes written",
			output:   "bytes_written: 1508",
			field:    "bytes_written",
			expected: 1508,
		},
		{
			name:     "close result",
			output:   "close_result: 0",
			field:    "close_result",
			expected: 0,
		},
		{
			name:     "field not found",
			output:   "some other output",
			field:    "missing_field",
			expected: 0,
		},
		{
			name: "field in multiline output",
			output: `[1/6] Halting device...
delete_result: 0
[2/6] Next step...
bytes_written: 1234
close_result: 0`,
			field:    "bytes_written",
			expected: 1234,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResult(tt.output, tt.field)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestExtractHexResult(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		field    string
		expected int64
	}{
		{
			name:     "file handle",
			output:   "file_handle: 0x12345678",
			field:    "file_handle",
			expected: 0x12345678,
		},
		{
			name:     "uppercase hex",
			output:   "file_handle: 0xABCDEF01",
			field:    "file_handle",
			expected: 0xABCDEF01,
		},
		{
			name:     "small hex value",
			output:   "file_handle: 0x1",
			field:    "file_handle",
			expected: 0x1,
		},
		{
			name:     "field not found",
			output:   "some other output",
			field:    "missing_field",
			expected: 0,
		},
		{
			name: "field in multiline output",
			output: `[1/6] Creating file...
file_handle: 0xDEADBEEF
[2/6] Writing...`,
			field:    "file_handle",
			expected: 0xDEADBEEF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHexResult(tt.output, tt.field)
			if result != tt.expected {
				t.Errorf("expected 0x%x, got 0x%x", tt.expected, result)
			}
		})
	}
}

func TestInjectCertsScript_Parse_Success(t *testing.T) {
	// Simulate successful GDB output
	output := `[1/6] Halting device...
[2/6] Setting up filename...
[3/6] Loading certificate to memory...
[4/6] Deleting old certificate...
delete_result: 0
[5/6] Creating new certificate file...
file_handle: 0x12345678
[6/6] Writing certificate data...
bytes_written: 1508
close_result: 0
[SUCCESS]
Device operation complete`

	script := &InjectCertsScript{
		certData: make([]byte, 1508), // Same size as bytes_written
	}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected Success=true")
	}

	if result.BytesWritten != 1508 {
		t.Errorf("expected BytesWritten=1508, got %d", result.BytesWritten)
	}

	deleteResult := result.GetDataInt("delete_result")
	if deleteResult != 0 {
		t.Errorf("expected delete_result=0, got %d", deleteResult)
	}

	fileHandle := result.GetDataInt64("file_handle")
	if fileHandle != 0x12345678 {
		t.Errorf("expected file_handle=0x12345678, got 0x%x", fileHandle)
	}

	closeResult := result.GetDataInt("close_result")
	if closeResult != 0 {
		t.Errorf("expected close_result=0, got %d", closeResult)
	}

	// Check steps
	if result.TotalSteps() != 6 {
		t.Errorf("expected 6 steps, got %d", result.TotalSteps())
	}
}

func TestInjectCertsScript_Parse_BytesMismatch(t *testing.T) {
	// Bytes written doesn't match expected size
	output := `[1/6] Halting device...
bytes_written: 100
close_result: 0
Device operation complete`

	script := &InjectCertsScript{
		certData: make([]byte, 1508), // Expected 1508 bytes
	}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected Success=false for byte mismatch")
	}

	if result.Error == nil {
		t.Error("expected error to be set for byte mismatch")
	}

	if !strings.Contains(result.Error.Error(), "bytes written mismatch") {
		t.Errorf("expected 'bytes written mismatch' error, got: %v", result.Error)
	}
}

func TestInjectCertsScript_Parse_CloseFailure(t *testing.T) {
	// Close operation failed
	output := `[1/6] Halting device...
bytes_written: 1508
close_result: -1
Device operation complete`

	script := &InjectCertsScript{
		certData: make([]byte, 1508),
	}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected Success=false for close failure")
	}

	if result.Error == nil {
		t.Error("expected error to be set for close failure")
	}

	if !strings.Contains(result.Error.Error(), "file close failed") {
		t.Errorf("expected 'file close failed' error, got: %v", result.Error)
	}
}

func TestInjectCertsScript_Parse_ExplicitFailure(t *testing.T) {
	// Explicit failure marker
	output := `[1/6] Halting device...
[2/6] Setting up filename...
[FAILED]
Error: Could not open file
Device operation failed`

	script := &InjectCertsScript{
		certData: make([]byte, 1508),
	}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected Success=false for explicit failure")
	}

	if result.Error == nil {
		t.Error("expected error to be set for explicit failure")
	}
}

func TestInjectCertsScript_Parse_ErrorMessage(t *testing.T) {
	// Parse error message from output
	output := `[1/6] Halting device...
Error: Device not responding
Device operation failed`

	script := &InjectCertsScript{
		certData: make([]byte, 1508),
	}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected Success=false for error message")
	}

	if result.Error == nil {
		t.Error("expected error to be set")
	}

	if !strings.Contains(result.Error.Error(), "Device not responding") {
		t.Errorf("expected error to contain 'Device not responding', got: %v", result.Error)
	}
}

func TestInjectCertsScript_Name(t *testing.T) {
	script := &InjectCertsScript{}
	if name := script.Name(); name != "inject_certs" {
		t.Errorf("expected name 'inject_certs', got %s", name)
	}
}

func TestInjectCertsScript_Template(t *testing.T) {
	script := &InjectCertsScript{}
	template := script.Template()

	if template == "" {
		t.Error("expected non-empty template")
	}

	// Template should contain expected GDB commands
	expectedCommands := []string{
		"target",
		"monitor",
		"restore",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(template, cmd) {
			t.Errorf("expected template to contain %q", cmd)
		}
	}
}
