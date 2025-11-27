package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muurk/smartap/internal/discovery"
)

// Messages for async operations
type scanStartMsg struct{}
type scanCompleteMsg struct {
	devices []*discovery.Device
	err     error
}

// discoveryKeyMap defines key bindings for the discovery screen
type discoveryKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Rescan  key.Binding
	Manual  key.Binding
	Quit    key.Binding
	Confirm key.Binding // For manual mode
	Cancel  key.Binding // For manual mode
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k discoveryKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Rescan, k.Manual, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k discoveryKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Rescan, k.Manual, k.Quit},
	}
}

// manualModeKeyMap defines key bindings for manual IP entry mode
type manualModeKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (m manualModeKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{m.Confirm, m.Cancel}
}

// FullHelp returns keybindings for the expanded help view
func (m manualModeKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{m.Confirm, m.Cancel},
	}
}

// scanningKeyMap defines key bindings for scanning mode
type scanningKeyMap struct {
	Manual key.Binding
	Quit   key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (s scanningKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{s.Manual, s.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (s scanningKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{s.Manual, s.Quit},
	}
}

// emptyScreenKeyMap defines key bindings for empty results screen
type emptyScreenKeyMap struct {
	Rescan key.Binding
	Manual key.Binding
	Quit   key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (e emptyScreenKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{e.Rescan, e.Manual, e.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (e emptyScreenKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{e.Rescan, e.Manual, e.Quit},
	}
}

// deviceItem wraps a Device for use with bubbles/list
type deviceItem struct {
	device *discovery.Device
}

// Implement list.Item interface
func (d deviceItem) FilterValue() string {
	// Filter by serial, IP, or hostname
	return d.device.Serial + " " + d.device.IP + " " + d.device.Hostname
}

// Title returns the device name for list display
func (d deviceItem) Title() string {
	if d.device.Serial == "manual" {
		return fmt.Sprintf("Manual: %s", d.device.IP)
	}
	return fmt.Sprintf("eValve%s", d.device.Serial)
}

// Description returns device details for list display
func (d deviceItem) Description() string {
	firmware := "Unknown"
	if fw, ok := d.device.Metadata["firmware"]; ok {
		firmware = fw
	}
	return fmt.Sprintf("%s:%d • Firmware: %s • Ready", d.device.IP, d.device.Port, firmware)
}

// deviceDelegate is a custom list delegate for rendering device cards
type deviceDelegate struct {
	width int
}

func (d deviceDelegate) Height() int { return 8 } // Card height including borders

func (d deviceDelegate) Spacing() int { return 1 } // Spacing between cards

func (d deviceDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d deviceDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	deviceItem, ok := item.(deviceItem)
	if !ok {
		return
	}

	device := deviceItem.device
	selected := index == m.Index()

	// Build device name
	var deviceName string
	if device.Serial == "manual" {
		deviceName = fmt.Sprintf("Manual: %s", device.IP)
	} else {
		deviceName = fmt.Sprintf("eValve%s", device.Serial)
	}

	// Get firmware version
	firmware := "Unknown"
	if fw, ok := device.Metadata["firmware"]; ok {
		firmware = fw
	}

	// Build content lines
	var content strings.Builder

	// Add selection indicator to device name
	if selected {
		content.WriteString(SelectedMenuItemStyle.Render("→ " + deviceName))
	} else {
		content.WriteString("  " + deviceName)
	}
	content.WriteString("\n\n")

	// Device details
	content.WriteString(fmt.Sprintf("  Serial:   %s\n", device.Serial))
	content.WriteString(fmt.Sprintf("  IP:       %s:%d\n", device.IP, device.Port))
	content.WriteString(fmt.Sprintf("  Firmware: %s\n", firmware))

	// Status with inline color styling (no border)
	statusStyle := lipgloss.NewStyle().Foreground(SecondaryColor).Bold(true)
	content.WriteString(fmt.Sprintf("  Status:   %s", statusStyle.Render("Ready")))

	// Create responsive card style
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(BorderColor).
		Padding(1, 2).
		MarginLeft(2)

	// Calculate card width (leave room for margins and borders)
	cardWidth := d.width - 6 // 2 for margin-left, 4 for border + padding
	if cardWidth < MinTerminalWidth-6 {
		cardWidth = MinTerminalWidth - 6
	}
	if cardWidth > MaxContentWidth-6 {
		cardWidth = MaxContentWidth - 6
	}

	cardStyle = cardStyle.Width(cardWidth)

	// Highlight selected card
	if selected {
		cardStyle = cardStyle.BorderForeground(HighlightColor)
	}

	fmt.Fprint(w, cardStyle.Render(content.String()))
}

// DiscoveryModel represents the device discovery screen state
type DiscoveryModel struct {
	// Discovery state
	Scanning   bool
	DeviceList list.Model
	Selected   bool
	Err        error

	// Manual IP entry state
	ManualMode bool
	IPInput    textinput.Model

	// UI state
	Width         int
	Height        int
	Spinner       spinner.Model
	ProgressBar   progress.Model
	ScanStartTime time.Time
	Help          help.Model
	Keys          discoveryKeyMap
	ManualKeys    manualModeKeyMap
	ScanningKeys  scanningKeyMap
	EmptyKeys     emptyScreenKeyMap
}

// NewDiscoveryModel creates a new discovery screen model
func NewDiscoveryModel() DiscoveryModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	// Initialize IP input
	ipInput := textinput.New()
	ipInput.Placeholder = "192.168.4.1"
	ipInput.CharLimit = 15 // Max length for IPv4 address
	ipInput.Width = 30

	// Initialize progress bar
	progressBar := progress.New(progress.WithDefaultGradient())
	progressBar.Width = 40

	// Initialize device list with custom delegate
	delegate := deviceDelegate{width: MinTerminalWidth}
	deviceList := list.New([]list.Item{}, delegate, 0, 0)
	deviceList.Title = "Discovered Devices"
	deviceList.SetShowStatusBar(false)
	deviceList.SetFilteringEnabled(true)
	deviceList.Styles.Title = TitleStyle

	// Initialize help
	h := help.New()

	// Initialize key bindings for normal mode
	keys := discoveryKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "configure"),
		),
		Rescan: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rescan"),
		),
		Manual: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "manual IP"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q", "quit"),
		),
	}

	// Initialize key bindings for manual entry mode
	manualKeys := manualModeKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}

	// Initialize key bindings for scanning mode
	scanningKeys := scanningKeyMap{
		Manual: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "manual IP"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}

	// Initialize key bindings for empty results
	emptyKeys := emptyScreenKeyMap{
		Rescan: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rescan"),
		),
		Manual: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "manual IP"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
	}

	return DiscoveryModel{
		Scanning:     false,
		DeviceList:   deviceList,
		Selected:     false,
		ManualMode:   false,
		IPInput:      ipInput,
		Spinner:      s,
		ProgressBar:  progressBar,
		Help:         h,
		Keys:         keys,
		ManualKeys:   manualKeys,
		ScanningKeys: scanningKeys,
		EmptyKeys:    emptyKeys,
	}
}

