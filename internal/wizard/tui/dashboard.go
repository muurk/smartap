package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/muurk/smartap/internal/deviceconfig"
)

// Message types for async operations
type applyCompleteMsg struct {
	config               *deviceconfig.DeviceConfig
	err                  error
	verificationDuration time.Duration
}

// ConfigSection represents which configuration section is active for inline editing
type ConfigSection int

const (
	SectionNone ConfigSection = iota
	SectionOutlets
	SectionWiFi
	SectionServer
)

// OutletEditorState tracks outlet selection editor state
type OutletEditorState struct {
	Cursor int // 0-7 for bitmask values
}

// WiFiEditorState tracks WiFi configuration editor state
type WiFiEditorState struct {
	Cursor        int             // Position in SSID list or password field
	PasswordInput textinput.Model // Password input field
}

// ServerEditorState tracks server configuration editor state
type ServerEditorState struct {
	EditingField FieldType       // DNS or Port
	DNSInput     textinput.Model // DNS input field
	PortInput    textinput.Model // Port input field
}

// FieldType represents which field is being edited in server config
type FieldType int

const (
	FieldDNS FieldType = iota
	FieldPort
)

// dashboardKeyMap defines key bindings for the dashboard screen
type dashboardKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Tab   key.Binding
	Enter key.Binding
	Back  key.Binding
	Quit  key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k dashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Enter, k.Back, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Tab, k.Enter},
		{k.Back, k.Quit},
	}
}

// DashboardModel represents the unified dashboard screen combining view + edit
type DashboardModel struct {
	// Device connection
	DeviceIP   string
	DevicePort int
	Client     *deviceconfig.Client

	// Configuration state
	CurrentConfig *deviceconfig.DeviceConfig // Original config from device
	PendingConfig *PendingDeviceConfig       // User's in-progress edits

	// UI state
	Width  int
	Height int

	// Navigation
	Cursor          int  // Which field is focused (0-6: outlets 0-3, wifi 4, dns 5, port 6)
	ShowingProgress bool // Progress modal visible (applying changes)
	ShowingSuccess  bool // Success modal visible
	ShowingFailure  bool // Failure modal visible
	ShowingHelp     bool // Help modal visible

	// Inline editing state (replaces modal states)
	EditingSection ConfigSection // Which section is being edited (Outlets, WiFi, Server, None)
	EditingField   int           // Which field within section (0-3 for outlets, 0 for WiFi, 0-1 for server)

	// Section-specific editors
	OutletEditor OutletEditorState
	WiFiEditor   WiFiEditorState
	ServerEditor ServerEditorState

	// WiFi configuration states (kept for warning modal and apply logic)
	ShowingWiFiWarning bool   // WiFi change warning modal visible
	WiFiSelectedSSID   string // Selected SSID from list
	WiFiSecurityType   string // Security type (WPA2 or OPEN)
	WiFiChangeApplied  bool   // Set to true when WiFi config was successfully applied

	// Per-section applying state (replaces global apply/preview/confirm)
	OutletsApplying bool
	WiFiApplying    bool
	ServerApplying  bool

	// Modal state (for result modals only)
	ModalCursor    int                        // For confirm/result modal buttons
	Spinner        spinner.Model              // Progress spinner for applying changes
	ProgressBar    progress.Model             // Progress bar component for applying changes
	ApplyStartTime time.Time                  // When apply started
	ApplyAttempt   int                        // Current retry attempt (1-based)
	MaxAttempts    int                        // Maximum retry attempts
	ApplyError     error                      // Error from apply operation
	VerifiedConfig *deviceconfig.DeviceConfig // Verified config after successful apply

	// Change tracking
	HasUnsavedChanges    bool
	LastSaved            time.Time
	VerificationDuration time.Duration // How long verification took (for success modal)
	SaveMessage          string        // e.g., "✓ Saved 2 seconds ago"

	// Navigation results
	BackRequested bool

	// Help
	Help help.Model
	Keys dashboardKeyMap
}

// PendingDeviceConfig tracks user's in-progress edits
type PendingDeviceConfig struct {
	// Diverter configuration (indices 0-3)
	Outlet1  int  // First press bitmask (0-7)
	Outlet2  int  // Second press bitmask (0-7)
	Outlet3  int  // Third press bitmask (0-7)
	K3Outlet bool // Third knob separation mode

	// Network configuration (indices 4-6)
	WiFiSSID string // WiFi network SSID (index 4)
	DNS      string // Server DNS hostname (index 5)
	Port     int    // Server port (index 6)
}

// NewDashboardModel creates a new dashboard with device config
func NewDashboardModel(ip string, port int, config *deviceconfig.DeviceConfig) DashboardModel {
	client := deviceconfig.NewClient(ip, port)

	// Initialize pending config from current config
	pending := &PendingDeviceConfig{
		// Diverter configuration
		Outlet1:  config.Outlet1,
		Outlet2:  config.Outlet2,
		Outlet3:  config.Outlet3,
		K3Outlet: config.K3Outlet,

		// Network configuration
		WiFiSSID: getFirstSSID(config.SSIDList),
		DNS:      config.DNS,
		Port:     config.Port,
	}

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	// Initialize progress bar
	progressBar := progress.New(progress.WithDefaultGradient())
	progressBar.Width = 40

	// Initialize text inputs
	dnsInput := textinput.New()
	dnsInput.Placeholder = "server.example.com"
	dnsInput.CharLimit = 253
	dnsInput.Width = 50

	portInput := textinput.New()
	portInput.Placeholder = "443"
	portInput.CharLimit = 5
	portInput.Width = 50

	wifiPasswordInput := textinput.New()
	wifiPasswordInput.Placeholder = "Enter password"
	wifiPasswordInput.EchoMode = textinput.EchoPassword
	wifiPasswordInput.EchoCharacter = '•'
	wifiPasswordInput.CharLimit = 63
	wifiPasswordInput.Width = 50

	// Initialize help
	h := help.New()

	// Initialize key bindings
	keys := dashboardKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}

	return DashboardModel{
		DeviceIP:          ip,
		DevicePort:        port,
		Client:            client,
		CurrentConfig:     config,
		PendingConfig:     pending,
		Cursor:            0,
		ModalCursor:       0,
		Spinner:           s,
		ProgressBar:       progressBar,
		HasUnsavedChanges: false,

		// Initialize inline editing state
		EditingSection: SectionNone,
		EditingField:   -1,

		// Initialize section editors
		OutletEditor: OutletEditorState{
			Cursor: 0,
		},
		WiFiEditor: WiFiEditorState{
			Cursor:        0,
			PasswordInput: wifiPasswordInput,
		},
		ServerEditor: ServerEditorState{
			EditingField: FieldDNS,
			DNSInput:     dnsInput,
			PortInput:    portInput,
		},
		BackRequested: false,
		Help:          h,
		Keys:          keys,
	}
}

