package protocol

import (
	"encoding/binary"
	"fmt"
)

// Protocol frame constants
const (
	ProtocolSync    = 0x7e
	ProtocolVersion = 0x03
	MinFrameSize    = 8  // Sync + Version + 4-byte ID + 2-byte length
	MinMessageSize  = 38 // Minimum message size with padding (from Ghidra FUN_0000650e)
)

// Message type constants (from Ghidra analysis and live capture)
// VERIFIED from COMPLETE-PROTOCOL-ANALYSIS.md (2025-11-21)
const (
	MsgTypeTelemetryBroadcast = 0x01 // Periodic status from DAT_000069b4 buffer (FUN_000067ae @ line 2776)
	MsgTypeOTA                = 0x05 // Over-the-air firmware update
	MsgTypeTelemetryResponse  = 0x29 // Response to telemetry query (FUN_00006928 @ line 2921)
	MsgTypeCommand            = 0x42 // Generic command/response (FUN_00006546 @ line 2520)
	MsgTypeExtended           = 0x44 // Extended command format
	MsgTypePressureMode       = 0x55 // Low pressure mode status (line 4762)
)

// Special message IDs (from Ghidra analysis)
const (
	MsgIDBroadcast = 0x0FFFFFFF // Used for periodic telemetry broadcasts (line 2776)
)

// Valve identifiers (from dual-valve 77-byte message analysis)
const (
	ValveIDCold = 0xca // Cold water valve (no temp sensor)
	ValveIDHot  = 0x6d // Hot water valve (has temp sensor)
)

// ProtocolFrame represents a parsed protocol frame
type ProtocolFrame struct {
	Sync      byte   // Should be 0x7e
	Version   byte   // Should be 0x03
	MessageID uint32 // 4-byte message counter/ID (little-endian)
	Length    uint16 // 2-byte payload length (little-endian)
	Payload   []byte // Message payload (variable length)
	Raw       []byte // Original frame bytes
}

// Message represents a decoded message
type Message interface {
	Type() byte
	String() string
}

// TelemetryResponseMessage (type 0x29) - Response to telemetry query
// Constructed by FUN_00006928 @ line 2921 in firmware
type TelemetryResponseMessage struct {
	MessageType byte   // 0x29
	Subtype     byte   // 0x11 typically
	Field       byte   // 0x80 typically
	Value       uint32 // 4-byte sensor value (little-endian)
	Padding     []byte // Zeros to 19 bytes
}

func (m *TelemetryResponseMessage) Type() byte { return m.MessageType }

func (m *TelemetryResponseMessage) String() string {
	return fmt.Sprintf("TelemetryResponse{subtype=0x%02x, field=0x%02x, value=%d (0x%08x)}",
		m.Subtype, m.Field, m.Value, m.Value)
}

// TelemetryBroadcastMessage (type 0x01) - Periodic unsolicited broadcast
// Sent with message ID 0x0FFFFFFF every ~1.8 seconds
// Payload from static firmware buffer DAT_000069b4 (19 bytes + padding to 38)
// Appears 2,024 times in captures (97% of all traffic)
// SOURCE: FUN_000067ae @ line 2776, called from FUN_000067ba @ line 1808
type TelemetryBroadcastMessage struct {
	MessageType   byte // [0] 0x01
	TelemetryType byte // [1] 0x11 (consistent telemetry marker)
	StatusType    byte // [2] 0x0f (data format indicator)

	// Fields at offsets 3-10 (8 bytes) - meanings TBD via testing
	Field1 uint32 // [3-6] LE: 0x08000000 in all captures
	Field2 uint32 // [7-10] LE: 0x55800000 (contains 0x55?)

	// Subfield
	SubType byte // [11] 0x03 in all captures

	// Data fields (to be mapped via device manipulation)
	DataFields []byte // [12-30] Various values (example: 00 00 50 7d 6d ca 12...)

	// Trailing marker
	TrailingByte byte // [31+] 0x29 (telemetry type marker?)

	Raw []byte // Complete payload for future analysis
}

func (m *TelemetryBroadcastMessage) Type() byte { return m.MessageType }

func (m *TelemetryBroadcastMessage) String() string {
	return fmt.Sprintf("TelemetryBroadcast{telemetry_type=0x%02x, status=0x%02x, field1=0x%08x, field2=0x%08x, subtype=0x%02x, data_len=%d}",
		m.TelemetryType, m.StatusType, m.Field1, m.Field2, m.SubType, len(m.DataFields))
}

