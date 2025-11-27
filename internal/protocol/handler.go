package protocol

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/muurk/smartap/internal/logging"
	"go.uber.org/zap"
)

// HandleMessage processes incoming WebSocket messages from the device
// Parses custom binary protocol with 0x7e 0x03 frames
func HandleMessage(conn *websocket.Conn, remoteAddr string, data []byte) error {
	// Check for 77-byte dual-valve message first (special case)
	if IsDualValveMessage(data) {
		return handleDualValveMessage(conn, remoteAddr, data)
	}

	// Try to parse as protocol frame (0x7e 0x03)
	if len(data) >= MinFrameSize && data[0] == ProtocolSync && data[1] == ProtocolVersion {
		return handleProtocolFrame(conn, remoteAddr, data)
	}

	// Try to parse as JSON
	var jsonMsg map[string]interface{}
	if err := json.Unmarshal(data, &jsonMsg); err == nil {
		logging.Info("Message parsed as JSON",
			zap.String("remote_addr", remoteAddr),
			zap.Any("parsed", jsonMsg),
		)
		return handleJSONMessage(conn, remoteAddr, jsonMsg)
	}

	// Unknown format - log as hex
	logging.Warn("Unknown message format",
		zap.String("remote_addr", remoteAddr),
		zap.Int("length", len(data)),
		zap.String("hex", hex.EncodeToString(data)),
	)

	return nil
}

// handleDualValveMessage processes 77-byte dual-valve status messages
func handleDualValveMessage(conn *websocket.Conn, remoteAddr string, data []byte) error {
	msg, err := ParseDualValveMessage(data)
	if err != nil {
		logging.Error("Failed to parse dual-valve message",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
			zap.String("hex", hex.EncodeToString(data)),
		)
		return err
	}

	// Log both valve states with detailed information
	logging.Info("🚰 Dual-valve status received",
		zap.String("remote_addr", remoteAddr),
		zap.Int("message_length", len(data)),
		zap.String("cold_valve", msg.ColdValve.String()),
		zap.String("hot_valve", msg.HotValve.String()),
	)

	// Log cold valve details
	if msg.ColdValve.PressureMode != nil {
		coldStatus := "disabled"
		if msg.ColdValve.PressureMode.Enabled != 0 {
			coldStatus = "enabled"
		}
		logging.Info("  ❄️  COLD valve",
			zap.String("remote_addr", remoteAddr),
			zap.String("valve_id", fmt.Sprintf("0x%02x", msg.ColdValve.ValveID)),
			zap.String("pressure_mode", coldStatus),
			zap.Bool("temp_sensor", msg.ColdValve.HasTempSensor),
		)
	}

	// Log hot valve details
	if msg.HotValve.PressureMode != nil {
		hotStatus := "disabled"
		if msg.HotValve.PressureMode.Enabled != 0 {
			hotStatus = "enabled"
		}
		logging.Info("  🔥 HOT valve",
			zap.String("remote_addr", remoteAddr),
			zap.String("valve_id", fmt.Sprintf("0x%02x", msg.HotValve.ValveID)),
			zap.String("pressure_mode", hotStatus),
			zap.Bool("temp_sensor", msg.HotValve.HasTempSensor),
		)
	}

	// TODO: Store valve states in device state manager
	return nil
}

// handleProtocolFrame processes binary protocol frames (0x7e 0x03)
func handleProtocolFrame(conn *websocket.Conn, remoteAddr string, data []byte) error {
	// Parse protocol frame
	frame, err := ParseProtocolFrame(data)
	if err != nil {
		logging.Error("Failed to parse protocol frame",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
			zap.String("hex", hex.EncodeToString(data)),
		)
		return err
	}

	// Parse message from payload
	msg, err := frame.ParseMessage()
	if err != nil {
		logging.Error("Failed to parse message payload",
			zap.String("remote_addr", remoteAddr),
			zap.String("frame", frame.String()),
			zap.Error(err),
		)
		return err
	}

	// Get message type name
	msgTypeName := GetMessageTypeName(msg.Type())

	// Log the decoded message
	logging.Info("📨 Decoded protocol message",
		zap.String("remote_addr", remoteAddr),
		zap.String("frame", frame.String()),
		zap.String("type", msgTypeName),
		zap.String("message", msg.String()),
	)

	// Dispatch based on message type
	switch m := msg.(type) {
	case *TelemetryBroadcastMessage:
		return handleTelemetryBroadcast(conn, remoteAddr, frame, m)
	case *TelemetryResponseMessage:
		return handleTelemetryResponse(conn, remoteAddr, frame, m)
	case *CommandMessage:
		return handleCommandMessageBinary(conn, remoteAddr, frame, m)
	case *PressureModeMessage:
		return handlePressureModeMessage(conn, remoteAddr, frame, m)
	case *UnknownMessage:
		logging.Warn("Unknown message type received",
			zap.String("remote_addr", remoteAddr),
			zap.Uint8("type", m.MessageType),
			zap.Int("data_len", len(m.Data)),
		)
		return nil
	}

	return nil
}

