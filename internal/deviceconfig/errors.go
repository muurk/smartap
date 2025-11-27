package deviceconfig

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
)

// Error types for device configuration operations

// ErrorType represents the category of error that occurred
type ErrorType int

const (
	// ErrTypeNetwork indicates a network-level error (connection refused, timeout, etc.)
	ErrTypeNetwork ErrorType = iota
	// ErrTypeAuth indicates an authentication failure (invalid credentials)
	ErrTypeAuth
	// ErrTypeHTTP indicates an HTTP-level error (non-200 status code)
	ErrTypeHTTP
	// ErrTypeParse indicates a parsing error (malformed JSON, invalid response)
	ErrTypeParse
	// ErrTypeValidation indicates a validation error (invalid configuration)
	ErrTypeValidation
	// ErrTypeTimeout indicates a request timeout
	ErrTypeTimeout
	// ErrTypeConnectionRefused indicates the device refused the connection
	ErrTypeConnectionRefused
	// ErrTypeDNS indicates a DNS resolution failure
	ErrTypeDNS
	// ErrTypeUnknown indicates an unknown or unexpected error
	ErrTypeUnknown
)

// NetworkErrorSubtype provides more specific network error classification
type NetworkErrorSubtype int

const (
	NetworkErrorGeneral NetworkErrorSubtype = iota
	NetworkErrorTimeout
	NetworkErrorConnectionRefused
	NetworkErrorDNS
	NetworkErrorHostUnreachable
	NetworkErrorNetworkUnreachable
)

// String returns a human-readable name for the error type
func (et ErrorType) String() string {
	switch et {
	case ErrTypeNetwork:
		return "Network Error"
	case ErrTypeAuth:
		return "Authentication Error"
	case ErrTypeHTTP:
		return "HTTP Error"
	case ErrTypeParse:
		return "Parse Error"
	case ErrTypeValidation:
		return "Validation Error"
	case ErrTypeTimeout:
		return "Timeout"
	case ErrTypeConnectionRefused:
		return "Connection Refused"
	case ErrTypeDNS:
		return "DNS Error"
	case ErrTypeUnknown:
		return "Unknown Error"
	default:
		return fmt.Sprintf("ErrorType(%d)", et)
	}
}

// DeviceError represents an error that occurred during device communication
type DeviceError struct {
	Type           ErrorType           // Category of error
	Message        string              // Human-readable error message
	StatusCode     int                 // HTTP status code (if applicable)
	Err            error               // Underlying error (if any)
	NetworkSubtype NetworkErrorSubtype // More specific network error type
	DeviceIP       string              // Device IP address (for context)
	Retryable      bool                // Whether the error is retryable
}

// Error implements the error interface
func (e *DeviceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error for error chain inspection
func (e *DeviceError) Unwrap() error {
	return e.Err
}

// ClassifyNetworkError analyzes an error and returns a more specific error type
func ClassifyNetworkError(err error, deviceIP string) *DeviceError {
	if err == nil {
		return nil
	}

	// Check for timeout errors
	if os.IsTimeout(err) {
		return &DeviceError{
			Type:           ErrTypeTimeout,
			Message:        "Request timed out",
			Err:            err,
			NetworkSubtype: NetworkErrorTimeout,
			DeviceIP:       deviceIP,
			Retryable:      true,
		}
	}

	// Check for DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &DeviceError{
			Type:           ErrTypeDNS,
			Message:        fmt.Sprintf("DNS resolution failed for %s", dnsErr.Name),
			Err:            err,
			NetworkSubtype: NetworkErrorDNS,
			DeviceIP:       deviceIP,
			Retryable:      false,
		}
	}

	// Check for connection refused
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if errors.Is(opErr.Err, syscall.ECONNREFUSED) {
			return &DeviceError{
				Type:           ErrTypeConnectionRefused,
				Message:        "Device refused connection",
				Err:            err,
				NetworkSubtype: NetworkErrorConnectionRefused,
				DeviceIP:       deviceIP,
				Retryable:      true,
			}
		}
		if errors.Is(opErr.Err, syscall.EHOSTUNREACH) {
			return &DeviceError{
				Type:           ErrTypeNetwork,
				Message:        "Host unreachable",
				Err:            err,
				NetworkSubtype: NetworkErrorHostUnreachable,
				DeviceIP:       deviceIP,
				Retryable:      true,
			}
		}
		if errors.Is(opErr.Err, syscall.ENETUNREACH) {
			return &DeviceError{
				Type:           ErrTypeNetwork,
				Message:        "Network unreachable",
				Err:            err,
				NetworkSubtype: NetworkErrorNetworkUnreachable,
				DeviceIP:       deviceIP,
				Retryable:      true,
			}
		}
	}

	// Check for URL errors
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// Recursively classify the underlying error
		return ClassifyNetworkError(urlErr.Err, deviceIP)
	}

	// Generic network error
	return &DeviceError{
		Type:           ErrTypeNetwork,
		Message:        "Network error occurred",
		Err:            err,
		NetworkSubtype: NetworkErrorGeneral,
		DeviceIP:       deviceIP,
		Retryable:      true,
	}
}

