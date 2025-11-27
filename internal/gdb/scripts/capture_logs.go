package scripts

import (
	"bufio"
	_ "embed"
	"fmt"
	"regexp"
	"strings"
	"time"
)

//go:embed templates/capture_logs.gdb.tmpl
var captureLogsTemplate string

// CaptureLogsScript implements log capture from device UART
type CaptureLogsScript struct {
	openocdHost string
	openocdPort int
	firmware    interface{} // *gdb.Firmware (avoid import cycle)
	onLog       func(timestamp time.Time, message string)
}

// NewCaptureLogsScript creates a new log capture script
func NewCaptureLogsScript(openocdHost string, openocdPort int, firmware interface{}) *CaptureLogsScript {
	return &CaptureLogsScript{
		openocdHost: openocdHost,
		openocdPort: openocdPort,
		firmware:    firmware,
	}
}

// SetLogCallback sets the callback function for log messages
func (s *CaptureLogsScript) SetLogCallback(callback func(timestamp time.Time, message string)) {
	s.onLog = callback
}

// Name returns the script name
func (s *CaptureLogsScript) Name() string {
	return "capture_logs"
}

// Template returns the embedded GDB script template
func (s *CaptureLogsScript) Template() string {
	return captureLogsTemplate
}

// Params returns the template parameters
func (s *CaptureLogsScript) Params() map[string]interface{} {
	return map[string]interface{}{
		"OpenOCDHost": s.openocdHost,
		"OpenOCDPort": s.openocdPort,
		"Firmware":    s.firmware,
	}
}

// Parse parses the GDB output and extracts log messages
func (s *CaptureLogsScript) Parse(output string) (*Result, error) {
	result := NewResult()
	result.AddStep("[1/3] Setting up log capture", "success", "")
	result.AddStep("[2/3] Log capture configured", "success", "")
	result.AddStep("[3/3] Resuming device", "success", "")

	// Parse log messages from output
	logs := s.parseLogMessages(output)

	result.SetData("log_count", len(logs))
	result.SetData("logs", logs)

	// Call callback for each log message
	if s.onLog != nil {
		for _, log := range logs {
			s.onLog(log.Timestamp, log.Message)
		}
	}

	// Log capture is considered successful if we set up the breakpoint
	// Even if no logs were captured (device might not be logging yet)
	if strings.Contains(output, "Breakpoint") || strings.Contains(output, "[2/3]") {
		result.Success = true
	} else {
		result.Success = false
		result.Error = fmt.Errorf("failed to set up log capture: breakpoint not set")
	}

	return result, nil
}

// LogEntry represents a captured log message
type LogEntry struct {
	Timestamp time.Time
	Message   string
}

// parseLogMessages extracts log messages from GDB output
func (s *CaptureLogsScript) parseLogMessages(output string) []LogEntry {
	var logs []LogEntry

	// Pattern: [LOG] <number>: <address> <string>
	// Example: [LOG] 1: 0x20001234:	"WiFi connecting..."
	logPattern := regexp.MustCompile(`\[LOG\]\s+\d+:\s+0x[0-9a-fA-F]+:\s+"([^"]+)"`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	startTime := time.Now()

	for scanner.Scan() {
		line := scanner.Text()

		// Match log pattern
		if matches := logPattern.FindStringSubmatch(line); matches != nil {
			message := matches[1]

			// Create log entry with timestamp relative to capture start
			logs = append(logs, LogEntry{
				Timestamp: startTime.Add(time.Duration(len(logs)) * time.Millisecond),
				Message:   message,
			})
		}
	}

	return logs
}

// Streaming implements Script.Streaming
// Log capture NEEDS streaming - it runs indefinitely until interrupted
func (s *CaptureLogsScript) Streaming() bool {
	return true
}
