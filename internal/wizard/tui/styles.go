package tui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/muurk/smartap/internal/version"
)

// Application branding constants
const (
	AppName       = "SMARTAP CONFIGURATION WIZARD"
	GitHubURL     = "github.com/muurk/smartap"
	GitHubFullURL = "https://github.com/muurk/smartap"
)

// AppVersion returns the application version from the centralized version package
func AppVersion() string {
	return version.Version
}

// Layout constants for responsive terminal width
const (
	MinTerminalWidth  = 72  // Minimum supported terminal width
	MaxContentWidth   = 120 // Maximum content width before capping
	DefaultBoxPadding = 2   // Default padding inside boxes
)

// Color palette
var (
	// Primary colors
	PrimaryColor   = lipgloss.Color("#7D56F4") // Purple
	SecondaryColor = lipgloss.Color("#43BF6D") // Green
	AccentColor    = lipgloss.Color("#FF8B94") // Pink
	WarningColor   = lipgloss.Color("#FFA500") // Orange
	ErrorColor     = lipgloss.Color("#FF0000") // Red

	// Neutral colors
	TextColor       = lipgloss.Color("#FFFFFF") // White
	SubtleColor     = lipgloss.Color("#626262") // Gray
	BorderColor     = lipgloss.Color("#7D56F4") // Purple (same as primary)
	HighlightColor  = lipgloss.Color("#43BF6D") // Green (same as secondary)
	BackgroundColor = lipgloss.Color("#1A1A1A") // Dark gray
)

// Common styles
var (
	// Title style - large, bold, centered
	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Padding(1, 0).
			MarginBottom(1)

	// Subtitle style
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(SubtleColor).
			Italic(true)

	// Menu item style (unselected)
	MenuItemStyle = lipgloss.NewStyle().
			PaddingLeft(4).
			Foreground(TextColor)

	// Menu item style (selected)
	SelectedMenuItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(HighlightColor).
				Bold(true)

	// Help text style
	HelpStyle = lipgloss.NewStyle().
			Foreground(SubtleColor).
			Padding(1, 0)

	// Error message style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ErrorColor)

	// Success message style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor)

	// Info box style
	InfoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	// Status bar style
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(SubtleColor).
			Background(BackgroundColor).
			Padding(0, 1)

	// Spinner style
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor)

	// List item style
	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	// Selected list item style
	SelectedListItemStyle = lipgloss.NewStyle().
				PaddingLeft(0).
				Foreground(HighlightColor).
				Bold(true)

	// Box style for containers
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(BorderColor).
			Padding(1, 2)

	// Focused input style
	FocusedInputStyle = lipgloss.NewStyle().
				Foreground(PrimaryColor).
				Bold(true)

	// Blurred input style
	BlurredInputStyle = lipgloss.NewStyle().
				Foreground(SubtleColor)

	// Success box style (for result screens)
	SuccessBoxStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	// Error box style (for result screens)
	ErrorBoxStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ErrorColor).
			Padding(1, 2)

	// Warning box style (for result screens)
	WarningBoxStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(WarningColor).
			Padding(1, 2)
)

// RenderTitle renders a title with consistent styling
func RenderTitle(text string) string {
	return TitleStyle.Render(text)
}

// RenderSubtitle renders a subtitle with consistent styling
func RenderSubtitle(text string) string {
	return SubtitleStyle.Render(text)
}

// RenderMenuItem renders a menu item with selection indicator
func RenderMenuItem(text string, selected bool) string {
	if selected {
		return SelectedMenuItemStyle.Render("→ " + text)
	}
	return MenuItemStyle.Render("  " + text)
}

// RenderHelp renders help text
func RenderHelp(text string) string {
	return HelpStyle.Render(text)
}

// RenderError renders an error message
func RenderError(text string) string {
	return ErrorStyle.Render("✗ " + text)
}

// RenderSuccess renders a success message
func RenderSuccess(text string) string {
	return SuccessStyle.Render("✓ " + text)
}

