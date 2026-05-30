package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorPrimary  = lipgloss.Color("62")
	colorSubdued  = lipgloss.Color("241")
	colorSensitive = lipgloss.Color("208")

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	styleDim   = lipgloss.NewStyle().Foreground(colorSubdued)
	styleSensitiveLabel = lipgloss.NewStyle().Foreground(colorSensitive)

	stylePaneActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary)

	stylePaneInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorSubdued)

	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	styleFilterTab = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(colorSubdued)

	styleFilterTabActive = lipgloss.NewStyle().
				Padding(0, 1).
				Foreground(colorPrimary).
				Bold(true)

	styleConfirmWarning = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)
)
