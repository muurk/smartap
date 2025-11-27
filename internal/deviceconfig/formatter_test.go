package deviceconfig

import (
	"strings"
	"testing"
)

// Test fixture: sample device configuration
func getSampleDeviceConfig() *DeviceConfig {
	return &DeviceConfig{
		SSIDList:     []string{"NETGEAR89", "MyHomeWiFi"},
		LowPowerMode: false,
		Serial:       "315260240",
		DNS:          "lb.smartap-tech.com",
		Port:         80,
		Outlet1:      1,
		Outlet2:      2,
		Outlet3:      4,
		K3Outlet:     true,
		SWVer:        "0x355",
		WNPVer:       "2.0.0",
		MAC:          "C4:BE:84:74:86:37",
	}
}

func TestDeviceConfig_Summary(t *testing.T) {
	dc := getSampleDeviceConfig()
	summary := dc.Summary()

	// Should be one line
	if strings.Count(summary, "\n") > 0 {
		t.Error("Summary() should return a single line")
	}

	// Should contain key information
	expectedParts := []string{"315260240", "lb.smartap-tech.com", "80", "0x355"}
	for _, part := range expectedParts {
		if !strings.Contains(summary, part) {
			t.Errorf("Summary() missing expected part: %s", part)
		}
	}
}

func TestDeviceConfig_FormatDeviceInfo(t *testing.T) {
	dc := getSampleDeviceConfig()
	info := dc.FormatDeviceInfo()

	// Should contain device information
	expectedParts := []string{
		"Device Information",
		"315260240",
		"C4:BE:84:74:86:37",
		"0x355",
		"2.0.0",
	}

	for _, part := range expectedParts {
		if !strings.Contains(info, part) {
			t.Errorf("FormatDeviceInfo() missing expected part: %s", part)
		}
	}
}

func TestDeviceConfig_FormatServerConfig(t *testing.T) {
	dc := getSampleDeviceConfig()
	config := dc.FormatServerConfig()

	expectedParts := []string{
		"Server Configuration",
		"lb.smartap-tech.com",
		"80",
		"http://",
	}

	for _, part := range expectedParts {
		if !strings.Contains(config, part) {
			t.Errorf("FormatServerConfig() missing expected part: %s", part)
		}
	}
}

func TestDeviceConfig_FormatWiFiConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *DeviceConfig
		expected []string
	}{
		{
			name: "with known networks",
			config: &DeviceConfig{
				SSIDList: []string{"NETGEAR89", "MyHomeWiFi"},
			},
			expected: []string{"WiFi Configuration", "NETGEAR89", "MyHomeWiFi"},
		},
		{
			name: "without known networks",
			config: &DeviceConfig{
				SSIDList: []string{},
			},
			expected: []string{"WiFi Configuration", "(none)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.FormatWiFiConfig()
			for _, exp := range tt.expected {
				if !strings.Contains(result, exp) {
					t.Errorf("FormatWiFiConfig() missing expected: %s", exp)
				}
			}
		})
	}
}

func TestDeviceConfig_FormatOutletConfig(t *testing.T) {
	dc := getSampleDeviceConfig()
	config := dc.FormatOutletConfig()

	expectedParts := []string{
		"Diverter Button Configuration",
		"1st Button Press",
		"2nd Button Press",
		"3rd Button Press",
		"3rd Knob Configuration",
		"SEPARATED",
	}

	for _, part := range expectedParts {
		if !strings.Contains(config, part) {
			t.Errorf("FormatOutletConfig() missing expected part: %s", part)
		}
	}
}

func TestDeviceConfig_FormatOutletConfig_K3Disabled(t *testing.T) {
	dc := getSampleDeviceConfig()
	dc.K3Outlet = false
	config := dc.FormatOutletConfig()

	if !strings.Contains(config, "STANDARD") {
		t.Error("FormatOutletConfig() should show STANDARD when K3 is disabled")
	}
}

func TestDeviceConfig_FormatCompact(t *testing.T) {
	dc := getSampleDeviceConfig()
	compact := dc.FormatCompact()

	// Should be relatively short (less than 10 lines)
	lines := strings.Split(compact, "\n")
	if len(lines) > 10 {
		t.Errorf("FormatCompact() should be compact, got %d lines", len(lines))
	}

	// Should contain key information
	expectedParts := []string{
		"315260240",
		"C4:BE:84:74:86:37",
		"0x355",
		"lb.smartap-tech.com",
	}

	for _, part := range expectedParts {
		if !strings.Contains(compact, part) {
			t.Errorf("FormatCompact() missing expected part: %s", part)
		}
	}
}

func TestDeviceConfig_FormatDetailed(t *testing.T) {
	dc := getSampleDeviceConfig()
	detailed := dc.FormatDetailed()

	// Should contain all sections
	expectedSections := []string{
		"SMARTAP DEVICE CONFIGURATION",
		"Device Information",
		"Server Configuration",
		"WiFi Configuration",
		"Diverter Button Configuration",
		"3rd Knob Configuration",
	}

	for _, section := range expectedSections {
		if !strings.Contains(detailed, section) {
			t.Errorf("FormatDetailed() missing section: %s", section)
		}
	}

	// Should have decorative box
	if !strings.Contains(detailed, "╔") || !strings.Contains(detailed, "╚") {
		t.Error("FormatDetailed() should have decorative box characters")
	}
}

