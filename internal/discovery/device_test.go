package discovery

import (
	"testing"
	"time"
)

func TestDevice_String(t *testing.T) {
	device := &Device{
		Serial:   "315260240",
		Hostname: "eValve315260240.local",
		IP:       "192.168.4.16",
		Port:     80,
	}

	expected := "Smartap Device 315260240 (eValve315260240.local) at 192.168.4.16:80"
	if device.String() != expected {
		t.Errorf("Device.String() = %v, want %v", device.String(), expected)
	}
}

func TestDevice_BaseURL(t *testing.T) {
	tests := []struct {
		name     string
		device   *Device
		expected string
	}{
		{
			name: "standard HTTP port",
			device: &Device{
				IP:   "192.168.4.16",
				Port: 80,
			},
			expected: "http://192.168.4.16:80",
		},
		{
			name: "custom port",
			device: &Device{
				IP:   "10.0.0.5",
				Port: 8080,
			},
			expected: "http://10.0.0.5:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.device.BaseURL(); got != tt.expected {
				t.Errorf("Device.BaseURL() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDevice_GetMetadata(t *testing.T) {
	device := &Device{
		Metadata: map[string]string{
			"path":    "/",
			"srcvers": "1D90645",
		},
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "existing key",
			key:      "path",
			expected: "/",
		},
		{
			name:     "another existing key",
			key:      "srcvers",
			expected: "1D90645",
		},
		{
			name:     "non-existent key",
			key:      "missing",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := device.GetMetadata(tt.key); got != tt.expected {
				t.Errorf("Device.GetMetadata(%v) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestDevice_GetMetadata_NilMap(t *testing.T) {
	device := &Device{
		Metadata: nil,
	}

	if got := device.GetMetadata("anything"); got != "" {
		t.Errorf("Device.GetMetadata() with nil map = %v, want empty string", got)
	}
}

func TestDevice_DiscoveredAt(t *testing.T) {
	now := time.Now()
	device := &Device{
		Serial:       "315260240",
		DiscoveredAt: now,
	}

	if device.DiscoveredAt != now {
		t.Errorf("Device.DiscoveredAt = %v, want %v", device.DiscoveredAt, now)
	}
}
