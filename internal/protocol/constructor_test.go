package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBuildProtocolFrame(t *testing.T) {
	tests := []struct {
		name        string
		messageID   uint32
		payload     []byte
		wantErr     bool
		checkFields func(t *testing.T, frame []byte)
	}{
		{
			name:      "empty payload with padding",
			messageID: 0x12345678,
			payload:   []byte{},
			wantErr:   false,
			checkFields: func(t *testing.T, frame []byte) {
				// Should be padded to 38 bytes
				if len(frame) != MinMessageSize {
					t.Errorf("frame size = %d, want %d", len(frame), MinMessageSize)
				}

				// Check sync and version
				if frame[0] != ProtocolSync {
					t.Errorf("sync byte = 0x%02x, want 0x%02x", frame[0], ProtocolSync)
				}
				if frame[1] != ProtocolVersion {
					t.Errorf("version = 0x%02x, want 0x%02x", frame[1], ProtocolVersion)
				}

				// Check message ID (little-endian)
				gotID := binary.LittleEndian.Uint32(frame[2:6])
				if gotID != 0x12345678 {
					t.Errorf("message ID = 0x%08x, want 0x12345678", gotID)
				}

				// Check length
				gotLen := binary.LittleEndian.Uint16(frame[6:8])
				if gotLen != 0 {
					t.Errorf("length = %d, want 0", gotLen)
				}

				// Check padding is zeros
				for i := 8; i < len(frame); i++ {
					if frame[i] != 0 {
						t.Errorf("padding byte %d = 0x%02x, want 0x00", i, frame[i])
					}
				}
			},
		},
		{
			name:      "small payload requires padding",
			messageID: 0xAABBCCDD,
			payload:   []byte{0x01, 0x02, 0x03},
			wantErr:   false,
			checkFields: func(t *testing.T, frame []byte) {
				// Should be padded to 38 bytes
				if len(frame) != MinMessageSize {
					t.Errorf("frame size = %d, want %d", len(frame), MinMessageSize)
				}

				// Check payload length field
				gotLen := binary.LittleEndian.Uint16(frame[6:8])
				if gotLen != 3 {
					t.Errorf("length = %d, want 3", gotLen)
				}

				// Check payload bytes
				if !bytes.Equal(frame[8:11], []byte{0x01, 0x02, 0x03}) {
					t.Errorf("payload = %v, want [0x01 0x02 0x03]", frame[8:11])
				}

				// Check padding
				for i := 11; i < len(frame); i++ {
					if frame[i] != 0 {
						t.Errorf("padding byte %d = 0x%02x, want 0x00", i, frame[i])
					}
				}
			},
		},
		{
			name:      "large payload no padding needed",
			messageID: 0x11223344,
			payload:   make([]byte, 100), // 100 bytes payload
			wantErr:   false,
			checkFields: func(t *testing.T, frame []byte) {
				// 8 byte header + 100 byte payload = 108 bytes
				if len(frame) != 108 {
					t.Errorf("frame size = %d, want 108", len(frame))
				}

				// Check length field
				gotLen := binary.LittleEndian.Uint16(frame[6:8])
				if gotLen != 100 {
					t.Errorf("length = %d, want 100", gotLen)
				}
			},
		},
		{
			name:      "maximum valid payload",
			messageID: 1,
			payload:   make([]byte, MaxPayloadSize),
			wantErr:   false,
			checkFields: func(t *testing.T, frame []byte) {
				if len(frame) != 8+MaxPayloadSize {
					t.Errorf("frame size = %d, want %d", len(frame), 8+MaxPayloadSize)
				}
			},
		},
		{
			name:      "payload too large",
			messageID: 1,
			payload:   make([]byte, MaxPayloadSize+1),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := BuildProtocolFrame(tt.messageID, tt.payload)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildProtocolFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFields != nil {
				tt.checkFields(t, frame)
			}
		})
	}
}

