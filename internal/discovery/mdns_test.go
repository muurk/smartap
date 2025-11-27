package discovery

import (
	"net"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"
)

func TestScanner_parseServiceEntry(t *testing.T) {
	scanner := NewScanner()

	tests := []struct {
		name       string
		entry      *zeroconf.ServiceEntry
		wantNil    bool
		wantSerial string
		wantIP     string
		wantPort   int
	}{
		{
			name: "valid Smartap device with IPv4",
			entry: &zeroconf.ServiceEntry{
				HostName: "eValve315260240.local.",
				Port:     80,
				AddrIPv4: []net.IP{net.ParseIP("192.168.4.16")},
				Text:     []string{"path=/", "srcvers=1D90645"},
			},
			wantNil:    false,
			wantSerial: "315260240",
			wantIP:     "192.168.4.16",
			wantPort:   80,
		},
		{
			name: "valid Smartap device without trailing dot",
			entry: &zeroconf.ServiceEntry{
				HostName: "eValve123456789.local",
				Port:     80,
				AddrIPv4: []net.IP{net.ParseIP("10.0.0.5")},
				Text:     []string{},
			},
			wantNil:    false,
			wantSerial: "123456789",
			wantIP:     "10.0.0.5",
			wantPort:   80,
		},
		{
			name: "valid device with custom port",
			entry: &zeroconf.ServiceEntry{
				HostName: "eValve999999999.local",
				Port:     8080,
				AddrIPv4: []net.IP{net.ParseIP("192.168.1.100")},
			},
			wantNil:    false,
			wantSerial: "999999999",
			wantIP:     "192.168.1.100",
			wantPort:   8080,
		},
		{
			name: "device with no port specified (should default to 80)",
			entry: &zeroconf.ServiceEntry{
				HostName: "eValve111111111.local",
				Port:     0,
				AddrIPv4: []net.IP{net.ParseIP("172.16.0.1")},
			},
			wantNil:    false,
			wantSerial: "111111111",
			wantIP:     "172.16.0.1",
			wantPort:   80,
		},
		{
			name: "non-Smartap device (wrong hostname pattern)",
			entry: &zeroconf.ServiceEntry{
				HostName: "someotherdevice.local",
				Port:     80,
				AddrIPv4: []net.IP{net.ParseIP("192.168.1.1")},
			},
			wantNil: true,
		},
		{
			name: "empty hostname",
			entry: &zeroconf.ServiceEntry{
				HostName: "",
				Port:     80,
				AddrIPv4: []net.IP{net.ParseIP("192.168.1.1")},
			},
			wantNil: true,
		},
		{
			name: "no IP address",
			entry: &zeroconf.ServiceEntry{
				HostName: "eValve315260240.local",
				Port:     80,
				AddrIPv4: []net.IP{},
				AddrIPv6: []net.IP{},
			},
			wantNil: true,
		},
		{
			name: "IPv6 only device",
			entry: &zeroconf.ServiceEntry{
				HostName: "eValve222222222.local",
				Port:     80,
				AddrIPv6: []net.IP{net.ParseIP("fe80::1")},
			},
			wantNil:    false,
			wantSerial: "222222222",
			wantIP:     "fe80::1",
			wantPort:   80,
		},
		{
			name: "device with both IPv4 and IPv6 (should prefer IPv4)",
			entry: &zeroconf.ServiceEntry{
				HostName: "eValve333333333.local",
				Port:     80,
				AddrIPv4: []net.IP{net.ParseIP("192.168.1.50")},
				AddrIPv6: []net.IP{net.ParseIP("fe80::2")},
			},
			wantNil:    false,
			wantSerial: "333333333",
			wantIP:     "192.168.1.50",
			wantPort:   80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := scanner.parseServiceEntry(tt.entry)

			if tt.wantNil {
				if device != nil {
					t.Errorf("parseServiceEntry() = %v, want nil", device)
				}
				return
			}

			if device == nil {
				t.Fatal("parseServiceEntry() = nil, want non-nil device")
			}

			if device.Serial != tt.wantSerial {
				t.Errorf("device.Serial = %v, want %v", device.Serial, tt.wantSerial)
			}

			if device.IP != tt.wantIP {
				t.Errorf("device.IP = %v, want %v", device.IP, tt.wantIP)
			}

			if device.Port != tt.wantPort {
				t.Errorf("device.Port = %v, want %v", device.Port, tt.wantPort)
			}

			if device.Hostname != tt.entry.HostName {
				t.Errorf("device.Hostname = %v, want %v", device.Hostname, tt.entry.HostName)
			}

			// Check that DiscoveredAt is recent (within last second)
			if time.Since(device.DiscoveredAt) > time.Second {
				t.Errorf("device.DiscoveredAt is not recent: %v", device.DiscoveredAt)
			}
		})
	}
}

