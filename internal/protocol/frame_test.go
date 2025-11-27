//go:build integration

package protocol

import (
	"bytes"
	"io"
	"testing"
)

func TestReadFrame(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
		verify  func(t *testing.T, frame *Frame)
	}{
		{
			name: "simple unmasked text frame",
			data: []byte{
				0x81, // FIN + text opcode
				0x05, // No mask, 5 byte payload
				'H', 'e', 'l', 'l', 'o',
			},
			wantErr: false,
			verify: func(t *testing.T, frame *Frame) {
				if !frame.FIN {
					t.Error("FIN should be true")
				}
				if frame.Opcode != OpcodeText {
					t.Errorf("opcode = 0x%02x, want 0x%02x (text)", frame.Opcode, OpcodeText)
				}
				if frame.Masked {
					t.Error("masked should be false")
				}
				if !bytes.Equal(frame.Payload, []byte("Hello")) {
					t.Errorf("payload = %v, want 'Hello'", frame.Payload)
				}
			},
		},
		{
			name: "masked binary frame",
			data: func() []byte {
				payload := []byte{0x01, 0x02, 0x03}
				maskKey := [4]byte{0xAA, 0xBB, 0xCC, 0xDD}
				masked := make([]byte, len(payload))
				for i := range payload {
					masked[i] = payload[i] ^ maskKey[i%4]
				}
				return append([]byte{
					0x82, // FIN + binary opcode
					0x83, // Mask bit + 3 byte payload
					maskKey[0], maskKey[1], maskKey[2], maskKey[3],
				}, masked...)
			}(),
			wantErr: false,
			verify: func(t *testing.T, frame *Frame) {
				if !frame.FIN {
					t.Error("FIN should be true")
				}
				if frame.Opcode != OpcodeBinary {
					t.Errorf("opcode = 0x%02x, want 0x%02x (binary)", frame.Opcode, OpcodeBinary)
				}
				if !frame.Masked {
					t.Error("masked should be true")
				}
				expected := []byte{0x01, 0x02, 0x03}
				if !bytes.Equal(frame.Payload, expected) {
					t.Errorf("payload = %v, want %v", frame.Payload, expected)
				}
			},
		},
		{
			name: "close frame",
			data: []byte{
				0x88, // FIN + close opcode
				0x00, // No payload
			},
			wantErr: false,
			verify: func(t *testing.T, frame *Frame) {
				if frame.Opcode != OpcodeClose {
					t.Errorf("opcode = 0x%02x, want 0x%02x (close)", frame.Opcode, OpcodeClose)
				}
				if len(frame.Payload) != 0 {
					t.Errorf("payload length = %d, want 0", len(frame.Payload))
				}
			},
		},
		{
			name: "ping frame",
			data: []byte{
				0x89, // FIN + ping opcode
				0x00, // No payload
			},
			wantErr: false,
			verify: func(t *testing.T, frame *Frame) {
				if frame.Opcode != OpcodePing {
					t.Errorf("opcode = 0x%02x, want 0x%02x (ping)", frame.Opcode, OpcodePing)
				}
			},
		},
		{
			name: "pong frame",
			data: []byte{
				0x8A, // FIN + pong opcode
				0x00, // No payload
			},
			wantErr: false,
			verify: func(t *testing.T, frame *Frame) {
				if frame.Opcode != OpcodePong {
					t.Errorf("opcode = 0x%02x, want 0x%02x (pong)", frame.Opcode, OpcodePong)
				}
			},
		},
		{
			name: "frame with extended payload length (16-bit)",
			data: func() []byte {
				payloadSize := 126
				payload := make([]byte, payloadSize)
				for i := range payload {
					payload[i] = byte(i % 256)
				}
				return append([]byte{
					0x82,                     // FIN + binary
					0x7E,                     // 126 = use next 2 bytes for length
					byte(payloadSize >> 8),   // Length high byte
					byte(payloadSize & 0xFF), // Length low byte
				}, payload...)
			}(),
			wantErr: false,
			verify: func(t *testing.T, frame *Frame) {
				if len(frame.Payload) != 126 {
					t.Errorf("payload length = %d, want 126", len(frame.Payload))
				}
			},
		},
		{
			name:    "incomplete frame (truncated header)",
			data:    []byte{0x81},
			wantErr: true,
		},
		{
			name: "incomplete frame (truncated payload)",
			data: []byte{
				0x81,     // FIN + text
				0x05,     // 5 byte payload
				'H', 'i', // Only 2 bytes instead of 5
			},
			wantErr: true,
		},
		{
			name: "incomplete masked frame (missing mask key)",
			data: []byte{
				0x82, // FIN + binary
				0x83, // Mask bit + 3 byte payload
				// Missing 4-byte mask key and payload
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(tt.data)
			frame, err := ReadFrame(r)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verify != nil {
				tt.verify(t, frame)
			}
		})
	}
}

func TestUnmaskPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		maskKey [4]byte
		want    []byte
	}{
		{
			name:    "simple unmasking",
			payload: []byte{0xAB, 0xBA, 0xCD, 0xDC},
			maskKey: [4]byte{0xAA, 0xBB, 0xCC, 0xDD},
			want:    []byte{0x01, 0x01, 0x01, 0x01},
		},
		{
			name:    "empty payload",
			payload: []byte{},
			maskKey: [4]byte{0x01, 0x02, 0x03, 0x04},
			want:    []byte{},
		},
		{
			name:    "payload longer than mask key",
			payload: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			maskKey: [4]byte{0x01, 0x01, 0x01, 0x01},
			want:    []byte{0x00, 0x03, 0x02, 0x05, 0x04, 0x07, 0x06, 0x09},
		},
		{
			name:    "all zero mask (no-op)",
			payload: []byte{0x11, 0x22, 0x33},
			maskKey: [4]byte{0x00, 0x00, 0x00, 0x00},
			want:    []byte{0x11, 0x22, 0x33},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// unmaskPayload modifies in place, so make a copy
			payload := make([]byte, len(tt.payload))
			copy(payload, tt.payload)

			got := unmaskPayload(payload, tt.maskKey)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("unmaskPayload() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFrame_OpcodeString(t *testing.T) {
	tests := []struct {
		opcode byte
		want   string
	}{
		{OpcodeText, "Text"},
		{OpcodeBinary, "Binary"},
		{OpcodeClose, "Close"},
		{OpcodePing, "Ping"},
		{OpcodePong, "Pong"},
		{0x05, "Unknown(0x05)"},
		{0xFF, "Unknown(0xff)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			frame := &Frame{Opcode: tt.opcode}
			got := frame.OpcodeString()
			if got != tt.want {
				t.Errorf("OpcodeString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFrame_String(t *testing.T) {
	frame := &Frame{
		FIN:     true,
		Opcode:  OpcodeBinary,
		Masked:  true,
		MaskKey: [4]byte{0xAA, 0xBB, 0xCC, 0xDD},
		Payload: []byte{0x01, 0x02, 0x03},
	}

	s := frame.String()
	if s == "" {
		t.Error("String() returned empty string")
	}

	// Verify it contains key information
	if !bytes.Contains([]byte(s), []byte("FIN=true")) {
		t.Error("String() should contain FIN flag")
	}
	if !bytes.Contains([]byte(s), []byte("Binary")) {
		t.Error("String() should contain opcode string")
	}
	if !bytes.Contains([]byte(s), []byte("Masked=true")) {
		t.Error("String() should contain masked flag")
	}
}

func TestReadFrame_EdgeCases(t *testing.T) {
	t.Run("EOF on first byte", func(t *testing.T) {
		r := bytes.NewReader([]byte{})
		_, err := ReadFrame(r)
		if err != io.EOF {
			t.Errorf("expected io.EOF, got %v", err)
		}
	})

	t.Run("fragmented frame (FIN=false)", func(t *testing.T) {
		data := []byte{
			0x01, // FIN=false, text opcode
			0x05, // 5 byte payload
			'H', 'e', 'l', 'l', 'o',
		}
		r := bytes.NewReader(data)
		frame, err := ReadFrame(r)
		if err != nil {
			t.Fatalf("ReadFrame() error = %v", err)
		}
		if frame.FIN {
			t.Error("FIN should be false for fragmented frame")
		}
	})

	t.Run("reserved bits set", func(t *testing.T) {
		data := []byte{
			0xF1, // FIN + all reserved bits + text opcode
			0x00, // No payload
		}
		r := bytes.NewReader(data)
		frame, err := ReadFrame(r)
		if err != nil {
			t.Fatalf("ReadFrame() error = %v", err)
		}
		if !frame.FIN {
			t.Error("FIN should be true")
		}
	})
}

// Benchmark tests
func BenchmarkReadFrame(b *testing.B) {
	data := []byte{
		0x82, // FIN + binary
		0x05, // 5 byte payload
		0x01, 0x02, 0x03, 0x04, 0x05,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(data)
		ReadFrame(r)
	}
}

func BenchmarkUnmaskPayload(b *testing.B) {
	payload := make([]byte, 1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}
	maskKey := [4]byte{0xAA, 0xBB, 0xCC, 0xDD}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy since unmaskPayload modifies in place
		p := make([]byte, len(payload))
		copy(p, payload)
		unmaskPayload(p, maskKey)
	}
}