// Init initializes the dashboard
func (m DashboardModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle result modals first (progress/success/failure/help)
	if m.ShowingHelp {
		return m.updateHelpModal(msg)
	} else if m.ShowingProgress {
		return m.updateProgressModal(msg)
	} else if m.ShowingSuccess {
		return m.updateSuccessModal(msg)
	} else if m.ShowingFailure {
		return m.updateFailureModal(msg)
	} else if m.ShowingWiFiWarning {
		// Keep WiFi warning modal (important safety check)
		return m.updateWiFiWarningModal(msg)
	}

	// Route to inline editor update functions based on editing state
	switch m.EditingSection {
	case SectionOutlets:
		return m.updateOutletEditor(msg)
	case SectionWiFi:
		return m.updateWiFiEditor(msg)
	case SectionServer:
		return m.updateServerEditor(msg)
	case SectionNone:
		// Normal navigation mode
		return m.updateNormalMode(msg)
	}

	// Fallback (should never reach here)
	return m.updateNormalMode(msg)
}

// updateNormalMode handles input when in normal navigation mode (no field being edited)
func (m DashboardModel) updateNormalMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Force quit immediately
			return m, tea.Quit

		case "q":
			// Quit should exit application, not go back
			// BackRequested should NOT be set - just quit
			return m, tea.Quit

		case "esc":
			// ESC no longer goes back to discovery (Task 1.3)
			// Will be used to cancel inline editing in Phase 3
			return m, nil

		case "up", "k":
			m.Cursor--
			if m.Cursor < 0 {
				m.Cursor = 9 // Wrap to bottom (Apply Server button)
			}

		case "down", "j":
			m.Cursor++
			if m.Cursor > 9 {
				m.Cursor = 0 // Wrap to top (First Press field)
			}

		case "tab":
			// Jump between section starts (0→5→7→0)
			// Navigation order: 0-4 Outlets, 5-6 WiFi, 7-9 Server
			if m.Cursor <= 4 {
				// Jump from Outlets to WiFi start
				m.Cursor = 5
			} else if m.Cursor <= 6 {
				// Jump from WiFi to Server start
				m.Cursor = 7
			} else {
				// Jump from Server back to Outlets start
				m.Cursor = 0
			}

		case "enter", " ":
			// Enter editing mode for focused field or trigger apply button
			return m.startEditing()

		case "?":
			// Show help modal
			m.ShowingHelp = true
		}
	}

	// Update change tracking
	m.updateChangeTracking()

	return m, nil
}

// startEditing determines which section/field to edit based on cursor position
// and transitions to the appropriate editing mode
//
// Navigation order (field → apply per section):
//   0-3: Outlet fields (First Press, Second Press, Third Press, K3 Mode)
//   4:   Apply Outlets button
//   5:   WiFi Network field
//   6:   Apply WiFi button
//   7:   Server DNS field
//   8:   Server Port field
//   9:   Apply Server button
func (m DashboardModel) startEditing() (tea.Model, tea.Cmd) {
	switch m.Cursor {
	case 0, 1, 2, 3:
		// Outlets section (fields 0-3: First Press, Second Press, Third Press, K3 Mode)
		m.EditingSection = SectionOutlets
		m.EditingField = m.Cursor
		m.OutletEditor.Cursor = m.getPendingValue(m.Cursor)
		return m, nil

	case 4:
		// Apply Outlets button
		return m.applyOutlets()

	case 5:
		// WiFi section (field 5: WiFi Network)
		m.EditingSection = SectionWiFi
		m.EditingField = 0
		m.WiFiEditor.Cursor = 0
		// Reset password input
		m.WiFiEditor.PasswordInput.SetValue("")
		return m, nil

	case 6:
		// Apply WiFi button
		return m.applyWiFi()

	case 7:
		// Server DNS field
		m.EditingSection = SectionServer
		m.EditingField = 0 // DNS
		m.ServerEditor.DNSInput.Focus()
		m.ServerEditor.DNSInput.SetValue(m.PendingConfig.DNS)
		m.ServerEditor.PortInput.Blur()
		return m, nil

	case 8:
		// Server Port field
		m.EditingSection = SectionServer
		m.EditingField = 1 // Port
		m.ServerEditor.PortInput.Focus()
		m.ServerEditor.PortInput.SetValue(fmt.Sprintf("%d", m.PendingConfig.Port))
		m.ServerEditor.DNSInput.Blur()
		return m, nil

	case 9:
		// Apply Server button
		return m.applyServer()
	}

	// Should not reach here, but return unchanged model if cursor is out of bounds
	return m, nil
}

// updateOutletEditor handles input when editing outlet fields inline
func (m DashboardModel) updateOutletEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel editing - return to normal mode without saving
			m.EditingSection = SectionNone
			m.EditingField = -1
			return m, nil

		case "up", "k":
			// Navigate up through options
			m.OutletEditor.Cursor--
			if m.OutletEditor.Cursor < 0 {
				if m.EditingField == 3 {
					// K3 mode has only 2 options (0-1)
					m.OutletEditor.Cursor = 1
				} else {
					// Outlet fields have 8 options (0-7)
					m.OutletEditor.Cursor = 7
				}
			}
			return m, nil

		case "down", "j":
			// Navigate down through options
			m.OutletEditor.Cursor++
			maxCursor := 7
			if m.EditingField == 3 {
				// K3 mode has only 2 options (0-1)
				maxCursor = 1
			}
			if m.OutletEditor.Cursor > maxCursor {
				m.OutletEditor.Cursor = 0
			}
			return m, nil

		case "0", "1", "2", "3", "4", "5", "6", "7":
			// Direct number input
			val, _ := strconv.Atoi(msg.String())

			// Validate range based on field type
			if m.EditingField == 3 {
				// K3 mode: only 0-1 valid
				if val > 1 {
					return m, nil // Ignore invalid input
				}
			}

			m.OutletEditor.Cursor = val
			// Fall through to apply selection
			fallthrough

		case "enter", " ":
			// Apply selection and exit editing mode
			switch m.EditingField {
			case 0:
				m.PendingConfig.Outlet1 = m.OutletEditor.Cursor
			case 1:
				m.PendingConfig.Outlet2 = m.OutletEditor.Cursor
			case 2:
				m.PendingConfig.Outlet3 = m.OutletEditor.Cursor
			case 3:
				// K3 mode toggle (0=Disabled, 1=Enabled)
				m.PendingConfig.K3Outlet = (m.OutletEditor.Cursor == 1)
			}

			// Update change tracking
			m.updateChangeTracking()

			// Exit editing mode
			m.EditingSection = SectionNone
			m.EditingField = -1

			return m, nil
		}
	}

	return m, nil
}

