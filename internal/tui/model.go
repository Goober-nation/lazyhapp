package tui

import (
	"lazyhapp/internal/core"
	"lazyhapp/internal/vpn"
	"os"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
)

const (
	HeaderHeight = 1
	FooterHeight = 4
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
	State          *core.AppState
	FocusedPanel   PanelID
	SelectedSub    int
	SelectedNode   int
	SelectedOption int // Index of selected option in Options panel

	// Scroll Offsets
	SubScrollOffset  int
	NodeScrollOffset int
	LogScrollOffset  int

	// VPN Client
	VpnClient   *vpn.Client
	Connected   bool
	CurrentNode string
	VpnLogChan  chan string

	// Layout
	Width         int
	Height        int
	ContentHeight int

	// Modals
	ActiveModal string // "add_sub", "add_sub_name", "remove_sub", "help", "reset_confirm"
	ModalInput  string
	tempSubUrl  string

	// Logs
	Logs      []string
	LogOffset int64

	// Viewport for logs
	LogViewport viewport.Model

	// Stats
	UpSpeed   string
	DownSpeed string
}

func InitialModel() Model {
	state, _ := core.LoadState()

	connected := false
	if state.VpnPid != 0 {
		proc, err := os.FindProcess(state.VpnPid)
		if err == nil {
			if err := proc.Signal(syscall.Signal(0)); err == nil {
				connected = true
			}
		}
	}

	vp := viewport.New(0, 0)
	vp.SetContent(strings.Join(state.Logs, "\n"))

	return Model{
		State:        state,
		FocusedPanel: PanelSubscriptions,
		SelectedSub:  0,
		SelectedNode: 0,
		VpnClient:    vpn.NewClient(),
		Logs:         state.Logs,
		Connected:    connected,
		CurrentNode:  state.CurrentNode,
		VpnLogChan:   make(chan string, 100),
		LogViewport:  vp,
	}
}

func (m Model) Init() tea.Cmd {
	return tickCmd() // Start polling logs immediately
}
