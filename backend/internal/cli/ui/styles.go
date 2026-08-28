package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Brand Colors
	Emerald = lipgloss.Color("#10b981")
	Blue    = lipgloss.Color("#3b82f6")
	Yellow  = lipgloss.Color("#f59e0b")
	Red     = lipgloss.Color("#ef4444")
	Purple  = lipgloss.Color("#a855f7")
	Cyan    = lipgloss.Color("#06b6d4")
	Gray    = lipgloss.Color("#888888")
	DarkGray = lipgloss.Color("#333333")

	// Base text styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#1e1e1e")).
			Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			MarginBottom(1)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(Gray)

	BoldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff"))

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff"))

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Red)

	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Yellow)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Cyan)

	// Box and border styles
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DarkGray).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	ActiveCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Emerald).
			Padding(0, 1).
			Margin(0, 0, 1, 0)
)

// TaskTypeBadge renders a styled inline badge for task types.
func TaskTypeBadge(taskType string) string {
	t := strings.ToUpper(strings.TrimSpace(taskType))
	var color lipgloss.Color

	switch t {
	case "TODO":
		color = lipgloss.Color("#ffffff")
	case "FIXME":
		color = Yellow
	case "BUG":
		color = Red
	case "HACK":
		color = Purple
	case "NOTE":
		color = Cyan
	default:
		color = Gray
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Render("[" + t + "]")
}

// StatusBadge renders a sleek inline badge for open, in_progress, resolved states.
func StatusBadge(status string) string {
	var color lipgloss.Color
	var label string

	switch strings.ToLower(strings.TrimSpace(status)) {
	case "open":
		color = Red
		label = "open"
	case "in_progress":
		color = Yellow
		label = "in progress"
	case "resolved":
		color = lipgloss.Color("#ffffff")
		label = "resolved"
	default:
		color = Gray
		label = strings.ToLower(status)
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Render("(" + label + ")")
}

// Banner returns the compact stylized logo for CodeTasker matching logo-kucuk.png (</CT>).
func Banner() string {
	logo := `
         ⣶  ⣠⠖⠉⠙⢶⢰⠏⢹⡏⢳
        ⢠⡏ ⣼⠃   ⠘  ⢸⡇⠈
   ⣠⣾⠃  ⣼⠃⢸⡇       ⢸⡇ ⠘⢷⣄
 ⣠⡾⠋   ⢀⡟ ⢸⠃       ⢸⡇   ⠙⢷⣄
⣾⡋     ⢸⠇ ⢸⡆       ⢸⡇     ⢙⣷
⠈⠻⣦⡀   ⣿  ⠘⣧       ⢸⡇   ⢀⣴⠟⠉
  ⠈⠻⣦⡀⢰⡇   ⠹⣆   ⢀⡇ ⢸⡇ ⢀⣴⠟⠁
    ⠈⠁⢼⠁    ⠈⠓⠒⠒⠉⠁ ⠛⠃ ⠈⠁
`
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffffff")).
		Bold(true).
		Render(logo)
}

// FormatCost formats dollar cost with 2 decimals.
func FormatCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}
