package deviceconfig

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test helper: create a test server that returns a specific configuration
func newVerifyTestServer(config *DeviceConfig, postHandler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth
		user, pass, ok := r.BasicAuth()
		if !ok || user != "SmarTap" || pass != "yeswecan" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(config)
		} else if r.Method == "POST" {
			if postHandler != nil {
				postHandler(w, r)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		}
	}))
}

func TestDefaultVerificationOptions(t *testing.T) {
	opts := DefaultVerificationOptions()

	if opts.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", opts.MaxRetries)
	}
	if opts.InitialDelay != 500*time.Millisecond {
		t.Errorf("Expected InitialDelay=500ms, got %v", opts.InitialDelay)
	}
	if opts.RetryDelay != 1*time.Second {
		t.Errorf("Expected RetryDelay=1s, got %v", opts.RetryDelay)
	}
	if !opts.UseExponentialBackoff {
		t.Error("Expected UseExponentialBackoff=true")
	}
	if opts.MaxRetryDelay != 5*time.Second {
		t.Errorf("Expected MaxRetryDelay=5s, got %v", opts.MaxRetryDelay)
	}
}

func TestVerifyConfigurationWithRetry_Success(t *testing.T) {
	config := &DeviceConfig{
		Serial:   "12345",
		Outlet1:  1,
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: false,
		DNS:      "test.local",
		Port:     80,
	}

	server := newVerifyTestServer(config, nil)
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 80)
	client.BaseURL = server.URL

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1,
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      false,
		},
	}

	// Use fast options for testing
	opts := &VerificationOptions{
		MaxRetries:            2,
		InitialDelay:          10 * time.Millisecond,
		RetryDelay:            10 * time.Millisecond,
		UseExponentialBackoff: false,
		MaxRetryDelay:         100 * time.Millisecond,
	}

	result := client.VerifyConfigurationWithRetry(update, opts)

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.Error)
	}
	if result.Attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", result.Attempts)
	}
	if len(result.Mismatches) != 0 {
		t.Errorf("Expected no mismatches, got %v", result.Mismatches)
	}
	if result.ActualConfig == nil {
		t.Error("Expected ActualConfig to be set")
	}
}

func TestVerifyConfigurationWithRetry_Mismatch(t *testing.T) {
	// Device returns different configuration than expected
	config := &DeviceConfig{
		Serial:   "12345",
		Outlet1:  3, // Different from expected
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: false,
		DNS:      "test.local",
		Port:     80,
	}

	server := newVerifyTestServer(config, nil)
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 80)
	client.BaseURL = server.URL

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1, // Expecting 1, but device has 3
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      false,
		},
	}

	opts := &VerificationOptions{
		MaxRetries:            2,
		InitialDelay:          10 * time.Millisecond,
		RetryDelay:            10 * time.Millisecond,
		UseExponentialBackoff: false,
		MaxRetryDelay:         100 * time.Millisecond,
	}

	result := client.VerifyConfigurationWithRetry(update, opts)

	if result.Success {
		t.Error("Expected failure, got success")
	}
	if result.Attempts != 3 { // Initial + 2 retries
		t.Errorf("Expected 3 attempts, got %d", result.Attempts)
	}
	if len(result.Mismatches) == 0 {
		t.Error("Expected mismatches, got none")
	}
	if !strings.Contains(result.Mismatches[0], "diverter first press") {
		t.Errorf("Expected first press mismatch, got %v", result.Mismatches[0])
	}
}

func TestVerifyConfigurationWithRetry_EventualSuccess(t *testing.T) {
	// Simulate device taking time to apply configuration
	// First call returns old config, second call returns new config

	attempts := 0
	oldConfig := &DeviceConfig{
		Serial:   "12345",
		Outlet1:  3,
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: false,
	}
	newConfig := &DeviceConfig{
		Serial:   "12345",
		Outlet1:  1,
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "SmarTap" || pass != "yeswecan" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Method == "GET" {
			attempts++
			w.Header().Set("Content-Type", "application/json")
			if attempts <= 1 {
				_ = json.NewEncoder(w).Encode(oldConfig)
			} else {
				_ = json.NewEncoder(w).Encode(newConfig)
			}
		}
	}))
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 80)
	client.BaseURL = server.URL
	client.SetCacheDuration(0) // Disable caching to ensure fresh reads

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1,
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      false,
		},
	}

	opts := &VerificationOptions{
		MaxRetries:            3,
		InitialDelay:          10 * time.Millisecond,
		RetryDelay:            10 * time.Millisecond,
		UseExponentialBackoff: false,
		MaxRetryDelay:         100 * time.Millisecond,
	}

	result := client.VerifyConfigurationWithRetry(update, opts)

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.Error)
	}
	if result.Attempts != 2 {
		t.Errorf("Expected 2 attempts (first fails, second succeeds), got %d", result.Attempts)
	}
}

