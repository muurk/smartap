package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RunOnceModel is a Bubble Tea model that renders once and exits.
// This is used for "run once and exit" output patterns rather than
// interactive TUIs.
type RunOnceModel struct {
	content string
	width   int
	height  int
	done    bool
}

// NewRunOnceModel creates a model that will render the given content and exit
func NewRunOnceModel(content string) RunOnceModel {
	width, height := GetTerminalSize()
	return RunOnceModel{
		content: content,
		width:   width,
		height:  height,
		done:    false,
	}
}

// Init implements tea.Model
func (m RunOnceModel) Init() tea.Cmd {
	// Immediately signal we're done after first render
	return tea.Quit
}

// Update implements tea.Model
func (m RunOnceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.WindowSizeMsg:
		// Update dimensions but we're about to quit anyway
		m.width, m.height = GetTerminalSize()
	}
	return m, nil
}

// View implements tea.Model
func (m RunOnceModel) View() string {
	return m.content
}

// RenderOnce renders content using Bubble Tea's rendering engine and immediately exits.
// This provides consistent terminal rendering without requiring user interaction.
func RenderOnce(content string) error {
	model := NewRunOnceModel(content)
	p := tea.NewProgram(model, tea.WithOutput(os.Stdout))
	_, err := p.Run()
	return err
}

// PrintStyled prints styled content directly to stdout without Bubble Tea.
// Use this for simpler output that doesn't need the full rendering engine.
func PrintStyled(content string) {
	fmt.Print(content)
}

// PrintStyledLine prints styled content with a newline.
func PrintStyledLine(content string) {
	fmt.Println(content)
}

// Printer provides methods for printing UI components to a writer.
// This is the primary way GDB commands should output styled content.
type Printer struct {
	out   io.Writer
	width int
}

// NewPrinter creates a new Printer that writes to the given writer.
// If w is nil, os.Stdout is used.
func NewPrinter(w io.Writer) *Printer {
	if w == nil {
		w = os.Stdout
	}
	return &Printer{
		out:   w,
		width: GetTerminalWidth(),
	}
}

// Width returns the current terminal width used by this printer
func (p *Printer) Width() int {
	return p.width
}

// Print writes content to the output
func (p *Printer) Print(content string) {
	_, _ = fmt.Fprint(p.out, content)
}

// Println writes content with a newline
func (p *Printer) Println(content string) {
	_, _ = fmt.Fprintln(p.out, content)
}

// PrintLines writes multiple lines
func (p *Printer) PrintLines(lines ...string) {
	for _, line := range lines {
		_, _ = fmt.Fprintln(p.out, line)
	}
}

// Newline prints an empty line
func (p *Printer) Newline() {
	_, _ = fmt.Fprintln(p.out)
}

// PrintHeader prints a command header box
func (p *Printer) PrintHeader(title, command string, params map[string]string) {
	p.Print(RenderHeader(title, command, params, p.width))
	p.Newline()
}

// PrintSuccess prints a success result box
func (p *Printer) PrintSuccess(title string, details map[string]string) {
	p.Print(RenderSuccessBox(title, details, p.width))
	p.Newline()
}

// PrintError prints an error result box with troubleshooting tips
func (p *Printer) PrintError(title string, err error, troubleshooting []string) {
	p.Print(RenderErrorBox(title, err, troubleshooting, p.width))
	p.Newline()
}

// PrintGDBOutput prints a GDB output box (for verbose mode)
func (p *Printer) PrintGDBOutput(output string) {
	p.Print(RenderGDBOutputBox(output, p.width))
	p.Newline()
}

// RenderHeader renders a command header box
func RenderHeader(title, command string, params map[string]string, width int) string {
	var b strings.Builder

	// Title line
	titleLine := HeaderTitleStyle.Render(strings.ToUpper(title))
	// Command line
	commandLine := HeaderCommandStyle.Render(command)

	// Build top section
	topSection := lipgloss.JoinVertical(lipgloss.Left, titleLine, commandLine)

	// Build params section
	var paramLines []string
	for key, value := range params {
		keyStyled := HeaderParamKeyStyle.Render(key + ":")
		valueStyled := HeaderParamValueStyle.Render(value)
		paramLines = append(paramLines, keyStyled+" "+valueStyled)
	}
	paramsSection := strings.Join(paramLines, "\n")

	// Divider
	dividerWidth := width - 6 // Account for border and padding
	if dividerWidth < 10 {
		dividerWidth = 10
	}
	divider := RenderHorizontalDivider(dividerWidth, "─")

	// Combine all sections
	content := lipgloss.JoinVertical(lipgloss.Left, topSection, divider, paramsSection)

	// Apply border
	bordered := HeaderBorderStyle(width).Render(content)
	b.WriteString(bordered)

	return b.String()
}

// RenderSuccessBox renders a success result box
func RenderSuccessBox(title string, details map[string]string, width int) string {
	var lines []string

	// Title with checkmark
	titleLine := SuccessTitleStyle.Render("   " + SuccessMarker + "  SUCCESS  ─  " + title)
	lines = append(lines, "")
	lines = append(lines, titleLine)
	lines = append(lines, "")

	// Details
	for key, value := range details {
		keyStyled := ResultKeyStyle.Render("   " + key + ":")
		valueStyled := ResultValueStyle.Render(value)
		lines = append(lines, keyStyled+" "+valueStyled)
	}

	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	return SuccessBoxStyle(width).Render(content)
}

// RenderErrorBox renders an error result box with troubleshooting
func RenderErrorBox(title string, err error, troubleshooting []string, width int) string {
	var lines []string

	// Title with X mark
	titleLine := ErrorTitleStyle.Render("   " + FailureMarker + "  FAILED  ─  " + title)
	lines = append(lines, "")
	lines = append(lines, titleLine)
	lines = append(lines, "")

	// Error message
	if err != nil {
		errorLine := ErrorMessageStyle.Render("   Error: " + err.Error())
		lines = append(lines, errorLine)
		lines = append(lines, "")
	}

	// Troubleshooting section
	if len(troubleshooting) > 0 {
		var troubleLines []string
		troubleLines = append(troubleLines, TroubleshootingTitleStyle.Render("Troubleshooting:"))
		troubleLines = append(troubleLines, "")
		for _, tip := range troubleshooting {
			troubleLines = append(troubleLines, TroubleshootingItemStyle.Render("  • "+tip))
		}

		troubleContent := strings.Join(troubleLines, "\n")
		troubleBox := TroubleshootingBoxStyle(width).Render(troubleContent)
		lines = append(lines, troubleBox)
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return ErrorBoxStyle(width).Render(content)
}

// RenderGDBOutputBox renders a GDB output box for verbose mode
func RenderGDBOutputBox(output string, width int) string {
	var lines []string

	// Title
	titleLine := GDBOutputTitleStyle.Render("GDB Output")

	// Content (preserve formatting)
	content := GDBOutputContentStyle.Render(output)

	lines = append(lines, titleLine)
	lines = append(lines, content)

	boxContent := strings.Join(lines, "\n")

	// Create box with title in border
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		BorderTop(true).
		Width(width - 4).
		Padding(0, 1).
		Render(boxContent)
}
