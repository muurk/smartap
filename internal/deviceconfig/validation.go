package deviceconfig

import (
	"fmt"
	"strings"
)

// ValidateDiverterBitmask validates a diverter button bitmask value.
// Valid range is 0-7 (3-bit bitmask for 3 outlets).
//
// Bitmask encoding:
//   - Bit 0 (1): Outlet 1
//   - Bit 1 (2): Outlet 2
//   - Bit 2 (4): Outlet 3
//
// Valid values:
//   - 0: No outlets (valid but unusual)
//   - 1: Outlet 1 only
//   - 2: Outlet 2 only
//   - 3: Outlets 1+2
//   - 4: Outlet 3 only
//   - 5: Outlets 1+3
//   - 6: Outlets 2+3
//   - 7: All three outlets
func ValidateDiverterBitmask(mask int) error {
	if mask < 0 || mask > 7 {
		return NewValidationError(fmt.Sprintf("diverter bitmask must be 0-7, got %d", mask))
	}
	return nil
}

// ValidateDiverterConfig validates a complete diverter configuration.
// Returns a slice of validation errors (empty if valid).
func ValidateDiverterConfig(config *DiverterConfig) []error {
	var errors []error

	if err := ValidateDiverterBitmask(config.FirstPress); err != nil {
		errors = append(errors, fmt.Errorf("first press: %w", err))
	}

	if err := ValidateDiverterBitmask(config.SecondPress); err != nil {
		errors = append(errors, fmt.Errorf("second press: %w", err))
	}

	if err := ValidateDiverterBitmask(config.ThirdPress); err != nil {
		errors = append(errors, fmt.Errorf("third press: %w", err))
	}

	return errors
}

// ValidateWiFiSSID validates a WiFi SSID.
// SSIDs must be non-empty and <= 32 characters (WiFi spec limit).
func ValidateWiFiSSID(ssid string) error {
	if ssid == "" {
		return NewValidationError("WiFi SSID cannot be empty")
	}
	if len(ssid) > 32 {
		return NewValidationError(fmt.Sprintf("WiFi SSID too long (max 32 chars): %d chars", len(ssid)))
	}
	return nil
}

// ValidateWiFiPassword validates a WiFi password.
// For WPA2: 8-63 characters (WiFi spec requirement)
// For Open: must be empty
func ValidateWiFiPassword(password string, securityType string) error {
	if securityType == "WPA2" {
		if password == "" {
			return NewValidationError("WiFi password required for WPA2 security")
		}
		if len(password) < 8 {
			return NewValidationError(fmt.Sprintf("WPA2 password too short (min 8 chars): %d chars", len(password)))
		}
		if len(password) > 63 {
			return NewValidationError(fmt.Sprintf("WPA2 password too long (max 63 chars): %d chars", len(password)))
		}
	} else if securityType == "OPEN" {
		if password != "" {
			return NewValidationError("WiFi password should be empty for open networks")
		}
	}
	return nil
}

// ValidateWiFiSecurityType validates the WiFi security type.
// Only "WPA2" and "OPEN" are supported by the device.
func ValidateWiFiSecurityType(securityType string) error {
	if securityType != "WPA2" && securityType != "OPEN" {
		return NewValidationError(fmt.Sprintf("WiFi security type must be 'WPA2' or 'OPEN', got '%s'", securityType))
	}
	return nil
}

