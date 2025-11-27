package deviceconfig

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"syscall"
	"testing"
)

func TestClassifyNetworkError_Timeout(t *testing.T) {
	// Create a timeout error
	err := &url.Error{
		Op:  "Get",
		URL: "http://192.168.4.16",
		Err: &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &timeoutError{},
		},
	}

	devErr := ClassifyNetworkError(err, "192.168.4.16")

	if devErr == nil {
		t.Fatal("Expected DeviceError, got nil")
	}

	if devErr.Type != ErrTypeTimeout {
		t.Errorf("Expected error type %v, got %v", ErrTypeTimeout, devErr.Type)
	}

	if devErr.NetworkSubtype != NetworkErrorTimeout {
		t.Errorf("Expected network subtype %v, got %v", NetworkErrorTimeout, devErr.NetworkSubtype)
	}

	if !devErr.Retryable {
		t.Error("Expected timeout error to be retryable")
	}
}

func TestClassifyNetworkError_ConnectionRefused(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "http://192.168.4.16",
		Err: &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: syscall.ECONNREFUSED,
		},
	}

	devErr := ClassifyNetworkError(err, "192.168.4.16")

	if devErr == nil {
		t.Fatal("Expected DeviceError, got nil")
	}

	if devErr.Type != ErrTypeConnectionRefused {
		t.Errorf("Expected error type %v, got %v", ErrTypeConnectionRefused, devErr.Type)
	}

	if devErr.NetworkSubtype != NetworkErrorConnectionRefused {
		t.Errorf("Expected network subtype %v, got %v", NetworkErrorConnectionRefused, devErr.NetworkSubtype)
	}

	if !devErr.Retryable {
		t.Error("Expected connection refused error to be retryable")
	}
}

func TestClassifyNetworkError_DNS(t *testing.T) {
	err := &net.DNSError{
		Err:        "no such host",
		Name:       "invalid.local",
		IsNotFound: true,
	}

	devErr := ClassifyNetworkError(err, "invalid.local")

	if devErr == nil {
		t.Fatal("Expected DeviceError, got nil")
	}

	if devErr.Type != ErrTypeDNS {
		t.Errorf("Expected error type %v, got %v", ErrTypeDNS, devErr.Type)
	}

	if devErr.NetworkSubtype != NetworkErrorDNS {
		t.Errorf("Expected network subtype %v, got %v", NetworkErrorDNS, devErr.NetworkSubtype)
	}

	if devErr.Retryable {
		t.Error("Expected DNS error to be non-retryable")
	}
}

func TestClassifyNetworkError_HostUnreachable(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "http://192.168.4.16",
		Err: &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: syscall.EHOSTUNREACH,
		},
	}

	devErr := ClassifyNetworkError(err, "192.168.4.16")

	if devErr == nil {
		t.Fatal("Expected DeviceError, got nil")
	}

	if devErr.Type != ErrTypeNetwork {
		t.Errorf("Expected error type %v, got %v", ErrTypeNetwork, devErr.Type)
	}

	if devErr.NetworkSubtype != NetworkErrorHostUnreachable {
		t.Errorf("Expected network subtype %v, got %v", NetworkErrorHostUnreachable, devErr.NetworkSubtype)
	}

	if !devErr.Retryable {
		t.Error("Expected host unreachable error to be retryable")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name: "Network error is retryable",
			err: &DeviceError{
				Type:      ErrTypeNetwork,
				Retryable: true,
			},
			retryable: true,
		},
		{
			name: "Auth error is not retryable",
			err: &DeviceError{
				Type:      ErrTypeAuth,
				Retryable: false,
			},
			retryable: false,
		},
		{
			name: "Validation error is not retryable",
			err: &DeviceError{
				Type:      ErrTypeValidation,
				Retryable: false,
			},
			retryable: false,
		},
		{
			name: "HTTP 500 error is retryable",
			err: &DeviceError{
				Type:       ErrTypeHTTP,
				StatusCode: 500,
				Retryable:  true,
			},
			retryable: true,
		},
		{
			name: "HTTP 404 error is not retryable",
			err: &DeviceError{
				Type:       ErrTypeHTTP,
				StatusCode: 404,
				Retryable:  false,
			},
			retryable: false,
		},
		{
			name:      "Unknown error is not retryable",
			err:       errors.New("unknown error"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestGetShortErrorMessage(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedText string
	}{
		{
			name: "Timeout error",
			err: &DeviceError{
				Type: ErrTypeTimeout,
			},
			expectedText: "Device not responding (timeout)",
		},
		{
			name: "Connection refused",
			err: &DeviceError{
				Type: ErrTypeConnectionRefused,
			},
			expectedText: "Device refused connection - is it in pairing mode?",
		},
		{
			name: "DNS error",
			err: &DeviceError{
				Type: ErrTypeDNS,
			},
			expectedText: "Cannot resolve device hostname",
		},
		{
			name: "Auth error",
			err: &DeviceError{
				Type: ErrTypeAuth,
			},
			expectedText: "Authentication failed - check credentials",
		},
		{
			name: "Host unreachable",
			err: &DeviceError{
				Type:           ErrTypeNetwork,
				NetworkSubtype: NetworkErrorHostUnreachable,
			},
			expectedText: "Device unreachable - check network connection",
		},
		{
			name: "HTTP 500",
			err: &DeviceError{
				Type:       ErrTypeHTTP,
				StatusCode: 500,
			},
			expectedText: "Device error (HTTP 500)",
		},
		{
			name: "Validation error",
			err: &DeviceError{
				Type:    ErrTypeValidation,
				Message: "Invalid bitmask value",
			},
			expectedText: "Invalid bitmask value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetShortErrorMessage(tt.err)
			if got != tt.expectedText {
				t.Errorf("GetShortErrorMessage() = %q, want %q", got, tt.expectedText)
			}
		})
	}
}

