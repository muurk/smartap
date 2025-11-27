package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Header represents a command header with title, command, and parameters.
// Used at the start of each GDB command to provide context.
type Header struct {
	Title   string            // e.g., "CERTIFICATE INJECTION"
	Command string            // e.g., "smartap-cfg gdb inject-certs"
	Params  map[string]string // e.g., {"Device": "localhost:3333", "Certificate": "..."}
	Width   int               // Terminal width for responsive rendering
}

// NewHeader creates a new header with the given values
func NewHeader(title, command string, params map[string]string) *Header {
	return &Header{
		Title:   title,
		Command: command,
		Params:  params,
		Width:   GetTerminalWidth(),
	}
}

// SetWidth sets the terminal width for responsive rendering
func (h *Header) SetWidth(width int) *Header {
	h.Width = width
	return h
}

// Render returns the styled header as a string
func (h *Header) Render() string {
	width := h.Width
	if width < MinTerminalWidth {
		width = MinTerminalWidth
	}

	var b strings.Builder

	// Title line - uppercase and bold
	titleLine := HeaderTitleStyle.Render(strings.ToUpper(h.Title))

	// Command line - muted
	commandLine := HeaderCommandStyle.Render(h.Command)

	// Build top section
	topSection := lipgloss.JoinVertical(lipgloss.Left, titleLine, commandLine)

	// Divider line
	dividerWidth := width - 6 // Account for border and padding
	if dividerWidth < 10 {
		dividerWidth = 10
	}
	divider := lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Render(strings.Repeat("─", dividerWidth))

	// Build params section
	var paramLines []string
	for key, value := range h.Params {
		// Format: "  Key:   Value" with aligned colons
		keyStyled := HeaderParamKeyStyle.Render(key + ":")
		valueStyled := HeaderParamValueStyle.Render(value)
		paramLines = append(paramLines, keyStyled+" "+valueStyled)
	}
	paramsSection := strings.Join(paramLines, "\n")

	// Combine all sections vertically
	var content string
	if len(h.Params) > 0 {
		content = lipgloss.JoinVertical(lipgloss.Left, topSection, divider, paramsSection)
	} else {
		content = topSection
	}

	// Apply rounded border with primary color
	bordered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(PrimaryColor).
		Width(width - 2). // Account for border characters
		Render(content)

	b.WriteString(bordered)
	return b.String()
}

// String implements fmt.Stringer
func (h *Header) String() string {
	return h.Render()
}

// HeaderConfig is a convenience type for creating headers
type HeaderConfig struct {
	Title   string
	Command string
	Params  map[string]string
}

// RenderCommandHeader is a convenience function to render a header directly
func RenderCommandHeader(config HeaderConfig) string {
	return NewHeader(config.Title, config.Command, config.Params).Render()
}
