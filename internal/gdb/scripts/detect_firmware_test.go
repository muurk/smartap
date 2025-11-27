package scripts

import (
	"fmt"
	"strings"
	"testing"
)

func TestDetectFirmwareScript_Parse_100PercentConfidence(t *testing.T) {
	// Simulate successful detection with 100% confidence (all signatures matched)
	output := `Checking firmware 0x355 (CC3200 ServicePack 1.32.0)
  sl_FsOpen    @ 0x20015c64: MATCH
  sl_FsRead    @ 0x20014b54: MATCH
  sl_FsWrite   @ 0x20014bf8: MATCH
  sl_FsClose   @ 0x2001555c: MATCH
  sl_FsDel     @ 0x20016ea8: MATCH
  sl_FsGetInfo @ 0x2001590c: MATCH
  uart_log     @ 0x20014f14: MATCH
  Confidence: 100% (7/7)

PERFECT MATCH FOUND - Stopping checks

DETECTED_VERSION=0x355
CONFIDENCE=100
MATCHES=7
STATUS=OK`

	script := &DetectFirmwareScript{}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected Success=true for 100% confidence")
	}

	// Check version
	version := result.GetDataString("version")
	if version != "0x355" {
		t.Errorf("expected version '0x355', got %s", version)
	}

	// Check confidence
	confidence := result.GetDataInt("confidence")
	if confidence != 100 {
		t.Errorf("expected confidence 100, got %d", confidence)
	}

	// Check matches
	matches := result.GetDataInt("matches")
	if matches != 7 {
		t.Errorf("expected 7 matches, got %d", matches)
	}

	// Check total
	total := result.GetDataInt("total")
	if total != 7 {
		t.Errorf("expected 7 total, got %d", total)
	}

	// Check status
	status := result.GetDataString("status")
	if status != "ok" {
		t.Errorf("expected status 'ok', got %s", status)
	}

	// Check step
	if result.TotalSteps() != 1 {
		t.Errorf("expected 1 step, got %d", result.TotalSteps())
	}

	if result.Steps[0].Status != "success" {
		t.Errorf("expected step status 'success', got %s", result.Steps[0].Status)
	}

	// Should contain confidence in message
	if !strings.Contains(result.Steps[0].Message, "100%") {
		t.Errorf("expected step message to contain confidence, got: %s", result.Steps[0].Message)
	}
}

func TestDetectFirmwareScript_Parse_PartialConfidence(t *testing.T) {
	// Simulate partial match with 57% confidence (4/7 signatures)
	output := `Checking firmware 0x355 (CC3200 ServicePack 1.32.0)
  sl_FsOpen    @ 0x20015c64: MATCH
  sl_FsRead    @ 0x20014b54: MATCH
  sl_FsWrite   @ 0x20014bf8: MISMATCH
  sl_FsClose   @ 0x2001555c: MATCH
  sl_FsDel     @ 0x20016ea8: MISMATCH
  sl_FsGetInfo @ 0x2001590c: MISMATCH
  uart_log     @ 0x20014f14: MATCH
  Confidence: 57% (4/7)

DETECTED_VERSION=0x355
CONFIDENCE=57
MATCHES=4
STATUS=PARTIAL`

	script := &DetectFirmwareScript{}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fail with confidence < 100%
	if result.Success {
		t.Error("expected Success=false for partial confidence")
	}

	if result.Error == nil {
		t.Error("expected error to be set for partial confidence")
	}

	if !strings.Contains(result.Error.Error(), "confidence too low") {
		t.Errorf("expected confidence error, got: %v", result.Error)
	}

	// Check parsed data
	version := result.GetDataString("version")
	if version != "0x355" {
		t.Errorf("expected version '0x355', got %s", version)
	}

	confidence := result.GetDataInt("confidence")
	if confidence != 57 {
		t.Errorf("expected confidence 57, got %d", confidence)
	}

	matches := result.GetDataInt("matches")
	if matches != 4 {
		t.Errorf("expected 4 matches, got %d", matches)
	}

	total := result.GetDataInt("total")
	if total != 7 {
		t.Errorf("expected 7 total, got %d", total)
	}

	// Check step reports failure
	if result.TotalSteps() != 1 {
		t.Errorf("expected 1 step, got %d", result.TotalSteps())
	}

	if result.Steps[0].Status != "failed" {
		t.Errorf("expected step status 'failed', got %s", result.Steps[0].Status)
	}
}