// Init initializes the discovery model
func (m DiscoveryModel) Init() tea.Cmd {
	// Start scanning immediately - send start message then begin scan
	return tea.Batch(
		func() tea.Msg { return scanStartMsg{} },
		scanDevices,
		m.Spinner.Tick,
	)
}

// Update handles messages and updates the model
func (m DiscoveryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.ManualMode {
			return m.updateManualMode(msg)
		}
		return m.updateNormalMode(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		// Update list size
		m.DeviceList.SetWidth(msg.Width - 4)
		m.DeviceList.SetHeight(msg.Height - 10) // Leave room for header/footer

	case scanStartMsg:
		m.Scanning = true
		m.ScanStartTime = time.Now()

	case scanCompleteMsg:
		m.Scanning = false
		m.Err = msg.err
		// Convert devices to list items
		items := make([]list.Item, len(msg.devices))
		for i, dev := range msg.devices {
			items[i] = deviceItem{device: dev}
		}
		m.DeviceList.SetItems(items)

	case spinner.TickMsg:
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}

	// Update list if not in manual mode or scanning
	if !m.ManualMode && !m.Scanning {
		m.DeviceList, cmd = m.DeviceList.Update(msg)
	}

	return m, cmd
}

// updateNormalMode handles keyboard input in normal device list mode
func (m DiscoveryModel) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		// Go back to main menu (in real integration)
		return m, tea.Quit

	case "enter", " ":
		// Get selected device from list
		if selectedItem := m.DeviceList.SelectedItem(); selectedItem != nil {
			m.Selected = true
			// In real integration, this would transition to config display
			return m, tea.Quit
		}

	case "r":
		// Rescan
		m.DeviceList.SetItems([]list.Item{})
		m.Err = nil
		return m, tea.Batch(
			func() tea.Msg { return scanStartMsg{} },
			scanDevices,
			m.Spinner.Tick,
		)

	case "m":
		// Switch to manual IP entry mode
		m.ManualMode = true
		m.IPInput.SetValue("")
		m.IPInput.Focus()
	}

	// Let the list handle up/down navigation
	return m, nil
}

