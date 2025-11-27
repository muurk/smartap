// Package config provides user configuration management for the Smartap project.
//
// This package manages a YAML-based configuration file that stores user-defined
// metadata for Smartap devices, including nicknames, outlet labels, and application
// preferences. The configuration follows OS-specific conventions for storage location.
//
// # Configuration File Location
//
// The configuration file is stored in platform-appropriate locations:
//   - Linux: $XDG_CONFIG_HOME/smartap/config.yaml or $HOME/.config/smartap/config.yaml
//   - macOS: $HOME/.config/smartap/config.yaml
//   - Windows: %LOCALAPPDATA%\smartap\config.yaml
//
// # Security
//
// IMPORTANT: This package NEVER stores sensitive credentials such as WiFi passwords
// or device authentication tokens. These are always prompted from the user when needed.
//
// # Usage Example
//
//	// Load the global registry
//	registry, err := config.LoadRegistry()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Add or update device metadata
//	device := &config.Device{
//	    Nickname: "Master Bathroom",
//	    Outlets: map[int]*config.OutletMeta{
//	        1: {Label: "Rain Head", Icon: "☔"},
//	        2: {Label: "Body Jets", Icon: "💦"},
//	        3: {Label: "Hand Shower", Icon: "🚰"},
//	    },
//	}
//	registry.Devices["ABC123"] = device
//
//	// Save changes atomically
//	if err := registry.Save(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Thread Safety
//
// The global registry uses sync.Once for safe initialization across goroutines.
// File operations are protected by a mutex to ensure atomic writes.
package config