func TestDetectFirmwareScript_Parse_LowConfidence(t *testing.T) {
	// Test with very low confidence (14% - 1/7)
	output := `Checking firmware 0x355 (CC3200 ServicePack 1.32.0)
  sl_FsOpen    @ 0x20015c64: MATCH
  sl_FsRead    @ 0x20014b54: MISMATCH
  sl_FsWrite   @ 0x20014bf8: MISMATCH
  sl_FsClose   @ 0x2001555c: MISMATCH
  sl_FsDel     @ 0x20016ea8: MISMATCH
  sl_FsGetInfo @ 0x2001590c: MISMATCH
  uart_log     @ 0x20014f14: MISMATCH
  Confidence: 14% (1/7)

DETECTED_VERSION=0x355
CONFIDENCE=14
MATCHES=1
STATUS=LOW`

	script := &DetectFirmwareScript{}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected Success=false for low confidence")
	}

	confidence := result.GetDataInt("confidence")
	if confidence != 14 {
		t.Errorf("expected confidence 14, got %d", confidence)
	}

	matches := result.GetDataInt("matches")
	if matches != 1 {
		t.Errorf("expected 1 match, got %d", matches)
	}

	// Step should be failed with low confidence message
	if result.Steps[0].Status != "failed" {
		t.Errorf("expected step status 'failed', got %s", result.Steps[0].Status)
	}

	if !strings.Contains(result.Steps[0].Message, "confidence") {
		t.Errorf("expected step message to mention confidence, got: %s", result.Steps[0].Message)
	}
}

func TestDetectFirmwareScript_Parse_UnknownFirmware(t *testing.T) {
	// Test with 0% confidence (no matches)
	output := `Checking firmware 0x355 (CC3200 ServicePack 1.32.0)
  sl_FsOpen    @ 0x20015c64: MISMATCH
  sl_FsRead    @ 0x20014b54: MISMATCH
  sl_FsWrite   @ 0x20014bf8: MISMATCH
  sl_FsClose   @ 0x2001555c: MISMATCH
  sl_FsDel     @ 0x20016ea8: MISMATCH
  sl_FsGetInfo @ 0x2001590c: MISMATCH
  uart_log     @ 0x20014f14: MISMATCH
  Confidence: 0% (0/7)

DETECTED_VERSION=UNKNOWN
CONFIDENCE=0
MATCHES=0
STATUS=UNKNOWN`

	script := &DetectFirmwareScript{}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected Success=false for unknown firmware")
	}

	if result.Error == nil {
		t.Error("expected error to be set for unknown firmware")
	}

	// Check parsed data
	version := result.GetDataString("version")
	if version != "UNKNOWN" {
		t.Errorf("expected version 'UNKNOWN', got %s", version)
	}

	confidence := result.GetDataInt("confidence")
	if confidence != 0 {
		t.Errorf("expected confidence 0, got %d", confidence)
	}

	status := result.GetDataString("status")
	if status != "unknown" {
		t.Errorf("expected status 'unknown', got %s", status)
	}

	// Step should indicate unknown firmware
	if result.Steps[0].Status != "failed" {
		t.Errorf("expected step status 'failed', got %s", result.Steps[0].Status)
	}

	if !strings.Contains(result.Steps[0].Message, "Unknown") || !strings.Contains(result.Steps[0].Message, "firmware") {
		t.Errorf("expected step message to mention unknown firmware, got: %s", result.Steps[0].Message)
	}
}