// CommandMessage (type 0x42) - Generic command/response
type CommandMessage struct {
	MessageType byte   // 0x42
	PayloadLen  byte   // Length + 5
	Field1      byte   // 0x01 typically
	Param1      uint32 // 4-byte parameter (little-endian)
	Data        []byte // Variable payload data
}

func (m *CommandMessage) Type() byte { return m.MessageType }

func (m *CommandMessage) String() string {
	return fmt.Sprintf("Command{payload_len=%d, field1=0x%02x, param1=%d (0x%08x), data_len=%d}",
		m.PayloadLen, m.Field1, m.Param1, m.Param1, len(m.Data))
}

// PressureModeMessage (type 0x55) - Low pressure mode status
type PressureModeMessage struct {
	MessageType byte // 0x55
	Subtype     byte // 0x04 typically
	Enabled     byte // Boolean: 0 or 1
}

func (m *PressureModeMessage) Type() byte { return m.MessageType }

func (m *PressureModeMessage) String() string {
	enabled := "disabled"
	if m.Enabled != 0 {
		enabled = "enabled"
	}
	return fmt.Sprintf("PressureMode{subtype=0x%02x, %s}", m.Subtype, enabled)
}

// ValveStatusMessage - Status message for hot or cold water valve
// Discovered in 77-byte dual-valve messages sent at connection initialization
type ValveStatusMessage struct {
	MessageType     byte                 // 0x01 (TelemetryBroadcast)
	ValveID         byte                 // 0xca (cold) or 0x6d (hot)
	PressureMode    *PressureModeMessage // Nested pressure mode status
	HasTempSensor   bool                 // True if valve has temperature sensor (hot valve only)
	TelemetryMarker byte                 // 0x29 if temperature sensor present
	Raw             []byte               // Raw message bytes
}

func (m *ValveStatusMessage) Type() byte { return m.MessageType }

func (m *ValveStatusMessage) String() string {
	valveType := "COLD"
	if m.ValveID == ValveIDHot {
		valveType = "HOT"
	}

	tempSensor := ""
	if m.HasTempSensor {
		tempSensor = ", temp_sensor=yes"
	}

	pressureStatus := "unknown"
	if m.PressureMode != nil {
		if m.PressureMode.Enabled != 0 {
			pressureStatus = "enabled"
		} else {
			pressureStatus = "disabled"
		}
	}

	return fmt.Sprintf("ValveStatus{valve=%s, valve_id=0x%02x, pressure_mode=%s%s}",
		valveType, m.ValveID, pressureStatus, tempSensor)
}

// DualValveMessage - Container for both hot and cold valve status
// These messages are 77 bytes total and sent together at connection start
type DualValveMessage struct {
	MessageType byte                // 0x01 (TelemetryBroadcast)
	ColdValve   *ValveStatusMessage // Frame 1 (37 bytes)
	HotValve    *ValveStatusMessage // Frame 2 (40 bytes)
}

func (m *DualValveMessage) Type() byte { return m.MessageType }

func (m *DualValveMessage) String() string {
	return fmt.Sprintf("DualValve{cold=%s, hot=%s}",
		m.ColdValve.String(), m.HotValve.String())
}

// UnknownMessage - Fallback for unrecognized message types
type UnknownMessage struct {
	MessageType byte
	Data        []byte
}

func (m *UnknownMessage) Type() byte { return m.MessageType }

func (m *UnknownMessage) String() string {
	return fmt.Sprintf("Unknown{type=0x%02x, len=%d}", m.MessageType, len(m.Data))
}

// IsDualValveMessage detects if the data is a 77-byte dual-valve message
// These messages appear at connection start and contain status for both valves
func IsDualValveMessage(data []byte) bool {
	// Must be exactly 77 bytes
	if len(data) != 77 {
		return false
	}

	// Frame 1 starts with 0x03 (missing 0x7e sync byte - TCP artifact)
	// Frame 2 starts at offset 37 with 0x7e 0x03
	if data[0] != ProtocolVersion {
		return false
	}

	if len(data) < 38 || data[37] != ProtocolSync || data[38] != ProtocolVersion {
		return false
	}

	// Both frames should have the same message ID (0x0fffffff typically)
	if len(data) >= 43 {
		msgID1 := binary.LittleEndian.Uint32(data[1:5])
		msgID2 := binary.LittleEndian.Uint32(data[39:43])
		if msgID1 != msgID2 {
			return false
		}
	}

	// Frame 2 should end with 0x29 (telemetry marker for hot valve temp sensor)
	if data[76] != MsgTypeTelemetryResponse {
		return false
	}

	return true
}

