package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	appName    = "smartap"
	configFile = "config.yaml"
)

var (
	// Global registry instance (loaded lazily)
	globalRegistry     *Registry
	globalRegistryOnce sync.Once
	globalRegistryErr  error

	// Mutex for thread-safe file operations
	fileMutex sync.Mutex
)

// GetConfigDir returns the OS-appropriate configuration directory for the application.
// This follows platform conventions:
//   - Linux: $XDG_CONFIG_HOME/smartap or $HOME/.config/smartap
//   - macOS: $HOME/.config/smartap (following XDG convention on macOS)
//   - Windows: %LOCALAPPDATA%\smartap
func GetConfigDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: Use LOCALAPPDATA
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			// Fallback to USERPROFILE\AppData\Local if LOCALAPPDATA not set
			userProfile := os.Getenv("USERPROFILE")
			if userProfile == "" {
				return "", fmt.Errorf("cannot determine user profile directory (LOCALAPPDATA and USERPROFILE not set)")
			}
			baseDir = filepath.Join(userProfile, "AppData", "Local", appName)
		} else {
			baseDir = filepath.Join(localAppData, appName)
		}

	case "darwin":
		// macOS: Use $HOME/.config/smartap (following modern XDG convention)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".config", appName)

	default:
		// Linux and other Unix-like systems: Use XDG_CONFIG_HOME or $HOME/.config
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			baseDir = filepath.Join(xdgConfigHome, appName)
		} else {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("cannot determine home directory: %w", err)
			}
			baseDir = filepath.Join(homeDir, ".config", appName)
		}
	}

	return baseDir, nil
}

// GetConfigPath returns the full path to the configuration file.
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, configFile), nil
}

// ensureConfigDir ensures the configuration directory exists.
// Creates the directory with appropriate permissions if it doesn't exist.
func ensureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	// Create directory with user-only permissions (0700)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return nil
}

// LoadRegistry loads the configuration registry from disk.
// If the file doesn't exist, returns a new default registry.
// Thread-safe - multiple calls will return the same instance.
func LoadRegistry() (*Registry, error) {
	globalRegistryOnce.Do(func() {
		globalRegistry, globalRegistryErr = loadRegistryFromDisk()
	})
	return globalRegistry, globalRegistryErr
}

// loadRegistryFromDisk performs the actual file loading.
func loadRegistryFromDisk() (*Registry, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config doesn't exist - return new default registry
		return NewRegistry(), nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var registry Registry
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate version
	if registry.Version != 1 {
		return nil, fmt.Errorf("unsupported config version: %d (expected 1)", registry.Version)
	}

	// Ensure maps are initialized
	if registry.Devices == nil {
		registry.Devices = make(map[string]*Device)
	}
	if registry.Preferences == nil {
		registry.Preferences = &Preferences{
			AutoDiscover:    true,
			DiscoverTimeout: 10,
			DefaultAuth: &AuthPrefs{
				Username: "SmarTap",
			},
		}
	}

	return &registry, nil
}

// Save saves the registry to disk.
// Performs an atomic write to prevent corruption on crash.
func (r *Registry) Save() error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Ensure config directory exists
	if err := ensureConfigDir(); err != nil {
		return fmt.Errorf("failed to ensure config directory exists: %w", err)
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Marshal to YAML with comments
	data, err := yaml.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Add header comment
	header := []byte(`# Smartap Configuration File
# This file stores user-defined metadata for Smartap devices.
#
# Security Note: WiFi passwords and device authentication credentials
# are NEVER stored in this file. They are always prompted when needed.
#
# Location: ` + configPath + `

`)
	data = append(header, data...)

	// Write to temporary file first (atomic write)
	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary config file: %w", err)
	}

	// Atomic rename (this is atomic on all platforms)
	if err := os.Rename(tmpPath, configPath); err != nil {
		// Clean up temp file on error
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// ReloadRegistry reloads the registry from disk, discarding any in-memory changes.
// This is useful for reading changes made by another process.
func ReloadRegistry() (*Registry, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// Reset the global registry
	globalRegistryOnce = sync.Once{}
	return LoadRegistry()
}

// GetGlobalRegistry returns the global registry instance.
// This is a convenience wrapper around LoadRegistry().
func GetGlobalRegistry() (*Registry, error) {
	return LoadRegistry()
}

// SaveGlobal saves the global registry instance to disk.
// This is a convenience wrapper for the most common use case.
func SaveGlobal() error {
	registry, err := LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	return registry.Save()
}

// CreateDefaultConfig creates a default configuration file with example data.
// This is useful for first-time setup or documentation purposes.
func CreateDefaultConfig() error {
	registry := NewRegistry()

	// Add example device
	exampleDevice := &Device{
		Nickname: "Example Bathroom Shower",
		LastIP:   "192.168.1.100",
		Outlets: map[int]*OutletMeta{
			1: {
				Label: "Rain Shower Head",
				Type:  "rain_head",
				Icon:  "☔",
			},
			2: {
				Label: "Body Jets",
				Type:  "body_jets",
				Icon:  "💦",
			},
			3: {
				Label: "Hand Shower",
				Type:  "hand_shower",
				Icon:  "🚰",
			},
		},
		Diverter: &DiverterMeta{
			FirstPress:  []int{1},
			SecondPress: []int{2},
			ThirdPress:  []int{3},
			K3Mode:      true,
		},
	}
	registry.Devices["000000000"] = exampleDevice

	return registry.Save()
}
