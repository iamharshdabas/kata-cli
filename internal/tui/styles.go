package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Lip Gloss Styles mapping to standard terminal ANSI color numbers (minimal & theme-compliant)
var (
	headerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")). // Bright Black / Dark Gray
			Padding(0, 2).
			Width(59)

	headerTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")). // Cyan
			Bold(true)

	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("4")). // Blue
				Bold(true).
				MarginTop(1).
				MarginBottom(1)

	tagHardStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")). // Red
			Bold(true)

	tagMedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")). // Yellow
			Bold(true)

	tagEasyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")). // Green
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // Gray

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // Gray

	// New styles for enhanced UX
	dangerTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)

	warningTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Bold(true)

	successTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)

	mutedTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	highlightTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("6")).
				Bold(true)
)

// Format the difficulty tag with colored output and emojis
func formatDifficulty(diff string) string {
	switch diff {
	case "Hard":
		return tagHardStyle.Render("[ 🔴 Boss  ]")
	case "Med":
		return tagMedStyle.Render("[ 🟡 Mid   ]")
	case "Easy":
		return tagEasyStyle.Render("[ 🟢 Chill ]")
	default:
		return "[" + diff + "]"
	}
}
