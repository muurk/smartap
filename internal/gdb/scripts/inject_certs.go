package scripts

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

//go:embed templates/inject_certs.gdb.tmpl
var injectCertsTemplate string

// InjectCertsScript implements certificate injection into device flash.
type InjectCertsScript struct {
	// firmware is the target firmware version info
	firmware interface{} // Will be *gdb.Firmware, but avoiding import cycle
	// certData is the certificate bytes (DER format)
	certData []byte
	// targetFile is the destination path on device (e.g., "/cert/129.der")
	targetFile string
	// openocdHost is the OpenOCD hostname
	openocdHost string
	// openocdPort is the OpenOCD port
	openocdPort int
	// certTempFile is the path to temporary cert file (set during execution)
	certTempFile string
}

// NewInjectCertsScript creates a new certificate injection script.
func NewInjectCertsScript(
	firmware interface{},
	certData []byte,
	targetFile string,
	openocdHost string,
	openocdPort int,
) *InjectCertsScript {
	return &InjectCertsScript{
		firmware:    firmware,
		certData:    certData,
		targetFile:  targetFile,
		openocdHost: openocdHost,
		openocdPort: openocdPort,
	}
}

// Name implements Script.Name
func (s *InjectCertsScript) Name() string {
	return "inject_certs"
}

// Template implements Script.Template
func (s *InjectCertsScript) Template() string {
	return injectCertsTemplate
}

// Params implements Script.Params
func (s *InjectCertsScript) Params() map[string]interface{} {
	// Convert filename to byte array for GDB script
	filenameBytes := []byte(s.targetFile)
	byteList := make([]int, len(filenameBytes))
	for i, b := range filenameBytes {
		byteList[i] = int(b)
	}

	// Validate certificate data first
	if len(s.certData) == 0 {
		return map[string]interface{}{
			"Error": "certificate data is empty (not loaded from embedded FS?)",
		}
	}

	// Create temporary file for certificate data
	// This will be used by GDB's "restore" command
	tmpFile, err := os.CreateTemp("", "smartap-cert-*.bin")
	if err != nil {
		// This will be caught during template rendering
		return map[string]interface{}{
			"Error": fmt.Sprintf("failed to create temp cert file: %v", err),
		}
	}
	defer tmpFile.Close()

	bytesWritten, err := tmpFile.Write(s.certData)
	if err != nil {
		os.Remove(tmpFile.Name())
		return map[string]interface{}{
			"Error": fmt.Sprintf("failed to write cert data: %v", err),
		}
	}

	// Verify all bytes were written
	if bytesWritten != len(s.certData) {
		os.Remove(tmpFile.Name())
		return map[string]interface{}{
			"Error": fmt.Sprintf("incomplete cert write: wrote %d of %d bytes", bytesWritten, len(s.certData)),
		}
	}

	s.certTempFile = tmpFile.Name()

	return map[string]interface{}{
		"OpenOCDHost":   s.openocdHost,
		"OpenOCDPort":   s.openocdPort,
		"CertSize":      len(s.certData),
		"TargetFile":    s.targetFile,
		"FilenameBytes": byteList,
		"CertTempFile":  s.certTempFile,
		"Firmware":      s.firmware,
	}
}

// Parse implements Script.Parse
func (s *InjectCertsScript) Parse(output string) (*Result, error) {
	result := NewResult()

	// Clean up temp file
	if s.certTempFile != "" {
		os.Remove(s.certTempFile)
	}

	// Extract step markers
	steps := parseInjectionSteps(output)
	for _, step := range steps {
		result.AddStep(step.Name, step.Status, step.Message)
	}

	// Parse results
	deleteResult := extractResult(output, "delete_result")
	fileHandle := extractHexResult(output, "file_handle")
	bytesWritten := extractResult(output, "bytes_written")
	closeResult := extractResult(output, "close_result")

	result.SetData("delete_result", deleteResult)
	result.SetData("file_handle", fileHandle)
	result.SetData("bytes_written", bytesWritten)
	result.SetData("close_result", closeResult)

	// Set bytes written in result
	if bytesWritten > 0 {
		result.BytesWritten = bytesWritten
	}

	// Check for success
	expectedSize := len(s.certData)
	if bytesWritten == expectedSize && closeResult == 0 {
		result.Success = true
	} else {
		result.Success = false
		if bytesWritten != expectedSize {
			result.Error = fmt.Errorf("bytes written mismatch: expected %d, got %d", expectedSize, bytesWritten)
		} else if closeResult != 0 {
			result.Error = fmt.Errorf("file close failed with result: %d", closeResult)
		}
	}

	// Check for explicit success/failure markers
	if strings.Contains(output, "[SUCCESS]") {
		result.Success = true
	} else if strings.Contains(output, "[FAILED]") {
		result.Success = false
		if result.Error == nil {
			result.Error = fmt.Errorf("certificate injection failed (see GDB output)")
		}
	}

	// Check for error messages
	if strings.Contains(output, "Error:") {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Error:") {
				result.Error = fmt.Errorf("%s", strings.TrimSpace(line))
				result.Success = false
				break
			}
		}
	}

	return result, nil
}

// parseInjectionSteps extracts step markers from injection output.
func parseInjectionSteps(output string) []Step {
	steps := make([]Step, 0)
	stepPattern := regexp.MustCompile(`\[(\d+)/(\d+)\]\s+(.+?)\s*(?:\.\.\.)?\s*$`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := stepPattern.FindStringSubmatch(line); matches != nil {
			current := matches[1]
			total := matches[2]
			description := matches[3]

			stepName := fmt.Sprintf("[%s/%s] %s", current, total, description)
			steps = append(steps, Step{
				Name:    stepName,
				Status:  "success",
				Message: "",
			})
		}
	}

	return steps
}

// extractResult extracts an integer result value from output.
// Looks for: result_name: 123
func extractResult(output, name string) int {
	pattern := regexp.MustCompile(fmt.Sprintf(`%s:\s*(-?\d+)`, name))
	matches := pattern.FindStringSubmatch(output)
	if matches == nil {
		return 0
	}

	val, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}

	return val
}

// extractHexResult extracts a hex result value from output.
// Looks for: result_name: 0x12345678
func extractHexResult(output, name string) int64 {
	pattern := regexp.MustCompile(fmt.Sprintf(`%s:\s*(0x[0-9a-fA-F]+)`, name))
	matches := pattern.FindStringSubmatch(output)
	if matches == nil {
		return 0
	}

	val, err := strconv.ParseInt(matches[1][2:], 16, 64)
	if err != nil {
		return 0
	}

	return val
}

// Streaming implements Script.Streaming
// Certificate injection doesn't need streaming - it completes in a few seconds
func (s *InjectCertsScript) Streaming() bool {
	return false
}