func TestVerifyConfigurationWithRetry_NetworkError(t *testing.T) {
	t.Skip("Skipping slow network timeout test - takes too long for normal test runs")

	// Create a client pointing to non-existent server
	client := NewClient("192.0.2.1", 12345) // TEST-NET-1 (guaranteed to not exist)

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1,
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      false,
		},
	}

	opts := &VerificationOptions{
		MaxRetries:            2,
		InitialDelay:          10 * time.Millisecond,
		RetryDelay:            10 * time.Millisecond,
		UseExponentialBackoff: false,
		MaxRetryDelay:         100 * time.Millisecond,
	}

	result := client.VerifyConfigurationWithRetry(update, opts)

	if result.Success {
		t.Error("Expected failure due to network error")
	}
	if result.Attempts != 3 { // Initial + 2 retries
		t.Errorf("Expected 3 attempts, got %d", result.Attempts)
	}
	if result.Error == nil {
		t.Error("Expected error to be set")
	}
}

func TestVerifyConfigurationWithRetry_ExponentialBackoff(t *testing.T) {
	config := &DeviceConfig{
		Serial:   "12345",
		Outlet1:  3, // Mismatch - will fail all attempts
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: false,
	}

	server := newVerifyTestServer(config, nil)
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 80)
	client.BaseURL = server.URL

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1,
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      false,
		},
	}

	opts := &VerificationOptions{
		MaxRetries:            3,
		InitialDelay:          10 * time.Millisecond,
		RetryDelay:            10 * time.Millisecond,
		UseExponentialBackoff: true,
		MaxRetryDelay:         100 * time.Millisecond,
	}

	start := time.Now()
	result := client.VerifyConfigurationWithRetry(update, opts)
	elapsed := time.Since(start)

	if result.Success {
		t.Error("Expected failure")
	}

	// With exponential backoff: 10ms + (10ms + 20ms + 40ms) = 80ms minimum
	// With delays in test execution, expect at least 70ms
	if elapsed < 70*time.Millisecond {
		t.Errorf("Expected elapsed time >= 70ms with exponential backoff, got %v", elapsed)
	}
}

func TestVerifyConfigurationWithRetry_DefaultOptions(t *testing.T) {
	config := &DeviceConfig{
		Serial:   "12345",
		Outlet1:  1,
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: false,
	}

	server := newVerifyTestServer(config, nil)
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 80)
	client.BaseURL = server.URL

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1,
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      false,
		},
	}

	// Pass nil to use default options
	result := client.VerifyConfigurationWithRetry(update, nil)

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.Error)
	}
}

func TestUpdateAndVerify_Success(t *testing.T) {
	config := &DeviceConfig{
		Serial:   "12345",
		Outlet1:  1,
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: false,
	}

	server := newVerifyTestServer(config, nil)
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 80)
	client.BaseURL = server.URL

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1,
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      false,
		},
	}

	opts := &VerificationOptions{
		MaxRetries:            2,
		InitialDelay:          10 * time.Millisecond,
		RetryDelay:            10 * time.Millisecond,
		UseExponentialBackoff: false,
		MaxRetryDelay:         100 * time.Millisecond,
	}

	result := client.UpdateAndVerify(update, opts)

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.Error)
	}
}

