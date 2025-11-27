package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// GDBOutput represents a box for displaying raw GDB output.
// Used in verbose mode to show the actual GDB commands and responses.
type GDBOutput struct {
	Title    string   // e.g., "GDB Output"
	Content  string   // The raw GDB output
	Lines    []string // Parsed output lines (for filtering)
	Width    int      // Terminal width
	MaxLines int      // Maximum lines to display (0 = unlimited)
}

// NewGDBOutput creates a new GDB output box
func NewGDBOutput(content string) *GDBOutput {
	return &GDBOutput{
		Title:    "GDB Output",
		Content:  content,
		Lines:    strings.Split(content, "\n"),
		Width:    GetTerminalWidth(),
		MaxLines: 0,
	}
}

// SetWidth sets the terminal width for responsive rendering
func (g *GDBOutput) SetWidth(width int) *GDBOutput {
	g.Width = width
	return g
}

// SetTitle sets a custom title for the box
func (g *GDBOutput) SetTitle(title string) *GDBOutput {
	g.Title = title
	return g
}

// SetMaxLines limits the number of lines displayed
func (g *GDBOutput) SetMaxLines(max int) *GDBOutput {
	g.MaxLines = max
	return g
}

// FilterLines filters the output to only show lines matching the given patterns.
// Useful for extracting specific GDB output (e.g., results, errors).
func (g *GDBOutput) FilterLines(patterns ...string) *GDBOutput {
	var filtered []string
	for _, line := range g.Lines {
		for _, pattern := range patterns {
			if strings.Contains(line, pattern) {
				filtered = append(filtered, line)
				break
			}
		}
	}
	g.Lines = filtered
	g.Content = strings.Join(filtered, "\n")
	return g
}

// FilterPrefix filters to only lines starting with given prefixes
func (g *GDBOutput) FilterPrefix(prefixes ...string) *GDBOutput {
	var filtered []string
	for _, line := range g.Lines {
		for _, prefix := range prefixes {
			if strings.HasPrefix(strings.TrimSpace(line), prefix) {
				filtered = append(filtered, line)
				break
			}
		}
	}
	g.Lines = filtered
	g.Content = strings.Join(filtered, "\n")
	return g
}

// ExtractResults extracts common GDB result patterns from the output.
// Returns a map of variable names to their values.
func (g *GDBOutput) ExtractResults() map[string]string {
	results := make(map[string]string)

	for _, line := range g.Lines {
		// Look for patterns like "variable_name: value" or "$variable = value"
		line = strings.TrimSpace(line)

		// Pattern: "name: value"
		if idx := strings.Index(line, ": "); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+2:])
			// Only capture if key looks like a variable (no spaces, reasonable length)
			if !strings.Contains(key, " ") && len(key) < 30 {
				results[key] = value
			}
		}
	}

	return results
}

// Render returns the styled GDB output box as a string
func (g *GDBOutput) Render() string {
	width := g.Width
	if width < MinTerminalWidth {
		width = MinTerminalWidth
	}

	// Apply max lines limit
	lines := g.Lines
	if g.MaxLines > 0 && len(lines) > g.MaxLines {
		lines = lines[:g.MaxLines]
		lines = append(lines, "... (output truncated)")
	}

	// Title styled
	titleStyled := GDBOutputTitleStyle.Render(g.Title)

	// Content styled (preserve monospace formatting)
	contentStyled := GDBOutputContentStyle.Render(strings.Join(lines, "\n"))

	// Combine title and content
	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyled, "", contentStyled)

	// Box with muted border
	boxWidth := width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Width(boxWidth).
		Padding(0, 1).
		MarginLeft(2).
		Render(inner)
}

// RenderCompact renders a more compact version showing only key results
func (g *GDBOutput) RenderCompact() string {
	width := g.Width
	if width < MinTerminalWidth {
		width = MinTerminalWidth
	}

	// Extract results
	results := g.ExtractResults()

	if len(results) == 0 {
		// Fallback to normal render if no results found
		return g.Render()
	}

	// Build compact output
	var lines []string
	for key, value := range results {
		line := GDBOutputContentStyle.Render(key + ": " + value)
		lines = append(lines, "  "+line)
	}

	// Title styled
	titleStyled := GDBOutputTitleStyle.Render(g.Title + " (summary)")

	// Combine
	inner := lipgloss.JoinVertical(lipgloss.Left, titleStyled, "", strings.Join(lines, "\n"))

	// Box
	boxWidth := width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(MutedColor).
		Width(boxWidth).
		Padding(0, 1).
		MarginLeft(2).
		Render(inner)
}

// String implements fmt.Stringer
func (g *GDBOutput) String() string {
	return g.Render()
}

// --- Convenience functions ---

// RenderGDBOutput renders a GDB output box with the given content
func RenderGDBOutput(content string) string {
	return NewGDBOutput(content).Render()
}

// RenderGDBResults renders a compact GDB results summary
func RenderGDBResults(content string) string {
	return NewGDBOutput(content).RenderCompact()
}

// ParseGDBStepOutput parses GDB output for step markers like "[1/9]", "[2/9]", etc.
// Returns a slice of steps detected in the output.
func ParseGDBStepOutput(output string) []Step {
	var steps []Step
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for patterns like "[1/9] Step name"
		if strings.HasPrefix(line, "[") {
			// Find the closing bracket
			closeBracket := strings.Index(line, "]")
			if closeBracket > 0 && closeBracket < 10 {
				// Parse step number and total
				bracket := line[1:closeBracket]
				parts := strings.Split(bracket, "/")
				if len(parts) == 2 {
					var stepNum, total int
					_, err1 := fmt.Sscanf(parts[0], "%d", &stepNum)
					_, err2 := fmt.Sscanf(parts[1], "%d", &total)

					if err1 == nil && err2 == nil {
						// Extract step name (everything after "] ")
						name := ""
						if len(line) > closeBracket+2 {
							name = strings.TrimSpace(line[closeBracket+1:])
							// Remove trailing "..." if present
							name = strings.TrimSuffix(name, "...")
						}

						steps = append(steps, Step{
							Number: stepNum,
							Name:   name,
							Status: StepComplete, // Assume complete if we see it in output
						})
					}
				}
			}
		}
	}

	return steps
}
