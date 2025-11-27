package protocol

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"
)

// Message constructor library for building protocol messages to send to Smartap device
// Based on verified Ghidra decompilation analysis (memory-analysis-export-ghidra.c)

const (
	// MaxPayloadSize is the maximum payload size (arbitrary safety limit)
	MaxPayloadSize = 1024

	// Reserved message IDs that should not be used for client-generated messages
	MsgIDBroadcastReserved = 0x0FFFFFFF // Reserved for device broadcasts
)

// Global message ID counter (thread-safe)
var messageIDCounter uint32 = 1

// BuildProtocolFrame constructs a complete protocol frame with header and padding
//
// Frame Structure (verified from Ghidra FUN_00006472 @ lines 2456-2490):
//
//	[0]     0x7e           Sync byte (ProtocolSync)
//	[1]     0x03           Version byte (ProtocolVersion)
//	[2-5]   message_id     Message ID (little-endian uint32)
//	[6-7]   length         Payload length (little-endian uint16)
//	[8+]    payload        Message payload bytes
//	[N+]    padding        Zero padding to MinFrameSize (38 bytes minimum)
//
// Parameters:
//   - messageID: Unique message identifier (use GenerateMessageID())
//   - payload: The message payload bytes (message type + data)
//
// Returns:
//   - Complete protocol frame ready to send via WebSocket
//   - Error if payload exceeds maximum size
//
// Source: FUN_00006472 builds header, FUN_0000650e adds padding
func BuildProtocolFrame(messageID uint32, payload []byte) ([]byte, error) {
	if len(payload) > MaxPayloadSize {
		return nil, fmt.Errorf("payload too large: %d bytes (max %d)", len(payload), MaxPayloadSize)
	}

	// Calculate total frame size (header 8 bytes + payload + padding)
	headerSize := MinFrameSize // 8 bytes
	payloadLen := len(payload)
	frameSize := headerSize + payloadLen

	// Ensure minimum message size of 38 bytes (per FUN_0000650e)
	if frameSize < MinMessageSize {
		frameSize = MinMessageSize
	}

	// Allocate frame buffer
	frame := make([]byte, frameSize)

	// Write header (FUN_00006472 @ line 2474-2483)
	frame[0] = ProtocolSync    // 0x7e
	frame[1] = ProtocolVersion // 0x03

	// Write message ID as little-endian (line 2476-2481)
	binary.LittleEndian.PutUint32(frame[2:6], messageID)

	// Write payload length as little-endian (line 2482-2483)
	binary.LittleEndian.PutUint16(frame[6:8], uint16(payloadLen))

	// Copy payload bytes (line 2685-2698)
	copy(frame[8:], payload)

	// Padding is automatically zero-filled by make()
	// This matches FUN_0000650e behavior @ line 2699

	return frame, nil
}

// BuildCommandMessage constructs a command message (type 0x42)
//
// Command messages are used for device control, configuration, and queries.
// Structure verified from Ghidra FUN_00006546 @ lines 2520-2571
//
// Payload Structure:
//
//	[0]     0x42           Message type (MsgTypeCommand)
//	[1]     len(data)+5    Total length + 5
//	[2]     0x01           Field marker
//	[3-6]   category       Category/command code (little-endian uint32)
//	[7+]    data           Variable length command data
//
// Parameters:
//   - messageID: Unique message ID (use GenerateMessageID())
//   - category: Command category/code (device-specific)
//   - data: Additional command data (can be empty)
//
// Returns:
//   - Complete protocol frame with command message payload
//
// Example:
//
//	msg, err := BuildCommandMessage(GenerateMessageID(), 0x1234, []byte{0x01, 0x02})
//
// Source: FUN_00006546 @ line 2520-2571
func BuildCommandMessage(messageID uint32, category uint32, data []byte) ([]byte, error) {
	// Calculate payload size: type(1) + length(1) + marker(1) + category(4) + data
	payloadSize := 1 + 1 + 1 + 4 + len(data)
	payload := make([]byte, payloadSize)

	// Write message type (line 2528)
	payload[0] = MsgTypeCommand // 0x42

	// Write length field = len(data) + 5 (line 2531-2532)
	payload[1] = byte(len(data) + 5)

	// Write marker byte (line 2533)
	payload[2] = 0x01

	// Write category as little-endian (line 2534-2541)
	binary.LittleEndian.PutUint32(payload[3:7], category)

	// Copy data bytes (line 2542-2553)
	if len(data) > 0 {
		copy(payload[7:], data)
	}

	// Wrap in protocol frame
	return BuildProtocolFrame(messageID, payload)
}

// BuildTelemetryQuery constructs a telemetry query message (type 0x29 query variant)
//
// Telemetry queries request specific sensor data or device state from the device.
// This is the inverse of the telemetry response message (FUN_00006928 @ lines 2921-2955)
//
// Payload Structure:
//
//	[0]     0x29           Message type (MsgTypeTelemetryResponse used for queries too)
//	[1]     0x11           Subtype (telemetry marker)
//	[2]     queryType      Which telemetry value to query
//	[3-18]  zeros          Zero padding
//
// Parameters:
//   - messageID: Unique message ID
//   - queryType: Type of telemetry to query (device-specific)
//
// Returns:
//   - Complete protocol frame with telemetry query payload
//
// Example:
//
//	msg, err := BuildTelemetryQuery(GenerateMessageID(), 0x80)
//
// Source: Reverse-engineered from FUN_00006928 response format
func BuildTelemetryQuery(messageID uint32, queryType uint8) ([]byte, error) {
	// Fixed 19-byte payload (matching response structure)
	payload := make([]byte, 19)

	// Write message type (from line 2949)
	payload[0] = MsgTypeTelemetryResponse // 0x29

	// Write subtype - consistent telemetry marker (line 2951)
	payload[1] = 0x11

	// Write query type
	payload[2] = queryType

	// Remaining bytes are zero-padded (already done by make())

	// Wrap in protocol frame
	return BuildProtocolFrame(messageID, payload)
}

