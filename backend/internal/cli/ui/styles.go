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
			Foreground(Emerald).
			MarginBottom(1)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(Gray)

	BoldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff"))

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Emerald)

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

// TaskTypeBadge renders a styled colored badge for task types.
func TaskTypeBadge(taskType string) string {
	var bg, fg lipgloss.Color
	fg = lipgloss.Color("#000000")

	switch strings.ToUpper(taskType) {
	case "TODO":
		bg = Emerald
	case "FIXME":
		bg = Yellow
	case "BUG":
		bg = Red
		fg = lipgloss.Color("#ffffff")
	case "HACK":
		bg = Purple
		fg = lipgloss.Color("#ffffff")
	case "NOTE":
		bg = Blue
		fg = lipgloss.Color("#ffffff")
	default:
		bg = Gray
		fg = lipgloss.Color("#ffffff")
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(fg).
		Background(bg).
		Padding(0, 1).
		Render(strings.ToUpper(taskType))
}

// StatusBadge renders a badge for open, in_progress, resolved states.
func StatusBadge(status string) string {
	var color lipgloss.Color
	var label string

	switch strings.ToLower(status) {
	case "open":
		color = Red
		label = "OPEN"
	case "in_progress":
		color = Yellow
		label = "IN PROGRESS"
	case "resolved":
		color = Emerald
		label = "RESOLVED"
	default:
		color = Gray
		label = strings.ToUpper(status)
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(color).
		Border(lipgloss.NormalBorder()).
		BorderForeground(color).
		Padding(0, 1).
		Render(label)
}

// Banner returns the stylized ASCII logo for CodeTasker.
func Banner() string {
	logo := `
   ______          __     ______           __             
  / ____/____  ___/ /__  /_  __/____ _ ___/ /__ ___   _____
 / /    / __ \/ _  / _ \  / /  / __ ` + "`" + `/ __  // _ \/ _ \/ ___/
/ /___ / /_/ / /_/ /  __/ / /  / /_/ /\__ \/  __/  __/ /    
\____/ \____/\__,_/\___/ /_/   \__,_/____/\___/\___//_/     
`
	return lipgloss.NewStyle().
		Foreground(Emerald).
		Bold(true).
		Render(logo)
}

// FormatCost formats dollar cost with 2 decimals.
func FormatCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}