// updateWiFiEditor handles input when editing WiFi configuration inline
func (m DashboardModel) updateWiFiEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel editing - return to normal mode without saving
			m.EditingSection = SectionNone
			m.EditingField = -1
			m.WiFiEditor.PasswordInput.Blur()
			m.WiFiEditor.PasswordInput.SetValue("")
			return m, nil

		case "up", "k":
			if m.WiFiEditor.PasswordInput.Focused() {
				// Move focus from password to SSID list
				m.WiFiEditor.PasswordInput.Blur()
				m.WiFiEditor.Cursor = len(m.CurrentConfig.SSIDList) - 1
			} else {
				// Navigate SSID list upward
				m.WiFiEditor.Cursor--
				if m.WiFiEditor.Cursor < 0 {
					// Wrap to password field
					m.WiFiEditor.PasswordInput.Focus()
					return m, textinput.Blink
				}
			}
			return m, nil

		case "down", "j":
			if m.WiFiEditor.PasswordInput.Focused() {
				// Stay in password field (bottom of list)
				return m, nil
			} else {
				// Navigate SSID list downward
				m.WiFiEditor.Cursor++
				if m.WiFiEditor.Cursor >= len(m.CurrentConfig.SSIDList) {
					// Move to password field
					m.WiFiEditor.PasswordInput.Focus()
					return m, textinput.Blink
				}
			}
			return m, nil

		case "enter":
			// Confirm WiFi config
			selectedSSID := m.getCurrentSSID()
			password := m.WiFiEditor.PasswordInput.Value()

			// Check if anything changed
			if selectedSSID == m.PendingConfig.WiFiSSID && password == "" {
				// No change - just exit editing
				m.EditingSection = SectionNone
				m.EditingField = -1
				m.WiFiEditor.PasswordInput.Blur()
				m.WiFiEditor.PasswordInput.SetValue("")
				return m, nil
			}

			// Apply changes to pending config
			m.PendingConfig.WiFiSSID = selectedSSID

			// Store password in WiFiEditor for later use during apply
			// (it will be retrieved when applyWiFi is called)

			// Update change tracking
			m.updateChangeTracking()

			// Exit editing mode
			m.EditingSection = SectionNone
			m.EditingField = -1
			m.WiFiEditor.PasswordInput.Blur()

			return m, nil
		}
	}

	// Update password input if focused
	if m.WiFiEditor.PasswordInput.Focused() {
		var cmd tea.Cmd
		m.WiFiEditor.PasswordInput, cmd = m.WiFiEditor.PasswordInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// getCurrentSSID returns the currently selected SSID based on WiFiEditor.Cursor
func (m DashboardModel) getCurrentSSID() string {
	if m.WiFiEditor.Cursor >= 0 && m.WiFiEditor.Cursor < len(m.CurrentConfig.SSIDList) {
		return m.CurrentConfig.SSIDList[m.WiFiEditor.Cursor]
	}
	return m.PendingConfig.WiFiSSID
}

// updateServerEditor handles input when editing server configuration inline
func (m DashboardModel) updateServerEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel editing - return to normal mode without saving
			m.EditingSection = SectionNone
			m.EditingField = -1
			m.ServerEditor.DNSInput.Blur()
			m.ServerEditor.PortInput.Blur()
			return m, nil

		case "enter":
			// Apply changes
			if m.EditingField == 0 {
				// DNS field
				value := m.ServerEditor.DNSInput.Value()
				if value != "" {
					m.PendingConfig.DNS = value
				}
			} else {
				// Port field
				value := m.ServerEditor.PortInput.Value()
				if port, err := strconv.Atoi(value); err == nil && port > 0 && port < 65536 {
					m.PendingConfig.Port = port
				}
			}

			// Update change tracking
			m.updateChangeTracking()

			// Exit editing mode
			m.EditingSection = SectionNone
			m.EditingField = -1
			m.ServerEditor.DNSInput.Blur()
			m.ServerEditor.PortInput.Blur()

			return m, nil
		}
	}

	// Update active input (pass keyboard events to textinput)
	var cmd tea.Cmd
	if m.EditingField == 0 {
		m.ServerEditor.DNSInput, cmd = m.ServerEditor.DNSInput.Update(msg)
	} else {
		m.ServerEditor.PortInput, cmd = m.ServerEditor.PortInput.Update(msg)
	}

	return m, cmd
}

// applyOutlets applies outlet configuration changes to the device
func (m DashboardModel) applyOutlets() (tea.Model, tea.Cmd) {
	// Check if there are changes
	if !m.hasOutletChanges() {
		// No changes to apply
		return m, nil
	}

	// Set showing progress
	m.ShowingProgress = true

	// Create config update with ONLY outlet changes
	update := &deviceconfig.ConfigUpdate{
		Diverter: &deviceconfig.DiverterConfig{
			FirstPress:  m.PendingConfig.Outlet1,
			SecondPress: m.PendingConfig.Outlet2,
			ThirdPress:  m.PendingConfig.Outlet3,
			K3Mode:      m.PendingConfig.K3Outlet,
		},
	}

	return m, applyConfigCmd(m.Client, update)
}

// applyWiFi applies WiFi configuration changes to the device
func (m DashboardModel) applyWiFi() (tea.Model, tea.Cmd) {
	// Check if there are changes
	password := m.WiFiEditor.PasswordInput.Value()
	if !m.hasWiFiChanges() && password == "" {
		// No changes to apply
		return m, nil
	}

	// Set WiFi fields for the warning modal
	m.WiFiSelectedSSID = m.PendingConfig.WiFiSSID
	m.WiFiSecurityType = "WPA2" // Assume WPA2 for now

	// Show warning modal
	m.ShowingWiFiWarning = true
	m.ModalCursor = 0 // Default to "Apply" button

	return m, nil
}

// applyServer applies server configuration changes to the device
func (m DashboardModel) applyServer() (tea.Model, tea.Cmd) {
	// Check if there are changes
	if !m.hasServerChanges() {
		// No changes to apply
		return m, nil
	}

	// Set showing progress
	m.ShowingProgress = true

	// Create config update with ONLY server changes
	update := &deviceconfig.ConfigUpdate{
		Server: &deviceconfig.ServerConfig{
			DNS:  m.PendingConfig.DNS,
			Port: m.PendingConfig.Port,
		},
	}

	return m, applyConfigCmd(m.Client, update)
}

// applyConfigCmd applies a configuration update to the device and verifies it
func applyConfigCmd(client *deviceconfig.Client, update *deviceconfig.ConfigUpdate) tea.Cmd {
	return func() tea.Msg {
		startTime := time.Now()

		// Apply the configuration
		err := client.UpdateConfiguration(update)
		if err != nil {
			return applyCompleteMsg{
				config: nil,
				err:    fmt.Errorf("configuration update failed: %w", err),
			}
		}

		// Wait a moment for device to apply changes
		time.Sleep(500 * time.Millisecond)

		// Verify by reading back the configuration
		config, verifyErr := client.GetConfiguration()
		if verifyErr != nil {
			return applyCompleteMsg{
				config: nil,
				err:    fmt.Errorf("configuration verification failed: %w", verifyErr),
			}
		}

		// Calculate total duration
		duration := time.Since(startTime)

		return applyCompleteMsg{
			config:               config,
			err:                  nil,
			verificationDuration: duration,
		}
	}
}

