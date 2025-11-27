package deviceconfig

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestRollbackManager_SaveSnapshot tests saving configuration snapshots
func TestRollbackManager_SaveSnapshot(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true,"swVer":"0x355","mac":"C4:BE:84:74:86:37"}`)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Username:   DefaultUsername,
		Password:   DefaultPassword,
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
	}

	rm := NewRollbackManager(client)

	// Save snapshot
	err := rm.SaveSnapshot("Before test update")
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Verify snapshot was saved
	snapshot := rm.GetLatestSnapshot()
	if snapshot == nil {
		t.Fatal("Expected snapshot to be saved")
	}

	if snapshot.Description != "Before test update" {
		t.Errorf("Expected description 'Before test update', got %q", snapshot.Description)
	}

	if snapshot.Config.Outlet1 != 1 || snapshot.Config.Outlet2 != 2 || snapshot.Config.Outlet3 != 4 {
		t.Errorf("Snapshot config mismatch: got [%d,%d,%d]", snapshot.Config.Outlet1, snapshot.Config.Outlet2, snapshot.Config.Outlet3)
	}
}

// TestRollbackManager_SnapshotLimit tests that only maxSnapshots are retained
func TestRollbackManager_SnapshotLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true}`)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Username:   DefaultUsername,
		Password:   DefaultPassword,
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
	}

	rm := NewRollbackManager(client)

	// Save more than maxSnapshots
	for i := 0; i < 15; i++ {
		err := rm.SaveSnapshot(fmt.Sprintf("Snapshot %d", i))
		if err != nil {
			t.Fatalf("SaveSnapshot %d failed: %v", i, err)
		}
	}

	// Should only have maxSnapshots (10) retained
	snapshots := rm.GetSnapshots()
	if len(snapshots) != 10 {
		t.Errorf("Expected 10 snapshots, got %d", len(snapshots))
	}

	// Oldest snapshot should be "Snapshot 5" (0-4 were removed)
	if snapshots[0].Description != "Snapshot 5" {
		t.Errorf("Expected oldest snapshot to be 'Snapshot 5', got %q", snapshots[0].Description)
	}

	// Newest snapshot should be "Snapshot 14"
	if snapshots[9].Description != "Snapshot 14" {
		t.Errorf("Expected newest snapshot to be 'Snapshot 14', got %q", snapshots[9].Description)
	}
}

// TestRollbackManager_GetSnapshots tests retrieving all snapshots
func TestRollbackManager_GetSnapshots(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true}`)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Username:   DefaultUsername,
		Password:   DefaultPassword,
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
	}

	rm := NewRollbackManager(client)

	// No snapshots initially
	snapshots := rm.GetSnapshots()
	if len(snapshots) != 0 {
		t.Errorf("Expected 0 snapshots initially, got %d", len(snapshots))
	}

	// Save some snapshots
	for i := 0; i < 3; i++ {
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		err := rm.SaveSnapshot(fmt.Sprintf("Snapshot %d", i))
		if err != nil {
			t.Fatalf("SaveSnapshot %d failed: %v", i, err)
		}
	}

	snapshots = rm.GetSnapshots()
	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}

	// Verify chronological order
	for i := 0; i < 3; i++ {
		if snapshots[i].Description != fmt.Sprintf("Snapshot %d", i) {
			t.Errorf("Expected snapshot %d to be 'Snapshot %d', got %q", i, i, snapshots[i].Description)
		}
	}

	// Verify timestamps are ordered
	for i := 1; i < 3; i++ {
		if snapshots[i].Timestamp.Before(snapshots[i-1].Timestamp) {
			t.Error("Snapshots are not in chronological order")
		}
	}
}

// TestRollbackManager_ClearSnapshots tests clearing all snapshots
func TestRollbackManager_ClearSnapshots(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true}`)
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Username:   DefaultUsername,
		Password:   DefaultPassword,
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
	}

	rm := NewRollbackManager(client)

	// Save some snapshots
	for i := 0; i < 3; i++ {
		rm.SaveSnapshot(fmt.Sprintf("Snapshot %d", i))
	}

	if len(rm.GetSnapshots()) != 3 {
		t.Fatal("Expected 3 snapshots before clear")
	}

	// Clear snapshots
	rm.ClearSnapshots()

	if len(rm.GetSnapshots()) != 0 {
		t.Errorf("Expected 0 snapshots after clear, got %d", len(rm.GetSnapshots()))
	}

	if rm.GetLatestSnapshot() != nil {
		t.Error("Expected GetLatestSnapshot to return nil after clear")
	}
}

