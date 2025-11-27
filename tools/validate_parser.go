//go:build ignore

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	// Import the protocol package
	"github.com/muurk/smartap/internal/protocol"
)

// CapturedMessage matches the structure from websocket.go
type CapturedMessage struct {
	Timestamp    string `json:"timestamp"`
	MessageNum   int    `json:"message_num"`
	RemoteAddr   string `json:"remote_addr"`
	Direction    string `json:"direction"`
	FrameType    string `json:"frame_type"`
	Opcode       byte   `json:"opcode"`
	FIN          bool   `json:"fin"`
	Masked       bool   `json:"masked"`
	PayloadLen   int    `json:"payload_length"`
	PayloadHex   string `json:"payload_hex"`
	PayloadAscii string `json:"payload_ascii"`
	RawFrameHex  string `json:"raw_frame_hex"`
}

// Statistics tracks parsing results
type Statistics struct {
	TotalMessages      int
	TotalFiles         int
	ParseSuccess       int
	ParseFailure       int
	MessageTypes       map[byte]int
	FailedMessages     []FailedMessage
	PayloadLengths     map[int]int
}

// FailedMessage stores information about parsing failures
type FailedMessage struct {
	File       string
	LineNumber int
	MessageNum int
	PayloadHex string
	Error      string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: validate_parser <directory-or-file>")
		fmt.Println("Example: validate_parser smartap-server/analysis/messages/")
		fmt.Println("         validate_parser capture-20251121-104043.jsonl")
		os.Exit(1)
	}

	path := os.Args[1]

	stats := Statistics{
		MessageTypes:   make(map[byte]int),
		FailedMessages: []FailedMessage{},
		PayloadLengths: make(map[int]int),
	}

	// Check if path is directory or file
	info, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Error accessing path: %v\n", err)
		os.Exit(1)
	}

	var files []string
	if info.IsDir() {
		// Find all JSONL files in directory
		pattern := filepath.Join(path, "*.jsonl")
		files, err = filepath.Glob(pattern)
		if err != nil {
			fmt.Printf("Error finding JSONL files: %v\n", err)
			os.Exit(1)
		}
		if len(files) == 0 {
			fmt.Printf("No JSONL files found in %s\n", path)
			os.Exit(1)
		}
	} else {
		// Single file
		files = []string{path}
	}

	fmt.Printf("=== Smartap Parser Validator ===\n")
	fmt.Printf("Files to process: %d\n\n", len(files))

	// Process each file
	for _, file := range files {
		processFile(file, &stats)
	}

	// Print results
	printStatistics(&stats)
}

func processFile(filename string, stats *Statistics) {
	stats.TotalFiles++

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file %s: %v\n", filename, err)
		return
	}

	lines := strings.Split(string(data), "\n")

	for lineNum, line := range lines {
		if line == "" {
			continue
		}

		var msg CapturedMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			fmt.Printf("Error parsing JSON in %s line %d: %v\n", filename, lineNum+1, err)
			continue
		}

		stats.TotalMessages++

		// Decode payload
		payload, err := hex.DecodeString(msg.PayloadHex)
		if err != nil {
			stats.ParseFailure++
			stats.FailedMessages = append(stats.FailedMessages, FailedMessage{
				File:       filename,
				LineNumber: lineNum + 1,
				MessageNum: msg.MessageNum,
				PayloadHex: msg.PayloadHex,
				Error:      fmt.Sprintf("hex decode error: %v", err),
			})
			continue
		}

		// Track payload length
		stats.PayloadLengths[len(payload)]++

		// Check for dual-valve message (76-77 bytes, starts with 0x03)
		if (len(payload) == 77 || len(payload) == 76) && payload[0] == 0x03 {
			// This is a dual-valve message, parse it specially
			dualValve, err := protocol.ParseDualValveMessage(payload)
			if err != nil {
				stats.ParseFailure++
				stats.FailedMessages = append(stats.FailedMessages, FailedMessage{
					File:       filename,
					LineNumber: lineNum + 1,
					MessageNum: msg.MessageNum,
					PayloadHex: msg.PayloadHex,
					Error:      fmt.Sprintf("dual-valve parse error: %v", err),
				})
				continue
			}
			// Success - count both messages in the dual-valve
			stats.ParseSuccess++
			stats.MessageTypes[dualValve.ColdValve.Type()]++
			if dualValve.HotValve != nil {
				stats.MessageTypes[dualValve.HotValve.Type()]++
			}
			continue
		}

		// Try to parse the protocol frame first
		frame, err := protocol.ParseProtocolFrame(payload)
		if err != nil {
			stats.ParseFailure++
			stats.FailedMessages = append(stats.FailedMessages, FailedMessage{
				File:       filename,
				LineNumber: lineNum + 1,
				MessageNum: msg.MessageNum,
				PayloadHex: msg.PayloadHex,
				Error:      fmt.Sprintf("frame parse error: %v", err),
			})
			continue
		}

		// Try to parse the message from the frame
		parsedMsg, err := frame.ParseMessage()
		if err != nil {
			stats.ParseFailure++
			stats.FailedMessages = append(stats.FailedMessages, FailedMessage{
				File:       filename,
				LineNumber: lineNum + 1,
				MessageNum: msg.MessageNum,
				PayloadHex: msg.PayloadHex,
				Error:      fmt.Sprintf("message parse error: %v", err),
			})
			continue
		}

		// Success!
		stats.ParseSuccess++
		stats.MessageTypes[parsedMsg.Type()]++
	}
}