// updateWiFiWarningModal handles input when WiFi warning modal is showing
// This modal is kept because it's an important safety check before changing WiFi
func (m DashboardModel) updateWiFiWarningModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Cancel - close warning
			m.ShowingWiFiWarning = false
			return m, nil

		case "left", "h":
			// Navigate to Apply button
			if m.ModalCursor > 0 {
				m.ModalCursor--
			}

		case "right", "l":
			// Navigate to Cancel button
			if m.ModalCursor < 1 {
				m.ModalCursor++
			}

		case "enter", " ":
			if m.ModalCursor == 0 {
				// Apply button - apply WiFi configuration
				wifiConfig := &deviceconfig.WiFiConfig{
					SSID:         m.WiFiSelectedSSID,
					Password:     m.WiFiEditor.PasswordInput.Value(),
					SecurityType: m.WiFiSecurityType,
				}

				// Apply WiFi configuration
				err := m.Client.UpdateWiFi(wifiConfig)
				if err != nil {
					// Show error - don't exit, let user try again
					m.ShowingWiFiWarning = false
					m.ShowingFailure = true
					m.ApplyError = err
					return m, nil
				}

				// Success! Update pending config and show success message
				// Note: We don't verify the config because WiFi will disconnect
				m.PendingConfig.WiFiSSID = m.WiFiSelectedSSID
				m.updateChangeTracking()
				m.ShowingWiFiWarning = false
				m.WiFiEditor.PasswordInput.SetValue("")

				// Show WiFi-specific success message (will exit on keypress)
				m.ShowingSuccess = true
				m.WiFiChangeApplied = true // Mark this as a WiFi change for custom success message
				m.ApplyError = nil         // No error
				return m, nil
			} else {
				// Cancel button - close warning
				m.ShowingWiFiWarning = false
			}
		}
	}

	return m, nil
}

// updateChangeTracking updates the HasUnsavedChanges flag
func (m *DashboardModel) updateChangeTracking() {
	// Check for changes in diverter configuration
	diverterChanged := (m.PendingConfig.Outlet1 != m.CurrentConfig.Outlet1 ||
		m.PendingConfig.Outlet2 != m.CurrentConfig.Outlet2 ||
		m.PendingConfig.Outlet3 != m.CurrentConfig.Outlet3 ||
		m.PendingConfig.K3Outlet != m.CurrentConfig.K3Outlet)

	// Check for changes in network configuration
	networkChanged := (m.PendingConfig.WiFiSSID != getFirstSSID(m.CurrentConfig.SSIDList) ||
		m.PendingConfig.DNS != m.CurrentConfig.DNS ||
		m.PendingConfig.Port != m.CurrentConfig.Port)

	m.HasUnsavedChanges = diverterChanged || networkChanged

	// Update save message
	if !m.LastSaved.IsZero() {
		elapsed := time.Since(m.LastSaved)
		if elapsed < 60*time.Second {
			m.SaveMessage = fmt.Sprintf("✓ Saved %d seconds ago", int(elapsed.Seconds()))
		} else {
			m.SaveMessage = ""
		}
	}
}

// updateSaveStatus updates the save status message for display
func (m *DashboardModel) updateSaveStatus() {
	if !m.LastSaved.IsZero() {
		elapsed := time.Since(m.LastSaved)
		if elapsed < 60*time.Second {
			m.SaveMessage = fmt.Sprintf("✓ Saved %d seconds ago", int(elapsed.Seconds()))
		} else {
			m.SaveMessage = ""
		}
	}
}

// renderDashboard renders the main dashboard view using RenderApplicationContainer
func (m DashboardModel) renderDashboard() string {
	// Update save status message (for "✓ Saved X seconds ago" display)
	m.updateSaveStatus()

	// Build the dashboard content
	content := m.renderDashboardContent()

	// Help text using bubbles/help
	helpText := m.Help.View(m.Keys)

	// Wrap with application container (full-screen layout with height)
	return RenderApplicationContainer(content, helpText, m.Width, m.Height)
}

// renderDashboardContent renders the main dashboard content (without container)
// Uses lipgloss.JoinVertical for layout, no width parameter needed
func (m DashboardModel) renderDashboardContent() string {
	// Device info line
	deviceInfo := fmt.Sprintf("Device: %s • %s:%d • FW: %s",
		m.getCurrentDeviceName(),
		m.DeviceIP,
		m.DevicePort,
		m.CurrentConfig.SWVer)

	deviceStyle := lipgloss.NewStyle().Foreground(TextColor)
	deviceLine := deviceStyle.Render(deviceInfo)

	// Status indicator (on separate line if present)
	var statusLine string
	if m.HasUnsavedChanges {
		statusStyle := lipgloss.NewStyle().Foreground(WarningColor).Bold(true)
		statusLine = statusStyle.Render("⚠ MODIFIED")
	} else if m.SaveMessage != "" {
		statusStyle := lipgloss.NewStyle().Foreground(SecondaryColor)
		statusLine = statusStyle.Render(m.SaveMessage)
	}

	// Divider - simple horizontal line
	divider := lipgloss.NewStyle().
		Foreground(BorderColor).
		Render(strings.Repeat("─", 60))

	// Sections - simple renders, no width parameter
	outletsSection := m.renderOutletsSection()
	wifiSection := m.renderWiFiSection()
	serverSection := m.renderServerSection()

	// Compose with JoinVertical
	return lipgloss.JoinVertical(lipgloss.Left,
		deviceLine,
		statusLine,
		divider,
		"",
		outletsSection,
		"",
		wifiSection,
		"",
		serverSection,
	)
}

// renderField renders a configuration field as a simple line (no box)
// Format: "→ Label          Value                      ▼" when selected
//         "  Label          Value                      ▼" when not selected
// Used for flat, simple field rendering without nested boxes
func (m DashboardModel) renderField(label string, value string, fieldIdx int, hasDropdown bool) string {
	isSelected := m.Cursor == fieldIdx

	// Build the line using lipgloss styles
	labelStyle := lipgloss.NewStyle().Width(18).Foreground(SubtleColor)
	valueStyle := lipgloss.NewStyle()

	if isSelected {
		labelStyle = labelStyle.Foreground(HighlightColor).Bold(true)
		valueStyle = valueStyle.Foreground(HighlightColor).Bold(true)
	}

	// Dropdown indicator
	indicator := ""
	if hasDropdown {
		indicator = " ▼"
	}

	// Selection arrow
	arrow := "  "
	if isSelected {
		arrow = "→ "
	}

	// Compose line using lipgloss.JoinHorizontal
	line := lipgloss.JoinHorizontal(lipgloss.Left,
		arrow,
		labelStyle.Render(label),
		valueStyle.Render(value+indicator),
	)

	return line
}

// isFieldChanged returns true if the field's pending value differs from the original
func (m DashboardModel) isFieldChanged(fieldIdx int) bool {
	switch fieldIdx {
	case 0:
		return m.PendingConfig.Outlet1 != m.CurrentConfig.Outlet1
	case 1:
		return m.PendingConfig.Outlet2 != m.CurrentConfig.Outlet2
	case 2:
		return m.PendingConfig.Outlet3 != m.CurrentConfig.Outlet3
	case 3:
		return m.PendingConfig.K3Outlet != m.CurrentConfig.K3Outlet
	case 5:
		return m.PendingConfig.WiFiSSID != getFirstSSID(m.CurrentConfig.SSIDList)
	case 7:
		return m.PendingConfig.DNS != m.CurrentConfig.DNS
	case 8:
		return m.PendingConfig.Port != m.CurrentConfig.Port
	}
	return false
}

