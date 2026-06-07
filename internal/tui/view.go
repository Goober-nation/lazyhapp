package tui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	activeBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")) // Vibrant blue/green

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255"))
)

func (m Model) View() string {
	// Account for header (1) + newline (1) + top/bottom padding (5+5 = 10)
	contentHeight := m.Height - 12
	if contentHeight < 10 {
		return "Terminal too small"
	}

	// Account for left/right padding (5+5 = 10)
	availableWidth := m.Width - 10
	panelWidth := availableWidth / 2
	panelHeight := contentHeight / 3

	topLeft := m.renderPanel("Subscriptions", m.renderSubscriptions(), PanelSubscriptions, panelWidth, panelHeight)
	topRight := m.renderPanel("Status", m.renderStatus(), PanelStatus, panelWidth, panelHeight)
	midLeft := m.renderPanel("Nodes", m.renderNodes(), PanelNodes, panelWidth, panelHeight)
	midRight := m.renderPanel("Logs", m.renderLogs(), PanelLogs, panelWidth, panelHeight)
	
	bottomPane := m.renderSystemInfo(availableWidth, panelHeight)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, topLeft, topRight)
	middleRow := lipgloss.JoinHorizontal(lipgloss.Top, midLeft, midRight)

	mainView := lipgloss.JoinVertical(lipgloss.Left, topRow, middleRow, bottomPane)
	
	// Apply padding around the calculated content
	finalView := lipgloss.NewStyle().
		Padding(5, 5).
		Render(mainView)

	header := " lazyhapp v0.1.0 "
	
	return fmt.Sprintf("%s\n%s", header, finalView)
}

func (m Model) renderPanel(title string, content string, id PanelID, w, h int) string {
	style := borderStyle
	if m.FocusedPanel == id {
		style = activeBorderStyle
	}

	return style.
		Width(w).
		Height(h).
		Render(fmt.Sprintf("%s\n%s", titleStyle.Render(title), content))
}

func (m Model) renderSubscriptions() string {
	var sb strings.Builder
	
	if len(m.State.Subscriptions) == 0 {
		sb.WriteString("No subscriptions added.\n")
	} else {
		for i, sub := range m.State.Subscriptions {
			prefix := "[ ]"
			if i == m.SelectedSub {
				prefix = "->"
			}
			sb.WriteString(fmt.Sprintf("%s %s\n", prefix, sub.Name))
		}
	}

	if m.ActiveModal == "add_sub" {
		sb.WriteString("\n" + strings.Repeat("-", 10) + "\n")
		sb.WriteString(fmt.Sprintf("URL: %s_", m.ModalInput))
	} else {
		sb.WriteString("\n(a: add sub)")
	}

	return sb.String()
}

func (m Model) renderNodes() string {
	if len(m.State.Subscriptions) == 0 || m.SelectedSub < 0 {
		return "Select a subscription first"
	}
	
	subID := m.State.Subscriptions[m.SelectedSub].ID
	var sb strings.Builder
	found := false
	for i, node := range m.State.Nodes {
		if node.SubscriptionID == subID {
			found = true
			prefix := "[ ]"
			if i == m.SelectedNode {
				prefix = "->"
			}
			sb.WriteString(fmt.Sprintf("%s %s (%dms)\n", prefix, node.Name, node.LastMeasuredPing))
		}
	}
	if !found {
		return "No nodes found for this subscription"
	}
	return sb.String()
}

func (m Model) renderStatus() string {
	status := "DISCONNECTED"
	if m.Connected {
		status = "CONNECTED"
	}
	
	node := "None"
	if m.CurrentNode != "" {
		node = m.CurrentNode
	}

	protocol := "N/A"
	if m.Connected {
		protocol = "Hysteria2"
	}

	return fmt.Sprintf("Status: %s\nCurrent Node: %s\nProtocol: %s\nPing: -- | Uptime: 00:00:00", status, node, protocol)
}

func (m Model) renderLogs() string {
	if len(m.Logs) == 0 {
		return "No logs available."
	}
	
	var sb strings.Builder
	start := 0
	if len(m.Logs) > 10 {
		start = len(m.Logs) - 10
	}
	
	for _, log := range m.Logs[start:] {
		sb.WriteString(log + "\n")
	}
	return sb.String()
}

func (m Model) renderSystemInfo(w, h int) string {
	sysInfo := fmt.Sprintf("OS: %s | Arch: %s | Core: %s\n", runtime.GOOS, runtime.GOARCH, "None")
	sysInfo += "------------------------------------------------------------\n"
	sysInfo += "esc: back | q: quit | Tab: cycle panels | j/k: scroll\n"
	sysInfo += "a: add sub | d: delete sub | r: refresh | ,.: switch sub"
	
	style := borderStyle
	if m.FocusedPanel == PanelSystemInfo {
		style = activeBorderStyle
	}
	
	return style.
		Width(w).
		Height(h).
		Render(sysInfo)
}
