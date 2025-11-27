//go:build integration

package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseProtocolFrame(t *testing.T) {
	tests := []struct {
		name    string
		frame   []byte
		wantErr bool
		verify  func(t *testing.T, pf *ProtocolFrame)
	}{
		{
			name: "valid telemetry response",
			frame: func() []byte {
				f := make([]byte, MinMessageSize)
				f[0] = ProtocolSync
				f[1] = ProtocolVersion
				binary.LittleEndian.PutUint32(f[2:6], 100) // Message ID
				binary.LittleEndian.PutUint16(f[6:8], 3)   // Payload length
				f[8] = MsgTypeTelemetryResponse
				f[9] = 0x11
				f[10] = 0x80
				return f
			}(),
			wantErr: false,
			verify: func(t *testing.T, pf *ProtocolFrame) {
				if pf.Sync != ProtocolSync {
					t.Errorf("sync = 0x%02x, want 0x%02x", pf.Sync, ProtocolSync)
				}
				if pf.Version != ProtocolVersion {
					t.Errorf("version = 0x%02x, want 0x%02x", pf.Version, ProtocolVersion)
				}
				if pf.MessageID != 100 {
					t.Errorf("messageID = %d, want 100", pf.MessageID)
				}
				if pf.Length != 3 {
					t.Errorf("length = %d, want 3", pf.Length)
				}
			},
		},
		{
			name:    "frame too small",
			frame:   make([]byte, 5),
			wantErr: true,
		},
		{
			name: "invalid sync byte",
			frame: func() []byte {
				f := make([]byte, MinMessageSize)
				f[0] = 0xFF // wrong
				f[1] = ProtocolVersion
				return f
			}(),
			wantErr: true,
		},
		{
			name: "invalid version",
			frame: func() []byte {
				f := make([]byte, MinMessageSize)
				f[0] = ProtocolSync
				f[1] = 0xFF // wrong
				return f
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf, err := ParseProtocolFrame(tt.frame)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseProtocolFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				tt.verify(t, pf)
			}
		})
	}
}