// renderSection renders a configuration section with title, fields, and apply button
// No box borders - just styled text for a flat, simple layout
func (m DashboardModel) renderSection(title string, fields []string, buttonLabel string, buttonIdx int, hasChanges bool) string {
	// Section title - bold, colored
	titleStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)
	titleLine := titleStyle.Render(title)

	// Build parts
	parts := []string{titleLine}

	// Add fields
	parts = append(parts, fields...)

	// Add apply button
	button := m.renderApplyButton(buttonLabel, buttonIdx, hasChanges)
	parts = append(parts, "", button)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderOutletFieldInline renders an outlet field in inline editing mode
func (m DashboardModel) renderOutletFieldInline(fieldIdx int) string {
	var options []string
	var currentValue int

	if fieldIdx == 3 {
		// K3 mode - just 2 options
		options = []string{"Disabled (sequential)", "Enabled (separate)"}
		if m.PendingConfig.K3Outlet {
			currentValue = 1
		}
	} else {
		// Outlet bitmask - 8 options
		for i := 0; i < 8; i++ {
			options = append(options, FormatBitmask(i))
		}
		currentValue = m.getPendingValue(fieldIdx)
	}

	var lines []string
	for i, opt := range options {
		cursor := "  "
		if i == m.OutletEditor.Cursor {
			cursor = "← "
		}

		indicator := "( )"
		if i == currentValue {
			indicator = "(•)"
		}

		style := lipgloss.NewStyle()
		if i == m.OutletEditor.Cursor {
			style = style.Foreground(HighlightColor)
		}

		line := style.Render(fmt.Sprintf("        %s %s %s", indicator, opt, cursor))
		lines = append(lines, line)
	}

	helpLine := lipgloss.NewStyle().
		Foreground(SubtleColor).
		Render("        ↑/↓ select • Enter confirm • Esc cancel")
	lines = append(lines, helpLine)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderOutletField renders a single outlet field as a simple line using renderField
// NOTE: Inline editing is handled by the section renderer inserting the inline editor
func (m DashboardModel) renderOutletField(fieldIdx int) string {
	label := m.getFieldLabel(fieldIdx)

	var value string
	if fieldIdx == 3 {
		// K3 mode
		if m.PendingConfig.K3Outlet {
			value = "[✓] Enabled"
		} else {
			value = "[ ] Disabled"
		}
	} else {
		value = FormatBitmask(m.getPendingValue(fieldIdx))
	}

	// Add change indicator if modified
	if m.isFieldChanged(fieldIdx) {
		value += " ⚠"
	}

	return m.renderField(label, value, fieldIdx, true)
}

// renderWiFiFieldInline renders WiFi field in inline editing mode
func (m DashboardModel) renderWiFiFieldInline() string {
	var lines []string

	// SSID options
	lines = append(lines, lipgloss.NewStyle().Foreground(SubtleColor).Render("      Select network:"))

	for i, ssid := range m.CurrentConfig.SSIDList {
		cursor := "  "
		if i == m.WiFiEditor.Cursor && !m.WiFiEditor.PasswordInput.Focused() {
			cursor = "← "
		}

		indicator := "( )"
		if ssid == m.PendingConfig.WiFiSSID {
			indicator = "(•)"
		}

		style := lipgloss.NewStyle()
		if i == m.WiFiEditor.Cursor && !m.WiFiEditor.PasswordInput.Focused() {
			style = style.Foreground(HighlightColor)
		}

		line := style.Render(fmt.Sprintf("        %s %s %s", indicator, ssid, cursor))
		lines = append(lines, line)
	}

	// Password field
	lines = append(lines, "")

	passwordCursor := "  "
	if m.WiFiEditor.PasswordInput.Focused() {
		passwordCursor = "← "
	}

	passwordLabel := lipgloss.NewStyle().Foreground(SubtleColor).Render("      Password: ")
	lines = append(lines, passwordLabel+m.WiFiEditor.PasswordInput.View()+" "+passwordCursor)

	// Help
	helpLine := lipgloss.NewStyle().
		Foreground(SubtleColor).
		Render("        ↑/↓ select • Enter confirm • Esc cancel")
	lines = append(lines, helpLine)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderServerFieldInline renders server field (DNS or Port) in inline editing mode
// fieldType: 0=DNS, 1=Port (matching EditingField values)
func (m DashboardModel) renderServerFieldInline(fieldType int) string {
	var input textinput.Model
	var helpText string

	if fieldType == 0 {
		input = m.ServerEditor.DNSInput
		helpText = "Enter DNS hostname (e.g., smartap.local)"
	} else {
		input = m.ServerEditor.PortInput
		helpText = "Enter port number (1-65535)"
	}

	helpStyle := lipgloss.NewStyle().Foreground(SubtleColor)

	lines := []string{
		helpStyle.Render("        " + helpText),
		"",
		"        " + input.View(),
		"",
		helpStyle.Render("        Enter confirm • Esc cancel"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderOutletsSection renders the Outlets configuration section
// Simple flat layout using renderSection helper
func (m DashboardModel) renderOutletsSection() string {
	fields := []string{
		m.renderOutletField(0),
		m.renderOutletField(1),
		m.renderOutletField(2),
		m.renderOutletField(3),
	}

	// If editing an outlet field, insert the inline editor after it
	if m.EditingSection == SectionOutlets {
		editedIdx := m.EditingField
		if editedIdx >= 0 && editedIdx < 4 {
			editor := m.renderOutletFieldInline(editedIdx)
			// Insert editor after the field
			newFields := make([]string, 0, len(fields)+1)
			for i, f := range fields {
				newFields = append(newFields, f)
				if i == editedIdx {
					newFields = append(newFields, editor)
				}
			}
			fields = newFields
		}
	}

	return m.renderSection("OUTLETS", fields, "Outlets", 4, m.hasOutletChanges())
}

// renderWiFiSection renders the WiFi configuration section
// Simple flat layout using renderSection helper
func (m DashboardModel) renderWiFiSection() string {
	fields := []string{
		m.renderWiFiField(),
	}

	// If editing WiFi, insert inline editor
	if m.EditingSection == SectionWiFi {
		editor := m.renderWiFiFieldInline()
		fields = append(fields, editor)
	}

	return m.renderSection("WIFI", fields, "WiFi", 6, m.hasWiFiChanges())
}

// renderWiFiField renders the WiFi network field as a simple line
func (m DashboardModel) renderWiFiField() string {
	value := m.PendingConfig.WiFiSSID
	if value == "" {
		value = "(not configured)"
	}
	if m.hasWiFiChanges() {
		value += " ⚠"
	}
	return m.renderField("Network", value, 5, true)
}

// renderServerSection renders the Server configuration section
// Simple flat layout using renderSection helper
func (m DashboardModel) renderServerSection() string {
	fields := []string{
		m.renderServerField(7), // DNS
		m.renderServerField(8), // Port
	}

	// If editing server, insert inline editor after the edited field
	if m.EditingSection == SectionServer {
		editor := m.renderServerFieldInline(m.EditingField)
		// Insert after appropriate field
		if m.EditingField == 0 {
			// Editing DNS - insert after first field
			fields = []string{fields[0], editor, fields[1]}
		} else {
			// Editing Port - append after second field
			fields = append(fields, editor)
		}
	}

	return m.renderSection("SERVER", fields, "Server", 9, m.hasServerChanges())
}

// renderServerField renders a server field (DNS or Port) as a simple line
func (m DashboardModel) renderServerField(fieldIdx int) string {
	var label, value string
	if fieldIdx == 7 {
		label = "Hostname"
		value = m.PendingConfig.DNS
	} else {
		label = "Port"
		value = fmt.Sprintf("%d", m.PendingConfig.Port)
	}
	if m.hasServerChanges() {
		value += " ⚠"
	}
	// Server fields don't have dropdown in collapsed view
	return m.renderField(label, value, fieldIdx, false)
}

// hasOutletChanges returns true if any outlet configuration has changed
func (m DashboardModel) hasOutletChanges() bool {
	return m.PendingConfig.Outlet1 != m.CurrentConfig.Outlet1 ||
		m.PendingConfig.Outlet2 != m.CurrentConfig.Outlet2 ||
		m.PendingConfig.Outlet3 != m.CurrentConfig.Outlet3 ||
		m.PendingConfig.K3Outlet != m.CurrentConfig.K3Outlet
}

// hasWiFiChanges returns true if WiFi configuration has changed
func (m DashboardModel) hasWiFiChanges() bool {
	// Get first SSID from list (current network)
	currentSSID := ""
	if len(m.CurrentConfig.SSIDList) > 0 {
		currentSSID = m.CurrentConfig.SSIDList[0]
	}
	return m.PendingConfig.WiFiSSID != currentSSID
}

// hasServerChanges returns true if server configuration has changed
func (m DashboardModel) hasServerChanges() bool {
	return m.PendingConfig.DNS != m.CurrentConfig.DNS ||
		m.PendingConfig.Port != m.CurrentConfig.Port
}

// renderApplyButton renders an Apply button for a configuration section
func (m DashboardModel) renderApplyButton(label string, fieldIdx int, hasChanges bool) string {
	buttonText := fmt.Sprintf("[Apply %s]", label)

	if hasChanges {
		buttonText += " ⚠ Modified"
	}

	buttonStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)

	// Highlight if cursor is on this button
	if m.Cursor == fieldIdx {
		buttonStyle = buttonStyle.
			Background(PrimaryColor).
			Foreground(BackgroundColor)
	}

	button := buttonStyle.Render(buttonText)

	// Center the button
	return lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(60).
		Render(button)
}

// renderWiFiWarningModal renders the WiFi change warning modal
// Refactored to use lipgloss.JoinVertical instead of strings.Builder
func (m DashboardModel) renderWiFiWarningModalContent() string {
	// Title
	titleStyle := lipgloss.NewStyle().Foreground(WarningColor).Bold(true)
	title := titleStyle.Render("⚠ WIFI NETWORK CHANGE WARNING")

	// Warning message lines
	warningStyle := lipgloss.NewStyle().Foreground(TextColor)
	warningLines := lipgloss.JoinVertical(lipgloss.Left,
		warningStyle.Render("  After applying this change:"),
		warningStyle.Render("  • Device will disconnect from current network"),
		warningStyle.Render("  • You will lose connection to the device"),
	)

	// Old -> New SSID
	changeStyle := lipgloss.NewStyle().Foreground(SubtleColor)
	networkInfo := lipgloss.JoinVertical(lipgloss.Left,
		changeStyle.Render(fmt.Sprintf("  Old Network: %s", m.PendingConfig.WiFiSSID)),
		changeStyle.Render(fmt.Sprintf("  New Network: %s", m.WiFiSelectedSSID)),
	)

	// Instructions
	instructionStyle := lipgloss.NewStyle().Foreground(SecondaryColor)
	instructions := lipgloss.JoinVertical(lipgloss.Left,
		instructionStyle.Render("  To continue:"),
		warningStyle.Render("  1. Connect to the same network on this computer"),
		warningStyle.Render("  2. Re-run this application to verify changes"),
	)

	// Buttons (Apply Changes / Cancel)
	applyButton := "Apply Changes"
	cancelButton := "Cancel"

	buttonStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(SubtleColor).
		Foreground(SubtleColor).
		Padding(0, 2).
		MarginRight(2)

	selectedButtonStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(HighlightColor).
		Foreground(HighlightColor).
		Bold(true).
		Padding(0, 2).
		MarginRight(2)

	var applyBtn, cancelBtn string
	if m.ModalCursor == 0 {
		applyBtn = selectedButtonStyle.Render("→ " + applyButton)
	} else {
		applyBtn = buttonStyle.Render("  " + applyButton)
	}

	if m.ModalCursor == 1 {
		cancelBtn = selectedButtonStyle.Render("→ " + cancelButton)
	} else {
		cancelBtn = buttonStyle.Render("  " + cancelButton)
	}

	buttons := "  " + lipgloss.JoinHorizontal(lipgloss.Left, applyBtn, cancelBtn)

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(SubtleColor)
	help := helpStyle.Render("  ←/→: Navigate  •  Enter: Confirm  •  Esc: Back")

	// Compose all content using lipgloss.JoinVertical
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		warningLines,
		"",
		networkInfo,
		"",
		instructions,
		"",
		buttons,
		"",
		help,
	)

	// Create modal box with Lipgloss
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(WarningColor).
		Padding(1, 2).
		Width(70) // Fixed comfortable width - centering handled by RenderModal

	return modalStyle.Render(content)
}

// Helper functions

func (m DashboardModel) getCurrentDeviceName() string {
	if m.CurrentConfig.Serial != "" {
		return fmt.Sprintf("eValve%s", m.CurrentConfig.Serial)
	}
	return "Unknown Device"
}

func (m DashboardModel) getFieldLabel(index int) string {
	// Navigation order (field → apply per section):
	//   0-3: Outlet fields, 4: Apply Outlets
	//   5: WiFi field, 6: Apply WiFi
	//   7-8: Server fields, 9: Apply Server
	switch index {
	case 0:
		return "First Button Press"
	case 1:
		return "Second Button Press"
	case 2:
		return "Third Button Press"
	case 3:
		return "Third Knob Mode"
	case 4:
		return "Apply Outlets"
	case 5:
		return "WiFi Network"
	case 6:
		return "Apply WiFi"
	case 7:
		return "Server DNS"
	case 8:
		return "Server Port"
	case 9:
		return "Apply Server"
	default:
		return ""
	}
}

func (m DashboardModel) getPendingValue(index int) int {
	switch index {
	case 0:
		return m.PendingConfig.Outlet1
	case 1:
		return m.PendingConfig.Outlet2
	case 2:
		return m.PendingConfig.Outlet3
	}
	return 0
}

// IsBackRequested returns true if user wants to go back
func (m DashboardModel) IsBackRequested() bool {
	return m.BackRequested
}

// updateProgressModal handles input during progress modal (blocks input)
func (m DashboardModel) updateProgressModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Block all input during progress
		return m, nil

	case spinner.TickMsg:
		// Update spinner animation
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd

	case applyCompleteMsg:
		// Apply operation completed
		m.ShowingProgress = false
		if msg.err != nil {
			// Show failure modal
			m.ShowingFailure = true
			m.ApplyError = msg.err
			m.ModalCursor = 0 // Default to "Retry" button
		} else {
			// Show success modal
			m.ShowingSuccess = true
			m.VerifiedConfig = msg.config
			m.HasUnsavedChanges = false
			m.LastSaved = time.Now()
			m.VerificationDuration = msg.verificationDuration // Store how long it took
			// Update current config
			if msg.config != nil {
				m.CurrentConfig = msg.config
			}
		}
		return m, nil
	}

	return m, nil
}

// updateSuccessModal handles input on success modal
func (m DashboardModel) updateSuccessModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ", "esc":
			// Close success modal and return to dashboard
			m.ShowingSuccess = false
			m.updateChangeTracking()
		}
	}

	return m, nil
}

