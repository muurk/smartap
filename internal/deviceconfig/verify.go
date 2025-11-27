package deviceconfig

import (
	"fmt"
	"time"
)

// VerificationOptions configures how configuration verification behaves
type VerificationOptions struct {
	// MaxRetries is the maximum number of verification attempts
	// Default: 3
	MaxRetries int

	// InitialDelay is the delay before the first verification attempt
	// This gives the device time to apply the configuration
	// Default: 500ms
	InitialDelay time.Duration

	// RetryDelay is the delay between retry attempts
	// Default: 1s
	RetryDelay time.Duration

	// UseExponentialBackoff enables exponential backoff for retries
	// If true, each retry delay is doubled (up to MaxRetryDelay)
	// Default: true
	UseExponentialBackoff bool

	// MaxRetryDelay is the maximum delay between retries when using exponential backoff
	// Default: 5s
	MaxRetryDelay time.Duration
}

// DefaultVerificationOptions returns sensible defaults for verification
func DefaultVerificationOptions() *VerificationOptions {
	return &VerificationOptions{
		MaxRetries:            3,
		InitialDelay:          500 * time.Millisecond,
		RetryDelay:            1 * time.Second,
		UseExponentialBackoff: true,
		MaxRetryDelay:         5 * time.Second,
	}
}

// VerificationResult contains the results of a configuration verification
type VerificationResult struct {
	// Success indicates whether verification succeeded
	Success bool

	// Attempts is the number of attempts made
	Attempts int

	// ActualConfig is the configuration retrieved from the device
	ActualConfig *DeviceConfig

	// Mismatches lists all detected mismatches between expected and actual config
	Mismatches []string

	// Error is any error that occurred during verification
	Error error
}

// VerifyConfigurationWithRetry verifies that a configuration was successfully applied to the device
// This function includes retry logic with exponential backoff for handling timing issues
func (c *Client) VerifyConfigurationWithRetry(expected *ConfigUpdate, opts *VerificationOptions) *VerificationResult {
	if opts == nil {
		opts = DefaultVerificationOptions()
	}

	result := &VerificationResult{
		Success:    false,
		Attempts:   0,
		Mismatches: []string{},
	}

	// Initial delay to give device time to apply changes
	time.Sleep(opts.InitialDelay)

	currentDelay := opts.RetryDelay

	// Retry loop
	for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
		result.Attempts++

		// Delay before retry (not on first attempt)
		if attempt > 0 {
			time.Sleep(currentDelay)

			// Exponential backoff
			if opts.UseExponentialBackoff {
				currentDelay *= 2
				if currentDelay > opts.MaxRetryDelay {
					currentDelay = opts.MaxRetryDelay
				}
			}
		}

		// Get current configuration
		current, err := c.GetConfiguration()
		if err != nil {
			result.Error = fmt.Errorf("attempt %d: failed to retrieve configuration: %w", attempt+1, err)
			// Don't give up on network errors - retry
			continue
		}

		result.ActualConfig = current

		// Verify configuration
		mismatches := verifyConfigurationMatch(expected, current)
		result.Mismatches = mismatches

		if len(mismatches) == 0 {
			// Success!
			result.Success = true
			return result
		}

		// If this isn't the last attempt, we'll retry
		if attempt < opts.MaxRetries {
			result.Error = fmt.Errorf("attempt %d: configuration mismatch (will retry)", attempt+1)
		} else {
			// Last attempt failed
			result.Error = fmt.Errorf("verification failed after %d attempts: %s", result.Attempts, formatMismatches(mismatches))
		}
	}

	return result
}

// verifyConfigurationMatch compares expected configuration with actual device config
// Returns a list of mismatches (empty if all matches)
func verifyConfigurationMatch(expected *ConfigUpdate, actual *DeviceConfig) []string {
	var mismatches []string

	// Verify diverter configuration
	if expected.Diverter != nil {
		if actual.Outlet1 != expected.Diverter.FirstPress {
			mismatches = append(mismatches, fmt.Sprintf("diverter first press: expected %d, got %d", expected.Diverter.FirstPress, actual.Outlet1))
		}
		if actual.Outlet2 != expected.Diverter.SecondPress {
			mismatches = append(mismatches, fmt.Sprintf("diverter second press: expected %d, got %d", expected.Diverter.SecondPress, actual.Outlet2))
		}
		if actual.Outlet3 != expected.Diverter.ThirdPress {
			mismatches = append(mismatches, fmt.Sprintf("diverter third press: expected %d, got %d", expected.Diverter.ThirdPress, actual.Outlet3))
		}
		if actual.K3Outlet != expected.Diverter.K3Mode {
			mismatches = append(mismatches, fmt.Sprintf("K3 mode: expected %v, got %v", expected.Diverter.K3Mode, actual.K3Outlet))
		}
	}

	// Verify server configuration
	if expected.Server != nil {
		if actual.DNS != expected.Server.DNS {
			mismatches = append(mismatches, fmt.Sprintf("DNS: expected %s, got %s", expected.Server.DNS, actual.DNS))
		}
		if actual.Port != expected.Server.Port {
			mismatches = append(mismatches, fmt.Sprintf("port: expected %d, got %d", expected.Server.Port, actual.Port))
		}
	}

	// Note: WiFi config cannot be verified (device doesn't return credentials)

	return mismatches
}

// formatMismatches creates a human-readable summary of mismatches
func formatMismatches(mismatches []string) string {
	if len(mismatches) == 0 {
		return "none"
	}
	if len(mismatches) == 1 {
		return mismatches[0]
	}
	result := fmt.Sprintf("%d mismatches: ", len(mismatches))
	for i, m := range mismatches {
		if i > 0 {
			result += "; "
		}
		result += m
	}
	return result
}

// UpdateAndVerify is a convenience method that updates configuration and verifies it was applied
// This combines UpdateConfiguration and VerifyConfigurationWithRetry in a single call
func (c *Client) UpdateAndVerify(update *ConfigUpdate, opts *VerificationOptions) *VerificationResult {
	// Update configuration
	err := c.UpdateConfiguration(update)
	if err != nil {
		return &VerificationResult{
			Success:  false,
			Attempts: 0,
			Error:    fmt.Errorf("update failed: %w", err),
		}
	}

	// Verify it was applied
	return c.VerifyConfigurationWithRetry(update, opts)
}

// VerifyDiverter is a convenience method to verify just the diverter configuration
func (c *Client) VerifyDiverter(expected *DiverterConfig, opts *VerificationOptions) *VerificationResult {
	update := &ConfigUpdate{
		Diverter: expected,
	}
	return c.VerifyConfigurationWithRetry(update, opts)
}

// VerifyServer is a convenience method to verify just the server configuration
func (c *Client) VerifyServer(expected *ServerConfig, opts *VerificationOptions) *VerificationResult {
	update := &ConfigUpdate{
		Server: expected,
	}
	return c.VerifyConfigurationWithRetry(update, opts)
}