func TestDetectFirmwareScript_Parse_MediumConfidence(t *testing.T) {
	// Test with medium confidence (85% - 6/7)
	output := `Checking firmware 0x355 (CC3200 ServicePack 1.32.0)
  sl_FsOpen    @ 0x20015c64: MATCH
  sl_FsRead    @ 0x20014b54: MATCH
  sl_FsWrite   @ 0x20014bf8: MATCH
  sl_FsClose   @ 0x2001555c: MATCH
  sl_FsDel     @ 0x20016ea8: MATCH
  sl_FsGetInfo @ 0x2001590c: MATCH
  uart_log     @ 0x20014f14: MISMATCH
  Confidence: 85% (6/7)

DETECTED_VERSION=0x355
CONFIDENCE=85
MATCHES=6
STATUS=MEDIUM`

	script := &DetectFirmwareScript{}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Medium confidence still fails (< 100%)
	if result.Success {
		t.Error("expected Success=false for medium confidence")
	}

	if result.Error == nil {
		t.Error("expected error to be set for medium confidence")
	}

	confidence := result.GetDataInt("confidence")
	if confidence != 85 {
		t.Errorf("expected confidence 85, got %d", confidence)
	}

	// Step should indicate medium confidence but still blocked
	if result.Steps[0].Status != "success" {
		t.Errorf("expected step status 'success', got %s", result.Steps[0].Status)
	}

	if !strings.Contains(result.Steps[0].Message, "medium") {
		t.Errorf("expected step message to mention medium confidence, got: %s", result.Steps[0].Message)
	}
}

func TestDetectFirmwareScript_Parse_MalformedOutput(t *testing.T) {
	// Test with malformed output (missing fields)
	output := `Some random output
No detection markers found`

	script := &DetectFirmwareScript{}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should handle gracefully with default values
	version := result.GetDataString("version")
	if version != "UNKNOWN" {
		t.Errorf("expected version 'UNKNOWN' for malformed output, got %s", version)
	}

	confidence := result.GetDataInt("confidence")
	if confidence != 0 {
		t.Errorf("expected confidence 0 for malformed output, got %d", confidence)
	}
}

func TestDetectFirmwareScript_Parse_VersionZero(t *testing.T) {
	// Test that version "0" is converted to "UNKNOWN"
	output := `DETECTED_VERSION=0
CONFIDENCE=0
MATCHES=0
STATUS=UNKNOWN`

	script := &DetectFirmwareScript{}

	result, err := script.Parse(output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	version := result.GetDataString("version")
	if version != "UNKNOWN" {
		t.Errorf("expected version '0' to be converted to 'UNKNOWN', got %s", version)
	}
}

func TestDetectFirmwareScript_Parse_ConfidenceCalculation(t *testing.T) {
	// Test that total is correctly back-calculated from confidence and matches
	// Note: The parser defaults to total=7 for the known firmware 0x355,
	// as that's the number of signatures checked
	testCases := []struct {
		name       string
		confidence int
		matches    int
		wantTotal  int // All should be 7 for firmware 0x355
	}{
		{"100% confidence", 100, 7, 7}, // 7 * 100 / 100 = 7
		{"85% confidence", 85, 6, 7},   // 6 * 100 / 85 ≈ 7
		{"57% confidence", 57, 4, 7},   // 4 * 100 / 57 ≈ 7
		{"14% confidence", 14, 1, 7},   // 1 * 100 / 14 ≈ 7
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := "DETECTED_VERSION=0x355\n"
			output += fmt.Sprintf("CONFIDENCE=%d\n", tc.confidence)
			output += fmt.Sprintf("MATCHES=%d\n", tc.matches)
			output += "STATUS=OK\n"

			script := &DetectFirmwareScript{}
			result, err := script.Parse(output)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}

			total := result.GetDataInt("total")
			if total != tc.wantTotal {
				t.Errorf("expected total %d, got %d", tc.wantTotal, total)
			}
		})
	}
}

func TestDetectFirmwareScript_Name(t *testing.T) {
	script := &DetectFirmwareScript{}
	if name := script.Name(); name != "detect_firmware" {
		t.Errorf("expected name 'detect_firmware', got %s", name)
	}
}

func TestDetectFirmwareScript_Template(t *testing.T) {
	script := &DetectFirmwareScript{}
	template := script.Template()

	if template == "" {
		t.Error("expected non-empty template")
	}

	// Template should contain expected GDB commands
	expectedCommands := []string{
		"target",
		"monitor",
		"DETECTED_VERSION",
		"CONFIDENCE",
		"MATCHES",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(template, cmd) {
			t.Errorf("expected template to contain %q", cmd)
		}
	}
}

