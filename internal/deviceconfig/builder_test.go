package deviceconfig

import (
	"testing"
)

// Sample device configuration for testing
var sampleConfig = &DeviceConfig{
	Serial:   "315260240",
	DNS:      "lb.smartap-tech.com",
	Port:     80,
	Outlet1:  1,
	Outlet2:  2,
	Outlet3:  4,
	K3Outlet: true,
	MAC:      "C4:BE:84:74:86:37",
}

// TestNewConfigBuilder tests builder initialization with and without current config
func TestNewConfigBuilder(t *testing.T) {
	t.Run("with current config", func(t *testing.T) {
		builder := NewConfigBuilder(sampleConfig)

		if builder.current != sampleConfig {
			t.Error("Builder should store reference to current config")
		}

		// Verify initial values copied from current config
		if builder.firstPress != 1 {
			t.Errorf("Expected firstPress=1, got %d", builder.firstPress)
		}
		if builder.secondPress != 2 {
			t.Errorf("Expected secondPress=2, got %d", builder.secondPress)
		}
		if builder.thirdPress != 4 {
			t.Errorf("Expected thirdPress=4, got %d", builder.thirdPress)
		}
		if !builder.k3Mode {
			t.Error("Expected k3Mode=true")
		}
		if builder.dns != "lb.smartap-tech.com" {
			t.Errorf("Expected dns='lb.smartap-tech.com', got '%s'", builder.dns)
		}
		if builder.port != 80 {
			t.Errorf("Expected port=80, got %d", builder.port)
		}
	})

	t.Run("without current config", func(t *testing.T) {
		builder := NewConfigBuilder(nil)

		if builder.current != nil {
			t.Error("Builder should have nil current config")
		}

		// Verify initial values are zero
		if builder.firstPress != 0 {
			t.Errorf("Expected firstPress=0, got %d", builder.firstPress)
		}
		if builder.dns != "" {
			t.Errorf("Expected empty dns, got '%s'", builder.dns)
		}
	})
}

// TestSetDiverterButton tests individual button configuration
func TestSetDiverterButton(t *testing.T) {
	builder := NewConfigBuilder(sampleConfig)

	builder.SetDiverterButton(1, 3).
		SetDiverterButton(2, 5).
		SetDiverterButton(3, 7)

	if !builder.diverterChanged {
		t.Error("diverterChanged should be true")
	}

	if builder.firstPress != 3 {
		t.Errorf("Expected firstPress=3, got %d", builder.firstPress)
	}
	if builder.secondPress != 5 {
		t.Errorf("Expected secondPress=5, got %d", builder.secondPress)
	}
	if builder.thirdPress != 7 {
		t.Errorf("Expected thirdPress=7, got %d", builder.thirdPress)
	}
}

// TestSetFirstSecondThirdPress tests convenience methods for button presses
func TestSetFirstSecondThirdPress(t *testing.T) {
	builder := NewConfigBuilder(nil)

	builder.SetFirstPress(1).
		SetSecondPress(2).
		SetThirdPress(4)

	if builder.firstPress != 1 {
		t.Errorf("Expected firstPress=1, got %d", builder.firstPress)
	}
	if builder.secondPress != 2 {
		t.Errorf("Expected secondPress=2, got %d", builder.secondPress)
	}
	if builder.thirdPress != 4 {
		t.Errorf("Expected thirdPress=4, got %d", builder.thirdPress)
	}
}

// TestBuilderSetSequentialOutlets tests the sequential configuration pattern
func TestBuilderSetSequentialOutlets(t *testing.T) {
	builder := NewConfigBuilder(nil)

	builder.SetSequentialOutlets()

	if builder.firstPress != 1 {
		t.Errorf("Expected firstPress=1, got %d", builder.firstPress)
	}
	if builder.secondPress != 2 {
		t.Errorf("Expected secondPress=2, got %d", builder.secondPress)
	}
	if builder.thirdPress != 4 {
		t.Errorf("Expected thirdPress=4, got %d", builder.thirdPress)
	}
	if !builder.diverterChanged {
		t.Error("diverterChanged should be true")
	}
}

// TestBuilderSetAllOutletsOn tests the all-outlets-on pattern
func TestBuilderSetAllOutletsOn(t *testing.T) {
	builder := NewConfigBuilder(nil)

	builder.SetAllOutletsOn()

	if builder.firstPress != 7 {
		t.Errorf("Expected firstPress=7, got %d", builder.firstPress)
	}
	if builder.secondPress != 7 {
		t.Errorf("Expected secondPress=7, got %d", builder.secondPress)
	}
	if builder.thirdPress != 7 {
		t.Errorf("Expected thirdPress=7, got %d", builder.thirdPress)
	}
}

