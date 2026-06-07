package tui

import (
	"fmt"
	"lazyhapp/internal/core"
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
	contentHeight := m.Height - 6
	if contentHeight < 15 {
		return "Terminal too small"
	}
	
	availableWidth := m.Width - 4
	panelWidth := availableWidth / 2
	panelHeight := contentHeight / 4
	
	leftCol := lipgloss.JoinVertical(lipgloss.Left, 
		m.renderPanel("Subscriptions", m.renderSubscriptions(panelHeight), PanelSubscriptions, panelWidth, panelHeight),
		m.renderPanel("Nodes", m.renderNodes(panelHeight), PanelNodes, panelWidth, panelHeight),
		m.renderPanel("Options", m.renderOptions(panelHeight), PanelOptions, panelWidth, panelHeight),
	)
	
	rightCol := m.renderPanel("Logs", m.renderLogs(panelHeight*3), PanelLogs, panelWidth, panelHeight*3)
	
	topSection := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
	bottomPane := m.renderSystemStatusInfo(availableWidth, panelHeight)
	
	mainView := lipgloss.JoinVertical(lipgloss.Left, topSection, bottomPane)
	
	finalView := lipgloss.NewStyle().
		Padding(0, 0).
		Render(mainView)
	
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Render(" lazyhapp v0.1.0 ")
	
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

func (m Model) renderSubscriptions(h int) string {
	var sb strings.Builder
	availableLines := h - 2
	if availableLines < 0 {
		availableLines = 0
	}
	if len(m.State.Subscriptions) == 0 {
		sb.WriteString("No subscriptions added.\n")
	} else {
		start := m.SelectedSub - (availableLines / 2)
		if start < 0 {
			start = 0
		}
		end := start + availableLines
		if end > len(m.State.Subscriptions) {
			end = len(m.State.Subscriptions)
			start = end - availableLines
			if start < 0 {
				start = 0
			}
		}
		for i := start; i < end; i++ {
			sub := m.State.Subscriptions[i]
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
	} else if m.ActiveModal == "reset_confirm" {
		sb.WriteString("\n" + strings.Repeat("-", 10) + "\n")
		sb.WriteString("Reset all data? (enter: yes / esc: no)")
	} else if m.ActiveModal == "remove_sub" {
		sb.WriteString("\n" + strings.Repeat("-", 10) + "\n")
		sb.WriteString("Delete subscription? (enter: yes / esc: no)")
	} else if m.ActiveModal == "add_sub_name" {
		sb.WriteString("\n" + strings.Repeat("-", 10) + "\n")
		sb.WriteString(fmt.Sprintf("Name for %s:\n%s_", m.tempSubUrl, m.ModalInput))
	} else {
		sb.WriteString("\n")
	}
	lines := strings.Split(sb.String(), "\n")

	if len(lines) > h {
		return strings.Join(lines[:h], "\n")
	}
	return sb.String()
}

func (m Model) renderNodes(h int) string {
	if len(m.State.Subscriptions) == 0 || m.SelectedSub < 0 {
		return "Select a subscription first"
	}
	subID := m.State.Subscriptions[m.SelectedSub].ID
	var filteredNodes []core.Node
	for _, node := range m.State.Nodes {
		if node.SubscriptionID == subID {
			filteredNodes = append(filteredNodes, node)
		}
	}
	if len(filteredNodes) == 0 {
		return "No nodes found for this subscription"
	}
	var sb strings.Builder
	availableLines := h - 1
	if availableLines < 0 {
		availableLines = 0
	}
	relativeSelected := m.SelectedNode
	if relativeSelected < 0 {
		relativeSelected = 0
	}
	if relativeSelected >= len(filteredNodes) {
		relativeSelected = len(filteredNodes) - 1
	}
	start := relativeSelected - (availableLines / 2)
	if start < 0 {
		start = 0
	}
	end := start + availableLines
	if end > len(filteredNodes) {
		end = len(filteredNodes)
		start = end - availableLines
		if start < 0 {
			start = 0
		}
	}
	for i := start; i < end; i++ {
		node := filteredNodes[i]
		checkbox := "[ ]"
		if m.Connected && m.CurrentNode == node.Name {
			checkbox = "[x]"
		}
		prefix := "  "
		if i == relativeSelected {
			prefix = "->"
		}
		sb.WriteString(fmt.Sprintf("%s %s %s (%dms)\n", prefix, checkbox, node.Name, node.LastMeasuredPing))
	}
	lines := strings.Split(sb.String(), "\n")
	if len(lines) > h {
		return strings.Join(lines[:h], "\n")
	}
	return sb.String()
}

func (m Model) renderLogs(h int) string {
	if len(m.Logs) == 0 {
		return "No logs available."
	}
	availableLines := h - 1
	if availableLines < 0 {
		availableLines = 0
	}

	start := 0
	if len(m.Logs) > availableLines {
		start = len(m.Logs) - availableLines
	}

	var sb strings.Builder
	availableWidth := m.Width - 4
	for _, log := range m.Logs[start:] {
		if len(log) > availableWidth {
			sb.WriteString(log[:availableWidth-3] + "...\n")
		} else {
			sb.WriteString(log + "\n")
		}
	}
	return sb.String()
}

func (m Model) renderOptions(h int) string {
	options := []string{
		"Auto-ping nodes: Enabled",
		"Default Protocol: Hysteria2",
		"Log level: Info",
		"DNS: System Default",
	}
	var sb strings.Builder
	sb.WriteString("Configuration:\n")
	for i, opt := range options {
		prefix := "[ ]"
		if i == m.SelectedOption {
			prefix = "->"
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", prefix, opt))
	}
	sb.WriteString("\n(o: change option)")
	lines := strings.Split(sb.String(), "\n")
	if len(lines) > h {
		return strings.Join(lines[:h], "\n")
	}
	return sb.String()
}

func (m Model) renderSystemStatusInfo(w, h int) string {
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
	
	up := "0 KB/s"
	down := "0 KB/s"
	if m.Connected {
		up = m.UpSpeed
		down = m.DownSpeed
	}
	
	sysInfo := fmt.Sprintf("Status: %s | Node: %s | Protocol: %s | Up: %s | Down: %s\n", status, node, protocol, up, down)
	sysInfo += fmt.Sprintf("OS: %s | Arch: %s\n", runtime.GOOS, runtime.GOARCH)
	sysInfo += "------------------------------------------------------------\n"
	sysInfo += "esc: back | q: quit | Tab: cycle panels | j/k: scroll\n"
	sysInfo += "a: add sub | d: delete sub | r: refresh | ,.: switch sub\n"
	sysInfo += "p: ping all | c: disconnect | o: change option"
	style := borderStyle
	if m.FocusedPanel == PanelSystemInfo {
		style = activeBorderStyle
	}
	return style.
		Width(w).
		Height(h).
		Render(sysInfo)
}

func (m Model) renderStatus() string { return "" }
func (m Model) renderSystemInfo(w, h int) string { return "" }
