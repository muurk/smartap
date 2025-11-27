package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
)

// StepStatus represents the current state of a step
type StepStatus int

const (
	StepPending  StepStatus = iota // Not yet started
	StepRunning                    // Currently executing
	StepComplete                   // Successfully completed
	StepFailed                     // Failed
	StepSkipped                    // Skipped
)

// Step represents a single step in a multi-step operation
type Step struct {
	Number  int        // Step number (1-based)
	Name    string     // Step description
	Status  StepStatus // Current status
	Message string     // Optional status message (e.g., "5s delay", "1,234 bytes")
}

// Progress represents a progress display with bar and step list
type Progress struct {
	Label      string  // e.g., "Injecting certificate..."
	Steps      []Step  // List of steps
	Current    int     // Current step (1-based)
	Total      int     // Total steps
	Percent    float64 // Progress percentage (0.0 - 1.0)
	Width      int     // Terminal width
	ShowBar    bool    // Whether to show progress bar
	ShowSteps  bool    // Whether to show step list
	bar        progress.Model
}

// NewProgress creates a new progress display
func NewProgress(label string, totalSteps int) *Progress {
	// Initialize progress bar with theme colors
	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
	)

	// Create steps
	steps := make([]Step, totalSteps)
	for i := 0; i < totalSteps; i++ {
		steps[i] = Step{
			Number: i + 1,
			Status: StepPending,
		}
	}

	return &Progress{
		Label:     label,
		Steps:     steps,
		Current:   0,
		Total:     totalSteps,
		Percent:   0,
		Width:     GetTerminalWidth(),
		ShowBar:   true,
		ShowSteps: true,
		bar:       bar,
	}
}

// SetWidth sets the terminal width for responsive rendering
func (p *Progress) SetWidth(width int) *Progress {
	p.Width = width
	// Adjust progress bar width
	barWidth := width - 20 // Leave room for percentage and step count
	if barWidth < 20 {
		barWidth = 20
	}
	if barWidth > 50 {
		barWidth = 50
	}
	p.bar = progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(barWidth),
	)
	return p
}

// SetStepNames sets the names for all steps
func (p *Progress) SetStepNames(names []string) *Progress {
	for i, name := range names {
		if i < len(p.Steps) {
			p.Steps[i].Name = name
		}
	}
	return p
}

// UpdateStep updates a specific step's status and optional message
func (p *Progress) UpdateStep(stepNumber int, status StepStatus, message string) {
	if stepNumber < 1 || stepNumber > len(p.Steps) {
		return
	}
	idx := stepNumber - 1
	p.Steps[idx].Status = status
	p.Steps[idx].Message = message

	// Update current step and percentage
	if status == StepRunning {
		p.Current = stepNumber
	} else if status == StepComplete || status == StepFailed || status == StepSkipped {
		// Count completed steps
		completed := 0
		for _, s := range p.Steps {
			if s.Status == StepComplete || s.Status == StepSkipped {
				completed++
			}
		}
		p.Percent = float64(completed) / float64(p.Total)
	}
}

// CompleteStep marks a step as complete
func (p *Progress) CompleteStep(stepNumber int, message string) {
	p.UpdateStep(stepNumber, StepComplete, message)
}

// FailStep marks a step as failed
func (p *Progress) FailStep(stepNumber int, message string) {
	p.UpdateStep(stepNumber, StepFailed, message)
}

// StartStep marks a step as running
func (p *Progress) StartStep(stepNumber int, message string) {
	p.UpdateStep(stepNumber, StepRunning, message)
}

// Render returns the styled progress display as a string
func (p *Progress) Render() string {
	var b strings.Builder

	// Label
	if p.Label != "" {
		b.WriteString(ProgressLabelStyle.Render(p.Label))
		b.WriteString("\n\n")
	}

	// Progress bar with percentage and step count
	if p.ShowBar {
		barLine := p.renderProgressBar()
		b.WriteString(barLine)
		b.WriteString("\n\n")
	}

	// Step list
	if p.ShowSteps {
		stepList := p.renderStepList()
		b.WriteString(stepList)
	}

	return b.String()
}

// renderProgressBar renders the progress bar line
func (p *Progress) renderProgressBar() string {
	// Get progress bar view
	barView := p.bar.ViewAs(p.Percent)

	// Calculate percentage display
	percentStr := fmt.Sprintf("%3.0f%%", p.Percent*100)

	// Step counter
	stepStr := fmt.Sprintf("[%d/%d]", p.Current, p.Total)

	// Combine: bar + percentage + step count
	return lipgloss.NewStyle().
		PaddingLeft(2).
		Render(fmt.Sprintf("%s  %s  %s", barView, percentStr, stepStr))
}

// renderStepList renders the list of steps
func (p *Progress) renderStepList() string {
	var lines []string

	for _, step := range p.Steps {
		line := p.renderStepLine(step)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderStepLine renders a single step line
func (p *Progress) renderStepLine(step Step) string {
	// Step prefix: "[1/9]"
	prefix := fmt.Sprintf("  [%d/%d]", step.Number, p.Total)

	// Status marker
	var marker string
	var nameStyle lipgloss.Style

	switch step.Status {
	case StepComplete:
		marker = StepMarkerComplete
		nameStyle = StepCompleteStyle
	case StepRunning:
		marker = StepMarkerRunning
		nameStyle = StepRunningStyle
	case StepFailed:
		marker = FailureMarker
		nameStyle = ErrorTitleStyle
	case StepSkipped:
		marker = "⊘"
		nameStyle = StepPendingStyle
	default: // StepPending
		marker = StepMarkerPending
		nameStyle = StepPendingStyle
	}

	// Build the line
	var b strings.Builder
	b.WriteString(prefix)
	b.WriteString(" ")
	b.WriteString(nameStyle.Render(step.Name))

	// Calculate padding for alignment
	// We want the marker to appear at a consistent column
	nameLen := lipgloss.Width(step.Name)
	maxNameLen := 45 // Max name length before wrapping
	padding := maxNameLen - nameLen
	if padding < 1 {
		padding = 1
	}
	b.WriteString(strings.Repeat(" ", padding))

	// Marker
	switch step.Status {
	case StepComplete:
		b.WriteString(StepCompleteStyle.Render(marker))
	case StepRunning:
		b.WriteString(StepRunningStyle.Render(marker))
	case StepFailed:
		b.WriteString(ErrorTitleStyle.Render(marker))
	default:
		b.WriteString(StepPendingStyle.Render(marker))
	}

	// Optional message
	if step.Message != "" {
		b.WriteString("  ")
		b.WriteString(StepNoteStyle.Render("(" + step.Message + ")"))
	}

	return b.String()
}

// String implements fmt.Stringer
func (p *Progress) String() string {
	return p.Render()
}

// StepCallback is the function signature for step progress updates.
// Commands call this to report progress.
type StepCallback func(stepNumber int, name string, status StepStatus, message string)
