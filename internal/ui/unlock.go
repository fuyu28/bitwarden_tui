package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type unlockModel struct {
	input    textinput.Model
	errMsg   string
	width    int
	height   int
}

func newUnlockModel() unlockModel {
	ti := textinput.New()
	ti.Placeholder = "Master password"
	ti.EchoMode = textinput.EchoPassword
	ti.Focus()
	return unlockModel{input: ti}
}

func (m unlockModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m unlockModel) Update(msg tea.Msg) (unlockModel, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m unlockModel) View() string {
	title := styleTitle.Render("bitwarden-tui")
	prompt := "Enter master password to unlock vault:"

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		prompt,
		m.input.View(),
	)
	if m.errMsg != "" {
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			"",
			styleConfirmWarning.Render(m.errMsg),
		)
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}
