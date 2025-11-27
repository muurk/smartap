package deviceconfig

import (
	"fmt"
)

// ConfigBuilder provides a fluent API for building configuration updates.
// It tracks changes and validates them before creating a ConfigUpdate.
//
// Example usage:
//
//	builder := NewConfigBuilder(currentConfig)
//	update, err := builder.
//	    SetDiverterButton(1, 1).     // First press -> Outlet 1
//	    SetDiverterButton(2, 2).     // Second press -> Outlet 2
//	    SetDiverterButton(3, 4).     // Third press -> Outlet 3
//	    EnableThirdKnob().
//	    SetServerDNS("my-server.local").
//	    Build()
type ConfigBuilder struct {
	// current holds the current device configuration (baseline)
	current *DeviceConfig

	// Diverter configuration changes
	diverterChanged bool
	firstPress      int
	secondPress     int
	thirdPress      int
	k3Mode          bool

	// WiFi configuration changes
	wifiChanged  bool
	ssid         string
	password     string
	securityType string

	// Server configuration changes
	serverChanged bool
	dns           string
	port          int
}

// NewConfigBuilder creates a new builder with the current device configuration as baseline.
// Pass nil if starting from scratch (all fields must be set explicitly).
func NewConfigBuilder(current *DeviceConfig) *ConfigBuilder {
	b := &ConfigBuilder{
		current: current,
	}

	// Initialize from current config if provided
	if current != nil {
		b.firstPress = current.Outlet1
		b.secondPress = current.Outlet2
		b.thirdPress = current.Outlet3
		b.k3Mode = current.K3Outlet
		b.dns = current.DNS
		b.port = current.Port
	}

	return b
}

// SetDiverterButton sets the outlet bitmask for a specific button press (1-3).
// buttonNumber: 1, 2, or 3
// bitmask: 0-7 (bit 0 = outlet 1, bit 1 = outlet 2, bit 2 = outlet 3)
//
// Examples:
//   - SetDiverterButton(1, 1) -> First press activates outlet 1
//   - SetDiverterButton(2, 3) -> Second press activates outlets 1+2 (1|2 = 3)
//   - SetDiverterButton(3, 7) -> Third press activates all outlets
func (b *ConfigBuilder) SetDiverterButton(buttonNumber int, bitmask int) *ConfigBuilder {
	b.diverterChanged = true

	switch buttonNumber {
	case 1:
		b.firstPress = bitmask
	case 2:
		b.secondPress = bitmask
	case 3:
		b.thirdPress = bitmask
	}

	return b
}

// SetFirstPress sets the outlet bitmask for the first button press.
func (b *ConfigBuilder) SetFirstPress(bitmask int) *ConfigBuilder {
	return b.SetDiverterButton(1, bitmask)
}

// SetSecondPress sets the outlet bitmask for the second button press.
func (b *ConfigBuilder) SetSecondPress(bitmask int) *ConfigBuilder {
	return b.SetDiverterButton(2, bitmask)
}

// SetThirdPress sets the outlet bitmask for the third button press.
func (b *ConfigBuilder) SetThirdPress(bitmask int) *ConfigBuilder {
	return b.SetDiverterButton(3, bitmask)
}

// SetSequentialOutlets configures sequential outlet activation (1 -> 2 -> 4).
// This is the most common configuration pattern.
func (b *ConfigBuilder) SetSequentialOutlets() *ConfigBuilder {
	b.diverterChanged = true
	b.firstPress = 1
	b.secondPress = 2
	b.thirdPress = 4
	return b
}

// SetAllOutletsOn configures all outlets to activate with each button press (7 -> 7 -> 7).
// Useful for testing or when all outlets should run simultaneously.
func (b *ConfigBuilder) SetAllOutletsOn() *ConfigBuilder {
	b.diverterChanged = true
	b.firstPress = 7
	b.secondPress = 7
	b.thirdPress = 7
	return b
}

// EnableThirdKnob enables third knob separation mode.
// When enabled, the third knob controls an outlet independently.
func (b *ConfigBuilder) EnableThirdKnob() *ConfigBuilder {
	b.diverterChanged = true
	b.k3Mode = true
	return b
}

// DisableThirdKnob disables third knob separation mode.
// The third knob will follow the normal button press sequence.
func (b *ConfigBuilder) DisableThirdKnob() *ConfigBuilder {
	b.diverterChanged = true
	b.k3Mode = false
	return b
}

// SetThirdKnobMode sets the third knob separation mode explicitly.
func (b *ConfigBuilder) SetThirdKnobMode(enabled bool) *ConfigBuilder {
	b.diverterChanged = true
	b.k3Mode = enabled
	return b
}

// SetWiFiSSID sets the WiFi network SSID.
func (b *ConfigBuilder) SetWiFiSSID(ssid string) *ConfigBuilder {
	b.wifiChanged = true
	b.ssid = ssid
	return b
}

// SetWiFiPassword sets the WiFi password.
// Only used for WPA2 networks. Leave empty for open networks.
func (b *ConfigBuilder) SetWiFiPassword(password string) *ConfigBuilder {
	b.wifiChanged = true
	b.password = password
	return b
}