func TestScanner_parseServiceEntry_Metadata(t *testing.T) {
	scanner := NewScanner()

	entry := &zeroconf.ServiceEntry{
		HostName: "eValve315260240.local",
		Port:     80,
		AddrIPv4: []net.IP{net.ParseIP("192.168.4.16")},
		Text:     []string{"path=/", "srcvers=1D90645", "flag", "version=1.0"},
	}

	device := scanner.parseServiceEntry(entry)
	if device == nil {
		t.Fatal("parseServiceEntry() = nil, want device")
	}

	// Check metadata parsing
	expectedMetadata := map[string]string{
		"path":    "/",
		"srcvers": "1D90645",
		"flag":    "", // Key without value
		"version": "1.0",
	}

	if len(device.Metadata) != len(expectedMetadata) {
		t.Errorf("device.Metadata has %d entries, want %d", len(device.Metadata), len(expectedMetadata))
	}

	for key, expectedValue := range expectedMetadata {
		if actualValue, ok := device.Metadata[key]; !ok {
			t.Errorf("device.Metadata missing key %q", key)
		} else if actualValue != expectedValue {
			t.Errorf("device.Metadata[%q] = %q, want %q", key, actualValue, expectedValue)
		}
	}
}

func TestNewScanner(t *testing.T) {
	scanner := NewScanner()

	if scanner == nil {
		t.Fatal("NewScanner() = nil, want scanner")
	}

	if scanner.Timeout != DefaultScanTimeout {
		t.Errorf("scanner.Timeout = %v, want %v", scanner.Timeout, DefaultScanTimeout)
	}
}

func TestSerialPattern(t *testing.T) {
	tests := []struct {
		hostname    string
		shouldMatch bool
		serial      string
	}{
		{"eValve315260240.local", true, "315260240"},
		{"eValve315260240.local.", true, "315260240"},
		{"eValve123456789.local", true, "123456789"},
		{"eValve1.local", true, "1"},
		{"eValve999999999999.local", true, "999999999999"},
		{"evalve315260240.local", false, ""}, // lowercase 'e'
		{"eValve.local", false, ""},          // no serial
		{"eValveABC.local", false, ""},       // non-numeric serial
		{"somedevice.local", false, ""},      // wrong prefix
		{"eValve315260240", false, ""},       // missing .local
		{"", false, ""},                      // empty
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			matches := serialPattern.FindStringSubmatch(tt.hostname)

			if tt.shouldMatch {
				if matches == nil || len(matches) < 2 {
					t.Errorf("serialPattern did not match %q", tt.hostname)
				} else if matches[1] != tt.serial {
					t.Errorf("serialPattern matched %q with serial %q, want %q", tt.hostname, matches[1], tt.serial)
				}
			} else {
				if matches != nil {
					t.Errorf("serialPattern matched %q, want no match", tt.hostname)
				}
			}
		})
	}
}

// Note: Integration tests with live mDNS discovery are in a separate test file
// that requires network access and should be run manually with:
// go test -tags=integration ./internal/discovery/
