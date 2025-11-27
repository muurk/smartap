package deviceconfig

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DeviceConfig represents the complete device configuration returned by GET /
// This matches the JSON structure returned by the Smartap device API.
//
// Note: The device returns malformed JSON with trailing HTML data. Callers should
// use CleanJSONResponse() to extract valid JSON before unmarshaling.
type DeviceConfig struct {
	// Network configuration
	SSIDList []string `json:"ssidList"` // List of WiFi networks device knows

	// Device settings
	LowPowerMode bool   `json:"lowPowerMode"` // Low power mode status
	Serial       string `json:"serial"`       // Device serial number
	DNS          string `json:"dns"`          // Server hostname
	Port         int    `json:"port"`         // Server port

	// Diverter button configuration (3-bit bitmasks)
	// These control which outlets open for each button press.
	// Bit 0 (1): Outlet 1, Bit 1 (2): Outlet 2, Bit 2 (4): Outlet 3
	Outlet1 int `json:"outlet1"` // First button press bitmask (0-7)
	Outlet2 int `json:"outlet2"` // Second button press bitmask (0-7)
	Outlet3 int `json:"outlet3"` // Third button press bitmask (0-7)

	// Third knob separation mode
	// When true, knob 3 controls a separate outlet independently
	// When false, knob 3 follows the normal button press sequence
	K3Outlet bool `json:"k3Outlet"`

	// Firmware versions
	SWVer  string `json:"swVer"`  // Software version (e.g., "0x355")
	WNPVer string `json:"wnpVer"` // WiFi network processor version

	// Hardware identifier
	MAC string `json:"mac"` // MAC address
}

// DiverterConfig represents configuration updates for the diverter button behavior.
// This is used to construct POST requests to update outlet configuration.
type DiverterConfig struct {
	// Button press configurations (3-bit bitmasks: 0-7)
	// Each value controls which outlets open when button is pressed N times
	FirstPress  int // Maps to __SL_P_OU1
	SecondPress int // Maps to __SL_P_OU2
	ThirdPress  int // Maps to __SL_P_OU3

	// Third knob separation mode
	K3Mode bool // Maps to __SL_P_K3O ("checked" or "no")
}

// WiFiConfig represents WiFi network configuration updates.
// This is used to construct POST requests to configure WiFi connectivity.
type WiFiConfig struct {
	SSID         string // Maps to __SL_P_USD
	Password     string // Maps to __SL_P_PSD (only for WPA2)
	SecurityType string // Maps to __SL_P_ENC ("WPA2" or "OPEN")
}

// ServerConfig represents server endpoint configuration updates.
type ServerConfig struct {
	DNS  string // Maps to __SL_P_DNS
	Port int    // Maps to __SL_P_PRT
}

// ConfigUpdate represents a complete configuration update.
// Fields are optional - only non-nil fields will be included in POST.
type ConfigUpdate struct {
	Diverter *DiverterConfig
	WiFi     *WiFiConfig
	Server   *ServerConfig
}

// CleanJSONResponse extracts valid JSON from the device's malformed response.
//
// The device returns valid JSON followed by trailing HTML data:
//
//	{"ssidList":...,"mac":"C4:BE:84:74:86:37"}"oldAppVer":"pkey:0000,315260240</div>"
//
// This function finds the end of the valid JSON object and truncates the rest.
func CleanJSONResponse(data []byte) ([]byte, error) {
	// Find the first '{' to locate JSON start
	start := -1
	for i, b := range data {
		if b == '{' {
			start = i
			break
		}
	}
	if start == -1 {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	// Find the matching closing '}' by tracking brace depth
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(data); i++ {
		b := data[i]

		// Handle string escaping
		if escaped {
			escaped = false
			continue
		}
		if b == '\\' {
			escaped = true
			continue
		}

		// Track string boundaries (braces inside strings don't count)
		if b == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if b == '{' {
				depth++
			} else if b == '}' {
				depth--
				if depth == 0 {
					// Found the end of the JSON object
					return data[start : i+1], nil
				}
			}
		}
	}

	return nil, fmt.Errorf("unclosed JSON object in response")
}

// ParseDeviceConfig parses the device configuration from raw response data.
// It automatically handles the malformed JSON response from the device.
func ParseDeviceConfig(data []byte) (*DeviceConfig, error) {
	// Clean the malformed response
	cleanData, err := CleanJSONResponse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to clean JSON response: %w", err)
	}

	// Unmarshal the clean JSON
	var config DeviceConfig
	if err := json.Unmarshal(cleanData, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device config: %w", err)
	}

	return &config, nil
}