// SetWiFiSecurity sets the WiFi security type.
// Valid values: "WPA2" or "OPEN"
func (b *ConfigBuilder) SetWiFiSecurity(securityType string) *ConfigBuilder {
	b.wifiChanged = true
	b.securityType = securityType
	return b
}

// SetWiFi sets all WiFi parameters at once.
// securityType: "WPA2" or "OPEN"
func (b *ConfigBuilder) SetWiFi(ssid, password, securityType string) *ConfigBuilder {
	b.wifiChanged = true
	b.ssid = ssid
	b.password = password
	b.securityType = securityType
	return b
}

// SetWiFiWPA2 configures WiFi with WPA2 security.
func (b *ConfigBuilder) SetWiFiWPA2(ssid, password string) *ConfigBuilder {
	return b.SetWiFi(ssid, password, "WPA2")
}

// SetWiFiOpen configures WiFi with no security (open network).
func (b *ConfigBuilder) SetWiFiOpen(ssid string) *ConfigBuilder {
	return b.SetWiFi(ssid, "", "OPEN")
}

// SetServerDNS sets the server hostname or IP address.
func (b *ConfigBuilder) SetServerDNS(dns string) *ConfigBuilder {
	b.serverChanged = true
	b.dns = dns
	return b
}

// SetServerPort sets the server port number.
func (b *ConfigBuilder) SetServerPort(port int) *ConfigBuilder {
	b.serverChanged = true
	b.port = port
	return b
}

// SetServer sets both server DNS and port at once.
func (b *ConfigBuilder) SetServer(dns string, port int) *ConfigBuilder {
	b.serverChanged = true
	b.dns = dns
	b.port = port
	return b
}

// HasChanges returns true if any configuration changes have been made.
func (b *ConfigBuilder) HasChanges() bool {
	return b.diverterChanged || b.wifiChanged || b.serverChanged
}

// Validate checks if the current configuration is valid.
// Returns an error if any validation rules are violated.
func (b *ConfigBuilder) Validate() error {
	// Validate diverter bitmasks (0-7 range)
	if b.diverterChanged {
		if b.firstPress < 0 || b.firstPress > 7 {
			return NewValidationError(fmt.Sprintf("first press bitmask must be 0-7, got %d", b.firstPress))
		}
		if b.secondPress < 0 || b.secondPress > 7 {
			return NewValidationError(fmt.Sprintf("second press bitmask must be 0-7, got %d", b.secondPress))
		}
		if b.thirdPress < 0 || b.thirdPress > 7 {
			return NewValidationError(fmt.Sprintf("third press bitmask must be 0-7, got %d", b.thirdPress))
		}
	}

	// Validate WiFi configuration
	if b.wifiChanged {
		if b.ssid == "" {
			return NewValidationError("WiFi SSID cannot be empty")
		}
		if b.securityType != "WPA2" && b.securityType != "OPEN" {
			return NewValidationError(fmt.Sprintf("WiFi security type must be 'WPA2' or 'OPEN', got '%s'", b.securityType))
		}
		if b.securityType == "WPA2" && b.password == "" {
			return NewValidationError("WiFi password required for WPA2 security")
		}
	}

	// Validate server configuration
	if b.serverChanged {
		if b.dns == "" {
			return NewValidationError("server DNS cannot be empty")
		}
		if b.port <= 0 || b.port > 65535 {
			return NewValidationError(fmt.Sprintf("server port must be 1-65535, got %d", b.port))
		}
	}

	return nil
}

// Build creates a ConfigUpdate from the builder's state.
// Only includes sections that have been modified.
// Returns an error if validation fails.
func (b *ConfigBuilder) Build() (*ConfigUpdate, error) {
	// Validate before building
	if err := b.Validate(); err != nil {
		return nil, err
	}

	// Build ConfigUpdate with only changed sections
	update := &ConfigUpdate{}

	if b.diverterChanged {
		update.Diverter = &DiverterConfig{
			FirstPress:  b.firstPress,
			SecondPress: b.secondPress,
			ThirdPress:  b.thirdPress,
			K3Mode:      b.k3Mode,
		}
	}

	if b.wifiChanged {
		update.WiFi = &WiFiConfig{
			SSID:         b.ssid,
			Password:     b.password,
			SecurityType: b.securityType,
		}
	}

	if b.serverChanged {
		update.Server = &ServerConfig{
			DNS:  b.dns,
			Port: b.port,
		}
	}

	return update, nil
}

// Reset clears all changes and restores builder to initial state.
func (b *ConfigBuilder) Reset() *ConfigBuilder {
	// Reset change flags
	b.diverterChanged = false
	b.wifiChanged = false
	b.serverChanged = false

	// Restore from current config if available
	if b.current != nil {
		b.firstPress = b.current.Outlet1
		b.secondPress = b.current.Outlet2
		b.thirdPress = b.current.Outlet3
		b.k3Mode = b.current.K3Outlet
		b.dns = b.current.DNS
		b.port = b.current.Port
	} else {
		// Clear all fields
		b.firstPress = 0
		b.secondPress = 0
		b.thirdPress = 0
		b.k3Mode = false
		b.ssid = ""
		b.password = ""
		b.securityType = ""
		b.dns = ""
		b.port = 0
	}

	return b
}
