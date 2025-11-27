package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muurk/smartap/internal/deviceconfig"
	"github.com/muurk/smartap/internal/discovery"
)

// Screen represents the current active screen in the application
type Screen string

const (
	ScreenDiscovery Screen = "discovery"
	ScreenDashboard Screen = "dashboard"
	ScreenSuccess   Screen = "success"
	ScreenFailure   Screen = "failure"
)

// Messages for screen transitions
type screenTransitionMsg struct {
	screen Screen
	data   interface{}
}

type goBackMsg struct{}
type quitMsg struct{}

// successKeyMap defines key bindings for the success screen
type successKeyMap struct {
	View     key.Binding
	Edit     key.Binding
	Discover key.Binding
	Quit     key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k successKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.View, k.Edit, k.Discover, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k successKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.View, k.Edit, k.Discover, k.Quit},
	}
}

// failureKeyMap defines key bindings for the failure screen
type failureKeyMap struct {
	Retry key.Binding
	Edit  key.Binding
	View  key.Binding
	Quit  key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k failureKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Retry, k.Edit, k.View, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k failureKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Retry, k.Edit, k.View, k.Quit},
	}
}

// AppModel is the top-level coordinator model that manages screen transitions
type AppModel struct {
	// Current screen state
	CurrentScreen  Screen
	PreviousScreen Screen

	// Screen models
	DiscoveryModel DiscoveryModel
	DashboardModel DashboardModel

	// Shared application state
	SelectedDevice *discovery.Device
	CurrentConfig  *deviceconfig.DeviceConfig
	PendingUpdate  *deviceconfig.ConfigUpdate
	LastError      error

	// Result state (from confirmation screen)
	VerifiedConfig *deviceconfig.DeviceConfig
	RolledBack     bool

	// UI state
	Width  int
	Height int

	// Help
	Help        help.Model
	SuccessKeys successKeyMap
	FailureKeys failureKeyMap
}

// NewAppModel creates a new application model starting at the specified screen
func NewAppModel(startScreen Screen, device *discovery.Device) AppModel {
	// Initialize help
	h := help.New()

	// Initialize key bindings for success screen
	successKeys := successKeyMap{
		View: key.NewBinding(
			key.WithKeys("enter", "v"),
			key.WithHelp("enter/v", "view"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Discover: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "discover"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}

	// Initialize key bindings for failure screen
	failureKeys := failureKeyMap{
		Retry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		View: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "view"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}

	model := AppModel{
		CurrentScreen:  startScreen,
		PreviousScreen: "",
		SelectedDevice: device,
		Help:           h,
		SuccessKeys:    successKeys,
		FailureKeys:    failureKeys,
	}

	// Initialize the starting screen
	switch startScreen {
	case ScreenDiscovery:
		model.DiscoveryModel = NewDiscoveryModel()
	}

	return model
}

// Init initializes the application
func (m AppModel) Init() tea.Cmd {
	// Initialize the current screen's model
	switch m.CurrentScreen {
	case ScreenDiscovery:
		return m.DiscoveryModel.Init()
	case ScreenDashboard:
		return m.DashboardModel.Init()
	default:
		return nil
	}
}

// Update handles all messages and routes them to the appropriate screen
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Propagate to all screens
		m.DiscoveryModel.Width = msg.Width
		m.DiscoveryModel.Height = msg.Height
		m.DashboardModel.Width = msg.Width
		m.DashboardModel.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global quit handler
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case screenTransitionMsg:
		return m.transitionTo(msg.screen, msg.data)

	case goBackMsg:
		return m.goBack()

	case quitMsg:
		return m, tea.Quit
	}

	// Route to current screen
	return m.updateCurrentScreen(msg)
}

