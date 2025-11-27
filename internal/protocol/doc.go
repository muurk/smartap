// Package protocol implements the Smartap device binary protocol.
//
// This package handles parsing, validation, and construction of binary protocol
// messages used by Smartap smart shower controllers. The protocol uses custom
// WebSocket frames with a specific binary format for device communication.
//
// # Protocol Overview
//
// Smartap devices communicate using binary messages with this structure:
//   - Frame sync byte: 0x7e
//   - Protocol version: 0x03
//   - Payload length: 2 bytes (little-endian)
//   - Message payload: Variable length
//   - Checksum: 1 byte (XOR of all bytes)
//
// # Message Types
//
// The protocol supports several message types:
//   - Device status: Current valve states, pressure modes, temperature sensors
//   - Valve control: Commands to control water valves
//   - Pressure mode: Enable/disable pressure-based flow control
//   - Dual-valve messages: 77-byte messages with two valve states
//
// # WebSocket Frame Format
//
// WebSocket frames from the device have this structure:
//   - FIN bit: 1 (single frame message)
//   - Opcode: 0x02 (binary frame)
//   - Mask bit: 1 (payload is masked)
//   - Mask key: 4 bytes
//   - Payload: Variable length (masked)
//
// The payload must be unmasked using XOR with the mask key before parsing.
//
// # Usage Example - Parsing
//
//	// Read WebSocket frame from connection
//	frame, err := protocol.ReadFrame(conn)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Parse protocol frame from payload
//	protoFrame, err := protocol.ParseProtocolFrame(frame.Payload)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Parse message from protocol frame
//	msg, err := protoFrame.ParseMessage()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Handle message based on type
//	switch msg.Type {
//	case protocol.MessageTypeStatus:
//	    fmt.Printf("Device status: %s\n", msg)
//	}
//
// # Usage Example - Construction
//
//	// Build a pressure mode set command
//	msgID := protocol.GenerateMessageID()
//	msg, err := protocol.BuildPressureModeSet(msgID, true)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Send via WebSocket
//	err = conn.WriteMessage(websocket.BinaryMessage, msg)
//
// # Message Analysis
//
// The package includes utilities for analyzing and debugging messages:
//   - Hex dumps with byte annotations
//   - Checksum validation
//   - Frame structure visualization
//   - Unknown message format detection
//
// # Dual-Valve Messages
//
// Smartap devices with dual valves (hot/cold water control) send 77-byte
// status messages with interleaved valve data:
//   - Bytes 0-37: Cold valve status
//   - Bytes 38-75: Hot valve status
//   - Byte 76: Message terminator (0x0a)
//
// Each valve status includes:
//   - Valve ID (0x01 for cold, 0x02 for hot)
//   - Pressure mode state (enabled/disabled)
//   - Temperature sensor presence
//   - Current flow metrics
//
// # Error Handling
//
// The package distinguishes between:
//   - Parse errors: Malformed message structure
//   - Validation errors: Invalid checksums or field values
//   - Protocol errors: Unexpected message types or versions
//
// All errors are wrapped with context for debugging.
//
// # Thread Safety
//
// All parsing and construction functions are stateless and safe for concurrent use.
// Message ID generation uses atomic operations for thread-safe ID assignment.
package protocol
