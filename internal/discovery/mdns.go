package discovery

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	// ServiceType is the mDNS service type for Smartap devices
	// Smartap devices advertise as "_http._tcp" services
	ServiceType = "_http._tcp"

	// ServiceDomain is the mDNS domain (typically "local.")
	ServiceDomain = "local."

	// DefaultScanTimeout is the default timeout for device discovery
	DefaultScanTimeout = 10 * time.Second

	// DefaultPort is the default HTTP port for Smartap devices
	DefaultPort = 80
)

// serialPattern matches Smartap device hostnames (e.g., "eValve315260240.local")
var serialPattern = regexp.MustCompile(`^eValve(\d+)\.local\.?$`)

// Scanner handles mDNS device discovery
type Scanner struct {
	// Timeout is the maximum time to wait for device discovery
	Timeout time.Duration
}

// NewScanner creates a new mDNS scanner with default settings
func NewScanner() *Scanner {
	return &Scanner{
		Timeout: DefaultScanTimeout,
	}
}

// ScanForDevices discovers all Smartap devices on the local network
// Returns a list of discovered devices or an error
func (s *Scanner) ScanForDevices() ([]*Device, error) {
	return s.ScanForDevicesWithContext(context.Background())
}

// ScanForDevicesWithContext discovers devices with a custom context
func (s *Scanner) ScanForDevicesWithContext(ctx context.Context) ([]*Device, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	// Channel to receive service entries
	entries := make(chan *zeroconf.ServiceEntry)
	devices := make([]*Device, 0)

	// Start the resolver
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS resolver: %w", err)
	}

	// Browse for services in a goroutine
	go func() {
		for entry := range entries {
			device := s.parseServiceEntry(entry)
			if device != nil {
				devices = append(devices, device)
			}
		}
	}()

	// Start browsing for HTTP services
	err = resolver.Browse(ctx, ServiceType, ServiceDomain, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to browse for mDNS services: %w", err)
	}

	// Wait for context to complete (timeout or cancellation)
	<-ctx.Done()

	return devices, nil
}

// WaitForDevice waits for a specific device by serial number
// Returns the device or an error if not found within timeout
func (s *Scanner) WaitForDevice(serial string) (*Device, error) {
	return s.WaitForDeviceWithContext(context.Background(), serial)
}

// WaitForDeviceWithContext waits for a specific device with a custom context
func (s *Scanner) WaitForDeviceWithContext(ctx context.Context, serial string) (*Device, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()

	// Channel to receive service entries
	entries := make(chan *zeroconf.ServiceEntry)
	deviceChan := make(chan *Device, 1)

	// Start the resolver
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create mDNS resolver: %w", err)
	}

	// Browse for services in a goroutine
	go func() {
		for entry := range entries {
			device := s.parseServiceEntry(entry)
			if device != nil && device.Serial == serial {
				deviceChan <- device
				cancel() // Found the device, cancel context
				return
			}
		}
	}()

	// Start browsing for HTTP services
	err = resolver.Browse(ctx, ServiceType, ServiceDomain, entries)
	if err != nil {
		return nil, fmt.Errorf("failed to browse for mDNS services: %w", err)
	}

	// Wait for device or timeout
	select {
	case device := <-deviceChan:
		return device, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("device with serial %s not found within timeout", serial)
	}
}

// parseServiceEntry converts a zeroconf service entry to a Device
// Returns nil if the entry is not a Smartap device
func (s *Scanner) parseServiceEntry(entry *zeroconf.ServiceEntry) *Device {
	// Check if hostname matches Smartap pattern (eValve{serial}.local)
	hostname := entry.HostName
	if hostname == "" {
		return nil
	}

	matches := serialPattern.FindStringSubmatch(hostname)
	if len(matches) < 2 {
		return nil
	}

	serial := matches[1]

	// Get IP address (prefer IPv4)
	var ip string
	for _, addr := range entry.AddrIPv4 {
		ip = addr.String()
		break
	}

	// Fallback to IPv6 if no IPv4
	if ip == "" && len(entry.AddrIPv6) > 0 {
		ip = entry.AddrIPv6[0].String()
	}

	if ip == "" {
		return nil
	}

	// Get port (default to 80 if not specified)
	port := entry.Port
	if port == 0 {
		port = DefaultPort
	}

	// Parse TXT records into metadata
	metadata := make(map[string]string)
	for _, txt := range entry.Text {
		// TXT records are in "key=value" format
		parts := strings.SplitN(txt, "=", 2)
		if len(parts) == 2 {
			metadata[parts[0]] = parts[1]
		} else {
			// Key without value
			metadata[parts[0]] = ""
		}
	}

	return &Device{
		Serial:       serial,
		Hostname:     hostname,
		IP:           ip,
		Port:         port,
		Metadata:     metadata,
		DiscoveredAt: time.Now(),
	}
}

// ScanForDevices is a convenience function to scan for devices with a custom timeout
func ScanForDevices(timeout time.Duration) ([]*Device, error) {
	scanner := NewScanner()
	scanner.Timeout = timeout
	return scanner.ScanForDevices()
}

// QuickScan performs a fast scan with a 3-second timeout
func QuickScan() ([]*Device, error) {
	scanner := NewScanner()
	scanner.Timeout = 3 * time.Second
	return scanner.ScanForDevices()
}

// FindDevice searches for a specific device by serial number with default timeout
func FindDevice(serial string) (*Device, error) {
	scanner := NewScanner()
	return scanner.WaitForDevice(serial)
}
