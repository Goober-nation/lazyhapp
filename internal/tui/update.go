package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"lazyhapp/internal/core"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.ContentHeight = m.Height - HeaderHeight - FooterHeight

		if m.ContentHeight >= 15 {
			availableWidth := m.Width - 4
			panelWidth := availableWidth / 2
			m.LogViewport.Width = panelWidth
			m.LogViewport.Height = m.ContentHeight
		}
		return m, nil


	case TickMsg:
		newLogs, offset := readLogFile(m.LogOffset)
		if len(newLogs) > 0 {
			m.Logs = append(m.Logs, newLogs...)
			m.LogOffset = offset
			m.State.Logs = m.Logs
			core.SaveState(m.State)
			m.updateLogViewport()
		}
		return m, tickCmd()

	case LogUpdateMsg:
		m.Logs = append(m.Logs, msg...)
		m.State.Logs = m.Logs
		core.SaveState(m.State)
		m.updateLogViewport()
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