// updateManualMode handles keyboard input in manual IP entry mode
func (m DiscoveryModel) updateManualMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "ctrl+c", "esc":
		// Cancel manual entry
		m.ManualMode = false
		m.IPInput.SetValue("")
		m.IPInput.Blur()
		return m, nil

	case "enter":
		value := m.IPInput.Value()
		if value != "" {
			// Create device from manual IP
			device := &discovery.Device{
				IP:           value,
				Port:         80,
				Hostname:     value,
				Serial:       "manual",
				DiscoveredAt: time.Now(),
			}
			// Add to list
			newItem := deviceItem{device: device}
			items := append([]list.Item{newItem}, m.DeviceList.Items()...)
			m.DeviceList.SetItems(items)
			m.DeviceList.Select(0) // Select the newly added item
			m.ManualMode = false
			m.IPInput.SetValue("")
			m.IPInput.Blur()
			return m, nil
		}
	}

	// Update the text input
	m.IPInput, cmd = m.IPInput.Update(msg)
	return m, cmd
}

// View renders the discovery screen
func (m DiscoveryModel) View() string {
	// Use default width if not set
	width := m.Width
	if width == 0 {
		width = 72
	}

	// Build main content area
	var content string
	if m.ManualMode {
		content = m.renderManualEntry()
	} else if m.Scanning {
		content = m.renderScanningEnhanced(width)
	} else {
		content = m.renderDeviceResults()
	}

	// Determine context-sensitive help text using bubbles/help
	var helpText string
	if m.ManualMode {
		helpText = m.Help.View(m.ManualKeys)
	} else if m.Scanning {
		helpText = m.Help.View(m.ScanningKeys)
	} else if len(m.DeviceList.Items()) > 0 {
		helpText = m.Help.View(m.Keys)
	} else {
		helpText = m.Help.View(m.EmptyKeys)
	}

	// Wrap with application container (full-screen layout with height)
	return RenderApplicationContainer(content, helpText, m.Width, m.Height)
}