// updateFailureModal handles input on failure modal
func (m DashboardModel) updateFailureModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			// Close failure modal
			m.ShowingFailure = false

		case "left", "h":
			if m.ModalCursor > 0 {
				m.ModalCursor--
			}

		case "right", "l":
			if m.ModalCursor < 1 {
				m.ModalCursor++
			}

		case "enter", " ":
			// Close failure modal (no retry - user can just apply again)
			m.ShowingFailure = false
		}
	}

	return m, nil
}

// updateHelpModal handles input when help modal is visible
func (m DashboardModel) updateHelpModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		// Any key closes the help modal
		m.ShowingHelp = false
	}

	return m, nil
}

// View renders the dashboard
func (m DashboardModel) View() string {
	// Handle modals first (progress, success, failure, help, wifi warning)
	if m.ShowingWiFiWarning {
		modal := m.renderWiFiWarningModalContent()
		return RenderModal("", modal, m.Width, m.Height)
	}
	if m.ShowingHelp {
		modal := m.renderHelpModalContent()
		return RenderModal("", modal, m.Width, m.Height)
	}
	if m.ShowingProgress {
		modal := m.renderProgressModalContent()
		return RenderModal("", modal, m.Width, m.Height)
	}
	if m.ShowingSuccess {
		modal := m.renderSuccessModalContent()
		return RenderModal("", modal, m.Width, m.Height)
	}
	if m.ShowingFailure {
		modal := m.renderFailureModalContent()
		return RenderModal("", modal, m.Width, m.Height)
	}

	// Normal dashboard view
	return m.renderDashboard()
}