// ParseDualValveMessage parses a 77-byte message containing both valve states
func ParseDualValveMessage(data []byte) (*DualValveMessage, error) {
	if len(data) != 77 {
		return nil, fmt.Errorf("dual valve message must be exactly 77 bytes, got %d", len(data))
	}

	// Frame 1: bytes 0-36 (37 bytes) - Cold valve
	// Missing 0x7e sync byte, starts with 0x03
	frame1Data := make([]byte, 38)
	frame1Data[0] = ProtocolSync // Reconstruct missing sync byte
	copy(frame1Data[1:], data[0:37])

	// Frame 2: bytes 37-76 (40 bytes) - Hot valve
	// Complete frame with 0x7e 0x03 header and trailing 0x29
	frame2Data := data[37:77]

	// Parse both frames
	coldValve, err := parseValveStatus(frame1Data, ValveIDCold)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cold valve: %w", err)
	}

	hotValve, err := parseValveStatus(frame2Data, ValveIDHot)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hot valve: %w", err)
	}

	return &DualValveMessage{
		MessageType: MsgTypeTelemetryBroadcast,
		ColdValve:   coldValve,
		HotValve:    hotValve,
	}, nil
}

// parseValveStatus decodes a single valve status frame
func parseValveStatus(data []byte, expectedValveID byte) (*ValveStatusMessage, error) {
	if len(data) < 38 {
		return nil, fmt.Errorf("valve status frame too short: %d bytes (minimum 38)", len(data))
	}

	// Parse as a protocol frame first
	frame, err := ParseProtocolFrame(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse frame: %w", err)
	}

	// Extract the telemetry broadcast payload
	if len(frame.Payload) < 1 || frame.Payload[0] != MsgTypeTelemetryBroadcast {
		return nil, fmt.Errorf("expected telemetry broadcast message (0x01), got 0x%02x", frame.Payload[0])
	}

	msg := &ValveStatusMessage{
		MessageType: MsgTypeTelemetryBroadcast,
		Raw:         data,
	}

	// Look for valve ID at payload offset 16 (4 bytes starting at offset 13)
	// Format: 0x00 0x50 0x7d [valve_id]
	if len(frame.Payload) >= 17 {
		valveIDOffset := 16
		msg.ValveID = frame.Payload[valveIDOffset]

		// Validate valve ID matches expected
		if msg.ValveID != expectedValveID {
			return nil, fmt.Errorf("valve ID mismatch: expected 0x%02x, got 0x%02x",
				expectedValveID, msg.ValveID)
		}
	}

	// Extract nested PressureMode message at offset 10
	if len(frame.Payload) >= 13 {
		nestedType := frame.Payload[10]
		if nestedType == MsgTypePressureMode {
			msg.PressureMode = &PressureModeMessage{
				MessageType: nestedType,
				Subtype:     frame.Payload[11],
				Enabled:     frame.Payload[12],
			}
		}
	}

	// Check for telemetry marker (0x29) at end - indicates temp sensor
	if len(data) > 38 && data[len(data)-1] == MsgTypeTelemetryResponse {
		msg.HasTempSensor = true
		msg.TelemetryMarker = MsgTypeTelemetryResponse
	}

	return msg, nil
}

// ParseProtocolFrame parses a protocol frame from raw bytes
func ParseProtocolFrame(data []byte) (*ProtocolFrame, error) {
	if len(data) < MinFrameSize {
		return nil, fmt.Errorf("frame too short: %d bytes (minimum %d)", len(data), MinFrameSize)
	}

	frame := &ProtocolFrame{
		Raw: data,
	}

	// Parse header
	frame.Sync = data[0]
	frame.Version = data[1]

	// Validate sync and version
	if frame.Sync != ProtocolSync {
		return nil, fmt.Errorf("invalid sync byte: 0x%02x (expected 0x%02x)", frame.Sync, ProtocolSync)
	}
	if frame.Version != ProtocolVersion {
		return nil, fmt.Errorf("invalid version: 0x%02x (expected 0x%02x)", frame.Version, ProtocolVersion)
	}

	// Parse message ID (4 bytes, little-endian)
	frame.MessageID = binary.LittleEndian.Uint32(data[2:6])

	// Parse length (2 bytes, little-endian)
	frame.Length = binary.LittleEndian.Uint16(data[6:8])

	// Extract payload
	payloadStart := 8
	if len(data) > payloadStart {
		frame.Payload = data[payloadStart:]
	}

	return frame, nil
}

