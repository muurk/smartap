package scripts

import (
	"testing"
	"time"
)

func TestNewResult(t *testing.T) {
	result := NewResult()

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Success != false {
		t.Error("expected Success to be false")
	}

	if result.Steps == nil {
		t.Error("expected Steps to be initialized")
	}

	if result.Data == nil {
		t.Error("expected Data to be initialized")
	}

	if len(result.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(result.Steps))
	}

	if len(result.Data) != 0 {
		t.Errorf("expected 0 data entries, got %d", len(result.Data))
	}
}

func TestResult_AddStep(t *testing.T) {
	result := NewResult()

	result.AddStep("[1/2] First step", "success", "completed")
	result.AddStep("[2/2] Second step", "failed", "error occurred")

	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(result.Steps))
	}

	step1 := result.Steps[0]
	if step1.Name != "[1/2] First step" {
		t.Errorf("expected Name '[1/2] First step', got %s", step1.Name)
	}
	if step1.Status != "success" {
		t.Errorf("expected Status 'success', got %s", step1.Status)
	}
	if step1.Message != "completed" {
		t.Errorf("expected Message 'completed', got %s", step1.Message)
	}

	step2 := result.Steps[1]
	if step2.Name != "[2/2] Second step" {
		t.Errorf("expected Name '[2/2] Second step', got %s", step2.Name)
	}
	if step2.Status != "failed" {
		t.Errorf("expected Status 'failed', got %s", step2.Status)
	}
	if step2.Message != "error occurred" {
		t.Errorf("expected Message 'error occurred', got %s", step2.Message)
	}
}

func TestResult_SetDataAndGetData(t *testing.T) {
	result := NewResult()

	result.SetData("string_key", "string_value")
	result.SetData("int_key", 42)
	result.SetData("int64_key", int64(123456))
	result.SetData("bool_key", true)

	// Test GetData
	if v := result.GetData("string_key"); v != "string_value" {
		t.Errorf("expected 'string_value', got %v", v)
	}

	if v := result.GetData("nonexistent"); v != nil {
		t.Errorf("expected nil for nonexistent key, got %v", v)
	}
}

func TestResult_GetDataString(t *testing.T) {
	result := NewResult()
	result.SetData("string_key", "value")
	result.SetData("int_key", 42)

	// Valid string
	if v := result.GetDataString("string_key"); v != "value" {
		t.Errorf("expected 'value', got %s", v)
	}

	// Non-string value
	if v := result.GetDataString("int_key"); v != "" {
		t.Errorf("expected empty string for non-string value, got %s", v)
	}

	// Nonexistent key
	if v := result.GetDataString("nonexistent"); v != "" {
		t.Errorf("expected empty string for nonexistent key, got %s", v)
	}
}

func TestResult_GetDataInt(t *testing.T) {
	result := NewResult()
	result.SetData("int_key", 42)
	result.SetData("string_key", "not an int")

	// Valid int
	if v := result.GetDataInt("int_key"); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}

	// Non-int value
	if v := result.GetDataInt("string_key"); v != 0 {
		t.Errorf("expected 0 for non-int value, got %d", v)
	}

	// Nonexistent key
	if v := result.GetDataInt("nonexistent"); v != 0 {
		t.Errorf("expected 0 for nonexistent key, got %d", v)
	}
}

func TestResult_GetDataInt64(t *testing.T) {
	result := NewResult()
	result.SetData("int64_key", int64(123456789))
	result.SetData("string_key", "not an int")

	// Valid int64
	if v := result.GetDataInt64("int64_key"); v != int64(123456789) {
		t.Errorf("expected 123456789, got %d", v)
	}

	// Non-int64 value
	if v := result.GetDataInt64("string_key"); v != 0 {
		t.Errorf("expected 0 for non-int64 value, got %d", v)
	}

	// Nonexistent key
	if v := result.GetDataInt64("nonexistent"); v != 0 {
		t.Errorf("expected 0 for nonexistent key, got %d", v)
	}
}

func TestResult_StepCounts(t *testing.T) {
	tests := []struct {
		name            string
		steps           []Step
		expectedTotal   int
		expectedSuccess int
		expectedFailed  int
	}{
		{
			name:            "no steps",
			steps:           []Step{},
			expectedTotal:   0,
			expectedSuccess: 0,
			expectedFailed:  0,
		},
		{
			name: "all success",
			steps: []Step{
				{Name: "Step 1", Status: "success"},
				{Name: "Step 2", Status: "success"},
				{Name: "Step 3", Status: "success"},
			},
			expectedTotal:   3,
			expectedSuccess: 3,
			expectedFailed:  0,
		},
		{
			name: "all failed",
			steps: []Step{
				{Name: "Step 1", Status: "failed"},
				{Name: "Step 2", Status: "failed"},
			},
			expectedTotal:   2,
			expectedSuccess: 0,
			expectedFailed:  2,
		},
		{
			name: "mixed",
			steps: []Step{
				{Name: "Step 1", Status: "success"},
				{Name: "Step 2", Status: "failed"},
				{Name: "Step 3", Status: "success"},
				{Name: "Step 4", Status: "skipped"},
			},
			expectedTotal:   4,
			expectedSuccess: 2,
			expectedFailed:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewResult()
			for _, step := range tt.steps {
				result.AddStep(step.Name, step.Status, step.Message)
			}

			if v := result.TotalSteps(); v != tt.expectedTotal {
				t.Errorf("expected %d total steps, got %d", tt.expectedTotal, v)
			}

			if v := result.SuccessSteps(); v != tt.expectedSuccess {
				t.Errorf("expected %d success steps, got %d", tt.expectedSuccess, v)
			}

			if v := result.FailedSteps(); v != tt.expectedFailed {
				t.Errorf("expected %d failed steps, got %d", tt.expectedFailed, v)
			}
		})
	}
}

func TestResult_Metadata(t *testing.T) {
	result := NewResult()

	// Set metadata
	result.Success = true
	result.Duration = 5 * time.Second
	result.BytesWritten = 1234
	result.BytesRead = 5678
	result.RawOutput = "GDB output"
	result.RawStderr = "GDB stderr"

	// Verify metadata
	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.Duration != 5*time.Second {
		t.Errorf("expected Duration 5s, got %s", result.Duration)
	}

	if result.BytesWritten != 1234 {
		t.Errorf("expected BytesWritten 1234, got %d", result.BytesWritten)
	}

	if result.BytesRead != 5678 {
		t.Errorf("expected BytesRead 5678, got %d", result.BytesRead)
	}

	if result.RawOutput != "GDB output" {
		t.Errorf("expected RawOutput 'GDB output', got %s", result.RawOutput)
	}

	if result.RawStderr != "GDB stderr" {
		t.Errorf("expected RawStderr 'GDB stderr', got %s", result.RawStderr)
	}
}
