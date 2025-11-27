package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// WebSocket frame opcodes
const (
	OpcodeContinuation = 0x0
	OpcodeText         = 0x1
	OpcodeBinary       = 0x2
	OpcodeClose        = 0x8
	OpcodePing         = 0x9
	OpcodePong         = 0xA
)

// Frame represents a WebSocket frame
type Frame struct {
	FIN     bool
	RSV1    bool
	RSV2    bool
	RSV3    bool
	Opcode  byte
	Masked  bool
	Length  uint64
	MaskKey [4]byte
	Payload []byte
	Raw     []byte // Original frame bytes for debugging
}

// ReadFrame reads a WebSocket frame from the reader
func ReadFrame(r io.Reader) (*Frame, error) {
	frame := &Frame{}

	// Read first two bytes
	header := make([]byte, 2)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("failed to read frame header: %w", err)
	}

	frame.Raw = append(frame.Raw, header...)

	// Parse first byte: FIN, RSV1-3, Opcode
	frame.FIN = (header[0] & 0x80) != 0
	frame.RSV1 = (header[0] & 0x40) != 0
	frame.RSV2 = (header[0] & 0x20) != 0
	frame.RSV3 = (header[0] & 0x10) != 0
	frame.Opcode = header[0] & 0x0F

	// Parse second byte: Mask, Payload length
	frame.Masked = (header[1] & 0x80) != 0
	payloadLen := uint64(header[1] & 0x7F)

	// Extended payload length
	if payloadLen == 126 {
		extLen := make([]byte, 2)
		if _, err := io.ReadFull(r, extLen); err != nil {
			return nil, fmt.Errorf("failed to read extended length: %w", err)
		}
		frame.Raw = append(frame.Raw, extLen...)
		frame.Length = uint64(binary.BigEndian.Uint16(extLen))
	} else if payloadLen == 127 {
		extLen := make([]byte, 8)
		if _, err := io.ReadFull(r, extLen); err != nil {
			return nil, fmt.Errorf("failed to read extended length: %w", err)
		}
		frame.Raw = append(frame.Raw, extLen...)
		frame.Length = binary.BigEndian.Uint64(extLen)
	} else {
		frame.Length = payloadLen
	}

	// Read mask key if present (client-to-server frames must be masked)
	if frame.Masked {
		if _, err := io.ReadFull(r, frame.MaskKey[:]); err != nil {
			return nil, fmt.Errorf("failed to read mask key: %w", err)
		}
		frame.Raw = append(frame.Raw, frame.MaskKey[:]...)
	}

	// Read payload
	if frame.Length > 0 {
		payload := make([]byte, frame.Length)
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, fmt.Errorf("failed to read payload: %w", err)
		}
		frame.Raw = append(frame.Raw, payload...)

		// Unmask payload if masked
		if frame.Masked {
			frame.Payload = unmaskPayload(payload, frame.MaskKey)
		} else {
			frame.Payload = payload
		}
	}

	return frame, nil
}

// unmaskPayload applies XOR mask to payload (WebSocket unmasking algorithm)
func unmaskPayload(payload []byte, maskKey [4]byte) []byte {
	unmasked := make([]byte, len(payload))
	for i := 0; i < len(payload); i++ {
		unmasked[i] = payload[i] ^ maskKey[i%4]
	}
	return unmasked
}

// OpcodeString returns a human-readable opcode name
func (f *Frame) OpcodeString() string {
	switch f.Opcode {
	case OpcodeContinuation:
		return "continuation"
	case OpcodeText:
		return "text"
	case OpcodeBinary:
		return "binary"
	case OpcodeClose:
		return "close"
	case OpcodePing:
		return "ping"
	case OpcodePong:
		return "pong"
	default:
		return fmt.Sprintf("unknown(0x%X)", f.Opcode)
	}
}

// String returns a debug representation of the frame
func (f *Frame) String() string {
	return fmt.Sprintf("Frame{FIN=%v, Opcode=%s, Masked=%v, Length=%d}",
		f.FIN, f.OpcodeString(), f.Masked, f.Length)
}