func printStatistics(stats *Statistics) {
	fmt.Printf("\n========================================\n")
	fmt.Printf("VALIDATION RESULTS\n")
	fmt.Printf("========================================\n\n")

	fmt.Printf("Files Processed:    %d\n", stats.TotalFiles)
	fmt.Printf("Total Messages:     %d\n", stats.TotalMessages)
	fmt.Printf("Parse Success:      %d (%.2f%%)\n", stats.ParseSuccess,
		float64(stats.ParseSuccess)/float64(stats.TotalMessages)*100)
	fmt.Printf("Parse Failure:      %d (%.2f%%)\n", stats.ParseFailure,
		float64(stats.ParseFailure)/float64(stats.TotalMessages)*100)

	fmt.Printf("\n----------------------------------------\n")
	fmt.Printf("MESSAGE TYPE DISTRIBUTION\n")
	fmt.Printf("----------------------------------------\n")
	for msgType, count := range stats.MessageTypes {
		typeName := getMessageTypeName(msgType)
		percentage := float64(count) / float64(stats.ParseSuccess) * 100
		fmt.Printf("Type 0x%02x (%s): %d (%.2f%%)\n", msgType, typeName, count, percentage)
	}

	fmt.Printf("\n----------------------------------------\n")
	fmt.Printf("PAYLOAD LENGTH DISTRIBUTION\n")
	fmt.Printf("----------------------------------------\n")
	for length, count := range stats.PayloadLengths {
		percentage := float64(count) / float64(stats.TotalMessages) * 100
		fmt.Printf("%d bytes: %d messages (%.2f%%)\n", length, count, percentage)
	}

	if len(stats.FailedMessages) > 0 {
		fmt.Printf("\n----------------------------------------\n")
		fmt.Printf("PARSE FAILURES (%d total)\n", len(stats.FailedMessages))
		fmt.Printf("----------------------------------------\n")

		// Show first 10 failures
		maxShow := 10
		if len(stats.FailedMessages) > maxShow {
			fmt.Printf("(Showing first %d of %d failures)\n\n", maxShow, len(stats.FailedMessages))
		}

		for i, failed := range stats.FailedMessages {
			if i >= maxShow {
				break
			}
			fmt.Printf("\nFailure #%d:\n", i+1)
			fmt.Printf("  File: %s (line %d, msg #%d)\n", failed.File, failed.LineNumber, failed.MessageNum)
			fmt.Printf("  Error: %s\n", failed.Error)
			// Show first 80 chars of hex
			hexPreview := failed.PayloadHex
			if len(hexPreview) > 80 {
				hexPreview = hexPreview[:80] + "..."
			}
			fmt.Printf("  Payload: %s\n", hexPreview)
		}
	}

	fmt.Printf("\n========================================\n")
	if stats.ParseFailure == 0 {
		fmt.Printf("✅ SUCCESS: All messages parsed successfully!\n")
	} else {
		fmt.Printf("⚠️  ISSUES FOUND: %d messages failed to parse\n", stats.ParseFailure)
	}
	fmt.Printf("========================================\n")
}

func getMessageTypeName(msgType byte) string {
	switch msgType {
	case 0x01:
		return "TelemetryBroadcast"
	case 0x05:
		return "OTA"
	case 0x29:
		return "TelemetryResponse"
	case 0x42:
		return "Command"
	case 0x44:
		return "Extended"
	case 0x55:
		return "PressureMode"
	default:
		return "Unknown"
	}
}
