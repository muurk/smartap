package server

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/muurk/smartap/internal/logging"
	"github.com/muurk/smartap/internal/protocol"
	"go.uber.org/zap"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10 //nolint:unused // Reserved for future ping/pong heartbeat

	// Maximum message size allowed from peer
	maxMessageSize = 8192 //nolint:unused // Reserved for future message size validation
)

// HandleWebSocketConnection manages a WebSocket connection after the HTTP 101 upgrade
// This function is called after we've sent the custom HTTP 101 response
// We parse WebSocket frames manually to unmask and log the actual payloads
// If analysisDir is non-empty, messages will be logged to that directory
func HandleWebSocketConnection(conn net.Conn, remoteAddr string, analysisDir string) error {
	logging.LogConnection(remoteAddr, "websocket_upgraded")

	defer func() {
		_ = conn.Close()
		logging.LogConnection(remoteAddr, "websocket_closed")
	}()

	// Message counter for analysis
	messageNum := 0

	// Main message receive loop
	for {
		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			logging.Info("Failed to set read deadline, connection may be closed",
				zap.String("remote_addr", remoteAddr),
				zap.Error(err),
			)
			return err
		}

		// Read and parse WebSocket frame
		frame, err := protocol.ReadFrame(conn)
		if err != nil {
			if err == io.EOF {
				logging.Info("Connection closed by device",
					zap.String("remote_addr", remoteAddr),
				)
			} else {
				logging.Info("Connection closed or error reading frame",
					zap.String("remote_addr", remoteAddr),
					zap.Error(err),
				)
			}
			break
		}

		messageNum++

		// Log the parsed frame
		logging.Info("WebSocket frame received",
			zap.String("remote_addr", remoteAddr),
			zap.Int("message_num", messageNum),
			zap.String("frame", frame.String()),
			zap.Int("payload_length", len(frame.Payload)),
		)

		// Handle based on opcode
		switch frame.Opcode {
		case protocol.OpcodeBinary:
			// Binary message - this is what the device sends
			logging.LogWebSocketMessage(remoteAddr, "received", 2, frame.Payload)

			// Save to analysis file (if enabled)
			SaveMessageToAnalysis(remoteAddr, messageNum, frame, analysisDir)

			// Try to parse the message (as we learn the protocol)
			if err := protocol.HandleMessage(nil, remoteAddr, frame.Payload); err != nil {
				logging.Error("Failed to handle binary message",
					zap.String("remote_addr", remoteAddr),
					zap.Error(err),
				)
			}

		case protocol.OpcodeText:
			// Text message
			logging.Info("Received text WebSocket message",
				zap.String("remote_addr", remoteAddr),
				zap.String("content", string(frame.Payload)),
			)
			SaveMessageToAnalysis(remoteAddr, messageNum, frame, analysisDir)

		case protocol.OpcodePing:
			// Respond with pong
			logging.Debug("Received ping, sending pong",
				zap.String("remote_addr", remoteAddr),
			)
			// TODO: Send pong frame

		case protocol.OpcodePong:
			logging.Debug("Received pong",
				zap.String("remote_addr", remoteAddr),
			)

		case protocol.OpcodeClose:
			logging.Info("Received close frame from device",
				zap.String("remote_addr", remoteAddr),
			)
			return nil

		default:
			logging.Warn("Received frame with unknown opcode",
				zap.String("remote_addr", remoteAddr),
				zap.String("opcode", frame.OpcodeString()),
			)
		}
	}

	return nil
}

// MessageAnalysis represents a captured message for analysis
type MessageAnalysis struct {
	Timestamp    time.Time `json:"timestamp"`
	MessageNum   int       `json:"message_num"`
	RemoteAddr   string    `json:"remote_addr"`
	Direction    string    `json:"direction"`
	FrameType    string    `json:"frame_type"`
	Opcode       byte      `json:"opcode"`
	FIN          bool      `json:"fin"`
	Masked       bool      `json:"masked"`
	PayloadLen   int       `json:"payload_length"`
	PayloadHex   string    `json:"payload_hex"`
	PayloadAscii string    `json:"payload_ascii"`
	RawFrameHex  string    `json:"raw_frame_hex"`
}