// ValidateWiFiConfig validates a complete WiFi configuration.
// Returns a slice of validation errors (empty if valid).
func ValidateWiFiConfig(config *WiFiConfig) []error {
	var errors []error

	if err := ValidateWiFiSSID(config.SSID); err != nil {
		errors = append(errors, err)
	}

	if err := ValidateWiFiSecurityType(config.SecurityType); err != nil {
		errors = append(errors, err)
	}

	if err := ValidateWiFiPassword(config.Password, config.SecurityType); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// ValidateServerDNS validates a server DNS hostname or IP address.
// Basic validation: non-empty, reasonable length.
func ValidateServerDNS(dns string) error {
	if dns == "" {
		return NewValidationError("server DNS/hostname cannot be empty")
	}
	if len(dns) > 253 {
		return NewValidationError(fmt.Sprintf("server DNS/hostname too long (max 253 chars): %d chars", len(dns)))
	}
	// Basic format check: should not contain spaces or other invalid chars
	if strings.ContainsAny(dns, " \t\n\r") {
		return NewValidationError("server DNS/hostname contains invalid whitespace characters")
	}
	return nil
}

// ValidateServerPort validates a server port number.
// Valid range: 1-65535
func ValidateServerPort(port int) error {
	if port <= 0 || port > 65535 {
		return NewValidationError(fmt.Sprintf("server port must be 1-65535, got %d", port))
	}
	return nil
}

// ValidateServerConfig validates a complete server configuration.
// Returns a slice of validation errors (empty if valid).
func ValidateServerConfig(config *ServerConfig) []error {
	var errors []error

	if err := ValidateServerDNS(config.DNS); err != nil {
		errors = append(errors, err)
	}

	if err := ValidateServerPort(config.Port); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// ValidateConfigUpdate validates a complete configuration update.
// This is the main validation entry point for configuration updates.
// Returns a slice of validation errors (empty if valid).
func ValidateConfigUpdate(update *ConfigUpdate) []error {
	var allErrors []error

	// Validate diverter config if present
	if update.Diverter != nil {
		errors := ValidateDiverterConfig(update.Diverter)
		allErrors = append(allErrors, errors...)
	}

	// Validate WiFi config if present
	if update.WiFi != nil {
		errors := ValidateWiFiConfig(update.WiFi)
		allErrors = append(allErrors, errors...)
	}

	// Validate server config if present
	if update.Server != nil {
		errors := ValidateServerConfig(update.Server)
		allErrors = append(allErrors, errors...)
	}

	// Check for logical conflicts
	conflicts := CheckLogicalConflicts(update)
	allErrors = append(allErrors, conflicts...)

	return allErrors
}

// CheckLogicalConflicts checks for logical conflicts in the configuration.
// These are configurations that are valid individually but problematic together.
func CheckLogicalConflicts(update *ConfigUpdate) []error {
	var conflicts []error

	// Check for all-zero diverter configuration (no outlets ever active)
	if update.Diverter != nil {
		if update.Diverter.FirstPress == 0 && update.Diverter.SecondPress == 0 && update.Diverter.ThirdPress == 0 {
			conflicts = append(conflicts, NewValidationError(
				"warning: all button presses set to 0 (no outlets will activate)",
			))
		}
	}

	// Check for duplicate button configurations (may be intentional but worth warning)
	if update.Diverter != nil {
		first := update.Diverter.FirstPress
		second := update.Diverter.SecondPress
		third := update.Diverter.ThirdPress

		if first == second && second == third && first != 0 {
			conflicts = append(conflicts, NewValidationError(
				fmt.Sprintf("warning: all button presses identical (%d) - pressing button multiple times will have no effect", first),
			))
		}
	}

	return conflicts
}

// ValidateDeviceConfig validates a complete device configuration returned by GET.
// This is useful for verifying configurations read from the device.
// Returns a slice of validation errors (empty if valid).
func ValidateDeviceConfig(config *DeviceConfig) []error {
	var errors []error

	// Validate outlet bitmasks
	if err := ValidateDiverterBitmask(config.Outlet1); err != nil {
		errors = append(errors, fmt.Errorf("outlet1: %w", err))
	}
	if err := ValidateDiverterBitmask(config.Outlet2); err != nil {
		errors = append(errors, fmt.Errorf("outlet2: %w", err))
	}
	if err := ValidateDiverterBitmask(config.Outlet3); err != nil {
		errors = append(errors, fmt.Errorf("outlet3: %w", err))
	}

	// Validate server configuration
	if err := ValidateServerDNS(config.DNS); err != nil {
		errors = append(errors, fmt.Errorf("dns: %w", err))
	}
	if err := ValidateServerPort(config.Port); err != nil {
		errors = append(errors, fmt.Errorf("port: %w", err))
	}

	// Validate serial number is present
	if config.Serial == "" {
		errors = append(errors, NewValidationError("serial number is empty"))
	}

	// Validate MAC address format (basic check)
	if config.MAC == "" {
		errors = append(errors, NewValidationError("MAC address is empty"))
	}

	return errors
}

// FormatValidationErrors formats a slice of validation errors into a user-friendly message.
func FormatValidationErrors(errors []error) string {
	if len(errors) == 0 {
		return "No validation errors"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Configuration validation failed with %d error(s):\n", len(errors)))

	for i, err := range errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}

	return sb.String()
}

// IsWarning checks if a validation error is a warning (non-fatal).
// Warnings have error messages starting with "warning:".
func IsWarning(err error) bool {
	// Check if it's a DeviceError and inspect the Message field
	if devErr, ok := err.(*DeviceError); ok {
		return strings.HasPrefix(devErr.Message, "warning:")
	}
	// Fallback to checking the error string
	return strings.Contains(err.Error(), "warning:")
}

// SeparateWarningsAndErrors separates validation errors into warnings and errors.
// Warnings are non-fatal issues that the user should be aware of.
// Errors are fatal issues that prevent configuration from being sent.
func SeparateWarningsAndErrors(errors []error) (warnings []error, criticalErrors []error) {
	for _, err := range errors {
		if IsWarning(err) {
			warnings = append(warnings, err)
		} else {
			criticalErrors = append(criticalErrors, err)
		}
	}
	return warnings, criticalErrors
}
