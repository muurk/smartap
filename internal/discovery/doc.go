// Package discovery provides mDNS-based device discovery for Smartap IoT devices.
//
// This package implements multicast DNS (mDNS) service discovery to automatically
// locate Smartap devices on the local network. Smartap devices advertise themselves
// using the "_http._tcp" service type.
//
// # Discovery Process
//
// The discovery process works as follows:
//  1. Broadcasts mDNS queries on the local network
//  2. Listens for service advertisements from Smartap devices
//  3. Filters responses to identify Smartap-specific services
//  4. Collects device information (hostname, IP, serial number, firmware version)
//  5. Returns a list of discovered devices after the timeout period
//
// # Usage Example
//
//	// Discover devices with 10-second timeout
//	devices, err := discovery.DiscoverDevices(10 * time.Second)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Print discovered devices
//	for _, device := range devices {
//	    fmt.Printf("Found: %s at %s (Serial: %s)\n",
//	        device.Hostname, device.IP, device.Serial)
//	}
//
// # Device Information
//
// Each discovered device includes:
//   - Hostname: Device's network hostname (e.g., "SmarTap-ABC123.local")
//   - IP: IPv4 address
//   - Port: HTTP configuration port (typically 80)
//   - Serial: Device serial number
//   - Firmware: Firmware version string
//
// # Network Requirements
//
// - Requires multicast support on the network interface
// - Devices must be on the same local network segment
// - Firewall must allow mDNS (UDP port 5353)
//
// # Thread Safety
//
// This package is safe for concurrent use. Multiple discovery sessions can run
// simultaneously without interference.
package discovery
