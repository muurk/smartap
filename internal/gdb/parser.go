package gdb

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/muurk/smartap/internal/gdb/scripts"
)

// Parser provides utilities for parsing GDB output into structured results.
// It uses compiled regex patterns for common GDB output patterns.
type Parser struct {
	// Compiled regex patterns for common GDB output formats
	stepPattern         *regexp.Regexp // Matches: [1/6] Step description...
	resultPattern       *regexp.Regexp // Matches: some_result: 123 or some_result = 0x1234
	hexPattern          *regexp.Regexp // Matches: 0x12ab34cd
	intPattern          *regexp.Regexp // Matches: integers
	successPattern      *regexp.Regexp // Matches: success, SUCCESS, OK, ok, ✓
	failurePattern      *regexp.Regexp // Matches: error, ERROR, FAIL, failed, ✗
	breakpointPattern   *regexp.Regexp // Matches: Breakpoint N at 0xADDR
	fileHandlePattern   *regexp.Regexp // Matches: $handle = 0x12345678
	bytesWrittenPattern *regexp.Regexp // Matches: bytes_written = 1234
}

// NewParser creates a new parser with compiled regex patterns.
func NewParser() *Parser {
	return &Parser{
		stepPattern:         regexp.MustCompile(`\[(\d+)/(\d+)\]\s+(.+?)(?:\.\.\.)?\s*$`),
		resultPattern:       regexp.MustCompile(`(\w+)\s*[=:]\s*(-?(?:0x)?[0-9a-fA-F]+)`),
		hexPattern:          regexp.MustCompile(`0x([0-9a-fA-F]+)`),
		intPattern:          regexp.MustCompile(`^-?\d+$`),
		successPattern:      regexp.MustCompile(`(?i)success|ok|✓|complete`),
		failurePattern:      regexp.MustCompile(`(?i)error|fail|✗|abort`),
		breakpointPattern:   regexp.MustCompile(`Breakpoint\s+(\d+)\s+at\s+(0x[0-9a-fA-F]+)`),
		fileHandlePattern:   regexp.MustCompile(`\$handle\s*=\s*(0x[0-9a-fA-F]+)`),
		bytesWrittenPattern: regexp.MustCompile(`bytes_written\s*[=:]\s*(\d+)`),
	}
}

// ParseSteps extracts step markers from GDB output.
// Looks for lines like:
//
//	[1/6] Halting device...
//	[2/6] Setting up filename...
//
// Returns a slice of Step structs with name and status.
// Status is determined by looking for success/failure keywords in subsequent lines.
func (p *Parser) ParseSteps(output string) []scripts.Step {
	lines := strings.Split(output, "\n")
	steps := make([]scripts.Step, 0)

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if matches := p.stepPattern.FindStringSubmatch(line); matches != nil {
			current := matches[1]
			total := matches[2]
			description := matches[3]

			stepName := fmt.Sprintf("[%s/%s] %s", current, total, description)

			// Look ahead a few lines to determine status
			status := "success"
			message := ""

			// Check next 3 lines for success/failure indicators
			for j := i + 1; j < i+4 && j < len(lines); j++ {
				nextLine := strings.TrimSpace(lines[j])

				// If we hit another step marker, stop looking
				if p.stepPattern.MatchString(nextLine) {
					break
				}

				// Check for failure indicators
				if p.failurePattern.MatchString(nextLine) {
					status = "failed"
					message = nextLine
					break
				}

				// Extract result values
				if matches := p.resultPattern.FindStringSubmatch(nextLine); matches != nil {
					message = nextLine
				}
			}

			steps = append(steps, scripts.Step{
				Name:    stepName,
				Status:  status,
				Message: message,
			})
		}
	}

	return steps
}

// ParseResult extracts a named result value from GDB output.
// Looks for patterns like:
//
//	result_name: 123
//	result_name = 0x1234
//	$result_name = 5678
//
// Returns the value as interface{} (could be int, int64, or string).
// Returns error if the pattern is not found.
func (p *Parser) ParseResult(output, name string) (interface{}, error) {
	// Build regex pattern for this specific name
	pattern := regexp.MustCompile(fmt.Sprintf(`%s\s*[=:]\s*(-?(?:0x)?[0-9a-fA-F]+)`, name))

	matches := pattern.FindStringSubmatch(output)
	if matches == nil {
		return nil, fmt.Errorf("result %q not found in output", name)
	}

	valueStr := matches[1]

	// Parse as hex if starts with 0x
	if strings.HasPrefix(valueStr, "0x") || strings.HasPrefix(valueStr, "0X") {
		val, err := strconv.ParseInt(valueStr[2:], 16, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse hex value %q: %w", valueStr, err)
		}
		return val, nil
	}

	// Parse as decimal integer
	val, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse int value %q: %w", valueStr, err)
	}

	return val, nil
}

