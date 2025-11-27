//go:build ignore

package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// MessageAnalysis matches the structure from websocket.go
type MessageAnalysis struct {
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: analyze-messages <jsonl-file>")
		fmt.Println("Example: analyze-messages smartap-server/analysis/messages/capture-20251121-030905.jsonl")
		os.Exit(1)
	}

	filename := os.Args[1]
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	lines := strings.Split(string(data), "\n")
	fmt.Printf("=== Smartap Message Analyzer ===\n")
	fmt.Printf("File: %s\n", filename)
	fmt.Printf("Messages: %d\n\n", len(lines)-1) // -1 for trailing newline

	for i, line := range lines {
		if line == "" {
			continue
		}

		var msg MessageAnalysis
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			fmt.Printf("Error parsing line %d: %v\n", i+1, err)
			continue
		}

		analyzeMessage(&msg)
	}
}

func analyzeMessage(msg *MessageAnalysis) {
	payload, err := hex.DecodeString(msg.PayloadHex)
	if err != nil {
		fmt.Printf("Error decoding hex: %v\n", err)
		return
	}

	fmt.Printf("========================================\n")
	fmt.Printf("Message #%d - %d bytes - %s\n", msg.MessageNum, len(payload), msg.Timestamp)
	fmt.Printf("========================================\n\n")

	// Dump as 32-bit little-endian words
	fmt.Println("32-bit Little-Endian Words:")
	fmt.Println("Offset  Hex        Decimal       Binary")
	fmt.Println("------  ---------- ------------- --------------------------------")

	for i := 0; i+4 <= len(payload); i += 4 {
		word := binary.LittleEndian.Uint32(payload[i : i+4])
		fmt.Printf("[%02d-%02d] 0x%08x %13d %032b\n", i, i+3, word, word, word)
	}

	// Handle remaining bytes
	rem := len(payload) % 4
	if rem > 0 {
		start := len(payload) - rem
		tail := payload[start:]
		fmt.Printf("[%02d-%02d] tail: ", start, len(payload)-1)
		for _, b := range tail {
			fmt.Printf("%02x ", b)
		}
		fmt.Printf("(decimal: ")
		for _, b := range tail {
			fmt.Printf("%d ", b)
		}
		fmt.Println(")")
	}

	fmt.Println()

	// Test checksum hypotheses
	fmt.Println("Checksum Analysis:")
	testChecksums(payload)
	fmt.Println()

	// Identify patterns
	fmt.Println("Pattern Detection:")
	detectPatterns(payload)
	fmt.Println()

	// Hex dump for reference
	fmt.Println("Hex Dump (16 bytes/line):")
	hexDump(payload)
	fmt.Println("\n")
}

