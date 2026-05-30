package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fuyu28/bitwarden_tui/internal/model"
	"github.com/fuyu28/bitwarden_tui/internal/repository"
)

type appState int

const (
	stateUnlock appState = iota
	stateMain
	stateError
)

type focus int

const (
	focusList focus = iota
	focusDetail
)

type listLoadedMsg struct {
	items []model.ListItem
	err   error
}

type detailLoadedMsg struct {
	item *model.Item
	err  error
}

type unlockDoneMsg struct{ err error }
type syncDoneMsg struct{ err error }

type App struct {
	state     appState
	focus     focus
	vaultRepo repository.VaultRepository

	unlock unlockModel
	list   listModel
	detail detailModel

	errMsg    string
	statusMsg string
	width     int
	height    int
}

func NewApp(vaultRepo repository.VaultRepository, clipRepo repository.ClipboardRepository) *App {
	unlocked, err := vaultRepo.IsUnlocked()
	var initialState appState
	var errMsg string
	if err != nil {
		initialState = stateError
		errMsg = fmt.Sprintf("rbw error: %v", err)
	} else if unlocked {
		initialState = stateMain
	} else {
		initialState = stateUnlock
	}

	return &App{
		state:     initialState,
		focus:     focusList,
		vaultRepo: vaultRepo,
		unlock:    newUnlockModel(),
		list:      newListModel(),
		detail:    newDetailModel(clipRepo),
		errMsg:    errMsg,
	}
}

func (a *App) Init() tea.Cmd {
	if a.state == stateMain {
		return a.loadList()
	}
	if a.state == stateUnlock {
		return a.unlock.Init()
	}
	return nil
}

func (a *App) loadList() tea.Cmd {
	return func() tea.Msg {
		items, err := a.vaultRepo.List()
		return listLoadedMsg{items: items, err: err}
	}
}

func (a *App) loadDetail(item model.ListItem) tea.Cmd {
	return func() tea.Msg {
		detail, err := a.vaultRepo.GetDetail(item.ID, item.Type)
		return detailLoadedMsg{item: detail, err: err}
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.resize()
		return a, nil

	case listLoadedMsg:
		if msg.err != nil {
			a.statusMsg = "List error: " + msg.err.Error()
		} else {
			a.list.setItems(msg.items)
			if sel := a.list.selectedItem(); sel != nil {
				return a, a.loadDetail(*sel)
			}
		}
		return a, nil

	case detailLoadedMsg:
		if msg.err != nil {
			a.statusMsg = "Detail error: " + msg.err.Error()
		} else {
			a.detail.setItem(msg.item)
		}
		return a, nil

	case unlockDoneMsg:
		if msg.err != nil {
			a.unlock.errMsg = "Unlock failed: " + msg.err.Error()
		} else {
			a.state = stateMain
			return a, a.loadList()
		}
		return a, nil

	case syncDoneMsg:
		if msg.err != nil {
			a.statusMsg = "Sync error: " + msg.err.Error()
		} else {
			a.statusMsg = "Synced!"
			return a, a.loadList()
		}
		return a, nil
	}

	if a.state == stateError {
		if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, keys.Quit) {
			return a, tea.Quit
		}
		return a, nil
	}

	if a.state == stateUnlock {
		return a.updateUnlock(msg)
	}

	return a.updateMain(msg)
}

func (a *App) updateUnlock(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter":
			pw := a.unlock.input.Value()
			a.unlock.input.SetValue("")
			vaultRepo := a.vaultRepo
			return a, func() tea.Msg {
				err := vaultRepo.Unlock(pw)
				return unlockDoneMsg{err: err}
			}
		case "ctrl+c", "q":
			return a, tea.Quit
		}
	}
	var cmd tea.Cmd
	a.unlock, cmd = a.unlock.Update(msg)
	return a, cmd
}

func (a *App) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, isKey := msg.(tea.KeyMsg)

	if isKey {
		// Global keys
		switch {
		case key.Matches(km, keys.Quit):
			return a, tea.Quit

		case key.Matches(km, keys.Tab):
			if a.focus == focusList {
				a.focus = focusDetail
				a.detail.focused = true
			} else {
				a.focus = focusList
				a.detail.focused = false
			}
			a.detail.updateContent()
			return a, nil

		case key.Matches(km, keys.Sync):
			vaultRepo := a.vaultRepo
			return a, func() tea.Msg {
				err := vaultRepo.Sync()
				return syncDoneMsg{err: err}
			}

		case key.Matches(km, keys.Filter1):
			a.list.setFilter("")
			return a, nil
		case key.Matches(km, keys.Filter2):
			a.list.setFilter(model.TypeLogin)
			return a, nil
		case key.Matches(km, keys.Filter3):
			a.list.setFilter(model.TypeCard)
			return a, nil
		case key.Matches(km, keys.Filter4):
			a.list.setFilter(model.TypeNote)
			return a, nil
		case key.Matches(km, keys.Filter5):
			a.list.setFilter(model.TypeSSH)
			return a, nil
		}
	}

	if a.focus == focusList {
		prevSel := a.list.selectedItem()
		var cmd tea.Cmd
		a.list, cmd = a.list.Update(msg)

		newSel := a.list.selectedItem()
		if newSel != nil && (prevSel == nil || prevSel.ID != newSel.ID) {
			return a, tea.Batch(cmd, a.loadDetail(*newSel))
		}
		return a, cmd
	}

	// focusDetail
	var cmd tea.Cmd
	a.detail, cmd = a.detail.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	if a.state == stateError {
		return styleConfirmWarning.Render(a.errMsg) + "\n\nPress q to quit."
	}

	if a.state == stateUnlock {
		a.unlock.width = a.width
		a.unlock.height = a.height
		return a.unlock.View()
	}

	// Main layout
	listWidth := a.width / 3
	detailWidth := a.width - listWidth

	contentHeight := a.height - 3

	listStyle := stylePaneInactive
	detailStyle := stylePaneInactive
	if a.focus == focusList {
		listStyle = stylePaneActive
	} else {
		detailStyle = stylePaneActive
	}

	listPane := listStyle.Width(listWidth - 2).Height(contentHeight - 2).Render(a.list.View())
	detailPane := detailStyle.Width(detailWidth - 2).Height(contentHeight - 2).Render(a.detail.View())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, listPane, detailPane)

	header := styleTitle.Render("bitwarden-tui")
	if a.statusMsg != "" {
		header += "  " + styleDim.Render(a.statusMsg)
	}

	statusBar := styleStatusBar.Width(a.width).Render(
		"/search  1-5 filter  y copy  tab focus  space toggle  r sync  q quit",
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		panes,
		statusBar,
	)
}

func (a *App) resize() {
	listWidth := a.width / 3
	detailWidth := a.width - listWidth
	contentHeight := a.height - 3

	a.list.setSize(listWidth-2, contentHeight-2)
	a.detail.setSize(detailWidth-2, contentHeight-2)
	a.unlock.width = a.width
	a.unlock.height = a.height
}

func (a *App) Model() tea.Model { return a }