// renderProgressModal renders the progress modal (applying configuration)
// Refactored to use lipgloss.JoinVertical instead of strings.Builder
func (m DashboardModel) renderProgressModalContent() string {
	// Title with spinner
	titleStyle := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true)
	title := titleStyle.Render(fmt.Sprintf("%s APPLYING CONFIGURATION", m.Spinner.View()))

	// Calculate progress (0-100%)
	// Simple two-stage progress: 0-50% = sending, 50-100% = verifying
	elapsed := time.Since(m.ApplyStartTime)
	elapsedRounded := elapsed.Round(100 * time.Millisecond)

	// Estimate progress based on elapsed time (10 second total estimated)
	baseProgress := min(int(elapsed.Seconds()*10), 50) // First 5 seconds = 0-50%
	if elapsed.Seconds() > 0.5 {
		// After 500ms (config sent), progress 50-100% based on verification time
		verifyProgress := min(int((elapsed.Seconds()-0.5)*10), 50)
		baseProgress = 50 + verifyProgress
	}
	percentage := min(baseProgress, 100)
	progressFloat := float64(percentage) / 100.0

	// Use bubbles/progress component
	progressBar := m.ProgressBar.ViewAs(progressFloat)
	progressLine := lipgloss.JoinHorizontal(lipgloss.Left, progressBar, fmt.Sprintf("  %d%%", percentage))

	// Progress steps
	successStyle := lipgloss.NewStyle().Foreground(SecondaryColor)
	var statusLines string
	if elapsed.Seconds() > 0.5 {
		// Config sent, now verifying
		attemptText := ""
		if m.MaxAttempts > 1 {
			attemptText = fmt.Sprintf(" (attempt %d/%d)", m.ApplyAttempt, m.MaxAttempts)
		}
		statusLines = lipgloss.JoinVertical(lipgloss.Left,
			successStyle.Render("✓ Configuration sent to device"),
			fmt.Sprintf("%s Verifying changes%s...", m.Spinner.View(), attemptText),
		)
	} else {
		// Still sending config
		statusLines = fmt.Sprintf("%s Sending configuration to device...", m.Spinner.View())
	}

	// Elapsed time
	timeStyle := lipgloss.NewStyle().Foreground(SubtleColor)
	elapsedText := timeStyle.Render(fmt.Sprintf("Elapsed: %s", elapsedRounded))

	// Compose all content using lipgloss.JoinVertical
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		progressLine,
		"",
		statusLines,
		"",
		elapsedText,
	)

	// Create modal box with Lipgloss
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 2).
		Width(60) // Fixed comfortable width - centering handled by RenderModal

	return modalStyle.Render(content)
}

// renderSuccessModal renders the success modal
// Refactored to use lipgloss.JoinVertical instead of strings.Builder
func (m DashboardModel) renderSuccessModalContent() string {
	titleStyle := lipgloss.NewStyle().Foreground(SecondaryColor).Bold(true)

	// Check if this was a WiFi configuration change
	if m.WiFiChangeApplied {
		// Title
		title := titleStyle.Render("✓ WIFI CONFIGURATION APPLIED!")

		// WiFi-specific success message
		successMsg := SuccessStyle.Render("✓ WiFi configuration applied!")
		connectingMsg := "The device is now connecting to the new network."

		// Instructions
		warningStyle := lipgloss.NewStyle().Foreground(WarningColor).Bold(true)
		instructions := lipgloss.JoinVertical(lipgloss.Left,
			warningStyle.Render("⚠ Important:"),
			"  • The device will disconnect from this network",
			"  • Connect your computer to the same WiFi network",
			"  • Re-run this application to verify the change",
		)

		infoStyle := lipgloss.NewStyle().Foreground(SecondaryColor)
		networkInfo := infoStyle.Render(fmt.Sprintf("New Network: %s", m.PendingConfig.WiFiSSID))

		// Exit button
		exitStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(HighlightColor).
			Foreground(HighlightColor).
			Bold(true).
			Padding(0, 2)

		exitButton := lipgloss.NewStyle().MarginLeft(15).Render(exitStyle.Render("Exit Application"))

		// Compose all content using lipgloss.JoinVertical
		content := lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			successMsg,
			"",
			connectingMsg,
			"",
			instructions,
			"",
			networkInfo,
			"",
			exitButton,
		)

		// Create modal box with Lipgloss
		modalStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(1, 2).
			Width(70) // Fixed comfortable width - centering handled by RenderModal

		return modalStyle.Render(content)
	}

	// Standard success modal for diverter config
	// Title
	title := titleStyle.Render("✓ CONFIGURATION APPLIED!")

	// Standard success message
	successMsg := SuccessStyle.Render("✓ Configuration updated successfully!")

	// Build content parts
	contentParts := []string{
		title,
		"",
		successMsg,
		"",
	}

	if m.VerifiedConfig != nil {
		contentParts = append(contentParts, "Your changes have been saved to the device", "")

		// Show verified config
		k3Status := "Disabled"
		if m.VerifiedConfig.K3Outlet {
			k3Status = "Enabled"
		}
		configDetails := lipgloss.JoinVertical(lipgloss.Left,
			"Verified new configuration:",
			fmt.Sprintf("  First Press:  %s", FormatBitmask(m.VerifiedConfig.Outlet1)),
			fmt.Sprintf("  Second Press: %s", FormatBitmask(m.VerifiedConfig.Outlet2)),
			fmt.Sprintf("  Third Press:  %s", FormatBitmask(m.VerifiedConfig.Outlet3)),
			fmt.Sprintf("  Third Knob:   %s", k3Status),
		)
		contentParts = append(contentParts, configDetails, "")
	}

	// Show verification duration
	if m.VerificationDuration > 0 {
		durationSeconds := m.VerificationDuration.Seconds()
		subtleStyle := lipgloss.NewStyle().Foreground(SubtleColor)
		durationText := subtleStyle.Render(fmt.Sprintf("Configuration verified in %.1f seconds", durationSeconds))
		contentParts = append(contentParts, durationText, "")
	}

	// Continue button
	continueStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(HighlightColor).
		Foreground(HighlightColor).
		Bold(true).
		Padding(0, 2)

	continueButton := lipgloss.NewStyle().MarginLeft(15).Render(continueStyle.Render("Continue"))
	contentParts = append(contentParts, continueButton)

	// Compose all content using lipgloss.JoinVertical
	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)

	// Create modal box with Lipgloss
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(SecondaryColor).
		Padding(1, 2).
		Width(60) // Fixed comfortable width - centering handled by RenderModal

	return modalStyle.Render(content)
}

