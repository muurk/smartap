package scripts

import (
	"time"
)

// Script represents a GDB operation that can be executed.
// All GDB operations (certificate injection, log capture, memory dump, etc.)
// implement this interface.
type Script interface {
	// Name returns a human-readable name for this script.
	// Used for logging and error messages.
	// Example: "inject_certs", "capture_logs", "dump_memory"
	Name() string

	// Template returns the GDB script template content.
	// The template uses Go text/template syntax and can access parameters
	// via the map returned by Params().
	// Example: "target extended-remote {{.OpenOCDHost}}:{{.OpenOCDPort}}\n..."
	Template() string

	// Params returns the parameters to be substituted into the template.
	// Keys are parameter names (e.g., "OpenOCDHost", "CertSize", "Firmware").
	// Values can be primitive types or structs with nested fields.
	// Example: map[string]interface{}{
	//     "OpenOCDHost": "localhost",
	//     "OpenOCDPort": 3333,
	//     "Firmware": firmwareStruct,
	// }
	Params() map[string]interface{}

	// Parse extracts structured results from GDB output.
	// The output parameter contains stdout from the GDB command.
	// Returns a Result with success/failure, parsed data, steps, etc.
	// Returns an error if parsing fails (use GDBParseError).
	Parse(output string) (*Result, error)

	// Streaming indicates whether this script should stream output in real-time.
	// When true, stdout/stderr are piped directly to os.Stdout/os.Stderr for
	// live monitoring. When false (default), output is buffered and returned
	// after completion.
	// Only long-running scripts like capture-logs should return true.
	Streaming() bool
}

// Result represents the outcome of executing a GDB script.
type Result struct {
	// Success indicates whether the overall operation succeeded.
	Success bool

	// Duration is how long the GDB script took to execute.
	Duration time.Duration

	// BytesWritten is the number of bytes written (for write operations).
	// Zero if not applicable.
	BytesWritten int

	// BytesRead is the number of bytes read (for read operations).
	// Zero if not applicable.
	BytesRead int

	// Steps contains progress information for multi-step operations.
	// Each step has a name, status, and optional message.
	Steps []Step

	// Data contains operation-specific parsed data.
	// For example:
	//   - "version": "0x355" (firmware detection)
	//   - "file_handle": 0x12345678 (file operations)
	//   - "delete_result": 0 (certificate injection)
	// Keys and values depend on the specific script.
	Data map[string]interface{}

	// Error contains the error if the operation failed.
	// nil if Success is true.
	Error error

	// RawOutput contains the complete stdout from GDB.
	// Useful for debugging parse errors.
	RawOutput string

	// RawStderr contains the complete stderr from GDB.
	// Useful for debugging execution errors.
	RawStderr string
}

// Step represents a single step in a multi-step GDB operation.
// Steps are extracted from echo statements in the GDB script:
//
//	echo [1/6] Halting device...\n
type Step struct {
	// Name is the step description.
	// Example: "[1/6] Halting device", "[2/6] Setting up filename"
	Name string

	// Status indicates the step outcome.
	// Values: "success", "failed", "skipped", "in_progress"
	Status string

	// Message provides additional context about the step.
	// Example: "1234 bytes written", "result: 0", "handle: 0x12345678"
	Message string
}

// NewResult creates a new Result with default values.
func NewResult() *Result {
	return &Result{
		Success: false,
		Steps:   make([]Step, 0),
		Data:    make(map[string]interface{}),
	}
}

// AddStep adds a step to the result.
func (r *Result) AddStep(name, status, message string) {
	r.Steps = append(r.Steps, Step{
		Name:    name,
		Status:  status,
		Message: message,
	})
}

// SetData sets a data value in the result.
func (r *Result) SetData(key string, value interface{}) {
	r.Data[key] = value
}

// GetData gets a data value from the result.
// Returns nil if the key doesn't exist.
func (r *Result) GetData(key string) interface{} {
	return r.Data[key]
}

// GetDataString gets a string data value from the result.
// Returns empty string if the key doesn't exist or value is not a string.
func (r *Result) GetDataString(key string) string {
	if v, ok := r.Data[key].(string); ok {
		return v
	}
	return ""
}

// GetDataInt gets an int data value from the result.
// Returns 0 if the key doesn't exist or value is not an int.
func (r *Result) GetDataInt(key string) int {
	if v, ok := r.Data[key].(int); ok {
		return v
	}
	return 0
}

// GetDataInt64 gets an int64 data value from the result.
// Returns 0 if the key doesn't exist or value is not an int64.
func (r *Result) GetDataInt64(key string) int64 {
	if v, ok := r.Data[key].(int64); ok {
		return v
	}
	return 0
}

// SuccessSteps returns the count of successful steps.
func (r *Result) SuccessSteps() int {
	count := 0
	for _, step := range r.Steps {
		if step.Status == "success" {
			count++
		}
	}
	return count
}

// FailedSteps returns the count of failed steps.
func (r *Result) FailedSteps() int {
	count := 0
	for _, step := range r.Steps {
		if step.Status == "failed" {
			count++
		}
	}
	return count
}

// TotalSteps returns the total number of steps.
func (r *Result) TotalSteps() int {
	return len(r.Steps)
}
