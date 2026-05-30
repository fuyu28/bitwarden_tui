package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fuyu28/bitwarden_tui/internal/infra/clipboard"
	"github.com/fuyu28/bitwarden_tui/internal/infra/copyq"
	"github.com/fuyu28/bitwarden_tui/internal/infra/rbw"
	"github.com/fuyu28/bitwarden_tui/internal/repository"
	"github.com/fuyu28/bitwarden_tui/internal/ui"
)

func selectClipboard() repository.ClipboardRepository {
	if _, err := exec.LookPath("copyq"); err == nil {
		return copyq.NewClient()
	}
	return clipboard.NewFallback()
}

func main() {
	vaultRepo := rbw.NewClient()
	clipRepo := selectClipboard()

	app := ui.NewApp(vaultRepo, clipRepo)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
