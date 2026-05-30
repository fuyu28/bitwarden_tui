package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fuyu28/bitwarden_tui/internal/model"
	"github.com/fuyu28/bitwarden_tui/internal/repository"
)

type field struct {
	label     string
	value     string
	sensitive bool
	masked    bool
}

type detailModel struct {
	viewport     viewport.Model
	item         *model.Item
	fields       []field
	cursor       int
	focused      bool
	clipRepo     repository.ClipboardRepository
	confirmCopy  bool
	statusMsg    string
	noteLines    []string
	noteCursor   int
}

func newDetailModel(clipRepo repository.ClipboardRepository) detailModel {
	vp := viewport.New(0, 0)
	return detailModel{
		viewport: vp,
		clipRepo: clipRepo,
	}
}

func (m *detailModel) setItem(item *model.Item) {
	m.item = item
	m.cursor = 0
	m.confirmCopy = false
	m.statusMsg = ""
	m.fields = buildFields(item)

	if item != nil && item.Type == model.TypeNote {
		m.noteLines = strings.Split(item.Notes, "\n")
		m.noteCursor = 0
	} else {
		m.noteLines = nil
	}
	m.updateContent()
}

func buildFields(item *model.Item) []field {
	if item == nil {
		return nil
	}
	var fields []field

	switch item.Type {
	case model.TypeLogin:
		d, _ := item.Detail.(*model.LoginDetail)
		if d == nil {
			break
		}
		fields = append(fields, field{label: "Username", value: d.Username})
		fields = append(fields, field{label: "Password", value: d.Password, sensitive: true, masked: true})
		if d.TOTP != "" {
			fields = append(fields, field{label: "TOTP", value: d.TOTP, sensitive: true, masked: true})
		}
		for i, uri := range d.URIs {
			fields = append(fields, field{label: fmt.Sprintf("URI %d", i+1), value: uri})
		}

	case model.TypeCard:
		d, _ := item.Detail.(*model.CardDetail)
		if d == nil {
			break
		}
		fields = append(fields, field{label: "Cardholder", value: d.CardholderName})
		fields = append(fields, field{label: "Brand", value: d.Brand})
		fields = append(fields, field{label: "Number", value: d.Number, sensitive: true, masked: true})
		fields = append(fields, field{label: "Expiry", value: d.ExpMonth + "/" + d.ExpYear})
		fields = append(fields, field{label: "Code/PIN", value: d.Code, sensitive: true, masked: true})

	case model.TypeSSH:
		d, _ := item.Detail.(*model.SSHKeyDetail)
		if d == nil {
			break
		}
		fields = append(fields, field{label: "Public Key", value: d.PublicKey})
		fields = append(fields, field{label: "Fingerprint", value: d.Fingerprint})
		fields = append(fields, field{label: "Private Key", value: d.PrivateKey, sensitive: true, masked: true})
	}

	if item.Notes != "" && item.Type != model.TypeNote {
		fields = append(fields, field{label: "Notes", value: item.Notes})
	}
	return fields
}

func (m *detailModel) displayValue(f field) string {
	if !f.sensitive || !f.masked {
		return f.value
	}
	return strings.Repeat("•", min(len(f.value), 8))
}

