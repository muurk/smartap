package deviceconfig

import (
	"fmt"
	"strings"
)

// Summary returns a one-line summary of the device configuration
func (dc *DeviceConfig) Summary() string {
	return fmt.Sprintf("Smartap %s @ %s:%d (FW: %s)", dc.Serial, dc.DNS, dc.Port, dc.SWVer)
}

// FormatDeviceInfo returns a formatted string with device identification information
func (dc *DeviceConfig) FormatDeviceInfo() string {
	var b strings.Builder

	b.WriteString("=== Device Information ===\n")
	b.WriteString(fmt.Sprintf("Serial Number:  %s\n", dc.Serial))
	b.WriteString(fmt.Sprintf("MAC Address:    %s\n", dc.MAC))
	b.WriteString(fmt.Sprintf("Firmware:       %s\n", dc.SWVer))
	b.WriteString(fmt.Sprintf("WNP Version:    %s\n", dc.WNPVer))
	b.WriteString(fmt.Sprintf("Low Power Mode: %v\n", dc.LowPowerMode))

	return b.String()
}

// FormatServerConfig returns a formatted string with server configuration
func (dc *DeviceConfig) FormatServerConfig() string {
	var b strings.Builder

	b.WriteString("=== Server Configuration ===\n")
	b.WriteString(fmt.Sprintf("DNS Hostname: %s\n", dc.DNS))
	b.WriteString(fmt.Sprintf("Port:         %d\n", dc.Port))
	b.WriteString(fmt.Sprintf("Full URL:     http://%s:%d\n", dc.DNS, dc.Port))

	return b.String()
}

// FormatWiFiConfig returns a formatted string with WiFi configuration
func (dc *DeviceConfig) FormatWiFiConfig() string {
	var b strings.Builder

	b.WriteString("=== WiFi Configuration ===\n")
	if len(dc.SSIDList) > 0 {
		b.WriteString(fmt.Sprintf("Known Networks: %s\n", strings.Join(dc.SSIDList, ", ")))
	} else {
		b.WriteString("Known Networks: (none)\n")
	}

	return b.String()
}

// FormatOutletConfig returns a formatted string with detailed outlet configuration
func (dc *DeviceConfig) FormatOutletConfig() string {
	var b strings.Builder

	b.WriteString("=== Diverter Button Configuration ===\n")
	b.WriteString(fmt.Sprintf("1st Button Press: %s (value: %d)\n", FormatBitmask(dc.Outlet1), dc.Outlet1))
	b.WriteString(fmt.Sprintf("2nd Button Press: %s (value: %d)\n", FormatBitmask(dc.Outlet2), dc.Outlet2))
	b.WriteString(fmt.Sprintf("3rd Button Press: %s (value: %d)\n", FormatBitmask(dc.Outlet3), dc.Outlet3))
	b.WriteString("\n")

	b.WriteString("=== 3rd Knob Configuration ===\n")
	if dc.K3Outlet {
		b.WriteString("3rd Knob Mode: SEPARATED (dedicated control)\n")
	} else {
		b.WriteString("3rd Knob Mode: STANDARD (follows button presses)\n")
	}

	return b.String()
}

// FormatCompact returns a compact multi-line format suitable for terminal display
func (dc *DeviceConfig) FormatCompact() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Device:  %s (MAC: %s)\n", dc.Serial, dc.MAC))
	b.WriteString(fmt.Sprintf("Firmware: %s\n", dc.SWVer))
	b.WriteString(fmt.Sprintf("Server:  %s:%d\n", dc.DNS, dc.Port))
	b.WriteString(fmt.Sprintf("Buttons: [1:%s] [2:%s] [3:%s]\n",
		FormatBitmask(dc.Outlet1),
		FormatBitmask(dc.Outlet2),
		FormatBitmask(dc.Outlet3)))
	b.WriteString(fmt.Sprintf("K3 Mode: %v\n", dc.K3Outlet))

	return b.String()
}

// FormatDetailed returns a comprehensive formatted string with all configuration details
func (dc *DeviceConfig) FormatDetailed() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("╔════════════════════════════════════════════════════════════════╗\n")
	b.WriteString("║              SMARTAP DEVICE CONFIGURATION                      ║\n")
	b.WriteString("╚════════════════════════════════════════════════════════════════╝\n")
	b.WriteString("\n")

	b.WriteString(dc.FormatDeviceInfo())
	b.WriteString("\n")
	b.WriteString(dc.FormatServerConfig())
	b.WriteString("\n")
	b.WriteString(dc.FormatWiFiConfig())
	b.WriteString("\n")
	b.WriteString(dc.FormatOutletConfig())

	return b.String()
}

// FormatBitmaskDetailed returns a detailed explanation of a diverter bitmask
func FormatBitmaskDetailed(mask int) string {
	outlets := DecodeBitmask(mask)
	if len(outlets) == 0 {
		return fmt.Sprintf("Bitmask %d: No outlets enabled (OFF)", mask)
	}

	outletNames := make([]string, len(outlets))
	for i, outlet := range outlets {
		outletNames[i] = fmt.Sprintf("Outlet %d", outlet)
	}

	return fmt.Sprintf("Bitmask %d: %s", mask, strings.Join(outletNames, " + "))
}

