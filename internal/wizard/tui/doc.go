// Package tui implements the terminal user interface for the Smartap configuration wizard.
//
// This package provides an interactive, full-screen TUI for discovering and configuring
// Smartap IoT devices. Built using the Bubble Tea framework, it follows the Elm architecture
// with immutable state updates and a clean Model-Update-View pattern.
//
// # Architecture
//
// The TUI is organized into three main screens:
//   - Discovery: Scan network for devices or enter IP manually
//   - Dashboard: View and edit device configuration
//   - Success/Failure: Display operation results
//
// All screens use a unified container pattern (RenderApplicationContainer) for consistent
// layout with header, content area, scrolling viewport, and context-sensitive footer.
//
// # Framework Components
//
// The TUI leverages Bubble Tea framework components throughout:
//   - bubbles/spinner: Loading indicators
//   - bubbles/textinput: Text entry fields with validation
//   - bubbles/progress: Progress bars for operations
//   - bubbles/list: Device lists with filtering
//   - bubbles/help: Context-aware help system
//   - bubbles/viewport: Scrolling for large content
//   - lipgloss: Styling and layout
//
// # Usage Example
//
//	// Create and run the wizard
//	app := tui.NewAppModel()
//	program := tea.NewProgram(app, tea.WithAltScreen())
//
//	if _, err := program.Run(); err != nil {
//	    log.Fatal(err)
//	}
//
// # Screen Flow
//
// The typical user flow through the wizard:
//
//  1. Discovery Screen:
//     - Automatically scans network for devices (mDNS)
//     - Displays found devices as cards with details
//     - Allows manual IP entry if device not found
//     - User selects device to configure
//
//  2. Dashboard Screen:
//     - Fetches current device configuration
//     - Displays configuration in three sections:
//       * Outlets: First/Second/Third press assignments + K3 mode
//       * WiFi: SSID selection + password entry
//       * Server: DNS hostname + port
//     - Inline editing - fields expand in place (no modal overlays)
//     - Per-section Apply buttons for independent updates
//     - Apply with automatic verification
//     - Clear modified indicators per section
//
//  3. Success/Failure Screen:
//     - Shows operation result
//     - Displays updated configuration on success
//     - Shows error details on failure
//     - Options to retry, edit, or exit
//
// # Inline Editing System
//
// The dashboard uses inline editing for all configuration fields:
//   - Press Enter on a field to expand it inline
//   - Field shows all editing options in place (full context visible)
//   - Arrow keys navigate options within expanded editor
//   - Enter confirms changes, ESC cancels
//   - Text inputs use bubbles/textinput with validation
//   - Changes tracked per-section with visual indicators
//
// # Key Bindings
//
// Each screen has context-aware key bindings:
//   - Discovery: ↑/↓ navigate, Enter select, r rescan, m manual IP, q quit
//   - Dashboard (Normal Mode): ↑/↓ navigate fields, Tab jump sections, Enter edit/apply, q quit
//   - Dashboard (Editing): ↑/↓ navigate options, Enter confirm, ESC cancel
//   - Success/Failure: Enter/v view, e edit again, d discover, q quit
//
// Help text automatically updates based on screen state (e.g., during scanning, manual entry).
//
// # Styling
//
// All styling uses lipgloss for consistency:
//   - Color palette: Cyan highlights, yellow warnings, green success, red errors
//   - Borders: Rounded borders for containers and modals
//   - Spacing: Consistent padding and margins
//   - Layout: Flexbox-style alignment and centering
//
// # State Management
//
// The TUI maintains immutable state with explicit updates:
//   - Models contain all state (no global variables)
//   - Update() returns new model + commands
//   - View() is pure function of model state
//   - Commands represent async operations
//
// # Error Handling
//
// Errors are handled gracefully with user-friendly messages:
//   - Network errors: "Could not connect to device"
//   - Validation errors: "Port must be between 1-65535"
//   - Device errors: "Device rejected configuration"
//   - Rollback: Automatic on verification failure
//
// # Accessibility
//
// The TUI is designed for accessibility:
//   - Keyboard-only navigation (no mouse required)
//   - Clear visual hierarchy
//   - High contrast colors
//   - Descriptive help text
//   - Screen reader friendly (via text content)
//
// # Thread Safety
//
// The Bubble Tea framework ensures thread safety through message passing.
// All model updates occur in a single goroutine, preventing race conditions.
package tui
