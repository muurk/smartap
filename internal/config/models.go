package config

import "time"

// Registry represents the entire user configuration file.
// This stores user-defined metadata for devices and application preferences.
type Registry struct {
	Version     int                `yaml:"version"`
	Devices     map[string]*Device `yaml:"devices,omitempty"` // Keyed by device serial number
	Preferences *Preferences       `yaml:"preferences,omitempty"`
}

// Device represents user-defined metadata for a single Smartap device.
// This is keyed by the device's serial number in the Registry.
type Device struct {
	Nickname string              `yaml:"nickname,omitempty"`  // User-friendly name
	LastIP   string              `yaml:"last_ip,omitempty"`   // Last known IP address
	LastSeen time.Time           `yaml:"last_seen,omitempty"` // Last discovery/connection time
	Outlets  map[int]*OutletMeta `yaml:"outlets,omitempty"`   // Outlet metadata (keyed by outlet number 1-3)
	Diverter *DiverterMeta       `yaml:"diverter,omitempty"`  // Last known diverter configuration
}

// OutletMeta represents user-defined metadata for a single outlet.
// This is purely client-side information - the device itself doesn't store outlet types.
type OutletMeta struct {
	Label string `yaml:"label"`          // User-defined label (e.g., "Rain Shower Head")
	Type  string `yaml:"type"`           // Outlet type identifier (e.g., "shower_head", "body_jets")
	Icon  string `yaml:"icon,omitempty"` // Optional emoji/icon for display
}

// DiverterMeta represents the last known diverter button configuration.
// This is stored for reference and quick restore functionality.
type DiverterMeta struct {
	FirstPress  []int `yaml:"first_press"`  // Outlet numbers for first button press
	SecondPress []int `yaml:"second_press"` // Outlet numbers for second button press
	ThirdPress  []int `yaml:"third_press"`  // Outlet numbers for third button press
	K3Mode      bool  `yaml:"k3_mode"`      // Third knob separation mode
}

// Preferences represents application-wide user preferences.
type Preferences struct {
	AutoDiscover    bool       `yaml:"auto_discover"`          // Enable automatic mDNS discovery on startup
	DiscoverTimeout int        `yaml:"discover_timeout"`       // mDNS discovery timeout in seconds
	DefaultAuth     *AuthPrefs `yaml:"default_auth,omitempty"` // Default authentication preferences
}

// AuthPrefs represents default authentication preferences.
// Note: Passwords are NEVER stored - they are always prompted from the user.
type AuthPrefs struct {
	Username string `yaml:"username"` // Default username (e.g., "SmarTap")
	// Password is NEVER stored in config file for security reasons
}

// NewRegistry creates a new Registry with default values.
func NewRegistry() *Registry {
	return &Registry{
		Version: 1,
		Devices: make(map[string]*Device),
		Preferences: &Preferences{
			AutoDiscover:    true,
			DiscoverTimeout: 10,
			DefaultAuth: &AuthPrefs{
				Username: "SmarTap",
			},
		},
	}
}

// GetDevice retrieves device metadata by serial number.
// Returns nil if the device doesn't exist in the registry.
func (r *Registry) GetDevice(serial string) *Device {
	return r.Devices[serial]
}

// EnsureDevice ensures a device entry exists in the registry.
// If the device doesn't exist, creates a new entry with default values.
// Returns the device entry (existing or newly created).
func (r *Registry) EnsureDevice(serial string) *Device {
	if r.Devices == nil {
		r.Devices = make(map[string]*Device)
	}

	if device, exists := r.Devices[serial]; exists {
		return device
	}

	// Create new device entry
	device := &Device{
		Outlets: make(map[int]*OutletMeta),
	}
	r.Devices[serial] = device
	return device
}

// UpdateDeviceLastSeen updates the last seen timestamp and IP for a device.
func (r *Registry) UpdateDeviceLastSeen(serial, ip string) {
	device := r.EnsureDevice(serial)
	device.LastSeen = time.Now()
	device.LastIP = ip
}

// SetOutletLabel sets or updates the outlet metadata for a device.
func (r *Registry) SetOutletLabel(serial string, outletNum int, label, typ, icon string) {
	device := r.EnsureDevice(serial)

	if device.Outlets == nil {
		device.Outlets = make(map[int]*OutletMeta)
	}

	device.Outlets[outletNum] = &OutletMeta{
		Label: label,
		Type:  typ,
		Icon:  icon,
	}
}

// UpdateDiverter updates the diverter configuration for a device.
func (r *Registry) UpdateDiverter(serial string, first, second, third []int, k3Mode bool) {
	device := r.EnsureDevice(serial)
	device.Diverter = &DiverterMeta{
		FirstPress:  first,
		SecondPress: second,
		ThirdPress:  third,
		K3Mode:      k3Mode,
	}
}

// SetDeviceNickname sets a user-friendly nickname for a device.
func (r *Registry) SetDeviceNickname(serial, nickname string) {
	device := r.EnsureDevice(serial)
	device.Nickname = nickname
}

// OutletTypeDefinitions maps outlet type identifiers to human-readable names.
// This is used for display and validation purposes.
var OutletTypeDefinitions = map[string]string{
	"none":        "No Device",
	"shower_head": "Shower Head",
	"rain_head":   "Rain Shower Head",
	"body_jets":   "Body Jets",
	"hand_shower": "Hand Shower",
	"tub_filler":  "Bathtub Filler",
	"steam":       "Steam Generator",
	"other":       "Other",
}

// OutletTypeIcons maps outlet type identifiers to default emoji icons.
var OutletTypeIcons = map[string]string{
	"none":        "⚪",
	"shower_head": "🚿",
	"rain_head":   "☔",
	"body_jets":   "💦",
	"hand_shower": "🚰",
	"tub_filler":  "🛁",
	"steam":       "💨",
	"other":       "🔧",
}