// NewNetworkError creates a network-level error with automatic classification
func NewNetworkError(message string, err error) *DeviceError {
	classified := ClassifyNetworkError(err, "")
	if classified != nil {
		classified.Message = message
		return classified
	}
	return &DeviceError{
		Type:      ErrTypeNetwork,
		Message:   message,
		Err:       err,
		Retryable: true,
	}
}

// NewAuthError creates an authentication error
func NewAuthError(message string) *DeviceError {
	return &DeviceError{
		Type:       ErrTypeAuth,
		Message:    message,
		StatusCode: http.StatusUnauthorized,
		Retryable:  false,
	}
}

// NewHTTPError creates an HTTP-level error
func NewHTTPError(statusCode int, message string) *DeviceError {
	retryable := statusCode >= 500 // Server errors are retryable
	return &DeviceError{
		Type:       ErrTypeHTTP,
		Message:    message,
		StatusCode: statusCode,
		Retryable:  retryable,
	}
}

// NewParseError creates a parsing error
func NewParseError(message string, err error) *DeviceError {
	return &DeviceError{
		Type:      ErrTypeParse,
		Message:   message,
		Err:       err,
		Retryable: false,
	}
}

// NewValidationError creates a validation error
func NewValidationError(message string) *DeviceError {
	return &DeviceError{
		Type:      ErrTypeValidation,
		Message:   message,
		Retryable: false,
	}
}

// IsNetworkError checks if an error is a network error (including timeout, connection refused, DNS, etc.)
func IsNetworkError(err error) bool {
	if devErr, ok := err.(*DeviceError); ok {
		return devErr.Type == ErrTypeNetwork ||
			devErr.Type == ErrTypeTimeout ||
			devErr.Type == ErrTypeConnectionRefused ||
			devErr.Type == ErrTypeDNS
	}
	return false
}

// IsAuthError checks if an error is an authentication error
func IsAuthError(err error) bool {
	if devErr, ok := err.(*DeviceError); ok {
		return devErr.Type == ErrTypeAuth
	}
	return false
}

// IsHTTPError checks if an error is an HTTP error
func IsHTTPError(err error) bool {
	if devErr, ok := err.(*DeviceError); ok {
		return devErr.Type == ErrTypeHTTP
	}
	return false
}

// IsParseError checks if an error is a parse error
func IsParseError(err error) bool {
	if devErr, ok := err.(*DeviceError); ok {
		return devErr.Type == ErrTypeParse
	}
	return false
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if devErr, ok := err.(*DeviceError); ok {
		return devErr.Type == ErrTypeValidation
	}
	return false
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	if devErr, ok := err.(*DeviceError); ok {
		return devErr.Retryable
	}
	// Unknown errors are not retryable by default
	return false
}

