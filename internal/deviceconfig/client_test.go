package deviceconfig

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Mock server response - valid device config
const mockDeviceResponse = `{"ssidList":["NETGEAR89"],"lowPowerMode":false,"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true,"swVer":"0x355","wnpVer":"2.:.0.000","mac":"C4:BE:84:74:86:37"}`

// Mock server response with trailing garbage (real device behavior)
const mockMalformedResponse = `{"ssidList":["NETGEAR89"],"lowPowerMode":false,"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true,"swVer":"0x355","wnpVer":"2.:.0.000","mac":"C4:BE:84:74:86:37"}"oldAppVer":"pkey:0000,315260240</div>"`

func TestNewClient(t *testing.T) {
	client := NewClient("192.168.4.16", 80)

	if client.BaseURL != "http://192.168.4.16:80" {
		t.Errorf("BaseURL = %s, want http://192.168.4.16:80", client.BaseURL)
	}

	if client.Username != DefaultUsername {
		t.Errorf("Username = %s, want %s", client.Username, DefaultUsername)
	}

	if client.Password != DefaultPassword {
		t.Errorf("Password = %s, want %s", client.Password, DefaultPassword)
	}

	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestNewClientWithURL(t *testing.T) {
	client := NewClientWithURL("http://192.168.4.16:8080")

	if client.BaseURL != "http://192.168.4.16:8080" {
		t.Errorf("BaseURL = %s, want http://192.168.4.16:8080", client.BaseURL)
	}
}

func TestSetTimeout(t *testing.T) {
	client := NewClient("192.168.4.16", 80)
	client.SetTimeout(5 * time.Second)

	if client.HTTPClient.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", client.HTTPClient.Timeout)
	}
}

func TestSetAuth(t *testing.T) {
	client := NewClient("192.168.4.16", 80)
	client.SetAuth("testuser", "testpass")

	if client.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", client.Username)
	}

	if client.Password != "testpass" {
		t.Errorf("Password = %s, want testpass", client.Password)
	}
}

func TestSetRetry(t *testing.T) {
	client := NewClient("192.168.4.16", 80)
	client.SetRetry(5, 2*time.Second)

	if client.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", client.MaxRetries)
	}

	if client.RetryDelay != 2*time.Second {
		t.Errorf("RetryDelay = %v, want 2s", client.RetryDelay)
	}
}

func TestPing_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	err := client.Ping()

	if err != nil {
		t.Errorf("Ping() error = %v, want nil", err)
	}
}

func TestPing_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	err := client.Ping()

	if err == nil {
		t.Error("Ping() should return error for auth failure")
	}

	if !IsAuthError(err) {
		t.Errorf("Ping() error should be auth error, got %T", err)
	}
}

func TestPing_NetworkFailure(t *testing.T) {
	// Use invalid URL to trigger network error
	client := NewClient("192.0.2.1", 80) // TEST-NET-1 (guaranteed unreachable)
	client.SetTimeout(100 * time.Millisecond)

	err := client.Ping()

	if err == nil {
		t.Error("Ping() should return error for network failure")
	}

	if !IsNetworkError(err) {
		t.Errorf("Ping() error should be network error, got %T: %v", err, err)
	}
}

func TestGetConfiguration_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != "GET" {
			t.Errorf("Request method = %s, want GET", r.Method)
		}

		// Check auth
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	config, err := client.GetConfiguration()

	if err != nil {
		t.Fatalf("GetConfiguration() error = %v, want nil", err)
	}

	if config.Serial != "315260240" {
		t.Errorf("Serial = %s, want 315260240", config.Serial)
	}

	if config.MAC != "C4:BE:84:74:86:37" {
		t.Errorf("MAC = %s, want C4:BE:84:74:86:37", config.MAC)
	}

	if config.Outlet1 != 1 {
		t.Errorf("Outlet1 = %d, want 1", config.Outlet1)
	}

	if !config.K3Outlet {
		t.Error("K3Outlet should be true")
	}
}

func TestGetConfiguration_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockMalformedResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	config, err := client.GetConfiguration()

	if err != nil {
		t.Fatalf("GetConfiguration() should handle malformed JSON, error = %v", err)
	}

	if config.Serial != "315260240" {
		t.Errorf("Serial = %s, want 315260240", config.Serial)
	}
}

func TestGetConfiguration_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	_, err := client.GetConfiguration()

	if err == nil {
		t.Error("GetConfiguration() should return error for auth failure")
	}

	if !IsAuthError(err) {
		t.Errorf("GetConfiguration() error should be auth error, got %T", err)
	}
}

