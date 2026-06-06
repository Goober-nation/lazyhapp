package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type SubscriptionResponse struct {
	Subscriptions []SubscriptionData `json:"subscriptions"`
}

type SubscriptionData struct {
	Name  string `json:"name"`
	Nodes []NodeData `json:"nodes"`
}

type NodeData struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Payload  string `json:"payload"`
}

func FetchSubscription(url string) ([]Subscription, []Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read error: %w", err)
	}

	var data SubscriptionResponse
	if err := json.Unmarshal(body, &data); err != nil || len(data.Subscriptions) == 0 {
		// Try parsing as a plain array of subscriptions
		var arrayData []SubscriptionData
		if err := json.Unmarshal(body, &arrayData); err != nil {
			return nil, nil, fmt.Errorf("json parse error: %w", err)
		}
		data.Subscriptions = arrayData
	}

	var subs []Subscription
	var nodes []Node

	for i, sd := range data.Subscriptions {
		subID := fmt.Sprintf("sub-%d", i)
		subs = append(subs, Subscription{
			ID:   subID,
			Name: sd.Name,
			URL:  url,
		})

		for j, nd := range sd.Nodes {
			nodes = append(nodes, Node{
				ID:             fmt.Sprintf("node-%d-%d", i, j),
				SubscriptionID: subID,
				Name:           nd.Name,
				Protocol:       nd.Protocol,
				ConfigPayload:  nd.Payload,
			})
		}
	}

	return subs, nodes, nil
}