// TestRollbackManager_RollbackToSnapshot tests rolling back to a snapshot
func TestRollbackManager_RollbackToSnapshot(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Return different config on subsequent GET requests
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if requestCount == 1 {
				// Original config
				fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true}`)
			} else {
				// After rollback (same as original)
				fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true}`)
			}
		} else if r.Method == http.MethodPost {
			// Accept POST updates
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Username:   DefaultUsername,
		Password:   DefaultPassword,
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
		MaxRetries: 0, // Disable retries for faster test
	}

	rm := NewRollbackManager(client)

	// Save snapshot
	err := rm.SaveSnapshot("Before bad update")
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	snapshot := rm.GetLatestSnapshot()
	if snapshot == nil {
		t.Fatal("Expected snapshot to exist")
	}

	// Rollback to snapshot
	result := rm.RollbackToSnapshot(snapshot)

	if !result.Success {
		t.Errorf("Expected rollback to succeed, got error: %v", result.Error)
	}

	if result.Attempts < 1 {
		t.Errorf("Expected at least 1 verification attempt, got %d", result.Attempts)
	}
}

// TestRollbackManager_RollbackToLatest tests rolling back to the most recent snapshot
func TestRollbackManager_RollbackToLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true}`)
		} else if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Username:   DefaultUsername,
		Password:   DefaultPassword,
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
		MaxRetries: 0,
	}

	rm := NewRollbackManager(client)

	// No snapshots - should fail
	result := rm.RollbackToLatest()
	if result.Success {
		t.Error("Expected rollback to fail when no snapshots exist")
	}
	if result.Error == nil || !strings.Contains(result.Error.Error(), "no snapshots available") {
		t.Errorf("Expected 'no snapshots available' error, got: %v", result.Error)
	}

	// Save snapshot
	rm.SaveSnapshot("Snapshot 1")

	// Rollback to latest
	result = rm.RollbackToLatest()
	if !result.Success {
		t.Errorf("Expected rollback to succeed, got error: %v", result.Error)
	}
}

// TestRollbackManager_RollbackToNil tests that rolling back to nil snapshot fails gracefully
func TestRollbackManager_RollbackToNil(t *testing.T) {
	client := &Client{
		BaseURL:    "http://192.168.4.16",
		Username:   DefaultUsername,
		Password:   DefaultPassword,
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
	}

	rm := NewRollbackManager(client)

	result := rm.RollbackToSnapshot(nil)
	if result.Success {
		t.Error("Expected rollback to nil snapshot to fail")
	}
	if result.Error == nil || !strings.Contains(result.Error.Error(), "snapshot is nil") {
		t.Errorf("Expected 'snapshot is nil' error, got: %v", result.Error)
	}
}

// TestRollbackManager_SafeUpdate_Success tests SafeUpdate with successful update
func TestRollbackManager_SafeUpdate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Return updated config
			fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":7,"outlet2":3,"outlet3":1,"k3Outlet":false}`)
		} else if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Username:   DefaultUsername,
		Password:   DefaultPassword,
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
		MaxRetries: 0,
	}

	rm := NewRollbackManager(client)

	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  7,
			SecondPress: 3,
			ThirdPress:  1,
			K3Mode:      false,
		},
	}

	result := rm.SafeUpdate(update, nil, "Test update")

	if !result.Success {
		t.Errorf("Expected update to succeed, got error: %v", result.Error)
	}

	if result.RollbackAttempted {
		t.Error("Expected no rollback to be attempted on successful update")
	}

	if len(rm.GetSnapshots()) != 1 {
		t.Errorf("Expected 1 snapshot to be saved, got %d", len(rm.GetSnapshots()))
	}
}

// TestRollbackManager_SafeUpdate_FailureWithRollback tests SafeUpdate with failed update and successful rollback
func TestRollbackManager_SafeUpdate_FailureWithRollback(t *testing.T) {
	getCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if getCount == 1 {
				// Initial snapshot
				fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true}`)
			} else if getCount == 2 {
				// After failed update (config didn't change - verification will fail)
				fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true}`)
			} else {
				// After rollback
				fmt.Fprint(w, `{"serial":"315260240","dns":"lb.smartap-tech.com","port":80,"outlet1":1,"outlet2":2,"outlet3":4,"k3Outlet":true}`)
			}
		} else if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := &Client{
		BaseURL:    server.URL,
		Username:   DefaultUsername,
		Password:   DefaultPassword,
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
		MaxRetries: 0,
	}

	rm := NewRollbackManager(client)

	// Intentionally create an update that will fail verification
	// (requesting 7,3,1 but device will report 1,2,4)
	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  7,
			SecondPress: 3,
			ThirdPress:  1,
			K3Mode:      false,
		},
	}

	result := rm.SafeUpdate(update, nil, "Test failing update")

	if result.Success {
		t.Error("Expected update to fail")
	}

	if !result.RollbackAttempted {
		t.Error("Expected rollback to be attempted")
	}

	if !result.RollbackSucceeded {
		t.Errorf("Expected rollback to succeed, got error: %v", result.RollbackResult.Error)
	}

	if result.Error == nil {
		t.Error("Expected error to be set")
	}

	// Should contain both update failure and rollback success info
	errorMsg := result.Error.Error()
	if !strings.Contains(errorMsg, "successfully rolled back") {
		t.Errorf("Expected error message to mention rollback success, got: %s", errorMsg)
	}
}

