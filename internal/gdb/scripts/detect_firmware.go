package scripts

import (
	_ "embed"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

//go:embed templates/detect_firmware.gdb.tmpl
var detectFirmwareTemplate string

// DetectFirmwareScript implements firmware version detection using signature matching.
type DetectFirmwareScript struct {
	openocdHost string
	openocdPort int
	firmwares   interface{} // []Firmware passed from caller to avoid import cycle
}

// NewDetectFirmwareScript creates a new firmware detection script.
// The firmwares parameter should be the result of gdb.LoadFirmwares().List()
func NewDetectFirmwareScript(openocdHost string, openocdPort int, firmwares interface{}) *DetectFirmwareScript {
	return &DetectFirmwareScript{
		openocdHost: openocdHost,
		openocdPort: openocdPort,
		firmwares:   firmwares,
	}
}

// Name implements Script.Name
func (s *DetectFirmwareScript) Name() string {
	return "detect_firmware"
}

// Template implements Script.Template
func (s *DetectFirmwareScript) Template() string {
	return detectFirmwareTemplate
}

// Params implements Script.Params
func (s *DetectFirmwareScript) Params() map[string]interface{} {
	// Use the firmwares passed to constructor
	firmwares := s.firmwares
	if firmwares == nil {
		firmwares = []interface{}{} // Empty slice if none provided
	}

	return map[string]interface{}{
		"OpenOCDHost": s.openocdHost,
		"OpenOCDPort": s.openocdPort,
		"Firmwares":   firmwares,
	}
}

// Parse implements Script.Parse
func (s *DetectFirmwareScript) Parse(output string) (*Result, error) {
	result := NewResult()

	// Parse machine-readable output section
	// Format:
	//   DETECTED_VERSION=0x355
	//   CONFIDENCE=100
	//   MATCHES=7
	//   STATUS=OK|UNKNOWN

	// Extract version
	versionPattern := regexp.MustCompile(`DETECTED_VERSION=([^\s\n]+)`)
	versionMatches := versionPattern.FindStringSubmatch(output)
	var version string
	if versionMatches != nil {
		version = versionMatches[1]
		// Convert "0" to empty string for unknown firmware
		if version == "0" {
			version = "UNKNOWN"
		}
	} else {
		version = "UNKNOWN"
	}

	// Extract confidence
	confidencePattern := regexp.MustCompile(`CONFIDENCE=(\d+)`)
	confidenceMatches := confidencePattern.FindStringSubmatch(output)
	var confidence int
	if confidenceMatches != nil {
		confidence, _ = strconv.Atoi(confidenceMatches[1])
	}

	// Extract matches
	matchesPattern := regexp.MustCompile(`MATCHES=(\d+)`)
	matchesMatches := matchesPattern.FindStringSubmatch(output)
	var matches int
	if matchesMatches != nil {
		matches, _ = strconv.Atoi(matchesMatches[1])
	}

	// Extract status
	statusPattern := regexp.MustCompile(`STATUS=([^\s\n]+)`)
	statusMatches := statusPattern.FindStringSubmatch(output)
	var status string
	if statusMatches != nil {
		status = strings.ToLower(statusMatches[1])
	} else {
		status = "unknown"
	}

	// Calculate total signatures from firmwares (assume 7 for version 0x355)
	// This could be extracted from output if needed
	total := 7
	if matches > 0 && confidence > 0 {
		// Back-calculate total from matches and confidence
		// confidence = (matches * 100) / total
		// total = (matches * 100) / confidence
		total = (matches * 100) / confidence
	}

	// Store results in data map
	result.SetData("version", version)
	result.SetData("confidence", confidence)
	result.SetData("matches", matches)
	result.SetData("total", total)
	result.SetData("status", status)

	// Add steps based on confidence
	if confidence == 100 {
		result.AddStep("[1/1] Firmware detection", "success", fmt.Sprintf("Version: %s (%d%% confidence)", version, confidence))
		result.Success = true
	} else if confidence >= 85 {
		result.AddStep("[1/1] Firmware detection", "success", fmt.Sprintf("Version: %s (%d%% confidence - medium)", version, confidence))
		result.Success = false // Not 100%, so operations should be blocked
		result.Error = fmt.Errorf("firmware confidence too low: %d%% (need 100%%)", confidence)
	} else if confidence > 0 {
		result.AddStep("[1/1] Firmware detection", "failed", fmt.Sprintf("Low confidence: %d%% (best match: %s)", confidence, version))
		result.Success = false
		result.Error = fmt.Errorf("firmware confidence too low: %d%% (need 100%%)", confidence)
	} else {
		result.AddStep("[1/1] Firmware detection", "failed", "Unknown firmware - no signatures matched")
		result.Success = false
		result.Error = fmt.Errorf("firmware unknown - no known signatures matched")
	}

	return result, nil
}

// Streaming implements Script.Streaming
// Firmware detection doesn't need streaming - it's a quick operation
func (s *DetectFirmwareScript) Streaming() bool {
	return false
}
