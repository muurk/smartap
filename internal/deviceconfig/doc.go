// Package deviceconfig provides an HTTP client for managing Smartap device configuration.
//
// This package implements a client for the Smartap device's local HTTP API, enabling
// reading and updating device configuration including diverter settings, server endpoints,
// and WiFi credentials. It includes automatic verification and rollback capabilities
// for safe configuration updates.
//
// # Configuration Categories
//
// The device configuration includes three main areas:
//   - Diverter: Button press mappings to water outlets (3-bit bitmask per button)
//   - Server: WebSocket server DNS name and port for device connectivity
//   - WiFi: Network SSID, password, and security type for device network access
//
// # Usage Example
//
//	// Create client for device at known IP
//	client := deviceconfig.NewClient("192.168.4.16", 80, "", "")
//
//	// Read current configuration
//	config, err := client.GetConfiguration()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Update diverter configuration with automatic verification
//	update := &deviceconfig.ConfigUpdate{
//	    Diverter: &deviceconfig.DiverterConfig{
//	        FirstPress:  1,  // Outlet 1 only
//	        SecondPress: 3,  // Outlets 1+2
//	        ThirdPress:  7,  // All outlets
//	        K3Mode:      true,
//	    },
//	}
//
//	result := client.UpdateAndVerify(update, nil)
//	if !result.Success {
//	    log.Fatalf("Update failed: %v", result.Error)
//	}
//
// # Safe Updates with Rollback
//
// The RollbackManager provides automatic configuration snapshots and rollback:
//
//	// Create rollback manager
//	rm := deviceconfig.NewRollbackManager(client)
//
//	// Perform safe update with automatic rollback on failure
//	result := rm.SafeUpdate(update, nil, "Update diverter settings")
//	if !result.Success {
//	    if result.RollbackSucceeded {
//	        log.Printf("Update failed but rolled back successfully")
//	    } else {
//	        log.Printf("Update AND rollback failed: %v", result.Error)
//	    }
//	}
//
// # Verification
//
// The UpdateAndVerify method automatically verifies configuration changes by:
//  1. Applying the configuration update via HTTP POST
//  2. Waiting for device to process changes
//  3. Reading back the configuration via HTTP GET
//  4. Comparing expected vs. actual values
//  5. Retrying up to 3 times if verification fails
//
// # Thread Safety
//
// Client instances are safe for concurrent use. The RollbackManager uses internal
// locking to protect snapshot operations.
//
// # Error Handling
//
// All errors are wrapped with context using %w for proper error chain handling.
// Network errors, timeout errors, and validation errors are clearly distinguished.
package deviceconfig
