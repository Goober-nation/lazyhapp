package core

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Subscription struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	RefreshDate string `json:"refresh_date"`
}

type Node struct {
	ID               string `json:"id"`
	SubscriptionID   string `json:"subscription_id"`
	Name             string `json:"name"`
	Protocol         string `json:"protocol"`
	ConfigPayload    string `json:"config_payload"`
	LastMeasuredPing int    `json:"last_measured_ping"`
}

type AppState struct {
	Subscriptions []Subscription `json:"subscriptions"`
	Nodes         []Node         `json:"nodes"`
	Logs          []string       `json:"logs"`
	VpnPid        int            `json:"vpn_pid"`
	CurrentNode   string         `json:"current_node"`
}

var configDirOverride string

func GetConfigDir() string {
	if configDirOverride != "" {
		return configDirOverride
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "lazyhapp")
}

func GetStateFilePath() string {
	return filepath.Join(GetConfigDir(), "state.json")
}

func LoadState() (*AppState, error) {
	path := GetStateFilePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &AppState{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state AppState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func SaveState(state *AppState) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(GetStateFilePath(), data, 0644)
}