func TestGetTroubleshootingHint(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedTexts []string // Texts that should appear in the hint
	}{
		{
			name: "Timeout error",
			err: &DeviceError{
				Type: ErrTypeTimeout,
			},
			expectedTexts: []string{
				"did not respond in time",
				"Troubleshooting:",
				"powered on",
				"WiFi network",
			},
		},
		{
			name: "Connection refused",
			err: &DeviceError{
				Type: ErrTypeConnectionRefused,
			},
			expectedTexts: []string{
				"refused the connection",
				"pairing mode",
				"WiFi hotspot",
			},
		},
		{
			name: "DNS error",
			err: &DeviceError{
				Type: ErrTypeDNS,
			},
			expectedTexts: []string{
				"resolve the device hostname",
				"IP address instead",
				"DNS settings",
			},
		},
		{
			name: "Auth error",
			err: &DeviceError{
				Type: ErrTypeAuth,
			},
			expectedTexts: []string{
				"Authentication failed",
				"SmarTap:yeswecan",
				"factory defaults",
			},
		},
		{
			name: "Host unreachable",
			err: &DeviceError{
				Type:           ErrTypeNetwork,
				NetworkSubtype: NetworkErrorHostUnreachable,
				DeviceIP:       "192.168.4.16",
			},
			expectedTexts: []string{
				"not reachable",
				"ping 192.168.4.16",
				"same network",
			},
		},
		{
			name: "HTTP 500 error",
			err: &DeviceError{
				Type:       ErrTypeHTTP,
				StatusCode: 500,
			},
			expectedTexts: []string{
				"HTTP 500",
				"firmware issue",
				"rebooting the device",
			},
		},
		{
			name: "Parse error",
			err: &DeviceError{
				Type: ErrTypeParse,
			},
			expectedTexts: []string{
				"Failed to parse",
				"firmware version",
				"0x355",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hint := GetTroubleshootingHint(tt.err)

			for _, expectedText := range tt.expectedTexts {
				if !strings.Contains(hint, expectedText) {
					t.Errorf("GetTroubleshootingHint() missing expected text %q\nGot: %s", expectedText, hint)
				}
			}
		})
	}
}

func TestNewHTTPError_RetryableForServerErrors(t *testing.T) {
	// HTTP 5xx errors should be retryable
	err500 := NewHTTPError(500, "Internal Server Error")
	if !err500.Retryable {
		t.Error("Expected HTTP 500 error to be retryable")
	}

	// HTTP 4xx errors should not be retryable
	err404 := NewHTTPError(404, "Not Found")
	if err404.Retryable {
		t.Error("Expected HTTP 404 error to be non-retryable")
	}
}

func TestErrorTypeString(t *testing.T) {
	tests := []struct {
		errorType ErrorType
		expected  string
	}{
		{ErrTypeNetwork, "Network Error"},
		{ErrTypeAuth, "Authentication Error"},
		{ErrTypeHTTP, "HTTP Error"},
		{ErrTypeParse, "Parse Error"},
		{ErrTypeValidation, "Validation Error"},
		{ErrTypeTimeout, "Timeout"},
		{ErrTypeConnectionRefused, "Connection Refused"},
		{ErrTypeDNS, "DNS Error"},
		{ErrTypeUnknown, "Unknown Error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.errorType.String(); got != tt.expected {
				t.Errorf("ErrorType.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// timeoutError is a mock error that implements timeout behavior
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