// TestThirdKnobMethods tests third knob mode configuration
func TestThirdKnobMethods(t *testing.T) {
	t.Run("EnableThirdKnob", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.EnableThirdKnob()

		if !builder.k3Mode {
			t.Error("Expected k3Mode=true")
		}
		if !builder.diverterChanged {
			t.Error("diverterChanged should be true")
		}
	})

	t.Run("DisableThirdKnob", func(t *testing.T) {
		builder := NewConfigBuilder(sampleConfig)
		builder.DisableThirdKnob()

		if builder.k3Mode {
			t.Error("Expected k3Mode=false")
		}
		if !builder.diverterChanged {
			t.Error("diverterChanged should be true")
		}
	})

	t.Run("SetThirdKnobMode", func(t *testing.T) {
		builder := NewConfigBuilder(nil)

		builder.SetThirdKnobMode(true)
		if !builder.k3Mode {
			t.Error("Expected k3Mode=true")
		}

		builder.SetThirdKnobMode(false)
		if builder.k3Mode {
			t.Error("Expected k3Mode=false")
		}
	})
}

// TestWiFiConfiguration tests WiFi configuration methods
func TestWiFiConfiguration(t *testing.T) {
	t.Run("SetWiFi all at once", func(t *testing.T) {
		builder := NewConfigBuilder(nil)

		builder.SetWiFi("MyNetwork", "password123", "WPA2")

		if !builder.wifiChanged {
			t.Error("wifiChanged should be true")
		}
		if builder.ssid != "MyNetwork" {
			t.Errorf("Expected ssid='MyNetwork', got '%s'", builder.ssid)
		}
		if builder.password != "password123" {
			t.Errorf("Expected password='password123', got '%s'", builder.password)
		}
		if builder.securityType != "WPA2" {
			t.Errorf("Expected securityType='WPA2', got '%s'", builder.securityType)
		}
	})

	t.Run("SetWiFiWPA2", func(t *testing.T) {
		builder := NewConfigBuilder(nil)

		builder.SetWiFiWPA2("SecureNetwork", "secret")

		if builder.ssid != "SecureNetwork" {
			t.Errorf("Expected ssid='SecureNetwork', got '%s'", builder.ssid)
		}
		if builder.password != "secret" {
			t.Errorf("Expected password='secret', got '%s'", builder.password)
		}
		if builder.securityType != "WPA2" {
			t.Errorf("Expected securityType='WPA2', got '%s'", builder.securityType)
		}
	})

	t.Run("SetWiFiOpen", func(t *testing.T) {
		builder := NewConfigBuilder(nil)

		builder.SetWiFiOpen("OpenNetwork")

		if builder.ssid != "OpenNetwork" {
			t.Errorf("Expected ssid='OpenNetwork', got '%s'", builder.ssid)
		}
		if builder.password != "" {
			t.Errorf("Expected empty password, got '%s'", builder.password)
		}
		if builder.securityType != "OPEN" {
			t.Errorf("Expected securityType='OPEN', got '%s'", builder.securityType)
		}
	})

	t.Run("Individual setters", func(t *testing.T) {
		builder := NewConfigBuilder(nil)

		builder.SetWiFiSSID("TestNetwork").
			SetWiFiPassword("testpass").
			SetWiFiSecurity("WPA2")

		if builder.ssid != "TestNetwork" {
			t.Errorf("Expected ssid='TestNetwork', got '%s'", builder.ssid)
		}
		if builder.password != "testpass" {
			t.Errorf("Expected password='testpass', got '%s'", builder.password)
		}
		if builder.securityType != "WPA2" {
			t.Errorf("Expected securityType='WPA2', got '%s'", builder.securityType)
		}
	})
}

