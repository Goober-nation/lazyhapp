package core

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchSubscription(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantSubs int
		wantNodes int
		wantErr  bool
	}{
		{
			name: "Valid structured JSON",
			body: `{
				"subscriptions": [
					{
						"name": "Sub 1",
						"nodes": [
							{"name": "Node 1", "protocol": "Hysteria2", "payload": "payload1"},
							{"name": "Node 2", "protocol": "Hysteria2", "payload": "payload2"}
						]
					}
				]
			}`,
			wantSubs: 1,
			wantNodes: 2,
			wantErr:  false,
		},
		{
			name: "Valid array JSON",
			body: `[
				{
					"name": "Sub 2",
					"nodes": [
						{"name": "Node 3", "protocol": "Hysteria2", "payload": "payload3"}
					]
				}
			]`,
			wantSubs: 1,
			wantNodes: 1,
			wantErr:  false,
		},
		{
			name: "Invalid JSON",
			body: `invalid json`,
			wantSubs: 0,
			wantNodes: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			subs, nodes, err := FetchSubscription(server.URL)

			if (err != nil) != tt.wantErr {
				t.Errorf("FetchSubscription() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(subs) != tt.wantSubs {
				t.Errorf("FetchSubscription() got %d subs, want %d", len(subs), tt.wantSubs)
			}
			if len(nodes) != tt.wantNodes {
				t.Errorf("FetchSubscription() got %d nodes, want %d", len(nodes), tt.wantNodes)
			}
		})
	}
}