func TestBuildCommandMessage(t *testing.T) {
	tests := []struct {
		name      string
		messageID uint32
		category  uint32
		data      []byte
		wantErr   bool
		verify    func(t *testing.T, frame []byte)
	}{
		{
			name:      "command with no data",
			messageID: 1,
			category:  0x1234,
			data:      []byte{},
			wantErr:   false,
			verify: func(t *testing.T, frame []byte) {
				// Validate frame
				if err := ValidateFrame(frame); err != nil {
					t.Errorf("invalid frame: %v", err)
				}

				// Check message type
				if frame[8] != MsgTypeCommand {
					t.Errorf("message type = 0x%02x, want 0x%02x", frame[8], MsgTypeCommand)
				}

				// Check length field (should be 5 for empty data)
				if frame[9] != 5 {
					t.Errorf("length field = %d, want 5", frame[9])
				}

				// Check marker
				if frame[10] != 0x01 {
					t.Errorf("marker = 0x%02x, want 0x01", frame[10])
				}

				// Check category
				gotCat := binary.LittleEndian.Uint32(frame[11:15])
				if gotCat != 0x1234 {
					t.Errorf("category = 0x%08x, want 0x00001234", gotCat)
				}
			},
		},
		{
			name:      "command with data",
			messageID: 2,
			category:  0xAABBCCDD,
			data:      []byte{0xFF, 0xEE, 0xDD},
			wantErr:   false,
			verify: func(t *testing.T, frame []byte) {
				// Check length field (should be len(data) + 5 = 8)
				if frame[9] != 8 {
					t.Errorf("length field = %d, want 8", frame[9])
				}

				// Check category
				gotCat := binary.LittleEndian.Uint32(frame[11:15])
				if gotCat != 0xAABBCCDD {
					t.Errorf("category = 0x%08x, want 0xAABBCCDD", gotCat)
				}

				// Check data
				if !bytes.Equal(frame[15:18], []byte{0xFF, 0xEE, 0xDD}) {
					t.Errorf("data = %v, want [0xFF 0xEE 0xDD]", frame[15:18])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := BuildCommandMessage(tt.messageID, tt.category, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("BuildCommandMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				tt.verify(t, frame)
			}
		})
	}
}

func TestBuildTelemetryQuery(t *testing.T) {
	tests := []struct {
		name      string
		messageID uint32
		queryType uint8
		verify    func(t *testing.T, frame []byte)
	}{
		{
			name:      "basic telemetry query",
			messageID: 100,
			queryType: 0x80,
			verify: func(t *testing.T, frame []byte) {
				// Validate frame
				if err := ValidateFrame(frame); err != nil {
					t.Errorf("invalid frame: %v", err)
				}

				// Check message type
				if frame[8] != MsgTypeTelemetryResponse {
					t.Errorf("message type = 0x%02x, want 0x%02x", frame[8], MsgTypeTelemetryResponse)
				}

				// Check subtype
				if frame[9] != 0x11 {
					t.Errorf("subtype = 0x%02x, want 0x11", frame[9])
				}

				// Check query type
				if frame[10] != 0x80 {
					t.Errorf("query type = 0x%02x, want 0x80", frame[10])
				}

				// Check padding is zeros
				for i := 11; i < 8+19; i++ {
					if frame[i] != 0 {
						t.Errorf("byte %d = 0x%02x, want 0x00", i, frame[i])
					}
				}
			},
		},
		{
			name:      "different query type",
			messageID: 200,
			queryType: 0x42,
			verify: func(t *testing.T, frame []byte) {
				if frame[10] != 0x42 {
					t.Errorf("query type = 0x%02x, want 0x42", frame[10])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := BuildTelemetryQuery(tt.messageID, tt.queryType)
			if err != nil {
				t.Errorf("BuildTelemetryQuery() error = %v", err)
				return
			}

			if tt.verify != nil {
				tt.verify(t, frame)
			}
		})
	}
}

func TestBuildPressureModeSet(t *testing.T) {
	tests := []struct {
		name      string
		messageID uint32
		enabled   bool
		wantValue byte
	}{
		{
			name:      "enable pressure mode",
			messageID: 1,
			enabled:   true,
			wantValue: 0x01,
		},
		{
			name:      "disable pressure mode",
			messageID: 2,
			enabled:   false,
			wantValue: 0x00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := BuildPressureModeSet(tt.messageID, tt.enabled)
			if err != nil {
				t.Errorf("BuildPressureModeSet() error = %v", err)
				return
			}

			// Validate frame
			if err := ValidateFrame(frame); err != nil {
				t.Errorf("invalid frame: %v", err)
			}

			// Check message type
			if frame[8] != MsgTypePressureMode {
				t.Errorf("message type = 0x%02x, want 0x%02x", frame[8], MsgTypePressureMode)
			}

			// Check subtype
			if frame[9] != 0x04 {
				t.Errorf("subtype = 0x%02x, want 0x04", frame[9])
			}

			// Check value
			if frame[10] != tt.wantValue {
				t.Errorf("value = 0x%02x, want 0x%02x", frame[10], tt.wantValue)
			}
		})
	}
}

func TestGenerateMessageID(t *testing.T) {
	// Generate 1000 message IDs
	ids := make(map[uint32]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateMessageID()

		// Check not reserved
		if id == MsgIDBroadcastReserved {
			t.Errorf("generated reserved ID: 0x%08x", id)
		}

		// Check not zero
		if id == 0 {
			t.Errorf("generated zero ID")
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("duplicate ID generated: 0x%08x", id)
		}
		ids[id] = true
	}

	// Should have generated 1000 unique IDs
	if len(ids) != 1000 {
		t.Errorf("generated %d unique IDs, want 1000", len(ids))
	}
}

func TestValidateFrame(t *testing.T) {
	tests := []struct {
		name    string
		frame   []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "too small",
			frame:   make([]byte, 20),
			wantErr: true,
			errMsg:  "frame too small",
		},
		{
			name: "invalid sync byte",
			frame: func() []byte {
				f := make([]byte, MinMessageSize)
				f[0] = 0xFF // wrong sync
				f[1] = ProtocolVersion
				return f
			}(),
			wantErr: true,
			errMsg:  "invalid sync byte",
		},
		{
			name: "invalid version",
			frame: func() []byte {
				f := make([]byte, MinMessageSize)
				f[0] = ProtocolSync
				f[1] = 0xFF // wrong version
				return f
			}(),
			wantErr: true,
			errMsg:  "invalid version",
		},
		{
			name: "valid empty frame",
			frame: func() []byte {
				f := make([]byte, MinMessageSize)
				f[0] = ProtocolSync
				f[1] = ProtocolVersion
				binary.LittleEndian.PutUint32(f[2:6], 1)
				binary.LittleEndian.PutUint16(f[6:8], 0)
				return f
			}(),
			wantErr: false,
		},
		{
			name: "valid command frame",
			frame: func() []byte {
				frame, _ := BuildCommandMessage(1, 0x1234, []byte{0x01})
				return frame
			}(),
			wantErr: false,
		},
		{
			name: "unknown message type",
			frame: func() []byte {
				f := make([]byte, MinMessageSize)
				f[0] = ProtocolSync
				f[1] = ProtocolVersion
				binary.LittleEndian.PutUint32(f[2:6], 1)
				binary.LittleEndian.PutUint16(f[6:8], 1)
				f[8] = 0xFF // unknown type
				return f
			}(),
			wantErr: true,
			errMsg:  "unknown message type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFrame(tt.frame)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !contains(err.Error(), tt.errMsg) {
					t.Errorf("error message = %v, want to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestCalculateHeaderChecksum(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   uint8
	}{
		{
			name:   "all zeros",
			header: make([]byte, 8),
			want:   3, // sum + 3
		},
		{
			name:   "sample header",
			header: []byte{0x7e, 0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00},
			want:   uint8((0x7e + 0x03 + 0x01 + 3) & 0xFF),
		},
		{
			name:   "header too short",
			header: []byte{0x7e, 0x03},
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateHeaderChecksum(tt.header)
			if got != tt.want {
				t.Errorf("CalculateHeaderChecksum() = 0x%02x, want 0x%02x", got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkBuildProtocolFrame(b *testing.B) {
	payload := make([]byte, 19)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildProtocolFrame(uint32(i), payload)
	}
}

func BenchmarkBuildCommandMessage(b *testing.B) {
	data := []byte{0x01, 0x02, 0x03}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildCommandMessage(uint32(i), 0x1234, data)
	}
}

func BenchmarkGenerateMessageID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateMessageID()
	}
}