// TestServerConfiguration tests server configuration methods
func TestServerConfiguration(t *testing.T) {
	t.Run("SetServer all at once", func(t *testing.T) {
		builder := NewConfigBuilder(nil)

		builder.SetServer("my-server.local", 8080)

		if !builder.serverChanged {
			t.Error("serverChanged should be true")
		}
		if builder.dns != "my-server.local" {
			t.Errorf("Expected dns='my-server.local', got '%s'", builder.dns)
		}
		if builder.port != 8080 {
			t.Errorf("Expected port=8080, got %d", builder.port)
		}
	})

	t.Run("Individual setters", func(t *testing.T) {
		builder := NewConfigBuilder(nil)

		builder.SetServerDNS("test.example.com").
			SetServerPort(443)

		if builder.dns != "test.example.com" {
			t.Errorf("Expected dns='test.example.com', got '%s'", builder.dns)
		}
		if builder.port != 443 {
			t.Errorf("Expected port=443, got %d", builder.port)
		}
	})
}

// TestHasChanges tests the change detection logic
func TestHasChanges(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		builder := NewConfigBuilder(sampleConfig)

		if builder.HasChanges() {
			t.Error("Expected no changes")
		}
	})

	t.Run("diverter changed", func(t *testing.T) {
		builder := NewConfigBuilder(sampleConfig)
		builder.SetFirstPress(3)

		if !builder.HasChanges() {
			t.Error("Expected changes")
		}
	})

	t.Run("wifi changed", func(t *testing.T) {
		builder := NewConfigBuilder(sampleConfig)
		builder.SetWiFiSSID("NewNetwork")

		if !builder.HasChanges() {
			t.Error("Expected changes")
		}
	})

	t.Run("server changed", func(t *testing.T) {
		builder := NewConfigBuilder(sampleConfig)
		builder.SetServerPort(443)

		if !builder.HasChanges() {
			t.Error("Expected changes")
		}
	})
}

// TestValidate tests validation logic
func TestValidate(t *testing.T) {
	t.Run("valid diverter config", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetSequentialOutlets()

		err := builder.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("invalid bitmask - too high", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetFirstPress(8) // Invalid: max is 7

		err := builder.Validate()
		if err == nil {
			t.Error("Expected validation error for bitmask > 7")
		}
		if !IsValidationError(err) {
			t.Error("Expected ValidationError type")
		}
	})

	t.Run("invalid bitmask - negative", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetSecondPress(-1)

		err := builder.Validate()
		if err == nil {
			t.Error("Expected validation error for negative bitmask")
		}
	})

	t.Run("valid WiFi WPA2", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetWiFiWPA2("TestNetwork", "password123")

		err := builder.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("invalid WiFi - empty SSID", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetWiFi("", "password", "WPA2")

		err := builder.Validate()
		if err == nil {
			t.Error("Expected validation error for empty SSID")
		}
	})

	t.Run("invalid WiFi - WPA2 without password", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetWiFi("TestNetwork", "", "WPA2")

		err := builder.Validate()
		if err == nil {
			t.Error("Expected validation error for WPA2 without password")
		}
	})

	t.Run("invalid WiFi - bad security type", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetWiFi("TestNetwork", "pass", "WEP")

		err := builder.Validate()
		if err == nil {
			t.Error("Expected validation error for invalid security type")
		}
	})

	t.Run("valid server config", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetServer("lb.smartap-tech.com", 80)

		err := builder.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("invalid server - empty DNS", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetServer("", 80)

		err := builder.Validate()
		if err == nil {
			t.Error("Expected validation error for empty DNS")
		}
	})

	t.Run("invalid server - port too low", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetServer("test.com", 0)

		err := builder.Validate()
		if err == nil {
			t.Error("Expected validation error for port=0")
		}
	})

	t.Run("invalid server - port too high", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetServer("test.com", 65536)

		err := builder.Validate()
		if err == nil {
			t.Error("Expected validation error for port > 65535")
		}
	})
}