// updateCurrentScreen routes updates to the currently active screen
func (m AppModel) updateCurrentScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.CurrentScreen {
	case ScreenDiscovery:
		updated, c := m.DiscoveryModel.Update(msg)
		m.DiscoveryModel = updated.(DiscoveryModel)
		cmd = c

		// Check if user selected a device
		if m.DiscoveryModel.Selected {
			m.SelectedDevice = m.DiscoveryModel.GetSelectedDevice()
			if m.SelectedDevice != nil {
				// Transition to unified dashboard (NEW FLOW)
				return m.transitionTo(ScreenDashboard, nil)
			}
		}

		// Check for quit (normal mode only, not during scan)
		if !m.DiscoveryModel.Scanning && !m.DiscoveryModel.ManualMode {
			if keyMsg, ok := msg.(tea.KeyMsg); ok {
				if keyMsg.String() == "q" || keyMsg.String() == "esc" {
					return m, tea.Quit
				}
			}
		}

	case ScreenDashboard:
		updated, c := m.DashboardModel.Update(msg)
		m.DashboardModel = updated.(DashboardModel)
		cmd = c

		// Check if user wants to go back
		if m.DashboardModel.IsBackRequested() {
			return m.goBack()
		}

	case ScreenSuccess:
		return m.handleSuccessScreen(msg)

	case ScreenFailure:
		return m.handleFailureScreen(msg)
	}

	return m, cmd
}

// handleSuccessScreen handles user input on the success screen
func (m AppModel) handleSuccessScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter", "v", "e":
			// View/edit configuration - go to dashboard
			m.CurrentConfig = m.VerifiedConfig
			return m.transitionTo(ScreenDashboard, nil)

		case "d":
			// Discover another device
			return m.transitionTo(ScreenDiscovery, nil)

		case "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

// handleFailureScreen handles user input on the failure screen
func (m AppModel) handleFailureScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "r", "e", "v":
			// Retry/edit/view - go to dashboard (dashboard handles all configuration)
			return m.transitionTo(ScreenDashboard, nil)

		case "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

// transitionTo transitions to a new screen
func (m AppModel) transitionTo(screen Screen, data interface{}) (tea.Model, tea.Cmd) {
	m.PreviousScreen = m.CurrentScreen
	m.CurrentScreen = screen

	var cmd tea.Cmd

	// Initialize the target screen with current state
	switch screen {
	case ScreenDiscovery:
		m.DiscoveryModel = NewDiscoveryModel()
		cmd = m.DiscoveryModel.Init()

	case ScreenDashboard:
		if m.SelectedDevice != nil {
			// Fetch config from device first
			client := deviceconfig.NewClient(m.SelectedDevice.IP, m.SelectedDevice.Port)
			config, err := client.GetConfiguration()
			if err != nil {
				// Store error and stay on discovery
				m.LastError = err
				return m, nil
			}
			m.CurrentConfig = config

			// Initialize dashboard with fetched config
			m.DashboardModel = NewDashboardModel(
				m.SelectedDevice.IP,
				m.SelectedDevice.Port,
				config,
			)
			// Copy terminal dimensions to new dashboard model
			m.DashboardModel.Width = m.Width
			m.DashboardModel.Height = m.Height
			cmd = m.DashboardModel.Init()
		}

	case ScreenSuccess:
		// Success screen doesn't need initialization
		cmd = nil

	case ScreenFailure:
		// Failure screen doesn't need initialization
		cmd = nil
	}

	return m, cmd
}

// goBack returns to the previous screen
func (m AppModel) goBack() (tea.Model, tea.Cmd) {
	switch m.CurrentScreen {
	case ScreenDiscovery:
		// Can't go back from discovery - quit instead
		return m, tea.Quit

	case ScreenDashboard:
		// Go back to discovery
		return m.transitionTo(ScreenDiscovery, nil)

	case ScreenSuccess, ScreenFailure:
		// Go back to dashboard
		return m.transitionTo(ScreenDashboard, nil)

	default:
		return m, tea.Quit
	}
}

// View renders the current screen
// Each screen handles its own container using RenderApplicationContainer()
func (m AppModel) View() string {
	switch m.CurrentScreen {
	case ScreenDiscovery:
		return m.DiscoveryModel.View()
	case ScreenDashboard:
		return m.DashboardModel.View()
	case ScreenSuccess:
		return m.renderSuccessScreen()
	case ScreenFailure:
		return m.renderFailureScreen()
	default:
		return "Unknown screen"
	}
}

// renderSuccessScreen renders the success result screen
func (m AppModel) renderSuccessScreen() string {
	// Build content (without container)
	content := m.buildSuccessContent()

	// Help text using bubbles/help
	helpText := m.Help.View(m.SuccessKeys)

	// Wrap with unified container
	return RenderApplicationContainer(content, helpText, m.Width, m.Height)
}

