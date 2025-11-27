package deviceconfig

import (
	"testing"
)

// Test data - real response from live device
const validDeviceResponse = `{"ssidList":["NETGEAR89"],"lowPowerMode":false,"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true,"swVer":"0x355","wnpVer":"2.:.0.000","mac":"C4:BE:84:74:86:37"}`

const malformedDeviceResponse = `{"ssidList":["NETGEAR89"],"lowPowerMode":false,"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true,"swVer":"0x355","wnpVer":"2.:.0.000","mac":"C4:BE:84:74:86:37"}"oldAppVer":"pkey:0000,315260240<\/div>"`

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "valid JSON only",
			input:   []byte(validDeviceResponse),
			want:    validDeviceResponse,
			wantErr: false,
		},
		{
			name:    "malformed response with trailing data",
			input:   []byte(malformedDeviceResponse),
			want:    validDeviceResponse,
			wantErr: false,
		},
		{
			name:    "JSON with leading whitespace",
			input:   []byte("  \n  " + validDeviceResponse),
			want:    validDeviceResponse,
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   []byte(""),
			wantErr: true,
		},
		{
			name:    "no JSON object",
			input:   []byte("this is not JSON"),
			wantErr: true,
		},
		{
			name:    "unclosed JSON",
			input:   []byte(`{"key":"value"`),
			wantErr: true,
		},
		{
			name:    "JSON with escaped quotes",
			input:   []byte(`{"key":"value with \"quotes\""}`),
			want:    `{"key":"value with \"quotes\""}`,
			wantErr: false,
		},
		{
			name:    "nested JSON objects",
			input:   []byte(`{"outer":{"inner":"value"},"key2":"value2"}`),
			want:    `{"outer":{"inner":"value"},"key2":"value2"}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CleanJSONResponse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CleanJSONResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("CleanJSONResponse() = %v, want %v", string(got), tt.want)
			}
		})
	}
}

func TestParseDeviceConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr bool
		check   func(*testing.T, *DeviceConfig)
	}{
		{
			name:    "valid device response",
			input:   []byte(validDeviceResponse),
			wantErr: false,
			check: func(t *testing.T, dc *DeviceConfig) {
				if dc.Serial != "315260240" {
					t.Errorf("Serial = %v, want 315260240", dc.Serial)
				}
				if dc.MAC != "C4:BE:84:74:86:37" {
					t.Errorf("MAC = %v, want C4:BE:84:74:86:37", dc.MAC)
				}
				if dc.Outlet1 != 1 {
					t.Errorf("Outlet1 = %v, want 1", dc.Outlet1)
				}
				if dc.Outlet2 != 2 {
					t.Errorf("Outlet2 = %v, want 2", dc.Outlet2)
				}
				if dc.Outlet3 != 4 {
					t.Errorf("Outlet3 = %v, want 4", dc.Outlet3)
				}
				if !dc.K3Outlet {
					t.Errorf("K3Outlet = %v, want true", dc.K3Outlet)
				}
				if dc.SWVer != "0x355" {
					t.Errorf("SWVer = %v, want 0x355", dc.SWVer)
				}
			},
		},
		{
			name:    "malformed device response",
			input:   []byte(malformedDeviceResponse),
			wantErr: false,
			check: func(t *testing.T, dc *DeviceConfig) {
				if dc.Serial != "315260240" {
					t.Errorf("Serial = %v, want 315260240", dc.Serial)
				}
			},
		},
		{
			name:    "invalid JSON",
			input:   []byte("not json"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDeviceConfig(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDeviceConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestDecodeBitmask(t *testing.T) {
	tests := []struct {
		name    string
		bitmask int
		want    []int
	}{
		{"no outlets", 0, []int{}},
		{"outlet 1 only", 1, []int{1}},
		{"outlet 2 only", 2, []int{2}},
		{"outlets 1+2", 3, []int{1, 2}},
		{"outlet 3 only", 4, []int{3}},
		{"outlets 1+3", 5, []int{1, 3}},
		{"outlets 2+3", 6, []int{2, 3}},
		{"all outlets", 7, []int{1, 2, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodeBitmask(tt.bitmask)
			if len(got) != len(tt.want) {
				t.Errorf("DecodeBitmask(%d) length = %v, want %v", tt.bitmask, len(got), len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("DecodeBitmask(%d)[%d] = %v, want %v", tt.bitmask, i, v, tt.want[i])
				}
			}
		})
	}
}

func TestEncodeBitmask(t *testing.T) {
	tests := []struct {
		name    string
		outlets []int
		want    int
	}{
		{"no outlets", []int{}, 0},
		{"outlet 1 only", []int{1}, 1},
		{"outlet 2 only", []int{2}, 2},
		{"outlets 1+2", []int{1, 2}, 3},
		{"outlet 3 only", []int{3}, 4},
		{"outlets 1+3", []int{1, 3}, 5},
		{"outlets 2+3", []int{2, 3}, 6},
		{"all outlets", []int{1, 2, 3}, 7},
		{"invalid outlet 0", []int{0}, 0},
		{"invalid outlet 4", []int{4}, 0},
		{"mixed valid and invalid", []int{0, 1, 4}, 1},
		{"duplicate outlets", []int{1, 1, 2}, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeBitmask(tt.outlets)
			if got != tt.want {
				t.Errorf("EncodeBitmask(%v) = %v, want %v", tt.outlets, got, tt.want)
			}
		})
	}
}

func TestFormatBitmask(t *testing.T) {
	tests := []struct {
		bitmask int
		want    string
	}{
		{0, "No outlets"},
		{1, "Outlet 1"},
		{2, "Outlet 2"},
		{3, "Outlets 1+2"},
		{4, "Outlet 3"},
		{5, "Outlets 1+3"},
		{6, "Outlets 2+3"},
		{7, "Outlets 1+2+3"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatBitmask(tt.bitmask)
			if got != tt.want {
				t.Errorf("FormatBitmask(%d) = %v, want %v", tt.bitmask, got, tt.want)
			}
		})
	}
}

func TestDiverterConfigToFormData(t *testing.T) {
	tests := []struct {
		name   string
		config DiverterConfig
		want   map[string]string
	}{
		{
			name: "standard sequential",
			config: DiverterConfig{
				FirstPress:  1,
				SecondPress: 2,
				ThirdPress:  4,
				K3Mode:      true,
			},
			want: map[string]string{
				"__SL_P_OU1": "1",
				"__SL_P_OU2": "2",
				"__SL_P_OU3": "4",
				"__SL_P_K3O": "checked",
			},
		},
		{
			name: "combined outlets",
			config: DiverterConfig{
				FirstPress:  3,
				SecondPress: 5,
				ThirdPress:  7,
				K3Mode:      false,
			},
			want: map[string]string{
				"__SL_P_OU1": "3",
				"__SL_P_OU2": "5",
				"__SL_P_OU3": "7",
				"__SL_P_K3O": "no",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ToFormData()
			for key, wantVal := range tt.want {
				gotVal := got.Get(key)
				if gotVal != wantVal {
					t.Errorf("ToFormData()[%s] = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestWiFiConfigToFormData(t *testing.T) {
	tests := []struct {
		name   string
		config WiFiConfig
		want   map[string]string
	}{
		{
			name: "WPA2 network",
			config: WiFiConfig{
				SSID:         "MyNetwork",
				Password:     "MyPassword",
				SecurityType: "WPA2",
			},
			want: map[string]string{
				"__SL_P_USD": "MyNetwork",
				"__SL_P_PSD": "MyPassword",
				"__SL_P_ENC": "WPA2",
				"__SL_P_CON": "connect",
			},
		},
		{
			name: "Open network",
			config: WiFiConfig{
				SSID:         "OpenNetwork",
				SecurityType: "OPEN",
			},
			want: map[string]string{
				"__SL_P_USD": "OpenNetwork",
				"__SL_P_ENC": "OPEN",
				"__SL_P_CON": "connect",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ToFormData()
			for key, wantVal := range tt.want {
				gotVal := got.Get(key)
				if gotVal != wantVal {
					t.Errorf("ToFormData()[%s] = %v, want %v", key, gotVal, wantVal)
				}
			}
			// For open networks, password should not be present
			if tt.config.SecurityType == "OPEN" {
				if got.Has("__SL_P_PSD") {
					t.Errorf("ToFormData() should not include __SL_P_PSD for open network")
				}
			}
		})
	}
}

func TestServerConfigToFormData(t *testing.T) {
	config := ServerConfig{
		DNS:  "my.server.com",
		Port: 8080,
	}

	got := config.ToFormData()

	if got.Get("__SL_P_DNS") != "my.server.com" {
		t.Errorf("ToFormData()[__SL_P_DNS] = %v, want my.server.com", got.Get("__SL_P_DNS"))
	}
	if got.Get("__SL_P_PRT") != "8080" {
		t.Errorf("ToFormData()[__SL_P_PRT] = %v, want 8080", got.Get("__SL_P_PRT"))
	}
}

func TestConfigUpdateToFormData(t *testing.T) {
	update := ConfigUpdate{
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

	got := update.ToFormData()

	// Check diverter params
	if got.Get("__SL_P_OU1") != "3" {
		t.Errorf("Expected __SL_P_OU1=3, got %s", got.Get("__SL_P_OU1"))
	}
	if got.Get("__SL_P_K3O") != "checked" {
		t.Errorf("Expected __SL_P_K3O=checked, got %s", got.Get("__SL_P_K3O"))
	}

	// Check server params
	if got.Get("__SL_P_DNS") != "test.server.com" {
		t.Errorf("Expected __SL_P_DNS=test.server.com, got %s", got.Get("__SL_P_DNS"))
	}
	if got.Get("__SL_P_PRT") != "443" {
		t.Errorf("Expected __SL_P_PRT=443, got %s", got.Get("__SL_P_PRT"))
	}
}

func TestDeviceConfigString(t *testing.T) {
	config := DeviceConfig{
		Serial:   "315260240",
		MAC:      "C4:BE:84:74:86:37",
		SWVer:    "0x355",
		DNS:      "lb.smartap-tech.com",
		Port:     80,
		SSIDList: []string{"NETGEAR89"},
		Outlet1:  1,
		Outlet2:  2,
		Outlet3:  4,
		K3Outlet: true,
	}

	got := config.String()

	// Check that important information is present
	mustContain := []string{
		"315260240",
		"C4:BE:84:74:86:37",
		"0x355",
		"lb.smartap-tech.com:80",
		"NETGEAR89",
		"Outlet 1",
		"Outlet 2",
		"Outlet 3",
		"true",
	}

	for _, substr := range mustContain {
		if !contains(got, substr) {
			t.Errorf("String() missing expected substring: %s\nGot: %s", substr, got)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}

// Benchmark tests
func BenchmarkCleanJSONResponse(b *testing.B) {
	data := []byte(malformedDeviceResponse)
	for i := 0; i < b.N; i++ {
		CleanJSONResponse(data)
	}
}

func BenchmarkParseDeviceConfig(b *testing.B) {
	data := []byte(malformedDeviceResponse)
	for i := 0; i < b.N; i++ {
		ParseDeviceConfig(data)
	}
}

func BenchmarkEncodeBitmask(b *testing.B) {
	outlets := []int{1, 2, 3}
	for i := 0; i < b.N; i++ {
		EncodeBitmask(outlets)
	}
}

func BenchmarkDecodeBitmask(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DecodeBitmask(7)
	}
}
