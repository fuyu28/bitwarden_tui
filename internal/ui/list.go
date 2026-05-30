package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fuyu28/bitwarden_tui/internal/model"
)

type listEntry struct {
	item model.ListItem
}

func (e listEntry) FilterValue() string { return e.item.Name + " " + e.item.User }
func (e listEntry) Title() string       { return e.item.Name }
func (e listEntry) Description() string { return string(e.item.Type) }

var filterTypes = []struct {
	label    string
	itemType model.ItemType
}{
	{"All", ""},
	{"Login", model.TypeLogin},
	{"Card", model.TypeCard},
	{"Note", model.TypeNote},
	{"SSH", model.TypeSSH},
}

type listModel struct {
	list       list.Model
	allItems   []model.ListItem
	typeFilter model.ItemType
	focused    bool
}

func newListModel() listModel {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)
	d.Styles.NormalTitle = lipgloss.NewStyle().Padding(0, 0, 0, 2)
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Padding(0, 0, 0, 1).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(colorPrimary).
		Foreground(colorPrimary)

	l := list.New(nil, d, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.DisableQuitKeybindings()
	l.KeyMap.Filter = key.NewBinding(key.WithKeys("/"))
	return listModel{list: l}
}

func (m *listModel) setItems(items []model.ListItem) {
	m.allItems = items
	m.applyFilter()
}

func (m *listModel) setFilter(t model.ItemType) {
	m.typeFilter = t
	m.applyFilter()
}

func (m *listModel) applyFilter() {
	var entries []list.Item
	for _, item := range m.allItems {
		if m.typeFilter == "" || item.Type == m.typeFilter {
			entries = append(entries, listEntry{item: item})
		}
	}
	m.list.SetItems(entries)
}

func (m listModel) selectedItem() *model.ListItem {
	sel := m.list.SelectedItem()
	if sel == nil {
		return nil
	}
	e, ok := sel.(listEntry)
	if !ok {
		return nil
	}
	return &e.item
}

func (m listModel) filterTabsView() string {
	var tabs []string
	for i, ft := range filterTypes {
		label := fmt.Sprintf("[%d]%s", i+1, ft.label)
		if ft.itemType == m.typeFilter {
			tabs = append(tabs, styleFilterTabActive.Render(label))
		} else {
			tabs = append(tabs, styleFilterTab.Render(label))
		}
	}
	return strings.Join(tabs, "")
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	tabs := m.filterTabsView()
	return lipgloss.JoinVertical(lipgloss.Left, tabs, m.list.View())
}

func (m *listModel) setSize(w, h int) {
	m.list.SetSize(w, h-1)
}
