package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

func FetchSubscription(urlStr string) ([]Subscription, []Node, error) {
	resp, err := http.Get(urlStr)
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

	// 1. Try structured JSON
	var data SubscriptionResponse
	if err := json.Unmarshal(body, &data); err == nil && len(data.Subscriptions) > 0 {
		return processStructuredData(data, urlStr)
	}

	// 2. Try JSON array
	var arrayData []SubscriptionData
	if err := json.Unmarshal(body, &arrayData); err == nil && len(arrayData) > 0 {
		data.Subscriptions = arrayData
		return processStructuredData(data, urlStr)
	}

	// 3. Try Base64 decoding
	decodedBody := string(body)
	if b64, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(body))); err == nil {
		decodedBody = string(b64)
	}

	// 4. Process as plain text list of links
	return processLinkList(decodedBody, urlStr)
}

func processStructuredData(data SubscriptionResponse, url string) ([]Subscription, []Node, error) {
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

func processLinkList(content, subURL string) ([]Subscription, []Node, error) {
	lines := strings.Split(content, "\n")
	var nodes []Node
	
	subID := "generic-sub"
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		u, err := url.Parse(line)
		if err != nil {
			continue
		}

		name := u.Fragment
		if name == "" {
			name = fmt.Sprintf("Node %d", i+1)
		}

		nodes = append(nodes, Node{
			ID:             fmt.Sprintf("node-gen-%d", i),
			SubscriptionID: subID,
			Name:           name,
			Protocol:       u.Scheme,
			ConfigPayload:  line,
		})
	}

	if len(nodes) == 0 {
		return nil, nil, fmt.Errorf("no valid VPN links found in response")
	}

	subs := []Subscription{
		{
			ID:   subID,
			Name: "Generic Subscription",
			URL:  subURL,
		},
	}

	return subs, nodes, nil
}
