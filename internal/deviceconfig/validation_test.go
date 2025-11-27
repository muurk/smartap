package deviceconfig

import (
	"testing"
)

// TestValidateDiverterBitmask tests diverter bitmask validation
func TestValidateDiverterBitmask(t *testing.T) {
	tests := []struct {
		name    string
		mask    int
		wantErr bool
	}{
		{"Valid: 0 (no outlets)", 0, false},
		{"Valid: 1 (outlet 1)", 1, false},
		{"Valid: 2 (outlet 2)", 2, false},
		{"Valid: 3 (outlets 1+2)", 3, false},
		{"Valid: 4 (outlet 3)", 4, false},
		{"Valid: 5 (outlets 1+3)", 5, false},
		{"Valid: 6 (outlets 2+3)", 6, false},
		{"Valid: 7 (all outlets)", 7, false},
		{"Invalid: negative", -1, true},
		{"Invalid: too high", 8, true},
		{"Invalid: way too high", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDiverterBitmask(tt.mask)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDiverterBitmask(%d) error = %v, wantErr %v", tt.mask, err, tt.wantErr)
			}
			if err != nil && !IsValidationError(err) {
				t.Errorf("Expected ValidationError, got %T", err)
			}
		})
	}
}

// TestValidateDiverterConfig tests complete diverter configuration validation
func TestValidateDiverterConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *DiverterConfig
		wantCount int // Number of errors expected
	}{
		{
			name: "Valid: sequential outlets",
			config: &DiverterConfig{
				FirstPress:  1,
				SecondPress: 2,
				ThirdPress:  4,
				K3Mode:      true,
			},
			wantCount: 0,
		},
		{
			name: "Valid: all outlets on",
			config: &DiverterConfig{
				FirstPress:  7,
				SecondPress: 7,
				ThirdPress:  7,
				K3Mode:      false,
			},
			wantCount: 0,
		},
		{
			name: "Invalid: first press too high",
			config: &DiverterConfig{
				FirstPress:  8,
				SecondPress: 2,
				ThirdPress:  4,
				K3Mode:      true,
			},
			wantCount: 1,
		},
		{
			name: "Invalid: multiple bad values",
			config: &DiverterConfig{
				FirstPress:  -1,
				SecondPress: 10,
				ThirdPress:  20,
				K3Mode:      false,
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateDiverterConfig(tt.config)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateDiverterConfig() got %d errors, want %d", len(errors), tt.wantCount)
				for i, err := range errors {
					t.Logf("  Error %d: %v", i+1, err)
				}
			}
		})
	}
}

// TestValidateWiFiSSID tests WiFi SSID validation
func TestValidateWiFiSSID(t *testing.T) {
	tests := []struct {
		name    string
		ssid    string
		wantErr bool
	}{
		{"Valid: normal SSID", "MyNetwork", false},
		{"Valid: with spaces", "My Home Network", false},
		{"Valid: with numbers", "Network123", false},
		{"Valid: max length (32 chars)", "12345678901234567890123456789012", false},
		{"Invalid: empty", "", true},
		{"Invalid: too long (33 chars)", "123456789012345678901234567890123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWiFiSSID(tt.ssid)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWiFiSSID(%q) error = %v, wantErr %v", tt.ssid, err, tt.wantErr)
			}
		})
	}
}

// TestValidateWiFiPassword tests WiFi password validation
func TestValidateWiFiPassword(t *testing.T) {
	tests := []struct {
		name         string
		password     string
		securityType string
		wantErr      bool
	}{
		{"Valid: WPA2 with 8 chars", "password", "WPA2", false},
		{"Valid: WPA2 with long password", "this-is-a-very-long-password-that-is-still-valid", "WPA2", false},
		{"Valid: WPA2 max length (63 chars)", "123456789012345678901234567890123456789012345678901234567890123", "WPA2", false},
		{"Valid: Open with empty password", "", "OPEN", false},
		{"Invalid: WPA2 empty password", "", "WPA2", true},
		{"Invalid: WPA2 too short (7 chars)", "1234567", "WPA2", true},
		{"Invalid: WPA2 too long (64 chars)", "1234567890123456789012345678901234567890123456789012345678901234", "WPA2", true},
		{"Invalid: Open with password", "password", "OPEN", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWiFiPassword(tt.password, tt.securityType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWiFiPassword(%q, %q) error = %v, wantErr %v", tt.password, tt.securityType, err, tt.wantErr)
			}
		})
	}
}

