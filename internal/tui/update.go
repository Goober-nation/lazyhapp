package tui

import (
	"context"
	"fmt"
	"lazyhapp/internal/core"
	"lazyhapp/internal/logger"
	"os"
	"bufio"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
)

// Msg for subscription fetch results
type SubFetchMsg struct {
	Subs  []core.Subscription
	Nodes []core.Node
	Err   error
}

// Msg for log updates from file
type LogUpdateMsg []string

// Msg for periodic polling
type TickMsg time.Time

// Msg for ping results
type PingResultMsg struct {
	Result core.PingResult
	Node   core.Node
}

type PingTickMsg struct {
	Node core.Node
}

func fetchSubCmd(url string, name string) tea.Cmd {
	return func() tea.Msg {
		subs, nodes, err := core.FetchSubscription(url, name)
		return SubFetchMsg{Subs: subs, Nodes: nodes, Err: err}
	}
}

func pingNodeCmd(ctx context.Context, node core.Node) tea.Cmd {
	return func() tea.Msg {
		res := core.PingNode(ctx, node)
		return PingResultMsg{Result: res, Node: node}
	}
}

func tickCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(time.Second)
		return TickMsg(time.Now())
	}
}

func schedulePingCmd(node core.Node, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		return PingTickMsg{Node: node}
	}
}

func readLogFile(offset int64) ([]string, int64) {
	file, err := os.Open("lazyhapp.log")
	if err != nil {
		return nil, offset
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, offset
	}

	if stat.Size() < offset {
		offset = 0
	}

	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, offset
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	newOffset, _ := file.Seek(0, io.SeekCurrent)
	return lines, newOffset
}

