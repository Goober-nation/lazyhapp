package tui

import (
	"lazyhapp/internal/core"
	"lazyhapp/internal/vpn"

	"github.com/charmbracelet/bubbletea"
)

type PanelID int

const (
	PanelSubscriptions PanelID = iota
	PanelNodes
	PanelLogs
	PanelOptions
	PanelStatus
	PanelSystemInfo // Combined System + Keybindings
)

type Model struct {
	State       *core.AppState
	FocusedPanel PanelID
	SelectedSub  int
	SelectedNode int
	
	// VPN Client
	VpnClient   *vpn.Client
	Connected   bool
	CurrentNode string

	// Layout
	Width       int
	Height      int

	// Modals
	ActiveModal string // "add_sub", "remove_sub", "help"
	ModalInput  string

	// Logs
	Logs        []string
}

func InitialModel() Model {
	state, _ := core.LoadState()
	return Model{
		State:        state,
		FocusedPanel: PanelSubscriptions,
		SelectedSub:  0,
		SelectedNode: 0,
		VpnClient:    vpn.NewClient(),
		Logs:         []string{"Application started..."},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}
