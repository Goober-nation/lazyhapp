package tui

import (
	"context"
	"fmt"
	"lazyhapp/internal/core"
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

// Msg for ping results
type PingResultMsg struct {
	Result core.PingResult
	Node   core.Node
}

type PingTickMsg struct {
	Node core.Node
}

func fetchSubCmd(url string) tea.Cmd {
	return func() tea.Msg {
		subs, nodes, err := core.FetchSubscription(url)
		return SubFetchMsg{Subs: subs, Nodes: nodes, Err: err}
	}
}

func pingNodeCmd(ctx context.Context, node core.Node) tea.Cmd {
	return func() tea.Msg {
		res := core.PingNode(ctx, node)
		return PingResultMsg{Result: res, Node: node}
	}
}

func schedulePingCmd(node core.Node, delay time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(delay)
		return PingTickMsg{Node: node}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case SubFetchMsg:
		if msg.Err != nil {
			m.Logs = append(m.Logs, fmt.Sprintf("Error fetching sub: %v", msg.Err))
		} else {
			m.State.Subscriptions = append(m.State.Subscriptions, msg.Subs...)
			m.State.Nodes = append(m.State.Nodes, msg.Nodes...)
			m.Logs = append(m.Logs, fmt.Sprintf("Successfully added %d subscriptions", len(msg.Subs)))
		}
		return m, nil

	case PingResultMsg:
		for i := range m.State.Nodes {
			if m.State.Nodes[i].ID == msg.Result.NodeID {
				m.State.Nodes[i].LastMeasuredPing = msg.Result.Ping
				break
			}
		}
		return m, schedulePingCmd(msg.Node, 5*time.Second)

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
			m.FocusedPanel = (m.FocusedPanel + 1) % 4
			return m, nil

		case "shift+tab":
			m.FocusedPanel = (m.FocusedPanel + 3) % 4
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
				m.Logs = append(m.Logs, "Entering URL...")
			}
			return m, nil

		case "d":
			if m.FocusedPanel == PanelSubscriptions {
				m.ActiveModal = "remove_sub"
				m.Logs = append(m.Logs, "Preparing to remove subscription...")
			}
			return m, nil

		case "r":
			if m.FocusedPanel == PanelSubscriptions && len(m.State.Subscriptions) > 0 {
				sub := m.State.Subscriptions[m.SelectedSub]
				m.Logs = append(m.Logs, fmt.Sprintf("Refreshing %s...", sub.Name))
				return m, fetchSubCmd(sub.URL)
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
					err := m.VpnClient.Connect(context.Background(), selectedNode.ConfigPayload)
					if err == nil {
						m.Connected = true
						m.CurrentNode = selectedNode.Name
						m.Logs = append(m.Logs, fmt.Sprintf("Connected to %s", selectedNode.Name))
					} else {
						m.Logs = append(m.Logs, fmt.Sprintf("Connection failed: %v", err))
					}
				}
			}
			return m, nil

		case "c":
			if m.FocusedPanel == PanelNodes {
				m.VpnClient.Disconnect()
				m.Connected = false
				m.CurrentNode = ""
				m.Logs = append(m.Logs, "Disconnected from VPN.")
			}
			return m, nil

		case "p":
			if m.FocusedPanel == PanelNodes {
				m.Logs = append(m.Logs, "Starting async ping check...")
				var cmds []tea.Cmd
				for _, node := range m.State.Nodes {
					cmds = append(cmds, pingNodeCmd(context.Background(), node))
				}
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) handleModalInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Debugging: log every key pressed when in modal using %s to avoid quote confusion
	m.Logs = append(m.Logs, fmt.Sprintf("Key pressed: %s", msg.String()))
	if len(m.Logs) > 20 {
		m.Logs = m.Logs[1:]
	}

	switch msg.String() {
	case "esc":
		m.ActiveModal = ""
		return m, nil
	case "enter":
		if m.ActiveModal == "add_sub" {
			url := m.ModalInput
			
			// Strip leading/trailing brackets if present (some terminals/paste modes)
			url = strings.Trim(url, "[] ")
			
			if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				url = "https://" + url
			}
			m.Logs = append(m.Logs, fmt.Sprintf("Fetching subscription: %s", url))
			cmd := fetchSubCmd(url)
			m.ActiveModal = ""
			return m, cmd
		}
		return m, nil
	case "backspace":
		if len(m.ModalInput) > 0 {
			m.ModalInput = m.ModalInput[:len(m.ModalInput)-1]
		}
		return m, nil
	default:
		// Avoid adding control sequences to the input
		var content string
		if len(msg.String()) == 1 {
			content = msg.String()
		} else {
			// Handle pasted blocks by stripping potential bracketed paste markers
			// \x1b[200~ and \x1b[201~
			content = strings.TrimPrefix(msg.String(), "\x1b[200~")
			content = strings.TrimSuffix(content, "\x1b[201~")
		}
		
		// Remove square brackets from input/pasted text
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
	}
}