func (m Model) addLog(msg string) {
	logger.Info("TUI", msg)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case TickMsg:
		newLogs, offset := readLogFile(m.LogOffset)
		if len(newLogs) > 0 {
			m.Logs = append(m.Logs, newLogs...)
			m.LogOffset = offset
			m.State.Logs = m.Logs
			core.SaveState(m.State)
		}
		return m, tickCmd()

	case LogUpdateMsg:
		m.Logs = append(m.Logs, msg...)
		m.State.Logs = m.Logs
		core.SaveState(m.State)
		return m, nil

	case SubFetchMsg:
		if msg.Err != nil {
			m.addLog(fmt.Sprintf("Error fetching sub: %v", msg.Err))
		} else {
			m.State.Subscriptions = append(m.State.Subscriptions, msg.Subs...)
			m.State.Nodes = append(m.State.Nodes, msg.Nodes...)
			m.addLog(fmt.Sprintf("Successfully added %d subscriptions", len(msg.Subs)))
			if err := core.SaveState(m.State); err != nil {
				m.addLog(fmt.Sprintf("Error saving state: %v", err))
			}
		}
		return m, nil

	case PingResultMsg:
		for i := range m.State.Nodes {
			if m.State.Nodes[i].ID == msg.Result.NodeID {
				m.State.Nodes[i].LastMeasuredPing = msg.Result.Ping
				break
			}
		}
		return m, nil

	case PingTickMsg:
		return m, pingNodeCmd(context.Background(), msg.Node)

	case tea.KeyMsg:
		if m.ActiveModal != "" {
			return m.handleModalInput(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			selectable := []PanelID{PanelSubscriptions, PanelNodes, PanelOptions}
			for i, p := range selectable {
				if p == m.FocusedPanel {
					m.FocusedPanel = selectable[(i+1)%len(selectable)]
					return m, nil
				}
			}
			m.FocusedPanel = selectable[0]
			return m, nil

		case "shift+tab":
			selectable := []PanelID{PanelSubscriptions, PanelNodes, PanelOptions}
			for i, p := range selectable {
				if p == m.FocusedPanel {
					m.FocusedPanel = selectable[(i-1+len(selectable))%len(selectable)]
					return m, nil
				}
			}
			m.FocusedPanel = selectable[len(selectable)-1]
			return m, nil

		case "j", "down":
			m.handleScrollDown()
			return m, nil

		case "k", "up":
			m.handleScrollUp()
			return m, nil

		case ",":
			if m.SelectedSub > 0 {
				m.SelectedSub--
				m.SelectedNode = 0
			}
			return m, nil

		case ".":
			if len(m.State.Subscriptions) > 0 && m.SelectedSub < len(m.State.Subscriptions)-1 {
				m.SelectedSub++
				m.SelectedNode = 0
			}
			return m, nil

		case "a":
			if m.FocusedPanel == PanelSubscriptions {
				m.ActiveModal = "add_sub"
				m.ModalInput = ""
				m.addLog("Entering URL...")
			}
			return m, nil

		case "d":
			if m.FocusedPanel == PanelSubscriptions {
				if len(m.State.Subscriptions) > 0 {
					m.ActiveModal = "remove_sub"
					m.addLog("Confirm deletion...")
				}
				return m, nil
			}
			return m, nil

		case "o":
			if m.FocusedPanel == PanelOptions {
				m.addLog("Option changed!")
				return m, nil
			}
			return m, nil

		case "r":
			if m.FocusedPanel == PanelSubscriptions && len(m.State.Subscriptions) > 0 {
				sub := m.State.Subscriptions[m.SelectedSub]
				m.addLog(fmt.Sprintf("Refreshing %s...", sub.Name))
				return m, fetchSubCmd(sub.URL, sub.Name)
			}
			return m, nil

		case "enter":
			if m.FocusedPanel == PanelNodes {
				subID := ""
				if len(m.State.Subscriptions) > 0 && m.SelectedSub < len(m.State.Subscriptions) {
					subID = m.State.Subscriptions[m.SelectedSub].ID
				}
				var selectedNode *core.Node
				for i, node := range m.State.Nodes {
					if node.SubscriptionID == subID && i == m.SelectedNode {
						selectedNode = &node
						break
					}
				}
				if selectedNode != nil {
					if m.Connected && m.CurrentNode == selectedNode.Name {
						m.VpnClient.Cleanup(m.State.VpnPid)
						m.Connected = false
						m.CurrentNode = ""
						m.State.VpnPid = 0
						m.State.CurrentNode = ""
						core.SaveState(m.State)
						m.addLog("Disconnected from VPN.")
					} else {
						m.addLog(fmt.Sprintf("Connecting to %s...", selectedNode.Name))
						pid, err := m.VpnClient.Connect(context.Background(), selectedNode.ConfigPayload)
						if err == nil {
							m.Connected = true
							m.CurrentNode = selectedNode.Name
							m.State.VpnPid = pid
							m.State.CurrentNode = selectedNode.Name
							core.SaveState(m.State)
							m.addLog(fmt.Sprintf("Connected successfully to %s", selectedNode.Name))
							return m, tickCmd()
						} else {
							m.addLog(fmt.Sprintf("Connection failed: %v", err))
						}
					}
				}
			}
			return m, nil

		case "c":
			if m.FocusedPanel == PanelNodes {
				if m.Connected {
					m.VpnClient.Cleanup(m.State.VpnPid)
					m.Connected = false
					m.CurrentNode = ""
					m.State.VpnPid = 0
					m.State.CurrentNode = ""
					core.SaveState(m.State)
					m.addLog("Disconnected from VPN.")
				}
			}
			return m, nil

		case "p":
			if m.FocusedPanel == PanelNodes {
				m.addLog("Starting async ping check...")
				var cmds []tea.Cmd
				for _, node := range m.State.Nodes {
					cmds = append(cmds, pingNodeCmd(context.Background(), node))
				}
				return m, tea.Batch(cmds...)
			}
			return m, nil

		case "ctrl+r":
			if m.FocusedPanel == PanelSubscriptions {
				m.ActiveModal = "reset_confirm"
				m.addLog("Reset requested...")
			}
			return m, nil

		case "ctrl+L":
			if m.FocusedPanel == PanelSubscriptions {
				m.ActiveModal = "reset_confirm"
				m.addLog("Reset requested...")
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) handleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.ActiveModal = ""
		return m, nil
	case "enter":
		if m.ActiveModal == "add_sub" {
			url := m.ModalInput
			url = strings.Trim(url, "[] ")
			if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "https://" + url
			}
			m.tempSubUrl = url
			m.ActiveModal = "add_sub_name"
			m.ModalInput = ""
			return m, nil
		} else if m.ActiveModal == "add_sub_name" {
			name := m.ModalInput
			if name == "" {
				name = "Unnamed Subscription"
			}
			m.addLog(fmt.Sprintf("Fetching subscription: %s (%s)", m.tempSubUrl, name))
			cmd := fetchSubCmd(m.tempSubUrl, name)
			m.ActiveModal = ""
			m.tempSubUrl = ""
			return m, cmd
		} else if m.ActiveModal == "reset_confirm" {
			m.VpnClient.Cleanup(m.State.VpnPid)
			m.State.Subscriptions = []core.Subscription{}
			m.State.Nodes = []core.Node{}
			m.State.VpnPid = 0
			m.State.CurrentNode = ""
			
			os.Remove("lazyhapp.log")
			if err := logger.Reset(); err != nil {
				m.addLog(fmt.Sprintf("Logger reset failed: %v", err))
			}
			m.Logs = []string{}
			m.LogOffset = 0
			
			if err := core.SaveState(m.State); err != nil {
				m.addLog(fmt.Sprintf("Reset failed: %v", err))
			} else {
				m.addLog("Application state fully reset.")
			}
			m.Connected = false
			m.CurrentNode = ""
			m.SelectedSub = 0
			m.SelectedNode = 0
			m.ActiveModal = ""
			return m, nil
		} else if m.ActiveModal == "remove_sub" {
			if len(m.State.Subscriptions) > 0 {
				subID := m.State.Subscriptions[m.SelectedSub].ID
				newSubs := []core.Subscription{}
				for i, sub := range m.State.Subscriptions {
					if i != m.SelectedSub {
						newSubs = append(newSubs, sub)
					}
				}
				m.State.Subscriptions = newSubs
				newNodes := []core.Node{}
				for _, node := range m.State.Nodes {
					if node.SubscriptionID != subID {
						newNodes = append(newNodes, node)
					}
				}
				m.State.Nodes = newNodes
				if err := core.SaveState(m.State); err != nil {
					m.addLog(fmt.Sprintf("Error saving state: %v", err))
				} else {
					m.addLog("Subscription deleted.")
				}
				if m.SelectedSub >= len(m.State.Subscriptions) {
					m.SelectedSub = len(m.State.Subscriptions) - 1
				}
				if m.SelectedSub < 0 {
					m.SelectedSub = 0
				}
				m.SelectedNode = 0
			}
			m.ActiveModal = ""
			return m, nil
		}
		return m, nil
	case "backspace":
		if len(m.ModalInput) > 0 {
			m.ModalInput = m.ModalInput[:len(m.ModalInput)-1]
		}
		return m, nil
	default:
		var content string
		if len(msg.String()) == 1 {
			content = msg.String()
		} else {
			content = strings.TrimPrefix(msg.String(), "\x1b[200~")
			content = strings.TrimSuffix(content, "\x1b[201~")
		}
		content = strings.ReplaceAll(content, "[", "")
		content = strings.ReplaceAll(content, "]", "")
		m.ModalInput += content
		return m, nil
	}
}

func (m *Model) handleScrollDown() {
	switch m.FocusedPanel {
	case PanelSubscriptions:
		if m.SelectedSub < len(m.State.Subscriptions)-1 {
			m.SelectedSub++
		}
	case PanelNodes:
		subID := ""
		if len(m.State.Subscriptions) > 0 && m.SelectedSub < len(m.State.Subscriptions) {
			subID = m.State.Subscriptions[m.SelectedSub].ID
		}
		count := 0
		for _, node := range m.State.Nodes {
			if node.SubscriptionID == subID {
				count++
			}
		}
		if m.SelectedNode < count-1 {
			m.SelectedNode++
		}
	case PanelOptions:
		if m.SelectedOption < 3 {
			m.SelectedOption++
		}
	}
}

func (m *Model) handleScrollUp() {
	switch m.FocusedPanel {
	case PanelSubscriptions:
		if m.SelectedSub > 0 {
			m.SelectedSub--
		}
	case PanelNodes:
		if m.SelectedNode > 0 {
			m.SelectedNode--
		}
	case PanelOptions:
		if m.SelectedOption > 0 {
			m.SelectedOption--
		}
	}
}
