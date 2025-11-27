package discovery

import (
	"fmt"
	"time"
)

// Device represents a discovered Smartap device on the network
type Device struct {
	// Serial is the device serial number (e.g., "315260240")
	Serial string

	// Hostname is the mDNS hostname (e.g., "eValve315260240.local")
	Hostname string

	// IP is the IPv4 address (e.g., "192.168.4.16")
	IP string

	// Port is the HTTP port (typically 80)
	Port int

	// MAC is the device MAC address (populated from HTTP call, not mDNS)
	MAC string

	// Metadata contains additional mDNS TXT record data
	// Common fields: "path=/", "srcvers=1D90645"
	Metadata map[string]string

	// DiscoveredAt is when the device was discovered
	DiscoveredAt time.Time
}

// String returns a human-readable string representation of the device
func (d *Device) String() string {
	return fmt.Sprintf("Smartap Device %s (%s) at %s:%d", d.Serial, d.Hostname, d.IP, d.Port)
}

// BaseURL returns the HTTP base URL for the device
func (d *Device) BaseURL() string {
	return fmt.Sprintf("http://%s:%d", d.IP, d.Port)
}

// GetMetadata retrieves a metadata value by key, or returns empty string if not found
func (d *Device) GetMetadata(key string) string {
	if d.Metadata == nil {
		return ""
	}
	return d.Metadata[key]
}
