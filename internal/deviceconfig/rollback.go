package deviceconfig

import (
	"fmt"
	"sync"
	"time"
)

// ConfigurationSnapshot represents a saved configuration state for rollback
type ConfigurationSnapshot struct {
	// Config is the saved device configuration
	Config *DeviceConfig

	// Timestamp when this snapshot was created
	Timestamp time.Time

	// Description of what operation this snapshot was taken before
	Description string
}

// RollbackManager manages configuration snapshots for rollback support
type RollbackManager struct {
	client *Client

	// snapshots stores configuration snapshots
	// Limited to last 10 snapshots to prevent unbounded growth
	snapshots []*ConfigurationSnapshot

	// maxSnapshots is the maximum number of snapshots to retain
	maxSnapshots int

	// mutex protects concurrent access to snapshots
	mutex sync.RWMutex
}

// NewRollbackManager creates a new rollback manager for a client
func NewRollbackManager(client *Client) *RollbackManager {
	return &RollbackManager{
		client:       client,
		snapshots:    make([]*ConfigurationSnapshot, 0, 10),
		maxSnapshots: 10,
	}
}

// SaveSnapshot captures the current device configuration as a snapshot
// This should be called before any configuration update
func (rm *RollbackManager) SaveSnapshot(description string) error {
	// Fetch current configuration
	config, err := rm.client.GetConfiguration()
	if err != nil {
		return fmt.Errorf("failed to fetch configuration for snapshot: %w", err)
	}

	// Create snapshot
	snapshot := &ConfigurationSnapshot{
		Config:      config,
		Timestamp:   time.Now(),
		Description: description,
	}

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	// Add snapshot to list
	rm.snapshots = append(rm.snapshots, snapshot)

	// Limit snapshot history
	if len(rm.snapshots) > rm.maxSnapshots {
		// Remove oldest snapshot
		rm.snapshots = rm.snapshots[1:]
	}

	return nil
}

// GetLatestSnapshot returns the most recent snapshot, or nil if no snapshots exist
func (rm *RollbackManager) GetLatestSnapshot() *ConfigurationSnapshot {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	if len(rm.snapshots) == 0 {
		return nil
	}

	return rm.snapshots[len(rm.snapshots)-1]
}

// GetSnapshots returns all snapshots in chronological order (oldest first)
func (rm *RollbackManager) GetSnapshots() []*ConfigurationSnapshot {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	// Return a copy to prevent external modification
	result := make([]*ConfigurationSnapshot, len(rm.snapshots))
	copy(result, rm.snapshots)
	return result
}

// ClearSnapshots removes all saved snapshots
func (rm *RollbackManager) ClearSnapshots() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.snapshots = make([]*ConfigurationSnapshot, 0, 10)
}

// RollbackToSnapshot restores the device configuration to a previous snapshot
func (rm *RollbackManager) RollbackToSnapshot(snapshot *ConfigurationSnapshot) *VerificationResult {
	if snapshot == nil {
		return &VerificationResult{
			Success: false,
			Error:   fmt.Errorf("snapshot is nil"),
		}
	}

	// Build configuration update from snapshot
	update := &ConfigUpdate{
		Diverter: &DiverterConfig{
			FirstPress:  snapshot.Config.Outlet1,
			SecondPress: snapshot.Config.Outlet2,
			ThirdPress:  snapshot.Config.Outlet3,
			K3Mode:      snapshot.Config.K3Outlet,
		},
		Server: &ServerConfig{
			DNS:  snapshot.Config.DNS,
			Port: snapshot.Config.Port,
		},
		// Note: WiFi cannot be rolled back as passwords aren't returned by device
	}

	// Apply configuration with verification
	return rm.client.UpdateAndVerify(update, nil)
}

// RollbackToLatest restores the device configuration to the most recent snapshot
// Returns error if no snapshots exist
func (rm *RollbackManager) RollbackToLatest() *VerificationResult {
	snapshot := rm.GetLatestSnapshot()
	if snapshot == nil {
		return &VerificationResult{
			Success: false,
			Error:   fmt.Errorf("no snapshots available for rollback"),
		}
	}

	return rm.RollbackToSnapshot(snapshot)
}