// RenderInfo renders an info box
func RenderInfo(text string) string {
	return InfoBoxStyle.Render(text)
}

// BuildHeaderContent creates header content with app name and GitHub URL
// Returns a string formatted for use in the application container
func BuildHeaderContent() string {
	left := lipgloss.NewStyle().
		Foreground(TextColor).
		Bold(true).
		Render(AppName + " v" + AppVersion())

	right := lipgloss.NewStyle().
		Foreground(SubtleColor).
		Render(GitHubURL)

	// Join with space in between
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

// BuildFooterContent creates footer content with help text
// Returns a styled string for use in the application container
func BuildFooterContent(helpText string) string {
	return lipgloss.NewStyle().
		Foreground(SubtleColor).
		Render(helpText)
}

// RenderApplicationContainer is the REQUIRED wrapper for all screens in the application.
// It provides:
// - Consistent full-screen panel using terminal width/height
// - Application header (name, version, GitHub URL)
// - Context-sensitive footer (help text)
// - Bordered outer container
// - Proper viewport support
//
// EVERY screen must use this function. Pattern:
//
//	func (m Model) View() string {
//	    content := m.buildContent()
//	    helpText := "context-specific help..."
//	    return RenderApplicationContainer(content, helpText, m.Width, m.Height)
//	}
//
// Parameters:
//   - content: The screen's main content (rendered separately)
//   - footerText: Context-sensitive help text for this screen
//   - terminalWidth: Current terminal width (from tea.WindowSizeMsg)
//   - terminalHeight: Current terminal height (from tea.WindowSizeMsg)
//
// Uses lipgloss.Place() to fill the entire terminal and pin footer to bottom
func RenderApplicationContainer(content string, footerText string, terminalWidth int, terminalHeight int) string {
	// Build header content
	header := BuildHeaderContent()

	// Build footer content
	footer := BuildFooterContent(footerText)

	// Create header section with bottom border
	headerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Bottom: "─"}).
		BorderForeground(BorderColor).
		Width(terminalWidth-4). // Leave room for outer border
		Padding(0, 1)

	styledHeader := headerStyle.Render(header)

	// Create footer section with top border
	footerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Top: "─"}).
		BorderForeground(BorderColor).
		Width(terminalWidth-4). // Leave room for outer border
		Padding(0, 1)

	styledFooter := footerStyle.Render(footer)

	// Create content area (viewport handles height for scrolling)
	// Note: No padding here - callers control their own content margins
	// This ensures Width(terminalWidth-4) is the actual usable content width
	contentStyle := lipgloss.NewStyle().
		Width(terminalWidth - 4) // Leave room for outer border

	styledContent := contentStyle.Render(content)

	// Combine header + content + footer vertically
	innerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		styledHeader,
		styledContent,
		styledFooter,
	)

	// Create outer border container with full terminal height for proper modal overlay background
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(BorderColor).
		Width(terminalWidth - 2).   // Account for border width
		Height(terminalHeight - 2). // Full height for proper background
		AlignVertical(lipgloss.Top) // Align content to top, preventing footer expansion

	bordered := borderStyle.Render(innerContent)

	// Use lipgloss.Place to fill the full terminal and ensure proper positioning
	return lipgloss.Place(
		terminalWidth,
		terminalHeight,
		lipgloss.Left,
		lipgloss.Top,
		bordered,
	)
}

// FormatBitmask formats an outlet bitmask (0-7) into a simple numbered format
// Returns formats like: "[0] None", "[1] Outlet 1", "[3] Outlets 1+2", "[7] Outlets 1+2+3"
func FormatBitmask(mask int) string {
	if mask == 0 {
		return "[0] None"
	}

	var outlets []string
	if mask&1 != 0 {
		outlets = append(outlets, "1")
	}
	if mask&2 != 0 {
		outlets = append(outlets, "2")
	}
	if mask&4 != 0 {
		outlets = append(outlets, "3")
	}

	if len(outlets) == 0 {
		return "[0] None"
	}

	if len(outlets) == 1 {
		return "[" + string(rune('0'+mask)) + "] Outlet " + outlets[0]
	}

	return "[" + string(rune('0'+mask)) + "] Outlets " + joinStrings(outlets, "+")
}