// renderScanningEnhanced renders a prominent, centered scanning progress display
// Renders the discovery screen using Lipgloss placement for automatic centering
func (m DiscoveryModel) renderScanningEnhanced(width int) string {
	elapsed := time.Since(m.ScanStartTime)
	elapsedSec := int(elapsed.Seconds())

	// Calculate progress (simulate - 10 second scan)
	progressPercent := min(100, (elapsedSec*100)/10)
	progressFloat := float64(progressPercent) / 100.0

	// Build content components
	title := fmt.Sprintf("%s SEARCHING FOR DEVICES", m.Spinner.View())
	subtitle := "Scanning your network for Smartap devices..."

	// Use bubbles/progress component (ViewAs already includes percentage display)
	progressBar := m.ProgressBar.ViewAs(progressFloat)
	elapsedText := fmt.Sprintf("Elapsed: %ds", elapsedSec)

	// Use lipgloss.JoinVertical for layout composition
	content := lipgloss.JoinVertical(lipgloss.Center,
		"", // Top spacing
		TitleStyle.Render(title),
		"",
		SubtitleStyle.Render(subtitle),
		"",
		progressBar,
		"",
		SubtitleStyle.Render(elapsedText),
		"", // Bottom spacing
	)

	// Use lipgloss.Place for centering (not manual padding!)
	// Height = 0 means "no vertical constraint" - let content determine height
	return lipgloss.Place(width, 0, lipgloss.Center, lipgloss.Top, content)
}

// renderDeviceResults renders the device list or "no devices found" message
func (m DiscoveryModel) renderDeviceResults() string {
	var b strings.Builder

	b.WriteString("\n")

	if m.Err != nil {
		// Error state
		b.WriteString(RenderError(fmt.Sprintf("Scan failed: %v", m.Err)))
		b.WriteString("\n\n")

		// Troubleshooting hints
		b.WriteString("  Troubleshooting:\n")
		b.WriteString("    • Ensure device is powered on\n")
		b.WriteString("    • Check that device is in pairing mode (LED flashing)\n")
		b.WriteString("    • Verify you're connected to device's WiFi hotspot\n")
		b.WriteString("    • Try increasing scan time (use 'r' to rescan)\n")

	} else if len(m.DeviceList.Items()) == 0 {
		// No devices found
		b.WriteString("  ")
		warningStyle := lipgloss.NewStyle().Foreground(WarningColor).Bold(true)
		b.WriteString(warningStyle.Render("⚠ No devices found on your network"))
		b.WriteString("\n\n")

		b.WriteString("  Troubleshooting:\n")
		b.WriteString("    • Ensure device is powered on\n")
		b.WriteString("    • Check that device is in pairing mode (LED flashing)\n")
		b.WriteString("    • Verify you're connected to device's WiFi hotspot\n")
		b.WriteString("    • Try increasing scan time (use 'r' to rescan)\n")
		b.WriteString("\n")

	} else {
		// Devices found - render the list
		b.WriteString(m.DeviceList.View())
	}

	return b.String()
}

// renderScanning renders the scanning progress indicator (legacy - keeping for compatibility)
func (m DiscoveryModel) renderScanning() string {
	elapsed := time.Since(m.ScanStartTime).Round(time.Second)
	status := fmt.Sprintf("%s Scanning network for Smartap devices... (%s)", m.Spinner.View(), elapsed)

	return SpinnerStyle.Render(status) + "\n\n"
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// renderManualEntry renders the manual IP entry dialog
func (m DiscoveryModel) renderManualEntry() string {
	var b strings.Builder

	b.WriteString(RenderSubtitle("Enter device IP address"))
	b.WriteString("\n\n")

	// Input box using textinput component
	b.WriteString("  IP Address: ")
	b.WriteString(m.IPInput.View())
	b.WriteString("\n\n")

	return b.String()
}

// GetSelectedDevice returns the selected device (if any)
func (m DiscoveryModel) GetSelectedDevice() *discovery.Device {
	if m.Selected {
		if selectedItem := m.DeviceList.SelectedItem(); selectedItem != nil {
			if item, ok := selectedItem.(deviceItem); ok {
				return item.device
			}
		}
	}
	return nil
}

// scanDevices is a command that performs device discovery
func scanDevices() tea.Msg {
	// Signal that scan is starting
	scanner := discovery.NewScanner()
	scanner.Timeout = 10 * time.Second

	devices, err := scanner.ScanForDevices()
	return scanCompleteMsg{
		devices: devices,
		err:     err,
	}
}