func (m *detailModel) updateContent() {
	if m.item == nil {
		m.viewport.SetContent(styleDim.Render("Select an item"))
		return
	}

	var sb strings.Builder
	header := styleTitle.Render(m.item.Name) + "  " + styleDim.Render("["+string(m.item.Type)+"]")
	sb.WriteString(header + "\n")
	if m.item.Folder != "" {
		sb.WriteString(styleDim.Render("Folder: "+m.item.Folder) + "\n")
	}
	sb.WriteString("\n")

	if m.item.Type == model.TypeNote {
		for i, line := range m.noteLines {
			prefix := "  "
			if m.focused && i == m.noteCursor {
				prefix = lipgloss.NewStyle().Foreground(colorPrimary).Render("> ")
			}
			sb.WriteString(fmt.Sprintf("%s%3d  %s\n", prefix, i+1, line))
		}
	} else {
		for i, f := range m.fields {
			cursor := "  "
			if m.focused && i == m.cursor {
				cursor = lipgloss.NewStyle().Foreground(colorPrimary).Render("> ")
			}
			label := fmt.Sprintf("%-14s", f.label)
			if f.sensitive {
				label = styleSensitiveLabel.Render(label)
			} else {
				label = styleDim.Render(label)
			}
			val := m.displayValue(f)
			sb.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, label, val))
		}
	}

	if m.confirmCopy {
		sb.WriteString("\n")
		sb.WriteString(styleConfirmWarning.Render("copyq not found. Value will be stored in plain text clipboard history."))
		sb.WriteString("\n")
		sb.WriteString("Press y again to confirm copy, or any other key to cancel.")
	}

	if m.statusMsg != "" {
		sb.WriteString("\n" + styleDim.Render(m.statusMsg))
	}

	m.viewport.SetContent(sb.String())
}

type copyDoneMsg struct{ err error }

func (m detailModel) Init() tea.Cmd { return nil }

func (m detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	if m.item == nil {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirmCopy {
			if msg.String() == "y" {
				return m.doCopySensitive()
			}
			m.confirmCopy = false
			m.statusMsg = "Copy cancelled."
			m.updateContent()
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Up):
			if m.item.Type == model.TypeNote {
				if m.noteCursor > 0 {
					m.noteCursor--
				}
			} else {
				if m.cursor > 0 {
					m.cursor--
				}
			}
			m.updateContent()

		case key.Matches(msg, keys.Down):
			if m.item.Type == model.TypeNote {
				if m.noteCursor < len(m.noteLines)-1 {
					m.noteCursor++
				}
			} else {
				if m.cursor < len(m.fields)-1 {
					m.cursor++
				}
			}
			m.updateContent()

		case key.Matches(msg, keys.Toggle):
			if m.item.Type != model.TypeNote && m.cursor < len(m.fields) {
				m.fields[m.cursor].masked = !m.fields[m.cursor].masked
				m.updateContent()
			}

		case key.Matches(msg, keys.Copy):
			return m.handleCopy()

		case key.Matches(msg, keys.CopyAll):
			if m.item.Type == model.TypeNote {
				return m.copyValue(m.item.Notes, false)
			}
		}

	case copyDoneMsg:
		if msg.err != nil {
			m.statusMsg = "Copy failed: " + msg.err.Error()
		} else {
			m.statusMsg = "Copied!"
		}
		m.updateContent()
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m detailModel) handleCopy() (detailModel, tea.Cmd) {
	if m.item.Type == model.TypeNote {
		if m.noteCursor < len(m.noteLines) {
			return m.copyValue(m.noteLines[m.noteCursor], false)
		}
		return m, nil
	}

	if m.cursor >= len(m.fields) {
		return m, nil
	}
	f := m.fields[m.cursor]
	if f.sensitive {
		if !m.clipRepo.SupportsHiddenCopy() {
			m.confirmCopy = true
			m.updateContent()
			return m, nil
		}
		return m.copyValue(f.value, true)
	}
	return m.copyValue(f.value, false)
}

func (m detailModel) doCopySensitive() (detailModel, tea.Cmd) {
	m.confirmCopy = false
	if m.cursor < len(m.fields) {
		return m.copyValue(m.fields[m.cursor].value, true)
	}
	return m, nil
}

func (m detailModel) copyValue(value string, sensitive bool) (detailModel, tea.Cmd) {
	var clipRepo = m.clipRepo
	return m, func() tea.Msg {
		var err error
		if sensitive {
			err = clipRepo.CopySensitive(value)
		} else {
			err = clipRepo.Copy(value)
		}
		return copyDoneMsg{err: err}
	}
}

func (m detailModel) View() string {
	return m.viewport.View()
}

func (m *detailModel) setSize(w, h int) {
	m.viewport.Width = w
	m.viewport.Height = h
	m.updateContent()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