func TestProtocolFrame_ParseMessage(t *testing.T) {
	tests := []struct {
		name     string
		frame    *ProtocolFrame
		wantType byte
		wantErr  bool
	}{
		{
			name: "telemetry response message",
			frame: &ProtocolFrame{
				Sync:      ProtocolSync,
				Version:   ProtocolVersion,
				MessageID: 1,
				Length:    3,
				Payload:   []byte{MsgTypeTelemetryResponse, 0x11, 0x80},
			},
			wantType: MsgTypeTelemetryResponse,
			wantErr:  false,
		},
		{
			name: "pressure mode message",
			frame: &ProtocolFrame{
				Sync:      ProtocolSync,
				Version:   ProtocolVersion,
				MessageID: 2,
				Length:    3,
				Payload:   []byte{MsgTypePressureMode, 0x04, 0x01},
			},
			wantType: MsgTypePressureMode,
			wantErr:  false,
		},
		{
			name: "command message",
			frame: &ProtocolFrame{
				Sync:      ProtocolSync,
				Version:   ProtocolVersion,
				MessageID: 3,
				Length:    6,
				Payload:   []byte{MsgTypeCommand, 0x05, 0x01, 0x34, 0x12, 0x00},
			},
			wantType: MsgTypeCommand,
			wantErr:  false,
		},
		{
			name: "unknown message type",
			frame: &ProtocolFrame{
				Sync:      ProtocolSync,
				Version:   ProtocolVersion,
				MessageID: 4,
				Length:    1,
				Payload:   []byte{0xFF},
			},
			wantType: 0xFF,
			wantErr:  false, // Unknown messages don't error, just return UnknownMessage
		},
		{
			name: "empty payload",
			frame: &ProtocolFrame{
				Sync:      ProtocolSync,
				Version:   ProtocolVersion,
				MessageID: 5,
				Length:    0,
				Payload:   []byte{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := tt.frame.ParseMessage()

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && msg.Type() != tt.wantType {
				t.Errorf("message type = 0x%02x, want 0x%02x", msg.Type(), tt.wantType)
			}
		})
	}
}

func TestIsDualValveMessage(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{
			name: "valid 77-byte dual valve message",
			data: func() []byte {
				msg := make([]byte, 77)
				msg[0] = 0x01  // Cold valve ID
				msg[38] = 0x02 // Hot valve ID
				msg[76] = 0x0a // Terminator
				return msg
			}(),
			want: true,
		},
		{
			name: "too short",
			data: make([]byte, 76),
			want: false,
		},
		{
			name: "too long",
			data: make([]byte, 78),
			want: false,
		},
		{
			name: "wrong cold valve ID",
			data: func() []byte {
				msg := make([]byte, 77)
				msg[0] = 0xFF // Wrong
				msg[38] = 0x02
				msg[76] = 0x0a
				return msg
			}(),
			want: false,
		},
		{
			name: "wrong hot valve ID",
			data: func() []byte {
				msg := make([]byte, 77)
				msg[0] = 0x01
				msg[38] = 0xFF // Wrong
				msg[76] = 0x0a
				return msg
			}(),
			want: false,
		},
		{
			name: "wrong terminator",
			data: func() []byte {
				msg := make([]byte, 77)
				msg[0] = 0x01
				msg[38] = 0x02
				msg[76] = 0xFF // Wrong
				return msg
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDualValveMessage(tt.data)
			if got != tt.want {
				t.Errorf("IsDualValveMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDualValveMessage(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		verify  func(t *testing.T, msg *DualValveMessage)
	}{
		{
			name: "valid dual valve message",
			data: func() []byte {
				msg := make([]byte, 77)
				msg[0] = 0x01  // Cold valve ID
				msg[38] = 0x02 // Hot valve ID
				msg[76] = 0x0a // Terminator
				return msg
			}(),
			wantErr: false,
			verify: func(t *testing.T, msg *DualValveMessage) {
				if msg.ColdValve.ValveID != 0x01 {
					t.Errorf("cold valve ID = 0x%02x, want 0x01", msg.ColdValve.ValveID)
				}
				if msg.HotValve.ValveID != 0x02 {
					t.Errorf("hot valve ID = 0x%02x, want 0x02", msg.HotValve.ValveID)
				}
			},
		},
		{
			name:    "invalid message (too short)",
			data:    make([]byte, 50),
			wantErr: true,
		},
		{
			name: "invalid message (wrong IDs)",
			data: func() []byte {
				msg := make([]byte, 77)
				msg[0] = 0xFF  // Wrong
				msg[38] = 0xFF // Wrong
				msg[76] = 0x0a
				return msg
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseDualValveMessage(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDualValveMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				tt.verify(t, msg)
			}
		})
	}
}

func TestMessage_String(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{
			name: "telemetry response",
			msg: &TelemetryResponseMessage{
				MessageType: MsgTypeTelemetryResponse,
				Subtype:     0x11,
				Field:       0x80,
				Value:       0x12345678,
			},
		},
		{
			name: "command message",
			msg: &CommandMessage{
				MessageType: MsgTypeCommand,
				PayloadLen:  5,
				Field1:      0x01,
				Param1:      0x1234,
				Data:        []byte{0x01, 0x02},
			},
		},
		{
			name: "pressure mode message",
			msg: &PressureModeMessage{
				MessageType: MsgTypePressureMode,
				Subtype:     0x04,
				Enabled:     0x01,
			},
		},
		{
			name: "unknown message",
			msg: &UnknownMessage{
				MessageType: 0xFF,
				Data:        []byte{0xFF, 0x01, 0x02},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify String() doesn't panic and returns non-empty
			s := tt.msg.String()
			if s == "" {
				t.Error("String() returned empty string")
			}
			// Verify it contains the message type
			if tt.msg.Type() != 0xFF && !bytes.Contains([]byte(s), []byte("0x")) {
				t.Error("String() should contain hex representation")
			}
		})
	}
}

func TestProtocolFrame_String(t *testing.T) {
	pf := &ProtocolFrame{
		Sync:      ProtocolSync,
		Version:   ProtocolVersion,
		MessageID: 100,
		Length:    5,
		Payload:   []byte{0x01, 0x02, 0x03, 0x04, 0x05},
	}

	s := pf.String()
	if s == "" {
		t.Error("String() returned empty string")
	}

	// Verify it contains key information
	if !bytes.Contains([]byte(s), []byte("0x7e")) {
		t.Error("String() should contain sync byte")
	}
	if !bytes.Contains([]byte(s), []byte("100")) {
		t.Error("String() should contain message ID")
	}
}

// Benchmark tests
func BenchmarkParseProtocolFrame(b *testing.B) {
	frame := make([]byte, MinMessageSize)
	frame[0] = ProtocolSync
	frame[1] = ProtocolVersion
	binary.LittleEndian.PutUint32(frame[2:6], 1)
	binary.LittleEndian.PutUint16(frame[6:8], 3)
	frame[8] = MsgTypeTelemetryResponse
	frame[9] = 0x11
	frame[10] = 0x80

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseProtocolFrame(frame)
	}
}

func BenchmarkIsDualValveMessage(b *testing.B) {
	data := make([]byte, 77)
	data[0] = 0x01
	data[38] = 0x02
	data[76] = 0x0a

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsDualValveMessage(data)
	}
}
