package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ConfirmDangerousOperation displays a warning box and prompts the user to type
// "I AGREE" to proceed with a dangerous operation. Returns true if the user
// confirmed, false otherwise.
func ConfirmDangerousOperation(title string, warnings []string, disclaimer string) bool {
	width := GetTerminalWidth()
	if width < MinTerminalWidth {
		width = MinTerminalWidth
	}

	var lines []string

	// Title with warning marker
	titleLine := lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(true).
		Render(fmt.Sprintf("   ⚠  WARNING  ─  %s", title))
	lines = append(lines, "")
	lines = append(lines, titleLine)
	lines = append(lines, "")

	// Warning bullet points
	for _, warning := range warnings {
		bulletStyle := lipgloss.NewStyle().Foreground(TextColor)
		lines = append(lines, bulletStyle.Render("   • "+warning))
	}
	lines = append(lines, "")

	// Disclaimer in muted text, word-wrapped
	if disclaimer != "" {
		disclaimerStyle := lipgloss.NewStyle().
			Foreground(MutedColor).
			Italic(true).
			Width(width - 12).
			PaddingLeft(3)
		lines = append(lines, disclaimerStyle.Render(disclaimer))
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")

	// Double border in orange/warning color
	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(WarningColor).
		Width(width-2).
		Padding(0, 2).
		Render(content)

	fmt.Println(box)
	fmt.Println()

	// Prompt for confirmation
	promptStyle := lipgloss.NewStyle().
		Foreground(WarningColor).
		Bold(true)
	fmt.Print(promptStyle.Render("To proceed, type \"I AGREE\" and press Enter: "))

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println()
		return false
	}

	// Check if user typed "I AGREE"
	input = strings.TrimSpace(input)
	if input == "I AGREE" {
		fmt.Println()
		return true
	}

	// User did not agree
	fmt.Println()
	cancelStyle := lipgloss.NewStyle().Foreground(MutedColor)
	fmt.Println(cancelStyle.Render("  Operation cancelled."))
	fmt.Println()
	return false
}

// FlashWriteConfirmation is a pre-configured confirmation for flash write operations
func FlashWriteConfirmation() bool {
	return ConfirmDangerousOperation(
		"FLASH WRITE OPERATION",
		[]string{
			"This operation will write to your device's flash memory",
			"Power cycle your SmartAP device before proceeding (unplug and replug power)",
			"Ensure OpenOCD has a stable JTAG connection",
			"Do not interrupt the operation once started",
		},
		"DISCLAIMER: This software is provided as-is, without warranty of any kind. "+
			"The authors accept no responsibility for any damage to your device. "+
			"By proceeding, you acknowledge that you understand the risks involved "+
			"in writing to device flash memory.",
	)
}
