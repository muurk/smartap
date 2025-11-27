package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// Color palette for GDB command UI
var (
	// Primary colors
	PrimaryColor = lipgloss.Color("#7D56F4") // Purple - headers, borders
	SuccessColor = lipgloss.Color("#43BF6D") // Green - success, checkmarks
	ErrorColor   = lipgloss.Color("#FF5555") // Red - errors, X marks
	WarningColor = lipgloss.Color("#FFA500") // Orange - warnings
	MutedColor   = lipgloss.Color("#626262") // Gray - secondary info
	TextColor    = lipgloss.Color("#FFFFFF") // White - main content
)

// Layout constants
const (
	MinTerminalWidth = 60  // Minimum supported terminal width
	MaxContentWidth  = 100 // Maximum content width before capping
	DefaultPadding   = 2   // Default padding inside boxes
)

// Shared styles for GDB command UI
var (
	// HeaderTitleStyle is for the main command title (e.g., "CERTIFICATE INJECTION")
	HeaderTitleStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				Bold(true).
				PaddingLeft(2)

	// HeaderCommandStyle is for the command path (e.g., "smartap-cfg gdb inject-certs")
	HeaderCommandStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				PaddingLeft(2)

	// HeaderParamKeyStyle is for parameter keys (e.g., "Device:")
	HeaderParamKeyStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				PaddingLeft(2)

	// HeaderParamValueStyle is for parameter values (e.g., "localhost:3333")
	HeaderParamValueStyle = lipgloss.NewStyle().
				Foreground(TextColor)

	// ProgressLabelStyle is for "Injecting certificate..."
	ProgressLabelStyle = lipgloss.NewStyle().
				Foreground(TextColor).
				PaddingLeft(2)

	// StepCompleteStyle is for completed step text
	StepCompleteStyle = lipgloss.NewStyle().
				Foreground(SuccessColor)

	// StepRunningStyle is for currently running step text
	StepRunningStyle = lipgloss.NewStyle().
				Foreground(WarningColor)

	// StepPendingStyle is for pending step text
	StepPendingStyle = lipgloss.NewStyle().
				Foreground(MutedColor)

	// StepNoteStyle is for optional notes in parentheses
	StepNoteStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true)

	// SuccessTitleStyle is for the success result title
	SuccessTitleStyle = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Bold(true)

	// ErrorTitleStyle is for the error result title
	ErrorTitleStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	// ErrorMessageStyle is for error message text
	ErrorMessageStyle = lipgloss.NewStyle().
				Foreground(ErrorColor)

	// ResultKeyStyle is for result detail keys
	ResultKeyStyle = lipgloss.NewStyle().
			Foreground(MutedColor).
			Width(15)

	// ResultValueStyle is for result detail values
	ResultValueStyle = lipgloss.NewStyle().
				Foreground(TextColor)

	// TroubleshootingTitleStyle is for "Troubleshooting:" headers
	TroubleshootingTitleStyle = lipgloss.NewStyle().
					Foreground(MutedColor).
					Bold(true)

	// TroubleshootingItemStyle is for troubleshooting bullet points
	TroubleshootingItemStyle = lipgloss.NewStyle().
					Foreground(MutedColor)

	// GDBOutputTitleStyle is for "GDB Output" header
	GDBOutputTitleStyle = lipgloss.NewStyle().
				Foreground(MutedColor).
				Bold(true)

	// GDBOutputContentStyle is for GDB output content
	GDBOutputContentStyle = lipgloss.NewStyle().
				Foreground(TextColor)
)

// Step status markers
const (
	StepMarkerComplete = "✓"
	StepMarkerRunning  = "●"
	StepMarkerPending  = "·"
	SuccessMarker      = "✓"
	FailureMarker      = "✗"
)

// GetTerminalWidth returns the current terminal width, with fallback
func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < MinTerminalWidth {
		return MinTerminalWidth
	}
	if width > MaxContentWidth {
		return MaxContentWidth
	}
	return width
}

// GetTerminalSize returns the current terminal width and height
func GetTerminalSize() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return MinTerminalWidth, 24 // Default fallback
	}
	if width < MinTerminalWidth {
		width = MinTerminalWidth
	}
	if width > MaxContentWidth {
		width = MaxContentWidth
	}
	return width, height
}

// HeaderBorderStyle returns the border style for command headers
func HeaderBorderStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(width - 2) // Account for border characters
}

// HeaderDividerStyle returns a horizontal divider for inside headers
func HeaderDividerStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Width(width - 4) // Account for border and padding
}

// SuccessBoxStyle returns the border style for success result boxes
func SuccessBoxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(SuccessColor).
		Width(width - 2).
		Padding(1, 2)
}

// ErrorBoxStyle returns the border style for error result boxes
func ErrorBoxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ErrorColor).
		Width(width - 2).
		Padding(1, 2)
}

// GDBOutputBoxStyle returns the border style for GDB output boxes
func GDBOutputBoxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Width(width - 4). // Smaller than full width
		Padding(0, 1)
}

// TroubleshootingBoxStyle returns the border style for troubleshooting sections
func TroubleshootingBoxStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Width(width - 8). // Indented within error box
		Padding(0, 1)
}

// ProgressBarStyle returns a style for the progress bar container
func ProgressBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		PaddingLeft(2)
}

// RenderHorizontalDivider creates a horizontal line of the specified width
func RenderHorizontalDivider(width int, char string) string {
	result := ""
	for i := 0; i < width; i++ {
		result += char
	}
	return lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Render(result)
}