// SaveMessageToAnalysis saves a message to the analysis directory
// If analysisDir is empty, this function does nothing (logging disabled)
func SaveMessageToAnalysis(remoteAddr string, messageNum int, frame *protocol.Frame, analysisDir string) {
	// Skip if analysis logging is disabled
	if analysisDir == "" {
		return
	}

	// Create filename with timestamp
	timestamp := time.Now()
	filename := filepath.Join(analysisDir, fmt.Sprintf("capture-%s.jsonl",
		timestamp.Format("20060102-150405")))

	// Prepare message analysis record
	analysis := MessageAnalysis{
		Timestamp:    timestamp,
		MessageNum:   messageNum,
		RemoteAddr:   remoteAddr,
		Direction:    "device->server",
		FrameType:    frame.OpcodeString(),
		Opcode:       frame.Opcode,
		FIN:          frame.FIN,
		Masked:       frame.Masked,
		PayloadLen:   len(frame.Payload),
		PayloadHex:   hex.EncodeToString(frame.Payload),
		PayloadAscii: toASCII(frame.Payload),
		RawFrameHex:  hex.EncodeToString(frame.Raw),
	}

	// Append to JSONL file (JSON Lines format - one JSON object per line)
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logging.Error("Failed to open analysis file",
			zap.String("filename", filename),
			zap.Error(err),
		)
		return
	}
	defer func() { _ = f.Close() }()

	// Write JSON line
	data, err := json.Marshal(analysis)
	if err != nil {
		logging.Error("Failed to marshal message analysis",
			zap.Error(err),
		)
		return
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		logging.Error("Failed to write to analysis file",
			zap.String("filename", filename),
			zap.Error(err),
		)
		return
	}

	logging.Debug("Saved message to analysis file",
		zap.String("filename", filename),
		zap.Int("message_num", messageNum),
	)
}

// toASCII converts bytes to ASCII string (non-printable chars become '.')
func toASCII(data []byte) string {
	result := make([]byte, len(data))
	for i, b := range data {
		if b >= 32 && b <= 126 {
			result[i] = b
		} else {
			result[i] = '.'
		}
	}
	return string(result)
}

// SendMessage sends a protocol message to the device via WebSocket
//
// This function wraps the message in a WebSocket binary frame and sends it.
// It's designed to work with messages built by the protocol.constructor package.
//
// Parameters:
//   - conn: The network connection to write to
//   - remoteAddr: Remote address for logging
//   - message: The complete protocol frame (from constructor functions)
//
// Returns:
//   - error if send fails
//
// Example usage:
//
//	msg, err := protocol.BuildPressureModeSet(protocol.GenerateMessageID(), true)
//	if err != nil {
//	    return err
//	}
//	err = SendMessage(conn, remoteAddr, msg)
func SendMessage(conn net.Conn, remoteAddr string, message []byte) error {
	// Validate the message before sending
	if err := protocol.ValidateFrame(message); err != nil {
		logging.Error("Invalid message frame",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
		)
		return fmt.Errorf("invalid message frame: %w", err)
	}

	// Build WebSocket binary frame
	// Format: [FIN + opcode] [length] [payload]
	frame := buildWebSocketFrame(message)

	// Set write deadline
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		logging.Error("Failed to set write deadline",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
		)
		return err
	}

	// Send the frame
	n, err := conn.Write(frame)
	if err != nil {
		logging.Error("Failed to send message",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
		)
		return fmt.Errorf("write failed: %w", err)
	}

	// Log the sent message
	logging.Info("📤 Sent message to device",
		zap.String("remote_addr", remoteAddr),
		zap.Int("bytes_written", n),
		zap.Int("message_length", len(message)),
		zap.String("message_hex", hex.EncodeToString(message)),
	)

	// Validate it matches Ghidra structures
	if err := protocol.ValidateFrame(message); err == nil {
		logging.Debug("  ✓ Message validated successfully",
			zap.String("remote_addr", remoteAddr),
		)
	}

	return nil
}

// buildWebSocketFrame wraps payload in WebSocket binary frame (server-to-client, unmasked)
func buildWebSocketFrame(payload []byte) []byte {
	// WebSocket frame format (server-to-client, unmasked):
	// [FIN + opcode] [payload length] [payload]

	var frame []byte
	payloadLen := len(payload)

	// Byte 1: FIN (1) + RSV (0,0,0) + Opcode (0x2 = binary)
	// 0x82 = 10000010 = FIN set, binary frame
	frame = append(frame, 0x82)

	// Byte 2: MASK (0) + Payload length
	// Server-to-client frames are NOT masked
	if payloadLen < 126 {
		// Small payload: length fits in 7 bits
		frame = append(frame, byte(payloadLen))
	} else if payloadLen < 65536 {
		// Medium payload: use 16-bit length
		frame = append(frame, 126)
		frame = append(frame, byte(payloadLen>>8))
		frame = append(frame, byte(payloadLen&0xFF))
	} else {
		// Large payload: use 64-bit length
		frame = append(frame, 127)
		for i := 7; i >= 0; i-- {
			frame = append(frame, byte((payloadLen>>(i*8))&0xFF))
		}
	}

	// Append payload
	frame = append(frame, payload...)

	return frame
}