// FormatK3Mode returns a human-readable description of K3 mode
func FormatK3Mode(enabled bool) string {
	if enabled {
		return "Enabled (separate)"
	}
	return "Disabled (sequential)"
}

// joinStrings joins a slice of strings with a separator
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// CalculateBoxWidth calculates the appropriate box width based on terminal width
// Uses full terminal width for maximum screen usage
func CalculateBoxWidth(terminalWidth int) int {
	if terminalWidth < MinTerminalWidth {
		return MinTerminalWidth
	}
	// Use full terminal width - no maximum cap for full-screen layout
	return terminalWidth
}

// SafePadding calculates safe padding that won't cause wrapping
// Returns 0 if width is too small for the requested padding
func SafePadding(width, requestedPadding int) int {
	if width < MinTerminalWidth {
		return 0
	}
	if requestedPadding*2 >= width {
		return 0
	}
	return requestedPadding
}

// SafeModalWidth calculates a safe modal width that respects terminal constraints
// Returns the minimum of requestedWidth and (terminalWidth - margin)
// Ensures modals never exceed terminal width and cause horizontal overflow
func SafeModalWidth(requestedWidth, terminalWidth int) int {
	// Leave margin for borders and padding (4 chars: 2 for border, 2 for spacing)
	maxWidth := terminalWidth - 4

	// Ensure we don't go below minimum usable width
	if maxWidth < 40 {
		maxWidth = 40 // Absolute minimum for usability
	}

	// Return the smaller of requested width and max width
	if requestedWidth < maxWidth {
		return requestedWidth
	}
	return maxWidth
}

// RenderModal renders result modals (progress, success, failure) centered on screen.
// NOTE: This is NOT used for configuration editing - that uses inline editors.
// Only used for temporary status overlays during apply operations.
//
// Uses lipgloss.Place() for automatic centering and overlay rendering.
//
// Parameters:
//   - background: The base view to overlay the modal on top of (currently unused but kept for API compatibility)
//   - modalContent: The styled modal content to display
//   - terminalWidth: Current terminal width
//   - terminalHeight: Current terminal height
//
// The modal content should already be styled with borders, padding, etc.
// RenderModal is DEPRECATED for dashboard use.
// All configuration status panels should use inline panel functions instead:
//   - renderInlineProgressPanel() for applying progress
//   - renderInlineSuccessPanel() for success feedback
//   - renderInlineErrorPanel() for error feedback
//   - renderInlineWiFiWarningPanel() for WiFi change warnings
//
// This function is kept only for the Help modal overlay, which benefits from
// covering the entire screen to focus user attention on help content.
//
// DO NOT use this for new features - use inline panels instead.
//
// Note: lipgloss.Place automatically handles the overlay, so the background parameter
// is currently ignored. It's kept for backwards compatibility.
func RenderModal(background string, modalContent string, terminalWidth int, terminalHeight int) string {
	// Use lipgloss.Place to center the modal on top of the background
	// The WithWhitespaceChars and WithWhitespaceForeground options create
	// a semi-transparent overlay effect by dimming the background
	return lipgloss.Place(
		terminalWidth,
		terminalHeight,
		lipgloss.Center,
		lipgloss.Center,
		modalContent,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("240")),
	)
}

// InlineEditorStyle returns styling for inline expanded editors
// Used when a configuration field is being edited inline (not in a modal)
func InlineEditorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.Border{
			Top:    "━",
			Bottom: "━",
			Left:   "┃",
			Right:  "┃",
		}).
		BorderForeground(PrimaryColor).
		Padding(0, 1)
}

// ExpandedFieldStyle returns styling for fields being edited inline
// Provides a subtle highlight to indicate the field is in edit mode
func ExpandedFieldStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Background(lipgloss.Color("236")) // Subtle dark gray highlight
}