// ToFormData converts DiverterConfig to URL-encoded form data for POST requests.
func (dc *DiverterConfig) ToFormData() url.Values {
	data := url.Values{}
	data.Set("__SL_P_OU1", strconv.Itoa(dc.FirstPress))
	data.Set("__SL_P_OU2", strconv.Itoa(dc.SecondPress))
	data.Set("__SL_P_OU3", strconv.Itoa(dc.ThirdPress))

	if dc.K3Mode {
		data.Set("__SL_P_K3O", "checked")
	} else {
		data.Set("__SL_P_K3O", "no")
	}

	return data
}

// ToFormData converts WiFiConfig to URL-encoded form data for POST requests.
func (wc *WiFiConfig) ToFormData() url.Values {
	data := url.Values{}
	data.Set("__SL_P_USD", wc.SSID)
	data.Set("__SL_P_ENC", wc.SecurityType)

	// Only include password for WPA2 networks
	if wc.SecurityType == "WPA2" && wc.Password != "" {
		data.Set("__SL_P_PSD", wc.Password)
	}

	// Trigger connection attempt
	data.Set("__SL_P_CON", "connect")

	return data
}

// ToFormData converts ServerConfig to URL-encoded form data for POST requests.
func (sc *ServerConfig) ToFormData() url.Values {
	data := url.Values{}
	data.Set("__SL_P_DNS", sc.DNS)
	data.Set("__SL_P_PRT", strconv.Itoa(sc.Port))
	return data
}

// ToFormData converts ConfigUpdate to URL-encoded form data for POST requests.
// Only non-nil fields are included in the output.
func (cu *ConfigUpdate) ToFormData() url.Values {
	data := url.Values{}

	if cu.Diverter != nil {
		for k, v := range cu.Diverter.ToFormData() {
			data[k] = v
		}
	}

	if cu.WiFi != nil {
		for k, v := range cu.WiFi.ToFormData() {
			data[k] = v
		}
	}

	if cu.Server != nil {
		for k, v := range cu.Server.ToFormData() {
			data[k] = v
		}
	}

	return data
}

// DecodeBitmask converts a 3-bit bitmask to a slice of outlet numbers.
// Example: 5 (binary 101) → [1, 3]
func DecodeBitmask(bitmask int) []int {
	outlets := []int{}
	for i := 0; i < 3; i++ {
		if bitmask&(1<<i) != 0 {
			outlets = append(outlets, i+1)
		}
	}
	return outlets
}

// EncodeBitmask converts a slice of outlet numbers to a 3-bit bitmask.
// Example: [1, 3] → 5 (binary 101)
func EncodeBitmask(outlets []int) int {
	bitmask := 0
	for _, outlet := range outlets {
		if outlet >= 1 && outlet <= 3 {
			bitmask |= (1 << (outlet - 1))
		}
	}
	return bitmask
}

// FormatBitmask returns a human-readable string for a bitmask.
// Example: 5 → "Outlets 1+3"
func FormatBitmask(bitmask int) string {
	outlets := DecodeBitmask(bitmask)
	if len(outlets) == 0 {
		return "No outlets"
	}

	parts := make([]string, len(outlets))
	for i, o := range outlets {
		parts[i] = strconv.Itoa(o)
	}

	return "Outlet" + func() string {
		if len(outlets) > 1 {
			return "s"
		}
		return ""
	}() + " " + strings.Join(parts, "+")
}

// String returns a human-readable summary of the device configuration.
func (dc *DeviceConfig) String() string {
	return fmt.Sprintf("Smartap Device %s (MAC: %s, FW: %s)\n"+
		"  Server: %s:%d\n"+
		"  WiFi: %s\n"+
		"  Diverter Config:\n"+
		"    1st press: %s\n"+
		"    2nd press: %s\n"+
		"    3rd press: %s\n"+
		"  3rd Knob Mode: %v",
		dc.Serial, dc.MAC, dc.SWVer,
		dc.DNS, dc.Port,
		strings.Join(dc.SSIDList, ", "),
		FormatBitmask(dc.Outlet1),
		FormatBitmask(dc.Outlet2),
		FormatBitmask(dc.Outlet3),
		dc.K3Outlet)
}
