package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	Purple = lipgloss.Color("#7C3AED")
	Green  = lipgloss.Color("#10B981")
	Red    = lipgloss.Color("#EF4444")
	Yellow = lipgloss.Color("#F59E0B")
	Dim    = lipgloss.Color("#6B7280")
	White  = lipgloss.Color("#F9FAFB")

	BoldStyle = lipgloss.NewStyle().Bold(true)
	DimStyle  = lipgloss.NewStyle().Foreground(Dim)

	SuccessStyle = lipgloss.NewStyle().Foreground(Green).Bold(true)
	ErrorStyle   = lipgloss.NewStyle().Foreground(Red).Bold(true)
	WarnStyle    = lipgloss.NewStyle().Foreground(Yellow)
	AccentStyle  = lipgloss.NewStyle().Foreground(Purple).Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Foreground(Purple).
			Bold(true).
			MarginBottom(1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(0, 1)
)

func Success(msg string) string {
	return SuccessStyle.Render("✓ " + msg)
}

func Error(msg string) string {
	return ErrorStyle.Render("✗ " + msg)
}

func Warn(msg string) string {
	return WarnStyle.Render("⚠ " + msg)
}

func Info(msg string) string {
	return AccentStyle.Render("● " + msg)
}