func TestGetConfiguration_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid JSON at all"))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	_, err := client.GetConfiguration()

	if err == nil {
		t.Error("GetConfiguration() should return error for invalid JSON")
	}

	if !IsParseError(err) {
		t.Errorf("GetConfiguration() error should be parse error, got %T: %v", err, err)
	}
}

func TestUpdateConfiguration_Success(t *testing.T) {
	receivedFormData := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != "POST" {
			t.Errorf("Request method = %s, want POST", r.Method)
		}

		// Check auth
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check content type
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %s, want application/x-www-form-urlencoded", contentType)
		}

		// Read form data
		err := r.ParseForm()
		if err != nil {
			t.Errorf("Failed to parse form: %v", err)
		}
		receivedFormData = r.Form.Encode()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  3,
			SecondPress: 5,
			ThirdPress:  7,
			K3Mode:      true,
		},
	}

	err := client.UpdateConfiguration(update)

	if err != nil {
		t.Fatalf("UpdateConfiguration() error = %v, want nil", err)
	}

	// Verify form data was sent correctly
	if !strings.Contains(receivedFormData, "__SL_P_OU1=3") {
		t.Errorf("Form data missing __SL_P_OU1=3, got: %s", receivedFormData)
	}
	if !strings.Contains(receivedFormData, "__SL_P_OU2=5") {
		t.Errorf("Form data missing __SL_P_OU2=5, got: %s", receivedFormData)
	}
	if !strings.Contains(receivedFormData, "__SL_P_K3O=checked") {
		t.Errorf("Form data missing __SL_P_K3O=checked, got: %s", receivedFormData)
	}
}

func TestUpdateConfiguration_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress: 1,
		},
	}

	err := client.UpdateConfiguration(update)

	if err == nil {
		t.Error("UpdateConfiguration() should return error for auth failure")
	}

	if !IsAuthError(err) {
		t.Errorf("UpdateConfiguration() error should be auth error, got %T", err)
	}
}

func TestUpdateDiverter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	config := &DiverterConfig{
		FirstPress:  1,
		SecondPress: 2,
		ThirdPress:  4,
		K3Mode:      true,
	}

	err := client.UpdateDiverter(config)

	if err != nil {
		t.Errorf("UpdateDiverter() error = %v, want nil", err)
	}
}

func TestUpdateWiFi(t *testing.T) {
	receivedFormData := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err := r.ParseForm()
		if err != nil {
			t.Errorf("Failed to parse form: %v", err)
		}
		receivedFormData = r.Form.Encode()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	config := &WiFiConfig{
		SSID:         "TestNetwork",
		Password:     "TestPassword123",
		SecurityType: "WPA2",
	}

	err := client.UpdateWiFi(config)

	if err != nil {
		t.Errorf("UpdateWiFi() error = %v, want nil", err)
	}

	// Verify WiFi parameters were sent
	if !strings.Contains(receivedFormData, "__SL_P_USD=TestNetwork") {
		t.Errorf("Form data missing SSID, got: %s", receivedFormData)
	}
	if !strings.Contains(receivedFormData, "__SL_P_PSD=TestPassword123") {
		t.Errorf("Form data missing password, got: %s", receivedFormData)
	}
}

func TestUpdateServer(t *testing.T) {
	receivedFormData := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		err := r.ParseForm()
		if err != nil {
			t.Errorf("Failed to parse form: %v", err)
		}
		receivedFormData = r.Form.Encode()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	config := &ServerConfig{
		DNS:  "test.server.com",
		Port: 443,
	}

	err := client.UpdateServer(config)

	if err != nil {
		t.Errorf("UpdateServer() error = %v, want nil", err)
	}

	// Verify server parameters were sent
	if !strings.Contains(receivedFormData, "__SL_P_DNS=test.server.com") {
		t.Errorf("Form data missing DNS, got: %s", receivedFormData)
	}
	if !strings.Contains(receivedFormData, "__SL_P_PRT=443") {
		t.Errorf("Form data missing port, got: %s", receivedFormData)
	}
}