// handleTelemetryBroadcast processes telemetry broadcast messages (0x01)
// These are periodic unsolicited status updates sent every ~1.8 seconds
// Message ID is always 0x0FFFFFFF for broadcasts
func handleTelemetryBroadcast(conn *websocket.Conn, remoteAddr string, frame *ProtocolFrame, msg *TelemetryBroadcastMessage) error {
	logging.Info("📡 Telemetry broadcast received",
		zap.String("remote_addr", remoteAddr),
		zap.Uint32("message_id", frame.MessageID),
		zap.String("telemetry_type", fmt.Sprintf("0x%02x", msg.TelemetryType)),
		zap.String("status_type", fmt.Sprintf("0x%02x", msg.StatusType)),
		zap.String("field1", fmt.Sprintf("0x%08x", msg.Field1)),
		zap.String("field2", fmt.Sprintf("0x%08x", msg.Field2)),
		zap.String("subtype", fmt.Sprintf("0x%02x", msg.SubType)),
		zap.Int("data_fields_len", len(msg.DataFields)),
		zap.String("data_fields_hex", hex.EncodeToString(msg.DataFields)),
	)

	// Log if this is a broadcast message ID
	if frame.MessageID == MsgIDBroadcast {
		logging.Debug("  → Confirmed broadcast message ID (0x0FFFFFFF)",
			zap.String("remote_addr", remoteAddr),
		)
	}

	// TODO: Store telemetry data, correlate fields with device state
	return nil
}

// handleTelemetryResponse processes telemetry response messages (0x29)
// These are responses to explicit telemetry queries
func handleTelemetryResponse(conn *websocket.Conn, remoteAddr string, frame *ProtocolFrame, msg *TelemetryResponseMessage) error {
	logging.Info("📊 Telemetry response received",
		zap.String("remote_addr", remoteAddr),
		zap.Uint32("message_id", frame.MessageID),
		zap.Uint8("subtype", msg.Subtype),
		zap.Uint32("value", msg.Value),
		zap.String("value_hex", fmt.Sprintf("0x%08x", msg.Value)),
	)

	// TODO: Store telemetry data, update device state
	return nil
}

// handleCommandMessageBinary processes binary command messages (0x42)
func handleCommandMessageBinary(conn *websocket.Conn, remoteAddr string, frame *ProtocolFrame, msg *CommandMessage) error {
	logging.Info("⚡ Command message received",
		zap.String("remote_addr", remoteAddr),
		zap.Uint32("message_id", frame.MessageID),
		zap.Uint32("param1", msg.Param1),
		zap.Int("data_len", len(msg.Data)),
	)

	// TODO: Handle command responses
	return nil
}

// handlePressureModeMessage processes pressure mode messages (0x55)
func handlePressureModeMessage(conn *websocket.Conn, remoteAddr string, frame *ProtocolFrame, msg *PressureModeMessage) error {
	status := "disabled"
	if msg.Enabled != 0 {
		status = "enabled"
	}

	logging.Info("💧 Pressure mode status",
		zap.String("remote_addr", remoteAddr),
		zap.Uint32("message_id", frame.MessageID),
		zap.String("status", status),
	)

	// TODO: Update device pressure mode state
	return nil
}

// handleJSONMessage processes JSON-formatted messages
func handleJSONMessage(conn *websocket.Conn, remoteAddr string, msg map[string]interface{}) error {
	// Check for message type field (common in many protocols)
	msgType, hasType := msg["type"]
	if hasType {
		logging.Info("JSON message has type field",
			zap.String("remote_addr", remoteAddr),
			zap.Any("type", msgType),
		)

		// Dispatch based on type
		// TODO: Implement handlers for specific message types as we discover them
		switch msgType {
		case "status":
			return handleStatusMessage(conn, remoteAddr, msg)
		case "telemetry":
			return handleTelemetryMessage(conn, remoteAddr, msg)
		case "command":
			return handleCommandMessage(conn, remoteAddr, msg)
		default:
			logging.Warn("Unknown message type",
				zap.String("remote_addr", remoteAddr),
				zap.Any("type", msgType),
			)
		}
	}

	// Log all fields for protocol analysis
	logging.Info("JSON message fields",
		zap.String("remote_addr", remoteAddr),
		zap.Any("fields", msg),
	)

	return nil
}

// handleStatusMessage processes device status messages
func handleStatusMessage(conn *websocket.Conn, remoteAddr string, msg map[string]interface{}) error {
	logging.Info("Device status message",
		zap.String("remote_addr", remoteAddr),
		zap.Any("status", msg),
	)
	// TODO: Parse specific status fields and update device state
	return nil
}

// handleTelemetryMessage processes device telemetry/sensor data
func handleTelemetryMessage(conn *websocket.Conn, remoteAddr string, msg map[string]interface{}) error {
	logging.Info("Device telemetry message",
		zap.String("remote_addr", remoteAddr),
		zap.Any("telemetry", msg),
	)
	// TODO: Parse sensor data and log/store
	return nil
}

// handleCommandMessage processes command responses from device
func handleCommandMessage(conn *websocket.Conn, remoteAddr string, msg map[string]interface{}) error {
	logging.Info("Device command response",
		zap.String("remote_addr", remoteAddr),
		zap.Any("command", msg),
	)
	// TODO: Match with pending command and handle response
	return nil
}

// SendCommand sends a command to the device
// This is a placeholder for future command functionality
func SendCommand(conn *websocket.Conn, remoteAddr string, command string, params map[string]interface{}) error {
	msg := map[string]interface{}{
		"type":    "command",
		"command": command,
		"params":  params,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	logging.Info("Sending command to device",
		zap.String("remote_addr", remoteAddr),
		zap.String("command", command),
		zap.Any("params", params),
	)

	return conn.WriteMessage(websocket.TextMessage, data)
}