// TestPromptBeforeDestructive tests the warning function for destructive changes
func TestPromptBeforeDestructive(t *testing.T) {
	tests := []struct {
		name           string
		current        *DeviceConfig
		update         *ConfigUpdate
		expectWarning  bool
		expectedSubstr string
	}{
		{
			name: "All buttons disabled",
			current: &DeviceConfig{
				Outlet1: 1,
				Outlet2: 2,
				Outlet3: 4,
			},
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  0,
					SecondPress: 0,
					ThirdPress:  0,
					K3Mode:      false,
				},
			},
			expectWarning:  true,
			expectedSubstr: "All diverter buttons will be disabled",
		},
		{
			name: "WiFi change warning",
			current: &DeviceConfig{
				Outlet1: 1,
				Outlet2: 2,
				Outlet3: 4,
			},
			update: &ConfigUpdate{
				WiFi: &WiFiConfig{
					SSID:         "NewNetwork",
					Password:     "password123",
					SecurityType: "WPA2",
				},
			},
			expectWarning:  true,
			expectedSubstr: "Changing WiFi configuration",
		},
		{
			name: "Open WiFi warning",
			current: &DeviceConfig{
				Outlet1: 1,
				Outlet2: 2,
				Outlet3: 4,
			},
			update: &ConfigUpdate{
				WiFi: &WiFiConfig{
					SSID:         "OpenNetwork",
					SecurityType: "OPEN",
				},
			},
			expectWarning:  true,
			expectedSubstr: "OPEN (no password) is a security risk",
		},
		{
			name: "Server change warning",
			current: &DeviceConfig{
				DNS:  "old.server.com",
				Port: 80,
			},
			update: &ConfigUpdate{
				Server: &ServerConfig{
					DNS:  "new.server.com",
					Port: 443,
				},
			},
			expectWarning:  true,
			expectedSubstr: "Changing server configuration",
		},
		{
			name: "Safe diverter update",
			current: &DeviceConfig{
				Outlet1: 1,
				Outlet2: 2,
				Outlet3: 4,
			},
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  1,
					SecondPress: 3,
					ThirdPress:  7,
					K3Mode:      true,
				},
			},
			expectWarning: false,
		},
		{
			name: "From all enabled to some disabled",
			current: &DeviceConfig{
				Outlet1: 7,
				Outlet2: 7,
				Outlet3: 7,
			},
			update: &ConfigUpdate{
				Diverter: &DiverterConfig{
					FirstPress:  1,
					SecondPress: 2,
					ThirdPress:  4,
					K3Mode:      false,
				},
			},
			expectWarning:  true,
			expectedSubstr: "disabling some outlets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning := PromptBeforeDestructive(tt.current, tt.update)

			if tt.expectWarning {
				if warning == "" {
					t.Error("Expected warning, got empty string")
				}
				if tt.expectedSubstr != "" && !strings.Contains(warning, tt.expectedSubstr) {
					t.Errorf("Expected warning to contain %q, got:\n%s", tt.expectedSubstr, warning)
				}
			} else {
				if warning != "" {
					t.Errorf("Expected no warning, got: %s", warning)
				}
			}
		})
	}
}

// TestSafeUpdateResult_String tests the String() method for SafeUpdateResult
func TestSafeUpdateResult_String(t *testing.T) {
	tests := []struct {
		name           string
		result         *SafeUpdateResult
		expectedSubstr string
	}{
		{
			name: "Successful update",
			result: &SafeUpdateResult{
				Success:     true,
				Description: "Test update",
				UpdateResult: &VerificationResult{
					Success:  true,
					Attempts: 2,
				},
			},
			expectedSubstr: "✅ Update succeeded",
		},
		{
			name: "Failed update with successful rollback",
			result: &SafeUpdateResult{
				Success:           false,
				Description:       "Test update",
				RollbackAttempted: true,
				RollbackSucceeded: true,
				UpdateResult: &VerificationResult{
					Success: false,
					Error:   fmt.Errorf("verification failed"),
				},
				RollbackResult: &VerificationResult{
					Success:  true,
					Attempts: 1,
				},
			},
			expectedSubstr: "successfully rolled back",
		},
		{
			name: "Failed update with failed rollback",
			result: &SafeUpdateResult{
				Success:           false,
				Description:       "Test update",
				RollbackAttempted: true,
				RollbackSucceeded: false,
				UpdateResult: &VerificationResult{
					Success: false,
					Error:   fmt.Errorf("verification failed"),
				},
				RollbackResult: &VerificationResult{
					Success: false,
					Error:   fmt.Errorf("rollback failed"),
				},
			},
			expectedSubstr: "rollback failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.result.String()
			if !strings.Contains(str, tt.expectedSubstr) {
				t.Errorf("Expected string to contain %q, got:\n%s", tt.expectedSubstr, str)
			}
		})
	}
}
