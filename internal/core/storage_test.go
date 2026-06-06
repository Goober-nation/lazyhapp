package core

import (
	"os"
	"testing"
)

func TestSaveLoadState(t *testing.T) {
	// Use a temporary directory for state
	tmpDir, err := os.MkdirTemp("", "lazyhapp_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configDirOverride = tmpDir

	state := &AppState{
		Subscriptions: []Subscription{
			{ID: "sub-1", Name: "Test Sub", URL: "http://example.com"},
		},
		Nodes: []Node{
			{ID: "node-1", SubscriptionID: "sub-1", Name: "Test Node"},
		},
	}

	if err := SaveState(state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	loadedState, err := LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if len(loadedState.Subscriptions) != 1 || loadedState.Subscriptions[0].Name != "Test Sub" {
		t.Errorf("Loaded state subscriptions incorrect: %+v", loadedState.Subscriptions)
	}
	if len(loadedState.Nodes) != 1 || loadedState.Nodes[0].Name != "Test Node" {
		t.Errorf("Loaded state nodes incorrect: %+v", loadedState.Nodes)
	}
}