func testChecksums(payload []byte) {
	if len(payload) == 0 {
		return
	}

	lastByte := payload[len(payload)-1]
	dataBytes := payload[:len(payload)-1]

	// Test 1: Sum of all bytes mod 256
	sum := uint32(0)
	for _, b := range dataBytes {
		sum += uint32(b)
	}
	sumMod256 := byte(sum & 0xFF)
	fmt.Printf("  Sum mod 256 (all):     0x%02x (%d) - ", sumMod256, sumMod256)
	if sumMod256 == lastByte {
		fmt.Println("✅ MATCH!")
	} else {
		fmt.Printf("❌ (expected 0x%02x)\n", lastByte)
	}

	// Test 1b: Sum of first 8 bytes (protocol header) + 3
	// Based on Ghidra analysis: FUN_00006472 line 2486
	if len(payload) >= 8 {
		headerSum := uint32(0)
		for i := 0; i < 8; i++ {
			headerSum += uint32(payload[i])
		}
		headerSumPlus3 := byte((headerSum + 3) & 0xFF)
		fmt.Printf("  Header sum + 3:        0x%02x (%d) - ", headerSumPlus3, headerSumPlus3)
		if headerSumPlus3 == lastByte {
			fmt.Println("✅ MATCH! (Ghidra algorithm)")
		} else {
			fmt.Printf("❌ (expected 0x%02x)\n", lastByte)
		}
	}

	// Test 2: XOR of all bytes
	xor := byte(0)
	for _, b := range dataBytes {
		xor ^= b
	}
	fmt.Printf("  XOR of all bytes:      0x%02x (%d) - ", xor, xor)
	if xor == lastByte {
		fmt.Println("✅ MATCH!")
	} else {
		fmt.Printf("❌ (expected 0x%02x)\n", lastByte)
	}

	// Test 3: Two's complement sum
	twosComp := byte((^sum + 1) & 0xFF)
	fmt.Printf("  Two's complement sum:  0x%02x (%d) - ", twosComp, twosComp)
	if twosComp == lastByte {
		fmt.Println("✅ MATCH!")
	} else {
		fmt.Printf("❌ (expected 0x%02x)\n", lastByte)
	}

	// Test 4: Negative sum mod 256
	negSum := byte((-int32(sum)) & 0xFF)
	fmt.Printf("  Negative sum mod 256:  0x%02x (%d) - ", negSum, negSum)
	if negSum == lastByte {
		fmt.Println("✅ MATCH!")
	} else {
		fmt.Printf("❌ (expected 0x%02x)\n", lastByte)
	}

	// Test 5: Sum of 16-bit words
	if len(dataBytes)%2 == 0 {
		sum16 := uint32(0)
		for i := 0; i+2 <= len(dataBytes); i += 2 {
			word := binary.LittleEndian.Uint16(dataBytes[i : i+2])
			sum16 += uint32(word)
		}
		sum16Mod256 := byte(sum16 & 0xFF)
		fmt.Printf("  Sum of 16-bit words:   0x%02x (%d) - ", sum16Mod256, sum16Mod256)
		if sum16Mod256 == lastByte {
			fmt.Println("✅ MATCH!")
		} else {
			fmt.Printf("❌ (expected 0x%02x)\n", lastByte)
		}
	}
}

func detectPatterns(payload []byte) {
	// Look for common byte patterns
	patterns := map[string]int{
		"7e03":   0,
		"0f1e":   0,
		"8055":   0,
		"507d":   0,
		"ffffff": 0,
	}

	hexStr := hex.EncodeToString(payload)
	for pattern := range patterns {
		count := strings.Count(hexStr, pattern)
		if count > 0 {
			patterns[pattern] = count
		}
	}

	fmt.Println("  Common patterns found:")
	for pattern, count := range patterns {
		if count > 0 {
			// Find positions
			positions := []int{}
			for i := 0; i <= len(hexStr)-len(pattern); i++ {
				if hexStr[i:i+len(pattern)] == pattern {
					positions = append(positions, i/2) // Convert to byte position
				}
			}
			fmt.Printf("    %s: %d occurrences at byte offsets: %v\n", pattern, count, positions)
		}
	}

	// Look for repeating blocks
	if len(payload) == 77 {
		// Check if it's two similar blocks
		block1 := payload[0:38]
		block2 := payload[38:76]

		fmt.Println("\n  77-byte message block analysis:")
		fmt.Println("    Block 1 (bytes 0-37):")
		fmt.Printf("      %x\n", block1)
		fmt.Println("    Block 2 (bytes 38-75):")
		fmt.Printf("      %x\n", block2)
		fmt.Printf("    Trailing byte: 0x%02x\n", payload[76])

		// Count differences
		diffs := 0
		for i := 0; i < len(block1) && i < len(block2); i++ {
			if block1[i] != block2[i] {
				diffs++
			}
		}
		fmt.Printf("    Differences between blocks: %d bytes (%.1f%% similar)\n",
			diffs, float64(38-diffs)/38.0*100)
	}
}

func hexDump(payload []byte) {
	for i := 0; i < len(payload); i += 16 {
		// Offset
		fmt.Printf("%04x  ", i)

		// Hex
		for j := 0; j < 16; j++ {
			if i+j < len(payload) {
				fmt.Printf("%02x ", payload[i+j])
			} else {
				fmt.Print("   ")
			}
			if j == 7 {
				fmt.Print(" ")
			}
		}

		// ASCII
		fmt.Print(" |")
		for j := 0; j < 16 && i+j < len(payload); j++ {
			b := payload[i+j]
			if b >= 32 && b <= 126 {
				fmt.Printf("%c", b)
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println("|")
	}
}
