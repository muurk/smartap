package deviceconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultUsername is the default HTTP Basic Auth username for Smartap devices
	DefaultUsername = "SmarTap"

	// DefaultPassword is the default HTTP Basic Auth password for Smartap devices
	DefaultPassword = "yeswecan"

	// DefaultTimeout is the default HTTP request timeout
	DefaultTimeout = 10 * time.Second

	// DefaultMaxRetries is the default number of retry attempts for failed requests
	DefaultMaxRetries = 3

	// DefaultRetryDelay is the default delay between retry attempts
	DefaultRetryDelay = 1 * time.Second

	// DefaultMaxRetryDelay is the maximum delay for exponential backoff
	DefaultMaxRetryDelay = 30 * time.Second

	// DefaultCacheDuration is the default cache validity duration
	DefaultCacheDuration = 30 * time.Second
)

// Client represents an HTTP client for communicating with a Smartap device
type Client struct {
	// BaseURL is the base URL for the device (e.g., "http://192.168.4.16")
	BaseURL string

	// Username for HTTP Basic Auth (default: "SmarTap")
	Username string

	// Password for HTTP Basic Auth (default: "yeswecan")
	Password string

	// HTTPClient is the underlying HTTP client
	HTTPClient *http.Client

	// MaxRetries is the maximum number of retry attempts for failed requests
	MaxRetries int

	// RetryDelay is the initial delay between retry attempts
	RetryDelay time.Duration

	// MaxRetryDelay is the maximum delay for exponential backoff
	MaxRetryDelay time.Duration

	// UseExponentialBackoff enables exponential backoff for retries
	UseExponentialBackoff bool

	// CacheDuration is how long to cache configuration (0 = no cache)
	CacheDuration time.Duration

	// cachedConfig is the cached configuration
	cachedConfig *DeviceConfig

	// cacheTime is when the cache was last updated
	cacheTime time.Time

	// cacheMutex protects the cache fields
	cacheMutex sync.RWMutex
}

// NewClient creates a new device configuration client
// ip: Device IP address (e.g., "192.168.4.16")
// port: Device HTTP port (typically 80)
func NewClient(ip string, port int) *Client {
	baseURL := fmt.Sprintf("http://%s:%d", ip, port)

	return &Client{
		BaseURL:               baseURL,
		Username:              DefaultUsername,
		Password:              DefaultPassword,
		HTTPClient:            &http.Client{Timeout: DefaultTimeout},
		MaxRetries:            DefaultMaxRetries,
		RetryDelay:            DefaultRetryDelay,
		MaxRetryDelay:         DefaultMaxRetryDelay,
		UseExponentialBackoff: true, // Enable by default
		CacheDuration:         DefaultCacheDuration,
	}
}

// NewClientWithURL creates a new client with a full base URL
// baseURL: Full base URL (e.g., "http://192.168.4.16:80")
func NewClientWithURL(baseURL string) *Client {
	return &Client{
		BaseURL:               baseURL,
		Username:              DefaultUsername,
		Password:              DefaultPassword,
		HTTPClient:            &http.Client{Timeout: DefaultTimeout},
		MaxRetries:            DefaultMaxRetries,
		RetryDelay:            DefaultRetryDelay,
		MaxRetryDelay:         DefaultMaxRetryDelay,
		UseExponentialBackoff: true, // Enable by default
		CacheDuration:         DefaultCacheDuration,
	}
}

// SetTimeout sets the HTTP request timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.HTTPClient.Timeout = timeout
}

// SetAuth sets custom HTTP Basic Auth credentials
func (c *Client) SetAuth(username, password string) {
	c.Username = username
	c.Password = password
}

// SetRetry configures retry behavior
func (c *Client) SetRetry(maxRetries int, retryDelay time.Duration) {
	c.MaxRetries = maxRetries
	c.RetryDelay = retryDelay
}

