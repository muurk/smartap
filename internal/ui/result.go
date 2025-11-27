package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ResultType indicates success or failure
type ResultType int

const (
	ResultSuccess ResultType = iota
	ResultFailure
	ResultWarning
)

// Result represents a result box (success, failure, or warning)
type Result struct {
	Type            ResultType        // Success, failure, or warning
	Title           string            // e.g., "Certificate injection complete"
	Details         map[string]string // Key-value details to display
	Error           error             // Error (for failure results)
	Troubleshooting []string          // Troubleshooting tips (for failure results)
	Width           int               // Terminal width
}

// NewSuccessResult creates a success result box
func NewSuccessResult(title string, details map[string]string) *Result {
	return &Result{
		Type:    ResultSuccess,
		Title:   title,
		Details: details,
		Width:   GetTerminalWidth(),
	}
}

// NewFailureResult creates a failure result box
func NewFailureResult(title string, err error, troubleshooting []string) *Result {
	return &Result{
		Type:            ResultFailure,
		Title:           title,
		Error:           err,
		Troubleshooting: troubleshooting,
		Width:           GetTerminalWidth(),
	}
}

// NewWarningResult creates a warning result box
func NewWarningResult(title string, details map[string]string) *Result {
	return &Result{
		Type:    ResultWarning,
		Title:   title,
		Details: details,
		Width:   GetTerminalWidth(),
	}
}

// SetWidth sets the terminal width for responsive rendering
func (r *Result) SetWidth(width int) *Result {
	r.Width = width
	return r
}

// AddDetail adds a detail key-value pair
func (r *Result) AddDetail(key, value string) *Result {
	if r.Details == nil {
		r.Details = make(map[string]string)
	}
	r.Details[key] = value
	return r
}

// Render returns the styled result box as a string
func (r *Result) Render() string {
	switch r.Type {
	case ResultSuccess:
		return r.renderSuccess()
	case ResultFailure:
		return r.renderFailure()
	case ResultWarning:
		return r.renderWarning()
	default:
		return r.renderSuccess()
	}
}

// renderSuccess renders a success result box
func (r *Result) renderSuccess() string {
	width := r.Width
	if width < MinTerminalWidth {
		width = MinTerminalWidth
	}

	var lines []string

	// Title with checkmark
	titleLine := SuccessTitleStyle.Render(fmt.Sprintf("   %s  SUCCESS  ─  %s", SuccessMarker, r.Title))
	lines = append(lines, "")
	lines = append(lines, titleLine)
	lines = append(lines, "")

	// Details
	for key, value := range r.Details {
		keyStyled := ResultKeyStyle.Render(fmt.Sprintf("   %s:", key))
		valueStyled := ResultValueStyle.Render(value)
		lines = append(lines, keyStyled+" "+valueStyled)
	}

	lines = append(lines, "")

	content := strings.Join(lines, "\n")

	// Double border in green
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(SuccessColor).
		Width(width - 2).
		Padding(0, 2).
		Render(content)
}

// renderFailure renders a failure result box
func (r *Result) renderFailure() string {
	width := r.Width
	if width < MinTerminalWidth {
		width = MinTerminalWidth
	}

	var lines []string

	// Title with X mark
	titleLine := ErrorTitleStyle.Render(fmt.Sprintf("   %s  FAILED  ─  %s", FailureMarker, r.Title))
	lines = append(lines, "")
	lines = append(lines, titleLine)
	lines = append(lines, "")

	// Error message
	if r.Error != nil {
		errorLine := ErrorMessageStyle.Render("   Error: " + r.Error.Error())
		lines = append(lines, errorLine)
		lines = append(lines, "")
	}

	// Troubleshooting section
	if len(r.Troubleshooting) > 0 {
		troubleBox := r.renderTroubleshootingBox(width)
		lines = append(lines, troubleBox)
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	// Double border in red
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ErrorColor).
		Width(width - 2).
		Padding(0, 2).
		Render(content)
}

// renderWarning renders a warning result box
func (r *Result) renderWarning() string {
	width := r.Width
	if width < MinTerminalWidth {
		width = MinTerminalWidth
	}

	var lines []string

	// Title with warning marker
	titleLine := lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(true).
		Render(fmt.Sprintf("   ⚠  WARNING  ─  %s", r.Title))
	lines = append(lines, "")
	lines = append(lines, titleLine)
	lines = append(lines, "")

	// Details
	for key, value := range r.Details {
		keyStyled := ResultKeyStyle.Render(fmt.Sprintf("   %s:", key))
		valueStyled := ResultValueStyle.Render(value)
		lines = append(lines, keyStyled+" "+valueStyled)
	}

	lines = append(lines, "")

	content := strings.Join(lines, "\n")

	// Double border in orange
	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(WarningColor).
		Width(width - 2).
		Padding(0, 2).
		Render(content)
}

// renderTroubleshootingBox renders the inner troubleshooting box
func (r *Result) renderTroubleshootingBox(width int) string {
	var lines []string

	// Title
	lines = append(lines, TroubleshootingTitleStyle.Render("Troubleshooting:"))
	lines = append(lines, "")

	// Bullet points
	for _, tip := range r.Troubleshooting {
		lines = append(lines, TroubleshootingItemStyle.Render("  • "+tip))
	}

	content := strings.Join(lines, "\n")

	// Inner box with muted border
	innerWidth := width - 12 // Indent within outer box
	if innerWidth < 40 {
		innerWidth = 40
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Width(innerWidth).
		Padding(0, 1).
		MarginLeft(3).
		Render(content)
}

// String implements fmt.Stringer
func (r *Result) String() string {
	return r.Render()
}

// --- Convenience functions for quick rendering ---

// RenderSuccess renders a success box with the given title and details
func RenderSuccess(title string, details map[string]string) string {
	return NewSuccessResult(title, details).Render()
}

// RenderFailure renders a failure box with the given title, error, and troubleshooting tips
func RenderFailure(title string, err error, troubleshooting []string) string {
	return NewFailureResult(title, err, troubleshooting).Render()
}

// RenderWarning renders a warning box with the given title and details
func RenderWarning(title string, details map[string]string) string {
	return NewWarningResult(title, details).Render()
}