// GetTroubleshootingHint returns user-friendly troubleshooting advice for an error
func GetTroubleshootingHint(err error) string {
	devErr, ok := err.(*DeviceError)
	if !ok {
		return "An unexpected error occurred. Please try again."
	}

	switch devErr.Type {
	case ErrTypeTimeout:
		return strings.Join([]string{
			"The device did not respond in time.",
			"Troubleshooting:",
			"  • Check that your device is powered on",
			"  • Verify you're connected to the device's WiFi network",
			"  • Try increasing the timeout duration",
			"  • Move closer to the device to improve signal strength",
		}, "\n")

	case ErrTypeConnectionRefused:
		return strings.Join([]string{
			"The device refused the connection.",
			"Troubleshooting:",
			"  • Ensure the device is in pairing mode (not connected to WiFi)",
			"  • Check that you're connected to the device's WiFi hotspot",
			"  • The device's HTTP server may not be running - try rebooting",
			"  • Verify the port number (default is 80)",
		}, "\n")

	case ErrTypeDNS:
		return strings.Join([]string{
			"Could not resolve the device hostname.",
			"Troubleshooting:",
			"  • Use the IP address instead of hostname",
			"  • Check your network DNS settings",
			"  • Verify you're on the same network as the device",
		}, "\n")

	case ErrTypeAuth:
		return strings.Join([]string{
			"Authentication failed.",
			"Troubleshooting:",
			"  • The default credentials are SmarTap:yeswecan",
			"  • Check if you've changed the device password",
			"  • Try resetting the device to factory defaults",
		}, "\n")

	case ErrTypeNetwork:
		hint := []string{"Network communication failed."}

		switch devErr.NetworkSubtype {
		case NetworkErrorHostUnreachable:
			hint = append(hint, "The device is not reachable on the network.",
				"Troubleshooting:",
				"  • Verify the device IP address is correct",
				"  • Check that you're on the same network as the device",
				"  • Ensure the device is powered on and connected",
				"  • Try pinging the device: ping "+devErr.DeviceIP)

		case NetworkErrorNetworkUnreachable:
			hint = append(hint, "Your computer cannot reach the device's network.",
				"Troubleshooting:",
				"  • Connect to the device's WiFi hotspot",
				"  • Check your network adapter settings",
				"  • Verify WiFi is enabled on your computer")

		default:
			hint = append(hint, "Troubleshooting:",
				"  • Check your network connection",
				"  • Verify the device is powered on",
				"  • Ensure you're connected to the correct network",
				"  • Try rebooting your computer's network adapter")
		}

		return strings.Join(hint, "\n")

	case ErrTypeHTTP:
		if devErr.StatusCode >= 500 {
			return strings.Join([]string{
				fmt.Sprintf("The device returned an error (HTTP %d).", devErr.StatusCode),
				"This is a device firmware issue.",
				"Troubleshooting:",
				"  • Try rebooting the device",
				"  • Check if a firmware update is available",
				"  • The device may need to be reset to factory defaults",
			}, "\n")
		}
		return fmt.Sprintf("The device returned HTTP error %d. Check the request parameters.", devErr.StatusCode)

	case ErrTypeParse:
		return strings.Join([]string{
			"Failed to parse the device's response.",
			"This may indicate a firmware issue or incompatibility.",
			"Troubleshooting:",
			"  • Check your firmware version (0x355 is tested)",
			"  • Try rebooting the device",
			"  • Contact support with your firmware version",
		}, "\n")

	case ErrTypeValidation:
		return "The configuration values are invalid. Check the error message for details."

	default:
		return "An error occurred. Please check the error message for details."
	}
}

// GetShortErrorMessage returns a concise, user-friendly error message
func GetShortErrorMessage(err error) string {
	devErr, ok := err.(*DeviceError)
	if !ok {
		return err.Error()
	}

	switch devErr.Type {
	case ErrTypeTimeout:
		return "Device not responding (timeout)"
	case ErrTypeConnectionRefused:
		return "Device refused connection - is it in pairing mode?"
	case ErrTypeDNS:
		return "Cannot resolve device hostname"
	case ErrTypeAuth:
		return "Authentication failed - check credentials"
	case ErrTypeNetwork:
		switch devErr.NetworkSubtype {
		case NetworkErrorHostUnreachable:
			return "Device unreachable - check network connection"
		case NetworkErrorNetworkUnreachable:
			return "Network unreachable - check WiFi connection"
		default:
			return "Network error - check connection"
		}
	case ErrTypeHTTP:
		return fmt.Sprintf("Device error (HTTP %d)", devErr.StatusCode)
	case ErrTypeParse:
		return "Failed to parse device response"
	case ErrTypeValidation:
		return devErr.Message
	default:
		return devErr.Message
	}
}
