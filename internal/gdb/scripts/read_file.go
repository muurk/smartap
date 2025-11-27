package scripts

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed templates/read_file.gdb.tmpl
var readFileTemplate string

// ReadFileScript implements file reading from device filesystem
type ReadFileScript struct {
	openocdHost string
	openocdPort int
	firmware    interface{} // *gdb.Firmware (avoid import cycle)
	remoteFile  string
	outputFile  string
	maxSize     int
}

// NewReadFileScript creates a new file reading script
func NewReadFileScript(openocdHost string, openocdPort int, firmware interface{}, remoteFile string, outputFile string, maxSize int) *ReadFileScript {
	return &ReadFileScript{
		openocdHost: openocdHost,
		openocdPort: openocdPort,
		firmware:    firmware,
		remoteFile:  remoteFile,
		outputFile:  outputFile,
		maxSize:     maxSize,
	}
}

// Name returns the script name
func (s *ReadFileScript) Name() string {
	return "read_file"
}

// Template returns the embedded GDB script template
func (s *ReadFileScript) Template() string {
	return readFileTemplate
}

// Params returns the template parameters
func (s *ReadFileScript) Params() map[string]interface{} {
	// Convert filename to byte array for GDB script
	filenameBytes := make([]int, len(s.remoteFile))
	for i, c := range s.remoteFile {
		filenameBytes[i] = int(c)
	}

	return map[string]interface{}{
		"OpenOCDHost":   s.openocdHost,
		"OpenOCDPort":   s.openocdPort,
		"Firmware":      s.firmware,
		"FilenameBytes": filenameBytes,
		"OutputFile":    s.outputFile,
		"MaxSize":       s.maxSize,
	}
}

// Parse parses the GDB output
func (s *ReadFileScript) Parse(output string) (*Result, error) {
	result := NewResult()
	result.AddStep("[1/6] Halting device", "success", "")
	result.AddStep("[2/6] Setting up filename", "success", "")
	result.AddStep("[3/6] Opening file for reading", "success", "")
	result.AddStep("[4/6] Reading file data", "success", "")
	result.AddStep("[5/6] Saving file data", "success", "")
	result.AddStep("[6/6] Closing file", "success", "")

	// Extract results from output
	fileHandle := extractHexResult(output, "file_handle")
	bytesRead := extractResult(output, "bytes_read")
	closeResult := extractResult(output, "close_result")

	result.SetData("file_handle", fileHandle)
	result.SetData("bytes_read", bytesRead)
	result.SetData("close_result", closeResult)
	result.SetData("remote_file", s.remoteFile)
	result.SetData("output_file", s.outputFile)

	// Check for success
	if strings.Contains(output, "[SUCCESS]") && bytesRead > 0 && closeResult == 0 {
		result.Success = true
		result.BytesRead = bytesRead
	} else {
		result.Success = false

		// Determine specific error
		if fileHandle == 0 || fileHandle < 0 {
			result.Error = fmt.Errorf("failed to open file %s: file may not exist (handle: 0x%x)", s.remoteFile, fileHandle)
		} else if bytesRead <= 0 {
			result.Error = fmt.Errorf("failed to read file: read returned %d bytes", bytesRead)
		} else if closeResult != 0 {
			result.Error = fmt.Errorf("file close failed: result=%d", closeResult)
		} else {
			result.Error = fmt.Errorf("file read operation failed")
		}
	}

	return result, nil
}

// Streaming implements Script.Streaming
// File reads don't need streaming - they complete quickly
func (s *ReadFileScript) Streaming() bool {
	return false
}