// BuildPressureModeSet constructs a pressure mode control message (type 0x55)
//
// Controls the low pressure mode feature of the device.
// Structure verified from Ghidra inline code @ line 4762
//
// Payload Structure:
//
//	[0]     0x55           Message type (MsgTypePressureMode)
//	[1]     0x04           Subtype/length indicator
//	[2]     value          0x00 = disabled, 0x01 = enabled
//
// Parameters:
//   - messageID: Unique message ID
//   - enabled: true to enable pressure mode, false to disable
//
// Returns:
//   - Complete protocol frame with pressure mode payload
//
// Example:
//
//	msg, err := BuildPressureModeSet(GenerateMessageID(), true)
//
// Source: Ghidra @ line 4762: local_28 = 0x55; local_27 = 4; local_26 = value
func BuildPressureModeSet(messageID uint32, enabled bool) ([]byte, error) {
	// Fixed 3-byte payload
	payload := make([]byte, 3)

	// Write message type (line 4762: local_28 = 0x55)
	payload[0] = MsgTypePressureMode // 0x55

	// Write subtype (line 4762: local_27 = 4)
	payload[1] = 0x04

	// Write value (line 4762: local_26 = value)
	if enabled {
		payload[2] = 0x01
	} else {
		payload[2] = 0x00
	}

	// Wrap in protocol frame
	return BuildProtocolFrame(messageID, payload)
}

// GenerateMessageID generates a unique message ID for outgoing messages
//
// Message IDs are used to correlate requests with responses. The device uses
// specific reserved IDs like 0x0FFFFFFF for broadcasts.
//
// This function generates sequential IDs starting from 1, skipping reserved values.
// Thread-safe using atomic operations.
//
// Returns:
//   - Unique message ID (never returns reserved IDs)
//
// Example:
//
//	msgID := GenerateMessageID()
//	msg, err := BuildCommandMessage(msgID, category, data)
func GenerateMessageID() uint32 {
	for {
		// Atomically increment counter
		id := atomic.AddUint32(&messageIDCounter, 1)

		// Skip reserved IDs
		if id == MsgIDBroadcastReserved {
			continue
		}

		// Handle overflow (wrap back to 1)
		if id == 0 {
			atomic.StoreUint32(&messageIDCounter, 1)
			continue
		}

		return id
	}
}

// ValidateFrame validates a protocol frame structure
//
// Checks that a frame has correct header, length, and structure.
// Useful for testing and debugging outgoing messages.
//
// Parameters:
//   - frame: The complete protocol frame to validate
//
// Returns:
//   - nil if frame is valid
//   - error describing what is wrong with the frame
//
// Validation checks:
//   - Minimum frame size (38 bytes)
//   - Sync byte (0x7e)
//   - Version byte (0x03)
//   - Length field matches actual payload
//   - Message type is recognized
func ValidateFrame(frame []byte) error {
	// Check minimum size
	if len(frame) < MinMessageSize {
		return fmt.Errorf("frame too small: %d bytes (minimum %d)", len(frame), MinMessageSize)
	}

	// Check sync byte
	if frame[0] != ProtocolSync {
		return fmt.Errorf("invalid sync byte: 0x%02x (expected 0x%02x)", frame[0], ProtocolSync)
	}

	// Check version
	if frame[1] != ProtocolVersion {
		return fmt.Errorf("invalid version: 0x%02x (expected 0x%02x)", frame[1], ProtocolVersion)
	}

	// Parse length field
	payloadLen := binary.LittleEndian.Uint16(frame[6:8])

	// Check that frame is large enough for declared payload
	requiredSize := 8 + int(payloadLen)
	if len(frame) < requiredSize {
		return fmt.Errorf("frame size %d smaller than header + payload (%d)", len(frame), requiredSize)
	}

	// Check that payload has valid message type
	if payloadLen > 0 {
		msgType := frame[8]
		if !isKnownMessageType(msgType) {
			return fmt.Errorf("unknown message type: 0x%02x", msgType)
		}
	}

	return nil
}

// isKnownMessageType checks if a message type is recognized
func isKnownMessageType(msgType byte) bool {
	switch msgType {
	case MsgTypeTelemetryBroadcast,
		MsgTypeOTA,
		MsgTypeTelemetryResponse,
		MsgTypeCommand,
		MsgTypeExtended,
		MsgTypePressureMode:
		return true
	default:
		return false
	}
}

// CalculateHeaderChecksum calculates the header checksum used by the firmware
//
// Based on Ghidra analysis, the firmware calculates a checksum of the header bytes.
// Formula from line 2684: sum of bytes 0-7 plus 3
//
// NOTE: This checksum does NOT appear in the actual protocol frames we've captured.
// It may be used internally by the firmware for validation but not transmitted.
// Including this function for completeness based on Ghidra code.
//
// Parameters:
//   - header: First 8 bytes of protocol frame
//
// Returns:
//   - Checksum value (sum of header bytes + 3)
//
// Source: Ghidra @ line 2684
func CalculateHeaderChecksum(header []byte) uint8 {
	if len(header) < 8 {
		return 0
	}

	var sum uint16
	for i := 0; i < 8; i++ {
		sum += uint16(header[i])
	}

	// Add 3 per Ghidra line 2684
	sum += 3

	return uint8(sum & 0xFF)
}