// TestValidateWiFiSecurityType tests WiFi security type validation
func TestValidateWiFiSecurityType(t *testing.T) {
	tests := []struct {
		name    string
		secType string
		wantErr bool
	}{
		{"Valid: WPA2", "WPA2", false},
		{"Valid: OPEN", "OPEN", false},
		{"Invalid: WEP", "WEP", true},
		{"Invalid: WPA", "WPA", true},
		{"Invalid: empty", "", true},
		{"Invalid: lowercase", "wpa2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWiFiSecurityType(tt.secType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWiFiSecurityType(%q) error = %v, wantErr %v", tt.secType, err, tt.wantErr)
			}
		})
	}
}

// TestValidateWiFiConfig tests complete WiFi configuration validation
func TestValidateWiFiConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *WiFiConfig
		wantCount int
	}{
		{
			name: "Valid: WPA2 network",
			config: &WiFiConfig{
				SSID:         "MyNetwork",
				Password:     "password123",
				SecurityType: "WPA2",
			},
			wantCount: 0,
		},
		{
			name: "Valid: Open network",
			config: &WiFiConfig{
				SSID:         "PublicWiFi",
				Password:     "",
				SecurityType: "OPEN",
			},
			wantCount: 0,
		},
		{
			name: "Invalid: empty SSID",
			config: &WiFiConfig{
				SSID:         "",
				Password:     "password",
				SecurityType: "WPA2",
			},
			wantCount: 1,
		},
		{
			name: "Invalid: multiple errors",
			config: &WiFiConfig{
				SSID:         "",
				Password:     "short",
				SecurityType: "WEP",
			},
			wantCount: 2, // empty SSID + bad security type (password not checked for invalid security type)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateWiFiConfig(tt.config)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateWiFiConfig() got %d errors, want %d", len(errors), tt.wantCount)
				for i, err := range errors {
					t.Logf("  Error %d: %v", i+1, err)
				}
			}
		})
	}
}

// TestValidateServerDNS tests server DNS validation
func TestValidateServerDNS(t *testing.T) {
	tests := []struct {
		name    string
		dns     string
		wantErr bool
	}{
		{"Valid: hostname", "lb.smartap-tech.com", false},
		{"Valid: IP address", "192.168.1.1", false},
		{"Valid: localhost", "localhost", false},
		{"Valid: long hostname", "this.is.a.very.long.hostname.example.com", false},
		{"Invalid: empty", "", true},
		{"Invalid: with spaces", "my server.com", true},
		{"Invalid: with tabs", "server\t.com", true},
		{"Invalid: with newline", "server\n.com", true},
		{"Invalid: too long (>253 chars)", string(make([]byte, 254)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerDNS(tt.dns)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServerDNS(%q) error = %v, wantErr %v", tt.dns, err, tt.wantErr)
			}
		})
	}
}