// Ping performs a simple health check on the device
// Returns nil if the device is reachable and responding
func (c *Client) Ping() error {
	req, err := http.NewRequest("GET", c.BaseURL+"/", nil)
	if err != nil {
		return NewNetworkError("failed to create ping request", err)
	}

	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return NewNetworkError("device unreachable", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return NewAuthError("authentication failed (check credentials)")
	}

	if resp.StatusCode != http.StatusOK {
		return NewHTTPError(resp.StatusCode, fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
	}

	return nil
}

// GetConfiguration retrieves the current device configuration
// Returns the parsed DeviceConfig struct or an error
// Uses cached configuration if available and fresh
func (c *Client) GetConfiguration() (*DeviceConfig, error) {
	// Check cache first (if caching is enabled)
	if c.CacheDuration > 0 {
		c.cacheMutex.RLock()
		if c.cachedConfig != nil && time.Since(c.cacheTime) < c.CacheDuration {
			// Cache is valid, return cached copy
			cached := *c.cachedConfig
			c.cacheMutex.RUnlock()
			return &cached, nil
		}
		c.cacheMutex.RUnlock()
	}

	var lastErr error
	currentDelay := c.RetryDelay

	// Retry loop with exponential backoff
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(currentDelay)

			// Exponential backoff
			if c.UseExponentialBackoff {
				currentDelay *= 2
				if currentDelay > c.MaxRetryDelay {
					currentDelay = c.MaxRetryDelay
				}
			}
		}

		config, err := c.getConfigurationAttempt()
		if err == nil {
			// Update cache
			if c.CacheDuration > 0 {
				c.cacheMutex.Lock()
				c.cachedConfig = config
				c.cacheTime = time.Now()
				c.cacheMutex.Unlock()
			}
			return config, nil
		}

		lastErr = err

		// Don't retry non-retryable errors
		if !IsRetryable(err) {
			return nil, err
		}
	}

	return nil, lastErr
}

// getConfigurationAttempt performs a single attempt to retrieve configuration
func (c *Client) getConfigurationAttempt() (*DeviceConfig, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/", nil)
	if err != nil {
		return nil, NewNetworkError("failed to create GET request", err)
	}

	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, NewNetworkError("GET request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, NewAuthError("authentication failed (check credentials)")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, NewHTTPError(resp.StatusCode, fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewNetworkError("failed to read response body", err)
	}

	// Clean malformed JSON response (device may append trailing data)
	cleanedBody, err := CleanJSONResponse(body)
	if err != nil {
		return nil, NewParseError("failed to clean JSON response", err)
	}

	// Parse JSON
	var config DeviceConfig
	if err := json.Unmarshal(cleanedBody, &config); err != nil {
		return nil, NewParseError("failed to parse JSON response", err)
	}

	return &config, nil
}

// UpdateConfiguration sends a configuration update to the device
// update: ConfigUpdate struct with fields to update
// Returns error if the update fails
// Invalidates the cache on successful update
func (c *Client) UpdateConfiguration(update *ConfigUpdate) error {
	var lastErr error
	currentDelay := c.RetryDelay

	// Retry loop with exponential backoff
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(currentDelay)

			// Exponential backoff
			if c.UseExponentialBackoff {
				currentDelay *= 2
				if currentDelay > c.MaxRetryDelay {
					currentDelay = c.MaxRetryDelay
				}
			}
		}

		err := c.updateConfigurationAttempt(update)
		if err == nil {
			// Invalidate cache after successful update
			c.InvalidateCache()
			return nil
		}

		lastErr = err

		// Don't retry non-retryable errors
		if !IsRetryable(err) {
			return err
		}
	}

	return lastErr
}

// updateConfigurationAttempt performs a single attempt to update configuration
func (c *Client) updateConfigurationAttempt(update *ConfigUpdate) error {
	// Convert update to form data
	formData := update.ToFormData()

	req, err := http.NewRequest("POST", c.BaseURL+"/", strings.NewReader(formData.Encode()))
	if err != nil {
		return NewNetworkError("failed to create POST request", err)
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return NewNetworkError("POST request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized {
		return NewAuthError("authentication failed (check credentials)")
	}

	// Device returns 200 OK or 204 No Content for successful updates
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		// Read error response if available
		body, _ := io.ReadAll(resp.Body)
		return NewHTTPError(resp.StatusCode, fmt.Sprintf("update failed with status %d: %s", resp.StatusCode, string(body)))
	}

	return nil
}

// UpdateDiverter updates the diverter button configuration
// config: DiverterConfig with button press bitmasks and K3 mode
func (c *Client) UpdateDiverter(config *DiverterConfig) error {
	update := &ConfigUpdate{
		Diverter: config,
	}
	return c.UpdateConfiguration(update)
}

// UpdateWiFi updates the WiFi configuration
// config: WiFiConfig with SSID, password, and security type
func (c *Client) UpdateWiFi(config *WiFiConfig) error {
	update := &ConfigUpdate{
		WiFi: config,
	}
	return c.UpdateConfiguration(update)
}

// UpdateServer updates the server configuration
// config: ServerConfig with DNS hostname and port
func (c *Client) UpdateServer(config *ServerConfig) error {
	update := &ConfigUpdate{
		Server: config,
	}
	return c.UpdateConfiguration(update)
}

// SetDiverterButtons sets the diverter button configuration with individual bitmasks
// This is a convenience method for setting button presses one at a time
func (c *Client) SetDiverterButtons(outlet1, outlet2, outlet3 int, k3Mode bool) error {
	config := &DiverterConfig{
		FirstPress:  outlet1,
		SecondPress: outlet2,
		ThirdPress:  outlet3,
		K3Mode:      k3Mode,
	}
	return c.UpdateDiverter(config)
}