// SafeUpdate performs a configuration update with automatic rollback on failure
// If verification fails, automatically attempts to roll back to the previous state
func (rm *RollbackManager) SafeUpdate(update *ConfigUpdate, opts *VerificationOptions, description string) *SafeUpdateResult {
	result := &SafeUpdateResult{
		Description: description,
	}

	// Save snapshot before update
	if err := rm.SaveSnapshot(description); err != nil {
		result.Error = fmt.Errorf("failed to save pre-update snapshot: %w", err)
		return result
	}

	// Attempt update with verification
	verifyResult := rm.client.UpdateAndVerify(update, opts)
	result.UpdateResult = verifyResult

	if verifyResult.Success {
		result.Success = true
		return result
	}

	// Update failed - attempt rollback
	result.RollbackAttempted = true
	snapshot := rm.GetLatestSnapshot()

	if snapshot == nil {
		result.Error = fmt.Errorf("update failed and no snapshot available for rollback: %w", verifyResult.Error)
		return result
	}

	rollbackResult := rm.RollbackToSnapshot(snapshot)
	result.RollbackResult = rollbackResult

	if rollbackResult.Success {
		result.RollbackSucceeded = true
		result.Error = fmt.Errorf("update failed (verification: %w), successfully rolled back to previous configuration", verifyResult.Error)
	} else {
		result.Error = fmt.Errorf("update failed (verification: %w) AND rollback failed: %w", verifyResult.Error, rollbackResult.Error)
	}

	return result
}

// SafeUpdateResult contains the results of a safe update operation
type SafeUpdateResult struct {
	// Success indicates whether the update succeeded
	Success bool

	// Description of the update operation
	Description string

	// UpdateResult contains the result of the update attempt
	UpdateResult *VerificationResult

	// RollbackAttempted indicates whether rollback was attempted
	RollbackAttempted bool

	// RollbackSucceeded indicates whether rollback succeeded (only valid if RollbackAttempted is true)
	RollbackSucceeded bool

	// RollbackResult contains the result of the rollback attempt (only valid if RollbackAttempted is true)
	RollbackResult *VerificationResult

	// Error contains any error that occurred
	Error error
}

// String returns a human-readable summary of the safe update result
func (r *SafeUpdateResult) String() string {
	if r.Success {
		return fmt.Sprintf("✅ Update succeeded: %s (verified in %d attempt(s))",
			r.Description, r.UpdateResult.Attempts)
	}

	if r.RollbackAttempted {
		if r.RollbackSucceeded {
			return fmt.Sprintf("⚠️  Update failed but successfully rolled back: %s\nUpdate error: %v\nRollback: successful after %d attempt(s)",
				r.Description, r.UpdateResult.Error, r.RollbackResult.Attempts)
		}
		return fmt.Sprintf("❌ Update failed and rollback failed: %s\nUpdate error: %v\nRollback error: %v",
			r.Description, r.UpdateResult.Error, r.RollbackResult.Error)
	}

	return fmt.Sprintf("❌ Update failed: %s\nError: %v",
		r.Description, r.Error)
}

// PromptBeforeDestructive checks if an update is potentially destructive and returns a warning message
// Returns empty string if the update is safe
func PromptBeforeDestructive(current *DeviceConfig, update *ConfigUpdate) string {
	warnings := []string{}

	// Check for diverter changes that disable all outlets
	if update.Diverter != nil {
		// Check if all three button presses are set to 0 (no outlets)
		if update.Diverter.FirstPress == 0 && update.Diverter.SecondPress == 0 && update.Diverter.ThirdPress == 0 {
			warnings = append(warnings, "⚠️  All diverter buttons will be disabled (no outlets enabled)")
		}

		// Check if changing from all outlets enabled to some disabled
		if current != nil {
			currentAllEnabled := current.Outlet1 == 7 && current.Outlet2 == 7 && current.Outlet3 == 7
			newSomeDisabled := (update.Diverter.FirstPress < 7) || (update.Diverter.SecondPress < 7) || (update.Diverter.ThirdPress < 7)

			if currentAllEnabled && newSomeDisabled {
				warnings = append(warnings, "⚠️  You are disabling some outlets that are currently all enabled")
			}
		}
	}

	// Check for WiFi changes (potentially destructive)
	if update.WiFi != nil {
		warnings = append(warnings, "⚠️  Changing WiFi configuration may disconnect the device from the network")

		// Additional warning for open WiFi
		if update.WiFi.SecurityType == "OPEN" {
			warnings = append(warnings, "⚠️  WARNING: Setting WiFi to OPEN (no password) is a security risk")
		}
	}

	// Check for server changes
	if update.Server != nil {
		if current != nil && (update.Server.DNS != current.DNS || update.Server.Port != current.Port) {
			warnings = append(warnings, "⚠️  Changing server configuration may affect device connectivity")
		}
	}

	if len(warnings) == 0 {
		return ""
	}

	// Build warning message
	msg := "⚠️  POTENTIALLY DESTRUCTIVE CHANGES DETECTED ⚠️\n\n"
	for _, w := range warnings {
		msg += w + "\n"
	}
	msg += "\nIt is recommended to save a snapshot before proceeding.\n"
	msg += "You can rollback to the previous configuration if something goes wrong.\n"

	return msg
}