// TestValidateServerPort tests server port validation
func TestValidateServerPort(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"Valid: 80", 80, false},
		{"Valid: 443", 443, false},
		{"Valid: 8080", 8080, false},
		{"Valid: 1 (min)", 1, false},
		{"Valid: 65535 (max)", 65535, false},
		{"Invalid: 0", 0, true},
		{"Invalid: negative", -1, true},
		{"Invalid: too high", 65536, true},
		{"Invalid: way too high", 100000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerPort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateServerPort(%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

// TestValidateServerConfig tests complete server configuration validation
func TestValidateServerConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *ServerConfig
		wantCount int
	}{
		{
			name: "Valid: standard server",
			config: &ServerConfig{
				DNS:  "lb.smartap-tech.com",
				Port: 80,
			},
			wantCount: 0,
		},
		{
			name: "Valid: custom port",
			config: &ServerConfig{
				DNS:  "my-server.local",
				Port: 8080,
			},
			wantCount: 0,
		},
		{
			name: "Invalid: empty DNS",
			config: &ServerConfig{
				DNS:  "",
				Port: 80,
			},
			wantCount: 1,
		},
		{
			name: "Invalid: bad port",
			config: &ServerConfig{
				DNS:  "server.com",
				Port: 0,
			},
			wantCount: 1,
		},
		{
			name: "Invalid: multiple errors",
			config: &ServerConfig{
				DNS:  "",
				Port: -1,
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateServerConfig(tt.config)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateServerConfig() got %d errors, want %d", len(errors), tt.wantCount)
				for i, err := range errors {
					t.Logf("  Error %d: %v", i+1, err)
				}
			}
		})
	}
}

// TestValidateConfigUpdate tests complete configuration update validation
func TestValidateConfigUpdate(t *testing.T) {
	tests := []struct {
		name      string
		update    *ConfigUpdate
		wantCount int
	}{
		{
			name: "Valid: diverter only",
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  1,
					SecondPress: 2,
					ThirdPress:  4,
					K3Mode:      true,
				},
			},
			wantCount: 0,
		},
		{
			name: "Valid: all sections",
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  1,
					SecondPress: 2,
					ThirdPress:  4,
					K3Mode:      true,
				},
				WiFi: &WiFiConfig{
					SSID:         "MyNetwork",
					Password:     "password123",
					SecurityType: "WPA2",
				},
				Server: &ServerConfig{
					DNS:  "my-server.com",
					Port: 443,
				},
			},
			wantCount: 0,
		},
		{
			name: "Invalid: bad diverter",
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  10,
					SecondPress: 2,
					ThirdPress:  4,
					K3Mode:      true,
				},
			},
			wantCount: 1,
		},
		{
			name: "Invalid: multiple sections with errors",
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  10,
					SecondPress: 2,
					ThirdPress:  4,
					K3Mode:      true,
				},
				WiFi: &WiFiConfig{
					SSID:         "",
					Password:     "pass",
					SecurityType: "WPA2",
				},
			},
			wantCount: 3, // 1 diverter error + 1 SSID error + 1 password too short
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidateConfigUpdate(tt.update)
			if len(errors) != tt.wantCount {
				t.Errorf("ValidateConfigUpdate() got %d errors, want %d", len(errors), tt.wantCount)
				for i, err := range errors {
					t.Logf("  Error %d: %v", i+1, err)
				}
			}
		})
	}
}

// TestCheckLogicalConflicts tests logical conflict detection
func TestCheckLogicalConflicts(t *testing.T) {
	tests := []struct {
		name         string
		update       *ConfigUpdate
		wantWarnings int
	}{
		{
			name: "No conflicts: normal config",
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  1,
					SecondPress: 2,
					ThirdPress:  4,
					K3Mode:      true,
				},
			},
			wantWarnings: 0,
		},
		{
			name: "Warning: all zeros",
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  0,
					SecondPress: 0,
					ThirdPress:  0,
					K3Mode:      false,
				},
			},
			wantWarnings: 1,
		},
		{
			name: "Warning: all identical (non-zero)",
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  3,
					SecondPress: 3,
					ThirdPress:  3,
					K3Mode:      false,
				},
			},
			wantWarnings: 1,
		},
		{
			name: "No warning: two identical (still useful)",
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  1,
					SecondPress: 1,
					ThirdPress:  2,
					K3Mode:      false,
				},
			},
			wantWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts := CheckLogicalConflicts(tt.update)
			if len(conflicts) != tt.wantWarnings {
				t.Errorf("CheckLogicalConflicts() got %d warnings, want %d", len(conflicts), tt.wantWarnings)
				for i, conflict := range conflicts {
					t.Logf("  Warning %d: %v", i+1, conflict)
				}
			}

			// Verify all conflicts are warnings
			for _, conflict := range conflicts {
				if !IsWarning(conflict) {
					t.Errorf("Expected warning, got error: %v", conflict)
				}
			}
		})
	}
}