// SetThirdKnobMode sets only the third knob separation mode
// This is a convenience method when you only want to change K3 mode
func (c *Client) SetThirdKnobMode(enabled bool) error {
	config := &DiverterConfig{
		K3Mode: enabled,
	}
	update := &ConfigUpdate{
		Diverter: config,
	}
	return c.UpdateConfiguration(update)
}

// SetSequentialOutlets configures outlets for sequential operation (common pattern)
// Button 1 -> Outlet 1, Button 2 -> Outlet 2, Button 3 -> Outlet 3
func (c *Client) SetSequentialOutlets(k3Mode bool) error {
	return c.SetDiverterButtons(1, 2, 4, k3Mode)
}

// SetAllOutletsOn configures all outlets to activate with each button press
// Useful for testing or when all outlets should run simultaneously
func (c *Client) SetAllOutletsOn() error {
	// Bitmask 7 = all three outlets (1 + 2 + 4)
	return c.SetDiverterButtons(7, 7, 7, false)
}

// VerifyConfiguration retrieves configuration and verifies it matches expected values
// This is useful after UpdateConfiguration to ensure the device applied the changes
func (c *Client) VerifyConfiguration(expected *ConfigUpdate) error {
	// Wait briefly for device to apply changes
	time.Sleep(500 * time.Millisecond)

	current, err := c.GetConfiguration()
	if err != nil {
		return fmt.Errorf("failed to retrieve configuration for verification: %w", err)
	}

	// Verify diverter configuration if provided
	if expected.Diverter != nil {
		if current.Outlet1 != expected.Diverter.FirstPress {
			return NewValidationError(fmt.Sprintf("diverter first press mismatch: expected %d, got %d", expected.Diverter.FirstPress, current.Outlet1))
		}
		if current.Outlet2 != expected.Diverter.SecondPress {
			return NewValidationError(fmt.Sprintf("diverter second press mismatch: expected %d, got %d", expected.Diverter.SecondPress, current.Outlet2))
		}
		if current.Outlet3 != expected.Diverter.ThirdPress {
			return NewValidationError(fmt.Sprintf("diverter third press mismatch: expected %d, got %d", expected.Diverter.ThirdPress, current.Outlet3))
		}
		if current.K3Outlet != expected.Diverter.K3Mode {
			return NewValidationError(fmt.Sprintf("K3 mode mismatch: expected %v, got %v", expected.Diverter.K3Mode, current.K3Outlet))
		}
	}

	// Verify server configuration if provided
	if expected.Server != nil {
		if current.DNS != expected.Server.DNS {
			return NewValidationError(fmt.Sprintf("DNS mismatch: expected %s, got %s", expected.Server.DNS, current.DNS))
		}
		if current.Port != expected.Server.Port {
			return NewValidationError(fmt.Sprintf("port mismatch: expected %d, got %d", expected.Server.Port, current.Port))
		}
	}

	// Note: WiFi config cannot be verified via GET (device doesn't return WiFi credentials)

	return nil
}

// GetFormData is a helper method that returns the raw form data for a configuration update
// This is useful for debugging or manual inspection
func (c *Client) GetFormData(update *ConfigUpdate) url.Values {
	return update.ToFormData()
}

// InvalidateCache clears the cached configuration, forcing the next GetConfiguration to fetch fresh data
func (c *Client) InvalidateCache() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.cachedConfig = nil
	c.cacheTime = time.Time{}
}

// SetCacheDuration sets the cache validity duration
// Set to 0 to disable caching entirely
func (c *Client) SetCacheDuration(duration time.Duration) {
	c.CacheDuration = duration
	if duration == 0 {
		// Disable caching - clear cache
		c.InvalidateCache()
	}
}

// GetCachedConfiguration returns the cached configuration without making a network request
// Returns nil if no valid cache exists
func (c *Client) GetCachedConfiguration() *DeviceConfig {
	c.cacheMutex.RLock()
	defer c.cacheMutex.RUnlock()

	if c.cachedConfig != nil && time.Since(c.cacheTime) < c.CacheDuration {
		cached := *c.cachedConfig
		return &cached
	}
	return nil
}

// RefreshConfiguration forces a fresh fetch from the device, bypassing and updating the cache
func (c *Client) RefreshConfiguration() (*DeviceConfig, error) {
	// Temporarily disable cache for this request
	oldDuration := c.CacheDuration
	c.CacheDuration = 0

	config, err := c.GetConfiguration()

	// Restore cache duration and update cache if successful
	c.CacheDuration = oldDuration
	if err == nil && c.CacheDuration > 0 {
		c.cacheMutex.Lock()
		c.cachedConfig = config
		c.cacheTime = time.Now()
		c.cacheMutex.Unlock()
	}

	return config, err
}