func TestVerifyConfiguration_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return config with expected values
		response := `{"ssidList":[],"lowPowerMode":false,"serial":"315260240","dns":"test.server.com","port":443,"outlet1":3,"outlet2":5,"outlet3":7,"k3Outlet":true,"swVer":"0x355","wnpVer":"2.:.0.000","mac":"C4:BE:84:74:86:37"}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	expected := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  3,
			SecondPress: 5,
			ThirdPress:  7,
			K3Mode:      true,
		},
		Server: &ServerConfig{
			DNS:  "test.server.com",
			Port: 443,
		},
	}

	err := client.VerifyConfiguration(expected)

	if err != nil {
		t.Errorf("VerifyConfiguration() error = %v, want nil", err)
	}
}

func TestVerifyConfiguration_Mismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return config with different values
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	expected := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  3,
			SecondPress: 5,
			ThirdPress:  7,
			K3Mode:      false,
		},
	}

	err := client.VerifyConfiguration(expected)

	if err == nil {
		t.Error("VerifyConfiguration() should return error for mismatch")
	}

	if !IsValidationError(err) {
		t.Errorf("VerifyConfiguration() error should be validation error, got %T: %v", err, err)
	}
}

func TestGetFormData(t *testing.T) {
	client := NewClient("192.168.4.16", 80)

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  3,
			SecondPress: 5,
			ThirdPress:  7,
			K3Mode:      true,
		},
	}

	formData := client.GetFormData(update)

	if formData.Get("__SL_P_OU1") != "3" {
		t.Errorf("FormData __SL_P_OU1 = %s, want 3", formData.Get("__SL_P_OU1"))
	}

	if formData.Get("__SL_P_K3O") != "checked" {
		t.Errorf("FormData __SL_P_K3O = %s, want checked", formData.Get("__SL_P_K3O"))
	}
}

// Benchmark tests
func BenchmarkGetConfiguration(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.GetConfiguration()
	}
}

func BenchmarkUpdateConfiguration(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress: 1,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.UpdateConfiguration(update)
	}
}

// Cache tests

func TestCaching_Enabled(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	client.SetCacheDuration(5 * time.Second)

	// First call should hit server
	config1, err := client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}

	// Second call should use cache
	config2, err := client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 request (cached), got %d", requestCount)
	}

	// Configs should be equal
	if config1.Serial != config2.Serial {
		t.Error("Cached config should match original")
	}
}

func TestCaching_Disabled(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	client.SetCacheDuration(0) // Disable caching

	// Both calls should hit server
	_, err := client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	_, err = client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 requests (no cache), got %d", requestCount)
	}
}

func TestCaching_Expiration(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	client.SetCacheDuration(100 * time.Millisecond)

	// First call
	_, err := client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	if requestCount != 1 {
		t.Errorf("Expected 1 request, got %d", requestCount)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Second call should hit server again
	_, err = client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 requests (cache expired), got %d", requestCount)
	}
}

func TestInvalidateCache(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	client.SetCacheDuration(5 * time.Second)

	// First call
	_, err := client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	// Invalidate cache
	client.InvalidateCache()

	// Next call should hit server again
	_, err = client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 requests (cache invalidated), got %d", requestCount)
	}
}

func TestGetCachedConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	client.SetCacheDuration(5 * time.Second)

	// No cache yet
	cached := client.GetCachedConfiguration()
	if cached != nil {
		t.Error("Expected nil for non-existent cache")
	}

	// Fetch config
	_, err := client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	// Should have cached value now
	cached = client.GetCachedConfiguration()
	if cached == nil {
		t.Error("Expected cached config")
	}

	if cached.Serial != "315260240" {
		t.Errorf("Cached serial = %s, want 315260240", cached.Serial)
	}
}

func TestRefreshConfiguration(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	client.SetCacheDuration(5 * time.Second)

	// First call - should cache
	_, err := client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	// Refresh should bypass cache
	_, err = client.RefreshConfiguration()
	if err != nil {
		t.Fatalf("RefreshConfiguration() error = %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 requests (refresh bypassed cache), got %d", requestCount)
	}

	// Next Get should use new cache
	_, err = client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 requests (cache from refresh), got %d", requestCount)
	}
}

func TestCacheInvalidatedAfterUpdate(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			requestCount++
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != DefaultUsername || password != DefaultPassword {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockDeviceResponse))
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)
	client.SetCacheDuration(5 * time.Second)

	// Get config (should cache)
	_, err := client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	// Update config (should invalidate cache)
	update := &ConfigUpdate{
		Diverter: &DiverterConfig{FirstPress: 3},
	}
	err = client.UpdateConfiguration(update)
	if err != nil {
		t.Fatalf("UpdateConfiguration() error = %v", err)
	}

	// Next Get should hit server (cache was invalidated)
	_, err = client.GetConfiguration()
	if err != nil {
		t.Fatalf("GetConfiguration() error = %v", err)
	}

	if requestCount != 2 {
		t.Errorf("Expected 2 GET requests (cache invalidated after update), got %d", requestCount)
	}
}

// Test SetDiverterButtons convenience method
func TestSetDiverterButtons(t *testing.T) {
	var capturedForm url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture form data
		if r.Method == http.MethodPost {
			r.ParseForm()
			capturedForm = r.Form
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	// Test setting diverter buttons
	err := client.SetDiverterButtons(1, 2, 4, true)
	if err != nil {
		t.Fatalf("SetDiverterButtons failed: %v", err)
	}

	// Verify form data
	if capturedForm.Get("__SL_P_OU1") != "1" {
		t.Errorf("Expected outlet1=1, got %s", capturedForm.Get("__SL_P_OU1"))
	}
	if capturedForm.Get("__SL_P_OU2") != "2" {
		t.Errorf("Expected outlet2=2, got %s", capturedForm.Get("__SL_P_OU2"))
	}
	if capturedForm.Get("__SL_P_OU3") != "4" {
		t.Errorf("Expected outlet3=4, got %s", capturedForm.Get("__SL_P_OU3"))
	}
	if capturedForm.Get("__SL_P_K3O") != "checked" {
		t.Errorf("Expected k3Mode=checked, got %s", capturedForm.Get("__SL_P_K3O"))
	}
}

// Test SetThirdKnobMode convenience method
func TestSetThirdKnobMode(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		expected string
	}{
		{"Enable K3 mode", true, "checked"},
		{"Disable K3 mode", false, "no"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedForm url.Values

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					r.ParseForm()
					capturedForm = r.Form
					w.WriteHeader(http.StatusNoContent)
				}
			}))
			defer server.Close()

			client := NewClientWithURL(server.URL)

			err := client.SetThirdKnobMode(tt.enabled)
			if err != nil {
				t.Fatalf("SetThirdKnobMode failed: %v", err)
			}

			if capturedForm.Get("__SL_P_K3O") != tt.expected {
				t.Errorf("Expected k3Mode=%s, got %s", tt.expected, capturedForm.Get("__SL_P_K3O"))
			}
		})
	}
}

// Test SetSequentialOutlets convenience method
func TestSetSequentialOutlets(t *testing.T) {
	var capturedForm url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			r.ParseForm()
			capturedForm = r.Form
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	// Test sequential outlets with K3 enabled
	err := client.SetSequentialOutlets(true)
	if err != nil {
		t.Fatalf("SetSequentialOutlets failed: %v", err)
	}

	// Verify sequential pattern: 1, 2, 4
	if capturedForm.Get("__SL_P_OU1") != "1" {
		t.Errorf("Expected outlet1=1 (Outlet 1 only), got %s", capturedForm.Get("__SL_P_OU1"))
	}
	if capturedForm.Get("__SL_P_OU2") != "2" {
		t.Errorf("Expected outlet2=2 (Outlet 2 only), got %s", capturedForm.Get("__SL_P_OU2"))
	}
	if capturedForm.Get("__SL_P_OU3") != "4" {
		t.Errorf("Expected outlet3=4 (Outlet 3 only), got %s", capturedForm.Get("__SL_P_OU3"))
	}
	if capturedForm.Get("__SL_P_K3O") != "checked" {
		t.Errorf("Expected k3Mode=checked, got %s", capturedForm.Get("__SL_P_K3O"))
	}
}

// Test SetAllOutletsOn convenience method
func TestSetAllOutletsOn(t *testing.T) {
	var capturedForm url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			r.ParseForm()
			capturedForm = r.Form
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := NewClientWithURL(server.URL)

	err := client.SetAllOutletsOn()
	if err != nil {
		t.Fatalf("SetAllOutletsOn failed: %v", err)
	}

	// Verify all outlets on pattern: 7, 7, 7 (all three outlets for each button)
	if capturedForm.Get("__SL_P_OU1") != "7" {
		t.Errorf("Expected outlet1=7 (all outlets), got %s", capturedForm.Get("__SL_P_OU1"))
	}
	if capturedForm.Get("__SL_P_OU2") != "7" {
		t.Errorf("Expected outlet2=7 (all outlets), got %s", capturedForm.Get("__SL_P_OU2"))
	}
	if capturedForm.Get("__SL_P_OU3") != "7" {
		t.Errorf("Expected outlet3=7 (all outlets), got %s", capturedForm.Get("__SL_P_OU3"))
	}
	if capturedForm.Get("__SL_P_K3O") != "no" {
		t.Errorf("Expected k3Mode=no (disabled), got %s", capturedForm.Get("__SL_P_K3O"))
	}
}