// FormatBitmaskTable returns a table showing all possible bitmask values
func FormatBitmaskTable() string {
	var b strings.Builder

	b.WriteString("=== Diverter Bitmask Reference ===\n")
	b.WriteString("Value | Outlets Enabled\n")
	b.WriteString("------+------------------\n")

	for i := 0; i <= 7; i++ {
		b.WriteString(fmt.Sprintf("  %d   | %s\n", i, FormatBitmask(i)))
	}

	b.WriteString("\nNote: Bitmasks combine outlets using binary flags:\n")
	b.WriteString("  Bit 0 (value 1) = Outlet 1\n")
	b.WriteString("  Bit 1 (value 2) = Outlet 2\n")
	b.WriteString("  Bit 2 (value 4) = Outlet 3\n")
	b.WriteString("  Combine values to enable multiple outlets\n")

	return b.String()
}

// FormatConfigUpdate returns a formatted string showing what will be changed
func (cu *ConfigUpdate) FormatChanges() string {
	var b strings.Builder
	changes := 0

	b.WriteString("=== Configuration Changes ===\n")

	if cu.Diverter != nil {
		b.WriteString("\nDiverter Configuration:\n")
		b.WriteString(fmt.Sprintf("  1st Press: %s (value: %d)\n", FormatBitmask(cu.Diverter.FirstPress), cu.Diverter.FirstPress))
		b.WriteString(fmt.Sprintf("  2nd Press: %s (value: %d)\n", FormatBitmask(cu.Diverter.SecondPress), cu.Diverter.SecondPress))
		b.WriteString(fmt.Sprintf("  3rd Press: %s (value: %d)\n", FormatBitmask(cu.Diverter.ThirdPress), cu.Diverter.ThirdPress))
		b.WriteString(fmt.Sprintf("  K3 Mode:   %v\n", cu.Diverter.K3Mode))
		changes++
	}

	if cu.Server != nil {
		b.WriteString("\nServer Configuration:\n")
		b.WriteString(fmt.Sprintf("  DNS:  %s\n", cu.Server.DNS))
		b.WriteString(fmt.Sprintf("  Port: %d\n", cu.Server.Port))
		changes++
	}

	if cu.WiFi != nil {
		b.WriteString("\nWiFi Configuration:\n")
		b.WriteString(fmt.Sprintf("  SSID:     %s\n", cu.WiFi.SSID))
		b.WriteString(fmt.Sprintf("  Security: %s\n", cu.WiFi.SecurityType))
		if cu.WiFi.SecurityType != "OPEN" {
			b.WriteString("  Password: ********\n")
		}
		changes++
	}

	if changes == 0 {
		b.WriteString("(no changes specified)\n")
	}

	return b.String()
}

// FormatDiff returns a formatted diff between two configurations
func FormatDiff(old, new *DeviceConfig) string {
	var b strings.Builder

	b.WriteString("=== Configuration Differences ===\n")

	hasChanges := false

	// Check diverter changes
	if old.Outlet1 != new.Outlet1 || old.Outlet2 != new.Outlet2 || old.Outlet3 != new.Outlet3 || old.K3Outlet != new.K3Outlet {
		b.WriteString("\nDiverter Configuration:\n")
		if old.Outlet1 != new.Outlet1 {
			b.WriteString(fmt.Sprintf("  1st Press: %s → %s\n", FormatBitmask(old.Outlet1), FormatBitmask(new.Outlet1)))
			hasChanges = true
		}
		if old.Outlet2 != new.Outlet2 {
			b.WriteString(fmt.Sprintf("  2nd Press: %s → %s\n", FormatBitmask(old.Outlet2), FormatBitmask(new.Outlet2)))
			hasChanges = true
		}
		if old.Outlet3 != new.Outlet3 {
			b.WriteString(fmt.Sprintf("  3rd Press: %s → %s\n", FormatBitmask(old.Outlet3), FormatBitmask(new.Outlet3)))
			hasChanges = true
		}
		if old.K3Outlet != new.K3Outlet {
			b.WriteString(fmt.Sprintf("  K3 Mode:   %v → %v\n", old.K3Outlet, new.K3Outlet))
			hasChanges = true
		}
	}

	// Check server changes
	if old.DNS != new.DNS || old.Port != new.Port {
		b.WriteString("\nServer Configuration:\n")
		if old.DNS != new.DNS {
			b.WriteString(fmt.Sprintf("  DNS:  %s → %s\n", old.DNS, new.DNS))
			hasChanges = true
		}
		if old.Port != new.Port {
			b.WriteString(fmt.Sprintf("  Port: %d → %d\n", old.Port, new.Port))
			hasChanges = true
		}
	}

	if !hasChanges {
		b.WriteString("\n(no differences detected)\n")
	}

	return b.String()
}
