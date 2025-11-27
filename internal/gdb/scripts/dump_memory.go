package scripts

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed templates/dump_memory.gdb.tmpl
var dumpMemoryTemplate string

// DumpMemoryScript implements memory dumping from device
type DumpMemoryScript struct {
	openocdHost  string
	openocdPort  int
	startAddress int64
	size         int
	outputFile   string
}

// NewDumpMemoryScript creates a new memory dump script
func NewDumpMemoryScript(openocdHost string, openocdPort int, startAddress int64, size int, outputFile string) *DumpMemoryScript {
	return &DumpMemoryScript{
		openocdHost:  openocdHost,
		openocdPort:  openocdPort,
		startAddress: startAddress,
		size:         size,
		outputFile:   outputFile,
	}
}

// Name returns the script name
func (s *DumpMemoryScript) Name() string {
	return "dump_memory"
}

// Template returns the embedded GDB script template
func (s *DumpMemoryScript) Template() string {
	return dumpMemoryTemplate
}

// Params returns the template parameters
func (s *DumpMemoryScript) Params() map[string]interface{} {
	endAddress := s.startAddress + int64(s.size)

	return map[string]interface{}{
		"OpenOCDHost":  s.openocdHost,
		"OpenOCDPort":  s.openocdPort,
		"StartAddress": s.startAddress,
		"EndAddress":   endAddress,
		"Size":         s.size,
		"OutputFile":   s.outputFile,
	}
}

// Parse parses the GDB output
func (s *DumpMemoryScript) Parse(output string) (*Result, error) {
	result := NewResult()
	result.AddStep("[1/3] Halting device", "success", "")
	result.AddStep(fmt.Sprintf("[2/3] Dumping memory from 0x%x (%d bytes)", s.startAddress, s.size), "success", "")
	result.AddStep("[3/3] Memory dump complete", "success", "")

	// Check for success marker
	if strings.Contains(output, "[SUCCESS]") {
		result.Success = true
		result.SetData("start_address", s.startAddress)
		result.SetData("size", s.size)
		result.SetData("output_file", s.outputFile)
	} else {
		result.Success = false
		result.Error = fmt.Errorf("memory dump failed: success marker not found")

		// Check for common errors
		if strings.Contains(output, "Cannot access memory") {
			result.Error = fmt.Errorf("cannot access memory at 0x%x: address may be invalid or not accessible", s.startAddress)
		} else if strings.Contains(output, "Error") || strings.Contains(output, "error") {
			// Extract error message
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "error") {
					result.Error = fmt.Errorf("memory dump failed: %s", strings.TrimSpace(line))
					break
				}
			}
		}
	}

	return result, nil
}

// Streaming implements Script.Streaming
// Memory dumps don't need streaming - they complete quickly
func (s *DumpMemoryScript) Streaming() bool {
	return false
}