func TestDetectFirmwareScript_Params(t *testing.T) {
	// Create mock firmwares list
	mockFirmwares := []interface{}{
		map[string]interface{}{
			"version": "0x355",
			"name":    "Test Firmware",
		},
	}

	script := NewDetectFirmwareScript("192.168.1.1", 4444, mockFirmwares)

	params := script.Params()
	if params == nil {
		t.Fatal("expected non-nil params")
	}

	if host := params["OpenOCDHost"]; host != "192.168.1.1" {
		t.Errorf("expected OpenOCDHost '192.168.1.1', got %v", host)
	}

	if port := params["OpenOCDPort"]; port != 4444 {
		t.Errorf("expected OpenOCDPort 4444, got %v", port)
	}

	// Check firmwares are passed through
	firmwares := params["Firmwares"]
	if firmwares == nil {
		t.Error("expected Firmwares to be set")
	}
}

func TestNewDetectFirmwareScript(t *testing.T) {
	mockFirmwares := []interface{}{
		map[string]interface{}{
			"version": "0x355",
		},
	}

	script := NewDetectFirmwareScript("localhost", 3333, mockFirmwares)

	if script == nil {
		t.Fatal("expected non-nil script")
	}

	if script.openocdHost != "localhost" {
		t.Errorf("expected openocdHost 'localhost', got %s", script.openocdHost)
	}

	if script.openocdPort != 3333 {
		t.Errorf("expected openocdPort 3333, got %d", script.openocdPort)
	}

	if script.firmwares == nil {
		t.Error("expected firmwares to be set")
	}
}

func TestNewDetectFirmwareScript_NilFirmwares(t *testing.T) {
	// Test with nil firmwares
	script := NewDetectFirmwareScript("localhost", 3333, nil)

	if script == nil {
		t.Fatal("expected non-nil script")
	}

	// Params should handle nil firmwares gracefully
	params := script.Params()
	firmwares := params["Firmwares"]
	if firmwares == nil {
		t.Error("expected Firmwares to be set to empty slice")
	}
}

// Test integration: full workflow
func TestDetectFirmwareScript_Integration(t *testing.T) {
	mockFirmwares := []interface{}{
		map[string]interface{}{
			"version": "0x355",
			"name":    "Smartap 0x355",
		},
	}

	script := NewDetectFirmwareScript("localhost", 3333, mockFirmwares)

	// Test basic properties
	if script.Name() != "detect_firmware" {
		t.Error("unexpected script name")
	}

	// Test params
	params := script.Params()
	if params["OpenOCDHost"] != "localhost" {
		t.Error("unexpected OpenOCDHost in params")
	}

	// Test template is not empty
	if len(script.Template()) == 0 {
		t.Error("template should not be empty")
	}

	// Test parsing success case (100% confidence)
	successOutput := `DETECTED_VERSION=0x355
CONFIDENCE=100
MATCHES=7
STATUS=OK`
	result, err := script.Parse(successOutput)
	if err != nil {
		t.Errorf("unexpected error parsing success output: %v", err)
	}
	if !result.Success {
		t.Error("expected successful parse result for 100% confidence")
	}

	// Test parsing failure case (< 100% confidence)
	failureOutput := `DETECTED_VERSION=0x355
CONFIDENCE=57
MATCHES=4
STATUS=PARTIAL`
	result, err = script.Parse(failureOutput)
	if err != nil {
		t.Errorf("unexpected error parsing failure output: %v", err)
	}
	if result.Success {
		t.Error("expected unsuccessful parse result for partial confidence")
	}

	// Test unknown firmware case
	unknownOutput := `DETECTED_VERSION=UNKNOWN
CONFIDENCE=0
MATCHES=0
STATUS=UNKNOWN`
	result, err = script.Parse(unknownOutput)
	if err != nil {
		t.Errorf("unexpected error parsing unknown output: %v", err)
	}
	if result.Success {
		t.Error("expected unsuccessful parse result for unknown firmware")
	}
}