// TestValidateDeviceConfig tests device configuration validation
func TestValidateDeviceConfig(t *testing.T) {
	validConfig := &DeviceConfig{
		Serial:   "315260240",
		DNS:      "lb.smartap-tech.com",
		Port:     80,
		Outlet1:  1,
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: true,
		MAC:      "C4:BE:84:74:86:37",
	}

	t.Run("Valid config", func(t *testing.T) {
		errors := ValidateDeviceConfig(validConfig)
		if len(errors) != 0 {
			t.Errorf("ValidateDeviceConfig() got %d errors, want 0", len(errors))
			for i, err := range errors {
				t.Logf("  Error %d: %v", i+1, err)
			}
		}
	})

	t.Run("Invalid: empty serial", func(t *testing.T) {
		config := *validConfig
		config.Serial = ""
		errors := ValidateDeviceConfig(&config)
		if len(errors) == 0 {
			t.Error("Expected validation error for empty serial")
		}
	})

	t.Run("Invalid: empty MAC", func(t *testing.T) {
		config := *validConfig
		config.MAC = ""
		errors := ValidateDeviceConfig(&config)
		if len(errors) == 0 {
			t.Error("Expected validation error for empty MAC")
		}
	})

	t.Run("Invalid: bad bitmask", func(t *testing.T) {
		config := *validConfig
		config.Outlet1 = 10
		errors := ValidateDeviceConfig(&config)
		if len(errors) == 0 {
			t.Error("Expected validation error for invalid bitmask")
		}
	})
}

// TestFormatValidationErrors tests error formatting
func TestFormatValidationErrors(t *testing.T) {
	t.Run("No errors", func(t *testing.T) {
		result := FormatValidationErrors(nil)
		if result != "No validation errors" {
			t.Errorf("Expected 'No validation errors', got %q", result)
		}
	})

	t.Run("Single error", func(t *testing.T) {
		errors := []error{
			NewValidationError("test error"),
		}
		result := FormatValidationErrors(errors)
		if !contains(result, "1 error") {
			t.Errorf("Expected '1 error' in output, got: %s", result)
		}
		if !contains(result, "test error") {
			t.Errorf("Expected 'test error' in output, got: %s", result)
		}
	})

	t.Run("Multiple errors", func(t *testing.T) {
		errors := []error{
			NewValidationError("error 1"),
			NewValidationError("error 2"),
			NewValidationError("error 3"),
		}
		result := FormatValidationErrors(errors)
		if !contains(result, "3 error") {
			t.Errorf("Expected '3 error' in output, got: %s", result)
		}
	})
}

// TestSeparateWarningsAndErrors tests warning/error separation
func TestSeparateWarningsAndErrors(t *testing.T) {
	errors := []error{
		NewValidationError("critical error 1"),
		NewValidationError("warning: this is a warning"),
		NewValidationError("critical error 2"),
		NewValidationError("warning: another warning"),
	}

	warnings, criticalErrors := SeparateWarningsAndErrors(errors)

	if len(warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(warnings))
	}

	if len(criticalErrors) != 2 {
		t.Errorf("Expected 2 critical errors, got %d", len(criticalErrors))
	}

	// Verify warnings are actually warnings
	for _, w := range warnings {
		if !IsWarning(w) {
			t.Errorf("Expected warning, got: %v", w)
		}
	}

	// Verify critical errors are not warnings
	for _, e := range criticalErrors {
		if IsWarning(e) {
			t.Errorf("Expected error, got warning: %v", e)
		}
	}
}

// Helper function contains() is defined in models_test.go