func TestFormatBitmaskDetailed(t *testing.T) {
	tests := []struct {
		mask     int
		contains []string
	}{
		{0, []string{"0", "No outlets"}},
		{1, []string{"1", "Outlet 1"}},
		{2, []string{"2", "Outlet 2"}},
		{3, []string{"3", "Outlet 1", "Outlet 2"}},
		{4, []string{"4", "Outlet 3"}},
		{7, []string{"7", "Outlet 1", "Outlet 2", "Outlet 3"}},
	}

	for _, tt := range tests {
		t.Run(FormatBitmask(tt.mask), func(t *testing.T) {
			result := FormatBitmaskDetailed(tt.mask)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatBitmaskDetailed(%d) missing: %s, got: %s", tt.mask, expected, result)
				}
			}
		})
	}
}

func TestFormatBitmaskTable(t *testing.T) {
	table := FormatBitmaskTable()

	// Should have header
	if !strings.Contains(table, "Diverter Bitmask Reference") {
		t.Error("FormatBitmaskTable() missing header")
	}

	// Should list all values 0-7
	for i := 0; i <= 7; i++ {
		expected := string(rune('0' + i))
		if !strings.Contains(table, expected) {
			t.Errorf("FormatBitmaskTable() missing value: %d", i)
		}
	}

	// Should have explanation
	if !strings.Contains(table, "binary flags") {
		t.Error("FormatBitmaskTable() missing explanation")
	}
}

func TestConfigUpdate_FormatChanges(t *testing.T) {
	tests := []struct {
		name     string
		update   *ConfigUpdate
		contains []string
	}{
		{
			name: "diverter changes",
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  1,
					SecondPress: 2,
					ThirdPress:  3,
					K3Mode:      true,
				},
			},
			contains: []string{"Diverter Configuration", "1st Press", "2nd Press", "3rd Press", "K3 Mode"},
		},
		{
			name: "server changes",
			update: &ConfigUpdate{
				Server: &ServerConfig{
					DNS:  "example.com",
					Port: 8080,
				},
			},
			contains: []string{"Server Configuration", "example.com", "8080"},
		},
		{
			name: "wifi changes",
			update: &ConfigUpdate{
				WiFi: &WiFiConfig{
					SSID:         "MyNetwork",
					Password:     "secret123",
					SecurityType: "WPA2",
				},
			},
			contains: []string{"WiFi Configuration", "MyNetwork", "WPA2", "********"},
		},
		{
			name: "wifi open network",
			update: &ConfigUpdate{
				WiFi: &WiFiConfig{
					SSID:         "OpenNet",
					SecurityType: "OPEN",
				},
			},
			contains: []string{"WiFi Configuration", "OpenNet", "OPEN"},
		},
		{
			name:     "no changes",
			update:   &ConfigUpdate{},
			contains: []string{"no changes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.update.FormatChanges()
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatChanges() missing: %s", expected)
				}
			}
		})
	}
}

func TestFormatDiff(t *testing.T) {
	oldConfig := getSampleDeviceConfig()
	newConfig := getSampleDeviceConfig()

	// Test no changes
	t.Run("no changes", func(t *testing.T) {
		diff := FormatDiff(oldConfig, newConfig)
		if !strings.Contains(diff, "no differences") {
			t.Error("FormatDiff() should show no differences for identical configs")
		}
	})

	// Test outlet changes
	t.Run("outlet changes", func(t *testing.T) {
		newConfig.Outlet1 = 3
		newConfig.Outlet2 = 5
		diff := FormatDiff(oldConfig, newConfig)

		if !strings.Contains(diff, "1st Press") {
			t.Error("FormatDiff() should show 1st press change")
		}
		if !strings.Contains(diff, "2nd Press") {
			t.Error("FormatDiff() should show 2nd press change")
		}
		if !strings.Contains(diff, "→") {
			t.Error("FormatDiff() should show arrow indicating change")
		}
	})

	// Test K3 mode change
	t.Run("K3 mode change", func(t *testing.T) {
		newConfig2 := getSampleDeviceConfig()
		newConfig2.K3Outlet = false
		diff := FormatDiff(oldConfig, newConfig2)

		if !strings.Contains(diff, "K3 Mode") {
			t.Error("FormatDiff() should show K3 mode change")
		}
		if !strings.Contains(diff, "true → false") {
			t.Error("FormatDiff() should show K3 mode value change")
		}
	})

	// Test server changes
	t.Run("server changes", func(t *testing.T) {
		newConfig3 := getSampleDeviceConfig()
		newConfig3.DNS = "new.server.com"
		newConfig3.Port = 8080
		diff := FormatDiff(oldConfig, newConfig3)

		if !strings.Contains(diff, "Server Configuration") {
			t.Error("FormatDiff() should show server configuration section")
		}
		if !strings.Contains(diff, "new.server.com") {
			t.Error("FormatDiff() should show new DNS")
		}
		if !strings.Contains(diff, "8080") {
			t.Error("FormatDiff() should show new port")
		}
	})
}

func TestDeviceConfig_String(t *testing.T) {
	// Test that existing String() method still works
	dc := getSampleDeviceConfig()
	str := dc.String()

	expectedParts := []string{
		"Smartap Device",
		"315260240",
		"C4:BE:84:74:86:37",
		"0x355",
		"lb.smartap-tech.com",
	}

	for _, part := range expectedParts {
		if !strings.Contains(str, part) {
			t.Errorf("String() missing expected part: %s", part)
		}
	}
}