// TestBuild tests the Build method and ConfigUpdate generation
func TestBuild(t *testing.T) {
	t.Run("diverter only", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetSequentialOutlets().EnableThirdKnob()

		update, err := builder.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if update.Diverter == nil {
			t.Fatal("Expected Diverter config")
		}
		if update.WiFi != nil {
			t.Error("Expected nil WiFi config")
		}
		if update.Server != nil {
			t.Error("Expected nil Server config")
		}

		if update.Diverter.FirstPress != 1 {
			t.Errorf("Expected FirstPress=1, got %d", update.Diverter.FirstPress)
		}
		if update.Diverter.SecondPress != 2 {
			t.Errorf("Expected SecondPress=2, got %d", update.Diverter.SecondPress)
		}
		if update.Diverter.ThirdPress != 4 {
			t.Errorf("Expected ThirdPress=4, got %d", update.Diverter.ThirdPress)
		}
		if !update.Diverter.K3Mode {
			t.Error("Expected K3Mode=true")
		}
	})

	t.Run("wifi only", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetWiFiWPA2("MyNetwork", "password123")

		update, err := builder.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if update.Diverter != nil {
			t.Error("Expected nil Diverter config")
		}
		if update.WiFi == nil {
			t.Fatal("Expected WiFi config")
		}
		if update.Server != nil {
			t.Error("Expected nil Server config")
		}

		if update.WiFi.SSID != "MyNetwork" {
			t.Errorf("Expected SSID='MyNetwork', got '%s'", update.WiFi.SSID)
		}
		if update.WiFi.Password != "password123" {
			t.Errorf("Expected Password='password123', got '%s'", update.WiFi.Password)
		}
		if update.WiFi.SecurityType != "WPA2" {
			t.Errorf("Expected SecurityType='WPA2', got '%s'", update.WiFi.SecurityType)
		}
	})

	t.Run("server only", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetServer("my-server.local", 8080)

		update, err := builder.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if update.Diverter != nil {
			t.Error("Expected nil Diverter config")
		}
		if update.WiFi != nil {
			t.Error("Expected nil WiFi config")
		}
		if update.Server == nil {
			t.Fatal("Expected Server config")
		}

		if update.Server.DNS != "my-server.local" {
			t.Errorf("Expected DNS='my-server.local', got '%s'", update.Server.DNS)
		}
		if update.Server.Port != 8080 {
			t.Errorf("Expected Port=8080, got %d", update.Server.Port)
		}
	})

	t.Run("all sections", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetSequentialOutlets().
			SetWiFiWPA2("TestNet", "pass").
			SetServer("test.com", 443)

		update, err := builder.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}

		if update.Diverter == nil {
			t.Error("Expected Diverter config")
		}
		if update.WiFi == nil {
			t.Error("Expected WiFi config")
		}
		if update.Server == nil {
			t.Error("Expected Server config")
		}
	})

	t.Run("validation failure", func(t *testing.T) {
		builder := NewConfigBuilder(nil)
		builder.SetFirstPress(10) // Invalid

		_, err := builder.Build()
		if err == nil {
			t.Error("Expected Build to fail validation")
		}
	})
}

// TestReset tests the Reset method
func TestReset(t *testing.T) {
	t.Run("reset to current config", func(t *testing.T) {
		builder := NewConfigBuilder(sampleConfig)

		// Make changes
		builder.SetAllOutletsOn().
			SetWiFiWPA2("NewNetwork", "newpass").
			SetServer("new.com", 443)

		if !builder.HasChanges() {
			t.Error("Expected changes before reset")
		}

		// Reset
		builder.Reset()

		if builder.HasChanges() {
			t.Error("Expected no changes after reset")
		}

		// Verify values restored
		if builder.firstPress != 1 {
			t.Errorf("Expected firstPress=1 after reset, got %d", builder.firstPress)
		}
		if builder.dns != "lb.smartap-tech.com" {
			t.Errorf("Expected dns='lb.smartap-tech.com' after reset, got '%s'", builder.dns)
		}
	})

	t.Run("reset without current config", func(t *testing.T) {
		builder := NewConfigBuilder(nil)

		builder.SetSequentialOutlets().SetServer("test.com", 80)

		builder.Reset()

		if builder.HasChanges() {
			t.Error("Expected no changes after reset")
		}

		// Verify values cleared
		if builder.firstPress != 0 {
			t.Errorf("Expected firstPress=0 after reset, got %d", builder.firstPress)
		}
		if builder.dns != "" {
			t.Errorf("Expected empty dns after reset, got '%s'", builder.dns)
		}
	})
}

// TestFluentAPI tests that methods return builder for chaining
func TestFluentAPI(t *testing.T) {
	builder := NewConfigBuilder(nil)

	// This should compile and execute without issues
	update, err := builder.
		SetSequentialOutlets().
		EnableThirdKnob().
		SetWiFiWPA2("TestNetwork", "password").
		SetServer("test.com", 80).
		Build()

	if err != nil {
		t.Fatalf("Fluent API failed: %v", err)
	}

	if update == nil {
		t.Fatal("Expected non-nil update")
	}

	// Verify all sections present
	if update.Diverter == nil || update.WiFi == nil || update.Server == nil {
		t.Error("Expected all sections configured via fluent API")
	}
}