// buildSuccessContent builds the success screen content
func (m AppModel) buildSuccessContent() string {
	var b strings.Builder

	b.WriteString(RenderTitle("✓ Configuration Updated Successfully!"))
	b.WriteString("\n\n")

	if m.VerifiedConfig != nil {
		b.WriteString(SuccessBoxStyle.Render("Verified configuration:"))
		b.WriteString("\n\n")

		// Show verified configuration
		config := fmt.Sprintf("  First Press:  %d (%s)\n", m.VerifiedConfig.Outlet1, deviceconfig.FormatBitmask(m.VerifiedConfig.Outlet1))
		config += fmt.Sprintf("  Second Press: %d (%s)\n", m.VerifiedConfig.Outlet2, deviceconfig.FormatBitmask(m.VerifiedConfig.Outlet2))
		config += fmt.Sprintf("  Third Press:  %d (%s)\n", m.VerifiedConfig.Outlet3, deviceconfig.FormatBitmask(m.VerifiedConfig.Outlet3))
		config += fmt.Sprintf("  K3 Mode:      %v", formatBool(m.VerifiedConfig.K3Outlet))

		b.WriteString(config)
		b.WriteString("\n\n")
	}

	b.WriteString("What would you like to do next?\n\n")

	b.WriteString(MenuItemStyle.Render("  Enter/v - View updated configuration"))
	b.WriteString("\n")
	b.WriteString(MenuItemStyle.Render("  e       - Make another change"))
	b.WriteString("\n")
	b.WriteString(MenuItemStyle.Render("  d       - Discover another device"))
	b.WriteString("\n")
	b.WriteString(MenuItemStyle.Render("  q       - Exit application"))
	b.WriteString("\n")

	return b.String()
}

// renderFailureScreen renders the failure result screen with rollback info
func (m AppModel) renderFailureScreen() string {
	// Build content (without container)
	content := m.buildFailureContent()

	// Help text using bubbles/help
	helpText := m.Help.View(m.FailureKeys)

	// Wrap with unified container
	return RenderApplicationContainer(content, helpText, m.Width, m.Height)
}

// buildFailureContent builds the failure screen content
func (m AppModel) buildFailureContent() string {
	var b strings.Builder

	b.WriteString(RenderTitle("✗ Configuration Update Failed"))
	b.WriteString("\n\n")

	if m.LastError != nil {
		errorBox := ErrorBoxStyle.Render(fmt.Sprintf("Error: %v", m.LastError))
		b.WriteString(errorBox)
		b.WriteString("\n\n")
	}

	if m.RolledBack {
		b.WriteString(WarningBoxStyle.Render("⚠ Automatically rolled back to previous configuration"))
		b.WriteString("\n\n")

		if m.CurrentConfig != nil {
			b.WriteString("Rollback successful - device restored to previous state:\n\n")

			config := fmt.Sprintf("  First Press:  %d (%s)\n", m.CurrentConfig.Outlet1, deviceconfig.FormatBitmask(m.CurrentConfig.Outlet1))
			config += fmt.Sprintf("  Second Press: %d (%s)\n", m.CurrentConfig.Outlet2, deviceconfig.FormatBitmask(m.CurrentConfig.Outlet2))
			config += fmt.Sprintf("  Third Press:  %d (%s)\n", m.CurrentConfig.Outlet3, deviceconfig.FormatBitmask(m.CurrentConfig.Outlet3))
			config += fmt.Sprintf("  K3 Mode:      %v", formatBool(m.CurrentConfig.K3Outlet))

			b.WriteString(config)
			b.WriteString("\n\n")
		}
	}

	// Troubleshooting hints
	b.WriteString("Troubleshooting:\n")
	b.WriteString("  • Check device is powered on and responsive\n")
	b.WriteString("  • Verify network connection to device\n")
	b.WriteString("  • Try refreshing configuration and retrying\n\n")

	b.WriteString("What would you like to do?\n\n")

	b.WriteString(MenuItemStyle.Render("  r - Retry the update"))
	b.WriteString("\n")
	b.WriteString(MenuItemStyle.Render("  e - Edit configuration again"))
	b.WriteString("\n")
	b.WriteString(MenuItemStyle.Render("  v - View current configuration"))
	b.WriteString("\n")
	b.WriteString(MenuItemStyle.Render("  q - Exit application"))
	b.WriteString("\n")

	return b.String()
}

// formatBool formats a boolean for display
func formatBool(b bool) string {
	if b {
		return "Enabled"
	}
	return "Disabled"
}