// renderFailureModal renders the failure modal with rollback info
// Refactored to use lipgloss.JoinVertical instead of strings.Builder
func (m DashboardModel) renderFailureModalContent() string {
	// Title
	titleStyle := lipgloss.NewStyle().Foreground(ErrorColor).Bold(true)
	title := titleStyle.Render("✗ CONFIGURATION UPDATE FAILED")

	// Build content parts
	contentParts := []string{title, ""}

	// Error message
	if m.ApplyError != nil {
		errorStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ErrorColor).
			Foreground(ErrorColor).
			Padding(0, 2).
			Bold(true)

		errorMsg := fmt.Sprintf("Error: %s", m.ApplyError.Error())
		contentParts = append(contentParts, errorStyle.Render(errorMsg), "")
	}

	// Rollback info (always show as we auto-rollback)
	warningStyle := lipgloss.NewStyle().Foreground(WarningColor).Bold(true)
	rollbackMsg := warningStyle.Render("⚠ Automatically rolled back to previous configuration")
	contentParts = append(contentParts, rollbackMsg, "")

	// Device restoration info
	k3Status := "Disabled"
	if m.CurrentConfig.K3Outlet {
		k3Status = "Enabled"
	}
	restoredConfig := lipgloss.JoinVertical(lipgloss.Left,
		"Your device has been restored to:",
		fmt.Sprintf("  • First Press:  %s", FormatBitmask(m.CurrentConfig.Outlet1)),
		fmt.Sprintf("  • Second Press: %s", FormatBitmask(m.CurrentConfig.Outlet2)),
		fmt.Sprintf("  • Third Press:  %s", FormatBitmask(m.CurrentConfig.Outlet3)),
		fmt.Sprintf("  • K3 Mode:      %s", k3Status),
	)
	contentParts = append(contentParts, restoredConfig, "")

	// Troubleshooting
	troubleshooting := lipgloss.JoinVertical(lipgloss.Left,
		"Troubleshooting:",
		"  • Check device is powered on and responsive",
		"  • Verify network connection to device",
	)
	contentParts = append(contentParts, troubleshooting, "")

	// Retry/Back buttons
	buttonStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(SubtleColor).
		Padding(0, 2)

	highlightedButtonStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(HighlightColor).
		Foreground(HighlightColor).
		Bold(true).
		Padding(0, 2)

	retryBtn := buttonStyle.Render("Retry")
	backBtn := buttonStyle.Render("Back")

	if m.ModalCursor == 0 {
		retryBtn = highlightedButtonStyle.Render("→ Retry")
	} else {
		backBtn = highlightedButtonStyle.Render("→ Back")
	}

	// Join buttons horizontally with spacing
	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Left,
		retryBtn,
		"         ",
		backBtn,
	)
	centeredButtons := lipgloss.NewStyle().MarginLeft(8).Render(buttonsRow)
	contentParts = append(contentParts, centeredButtons)

	// Compose all content using lipgloss.JoinVertical
	content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)

	// Create modal box with Lipgloss
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ErrorColor).
		Padding(1, 2).
		Width(70) // Fixed comfortable width - centering handled by RenderModal

	return modalStyle.Render(content)
}

// renderHelpModal renders the help modal explaining outlet configurations
// Refactored to use lipgloss.JoinVertical instead of strings.Builder
func (m DashboardModel) renderHelpModalContent() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Bold(true)
	title := titleStyle.Render("OUTLET CONFIGURATION HELP")

	// Subtitle style
	subtitleStyle := lipgloss.NewStyle().
		Foreground(SecondaryColor).
		Bold(true)

	// Understanding Outlet Combinations
	outletCombos := lipgloss.JoinVertical(lipgloss.Left,
		subtitleStyle.Render("Understanding Outlet Combinations:"),
		"  [0] No Outlets        - Disabled (nothing turns on)",
		"  [1] Outlet 1          - Only outlet 1 active",
		"  [2] Outlet 2          - Only outlet 2 active",
		"  [3] Outlets 1+2       - Both outlets 1 and 2 active",
		"  [4] Outlet 3          - Only outlet 3 active",
		"  [5] Outlets 1+3       - Both outlets 1 and 3 active",
		"  [6] Outlets 2+3       - Both outlets 2 and 3 active",
		"  [7] Outlets 1+2+3     - All three outlets active",
	)

	// Third Knob Mode
	knobMode := lipgloss.JoinVertical(lipgloss.Left,
		subtitleStyle.Render("Third Knob Mode:"),
		"  [✓] Enabled  - Outlet 3 controlled separately",
		"  [ ] Disabled - All outlets on one knob",
	)

	// Instructions
	instructions := "Press any key to close this help screen"

	// Compose all content using lipgloss.JoinVertical
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		outletCombos,
		"",
		knobMode,
		"",
		instructions,
	)

	// Create modal box with Lipgloss
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Padding(1, 2).
		Width(70) // Fixed comfortable width - centering handled by RenderModal

	return modalStyle.Render(content)
}

// getFirstSSID returns the first SSID from the list, or empty string if none
func getFirstSSID(ssidList []string) string {
	if len(ssidList) > 0 {
		return ssidList[0]
	}
	return ""
}