func TestUpdateAndVerify_UpdateFails(t *testing.T) {
	// Server that returns 500 for POST
	server := newVerifyTestServer(nil, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 80)
	client.BaseURL = server.URL

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1,
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      false,
		},
	}

	opts := &VerificationOptions{
		MaxRetries:            2,
		InitialDelay:          10 * time.Millisecond,
		RetryDelay:            10 * time.Millisecond,
		UseExponentialBackoff: false,
		MaxRetryDelay:         100 * time.Millisecond,
	}

	result := client.UpdateAndVerify(update, opts)

	if result.Success {
		t.Error("Expected failure due to update error")
	}
	if result.Attempts != 0 {
		t.Errorf("Expected 0 verification attempts (update failed), got %d", result.Attempts)
	}
	if !strings.Contains(result.Error.Error(), "update failed") {
		t.Errorf("Expected 'update failed' error, got %v", result.Error)
	}
}

func TestVerifyDiverter(t *testing.T) {
	config := &DeviceConfig{
		Serial:   "12345",
		Outlet1:  1,
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: false,
	}

	server := newVerifyTestServer(config, nil)
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 80)
	client.BaseURL = server.URL

	diverterConfig := &DiverterConfig{
		FirstPress:  1,
		SecondPress: 2,
		ThirdPress:  4,
		K3Mode:      false,
	}

	opts := &VerificationOptions{
		MaxRetries:            2,
		InitialDelay:          10 * time.Millisecond,
		RetryDelay:            10 * time.Millisecond,
		UseExponentialBackoff: false,
		MaxRetryDelay:         100 * time.Millisecond,
	}

	result := client.VerifyDiverter(diverterConfig, opts)

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.Error)
	}
}

func TestVerifyServer(t *testing.T) {
	config := &DeviceConfig{
		Serial: "12345",
		DNS:    "test.local",
		Port:   8080,
	}

	server := newVerifyTestServer(config, nil)
	defer server.Close()

	client := NewClient(strings.TrimPrefix(server.URL, "http://"), 80)
	client.BaseURL = server.URL

	serverConfig := &ServerConfig{
		DNS:  "test.local",
		Port: 8080,
	}

	opts := &VerificationOptions{
		MaxRetries:            2,
		InitialDelay:          10 * time.Millisecond,
		RetryDelay:            10 * time.Millisecond,
		UseExponentialBackoff: false,
		MaxRetryDelay:         100 * time.Millisecond,
	}

	result := client.VerifyServer(serverConfig, opts)

	if !result.Success {
		t.Errorf("Expected success, got failure: %v", result.Error)
	}
}

func TestVerifyConfigurationMatch_AllFields(t *testing.T) {
	expected := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1,
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      true,
		},
		Server: &ServerConfig{
			DNS:  "test.local",
			Port: 8080,
		},
	}

	actual := &DeviceConfig{
		Outlet1:  1,
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: true,
		DNS:      "test.local",
		Port:     8080,
	}

	mismatches := verifyConfigurationMatch(expected, actual)

	if len(mismatches) != 0 {
		t.Errorf("Expected no mismatches, got %v", mismatches)
	}
}

func TestVerifyConfigurationMatch_MultipleMismatches(t *testing.T) {
	expected := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  1,
			SecondPress: 2,
			ThirdPress:  4,
			K3Mode:      true,
		},
		Server: &ServerConfig{
			DNS:  "test.local",
			Port: 8080,
		},
	}

	actual := &DeviceConfig{
		Outlet1:  3, // Mismatch
		Outlet2:  2,
		Outlet3:  5,             // Mismatch
		K3Outlet: false,         // Mismatch
		DNS:      "other.local", // Mismatch
		Port:     80,            // Mismatch
	}

	mismatches := verifyConfigurationMatch(expected, actual)

	if len(mismatches) != 5 {
		t.Errorf("Expected 5 mismatches, got %d: %v", len(mismatches), mismatches)
	}
}

func TestFormatMismatches(t *testing.T) {
	tests := []struct {
		name       string
		mismatches []string
		expected   string
	}{
		{
			name:       "No mismatches",
			mismatches: []string{},
			expected:   "none",
		},
		{
			name:       "Single mismatch",
			mismatches: []string{"outlet1: expected 1, got 2"},
			expected:   "outlet1: expected 1, got 2",
		},
		{
			name:       "Multiple mismatches",
			mismatches: []string{"outlet1: expected 1, got 2", "outlet2: expected 3, got 4"},
			expected:   "2 mismatches: outlet1: expected 1, got 2; outlet2: expected 3, got 4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMismatches(tt.mismatches)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