// ParseHexValue extracts a hex value from output.
// Returns the value as int64.
// Example: "0x20015c64" -> 536944740
func (p *Parser) ParseHexValue(output string) (int64, error) {
	matches := p.hexPattern.FindStringSubmatch(output)
	if matches == nil {
		return 0, fmt.Errorf("no hex value found in: %s", output)
	}

	val, err := strconv.ParseInt(matches[1], 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse hex value %q: %w", matches[1], err)
	}

	return val, nil
}

// ParseIntValue extracts an integer value from output.
// Returns the value as int64.
func (p *Parser) ParseIntValue(output string) (int64, error) {
	// Find first integer in the output
	matches := regexp.MustCompile(`-?\d+`).FindString(output)
	if matches == "" {
		return 0, fmt.Errorf("no integer found in: %s", output)
	}

	val, err := strconv.ParseInt(matches, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer %q: %w", matches, err)
	}

	return val, nil
}

// DetectErrors scans GDB output for error indicators.
// Returns an error if any failure patterns are found.
func (p *Parser) DetectErrors(output string) error {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for common GDB error patterns
		if strings.Contains(line, "Cannot access memory at address") {
			return fmt.Errorf("GDB memory access error: %s", line)
		}

		if strings.Contains(line, "Connection refused") ||
			strings.Contains(line, "Connection timed out") {
			return &GDBConnectionError{
				Host: "unknown",
				Port: 0,
				Err:  fmt.Errorf("%s", line),
			}
		}

		if strings.Contains(line, "No such file or directory") {
			return fmt.Errorf("GDB file not found: %s", line)
		}

		if strings.Contains(line, "Remote communication error") {
			return fmt.Errorf("GDB communication error: %s", line)
		}
	}

	return nil
}

// HasSuccess checks if the output contains success indicators.
func (p *Parser) HasSuccess(output string) bool {
	return p.successPattern.MatchString(output)
}

// HasFailure checks if the output contains failure indicators.
func (p *Parser) HasFailure(output string) bool {
	return p.failurePattern.MatchString(output)
}

// ParseFileHandle extracts a file handle from GDB output.
// Looks for: $handle = 0x12345678
func (p *Parser) ParseFileHandle(output string) (int64, error) {
	matches := p.fileHandlePattern.FindStringSubmatch(output)
	if matches == nil {
		return 0, fmt.Errorf("no file handle found in output")
	}

	hexStr := matches[1][2:] // Remove "0x" prefix
	val, err := strconv.ParseInt(hexStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse file handle %q: %w", matches[1], err)
	}

	return val, nil
}

// ParseBytesWritten extracts bytes written count from output.
// Looks for: bytes_written: 1234 or bytes_written = 1234
func (p *Parser) ParseBytesWritten(output string) (int, error) {
	matches := p.bytesWrittenPattern.FindStringSubmatch(output)
	if matches == nil {
		return 0, fmt.Errorf("no bytes_written found in output")
	}

	val, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse bytes_written %q: %w", matches[1], err)
	}

	return val, nil
}

// ExtractLines extracts lines matching a pattern from output.
// Returns all matching lines as a slice.
func (p *Parser) ExtractLines(output, pattern string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}

	lines := strings.Split(output, "\n")
	matches := make([]string, 0)

	for _, line := range lines {
		if re.MatchString(line) {
			matches = append(matches, strings.TrimSpace(line))
		}
	}

	return matches, nil
}

// ParseKeyValue extracts key-value pairs from output.
// Looks for patterns like:
//
//	key: value
//	key = value
//	key -> value
//
// Returns a map of all found key-value pairs.
func (p *Parser) ParseKeyValue(output string) map[string]string {
	result := make(map[string]string)

	// Pattern for key-value pairs (supports :, =, ->)
	pattern := regexp.MustCompile(`(\w+)\s*(?:[:=]|->)\s*(.+)$`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := pattern.FindStringSubmatch(line); matches != nil {
			key := matches[1]
			value := strings.TrimSpace(matches[2])
			result[key] = value
		}
	}

	return result
}