// ParseMessage decodes the payload into a specific message type
func (f *ProtocolFrame) ParseMessage() (Message, error) {
	if len(f.Payload) == 0 {
		return nil, fmt.Errorf("empty payload")
	}

	msgType := f.Payload[0]

	switch msgType {
	case MsgTypeTelemetryBroadcast:
		return parseTelemetryBroadcast(f.Payload)
	case MsgTypeTelemetryResponse:
		return parseTelemetryResponse(f.Payload)
	case MsgTypeCommand:
		return parseCommandMessage(f.Payload)
	case MsgTypePressureMode:
		return parsePressureModeMessage(f.Payload)
	default:
		return &UnknownMessage{
			MessageType: msgType,
			Data:        f.Payload[1:],
		}, nil
	}
}

// parseTelemetryResponse decodes a telemetry response message (0x29)
func parseTelemetryResponse(payload []byte) (*TelemetryResponseMessage, error) {
	if len(payload) < 7 {
		return nil, fmt.Errorf("telemetry response payload too short: %d bytes (minimum 7)", len(payload))
	}

	msg := &TelemetryResponseMessage{
		MessageType: payload[0],
		Subtype:     payload[1],
		Field:       payload[2],
	}

	// Parse 4-byte value (little-endian)
	msg.Value = binary.LittleEndian.Uint32(payload[3:7])

	// Store padding if present
	if len(payload) > 7 {
		msg.Padding = payload[7:]
	}

	return msg, nil
}

// parseTelemetryBroadcast decodes a telemetry broadcast message (0x01)
// This is the periodic heartbeat message from static buffer DAT_000069b4
func parseTelemetryBroadcast(payload []byte) (*TelemetryBroadcastMessage, error) {
	if len(payload) < 20 {
		return nil, fmt.Errorf("telemetry broadcast too short: %d bytes (minimum 20)", len(payload))
	}

	msg := &TelemetryBroadcastMessage{
		MessageType:   payload[0],
		TelemetryType: payload[1],
		StatusType:    payload[2],
		Raw:           payload,
	}

	// Parse 8-byte field section (offsets 3-10)
	if len(payload) >= 11 {
		msg.Field1 = binary.LittleEndian.Uint32(payload[3:7])
		msg.Field2 = binary.LittleEndian.Uint32(payload[7:11])
	}

	// Parse subtype
	if len(payload) >= 12 {
		msg.SubType = payload[11]
	}

	// Extract data fields and trailing byte
	if len(payload) > 12 {
		endIdx := len(payload)
		// Check for trailing 0x29 byte
		if len(payload) > 20 && payload[len(payload)-1] == MsgTypeTelemetryResponse {
			msg.TrailingByte = payload[len(payload)-1]
			endIdx = len(payload) - 1
		}
		msg.DataFields = payload[12:endIdx]
	}

	return msg, nil
}

// parseCommandMessage decodes a command message (0x42)
func parseCommandMessage(payload []byte) (*CommandMessage, error) {
	if len(payload) < 7 {
		return nil, fmt.Errorf("command payload too short: %d bytes (minimum 7)", len(payload))
	}

	msg := &CommandMessage{
		MessageType: payload[0],
		PayloadLen:  payload[1],
		Field1:      payload[2],
	}

	// Parse 4-byte parameter (little-endian)
	msg.Param1 = binary.LittleEndian.Uint32(payload[3:7])

	// Store remaining data
	if len(payload) > 7 {
		msg.Data = payload[7:]
	}

	return msg, nil
}

// parsePressureModeMessage decodes a pressure mode message (0x55)
func parsePressureModeMessage(payload []byte) (*PressureModeMessage, error) {
	if len(payload) < 3 {
		return nil, fmt.Errorf("pressure mode payload too short: %d bytes (minimum 3)", len(payload))
	}

	return &PressureModeMessage{
		MessageType: payload[0],
		Subtype:     payload[1],
		Enabled:     payload[2],
	}, nil
}

// String returns a human-readable representation of the frame
func (f *ProtocolFrame) String() string {
	return fmt.Sprintf("Frame{sync=0x%02x, ver=0x%02x, id=%d (0x%08x), len=%d, payload=%d bytes}",
		f.Sync, f.Version, f.MessageID, f.MessageID, f.Length, len(f.Payload))
}

// GetMessageTypeName returns a human-readable name for a message type
func GetMessageTypeName(msgType byte) string {
	switch msgType {
	case MsgTypeTelemetryBroadcast:
		return "TelemetryBroadcast"
	case MsgTypeOTA:
		return "OTA"
	case MsgTypeTelemetryResponse:
		return "TelemetryResponse"
	case MsgTypeCommand:
		return "Command"
	case MsgTypeExtended:
		return "Extended"
	case MsgTypePressureMode:
		return "PressureMode"
	default:
		return fmt.Sprintf("Unknown(0x%02x)", msgType)
	}
}
