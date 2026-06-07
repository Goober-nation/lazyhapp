package core

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"lazyhapp/internal/logger"
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

func FetchSubscription(urlStr string, name string) ([]Subscription, []Node, error) {
	logger.Info("CORE", fmt.Sprintf("Fetching subscription: %s", urlStr))
	resp, err := http.Get(urlStr)
	if err != nil {
		logger.Error("CORE", fmt.Sprintf("Network error fetching %s: %v", urlStr, err))
		return nil, nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("CORE", fmt.Sprintf("Server returned status %d for %s", resp.StatusCode, urlStr))
		return nil, nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read error: %w", err)
	}

	// 1. Try structured JSON
	var data SubscriptionResponse
	if err := json.Unmarshal(body, &data); err == nil && len(data.Subscriptions) > 0 {
		return processStructuredData(data, urlStr, name)
	}

	// 2. Try JSON array
	var arrayData []SubscriptionData
	if err := json.Unmarshal(body, &arrayData); err == nil && len(arrayData) > 0 {
		data.Subscriptions = arrayData
		return processStructuredData(data, urlStr, name)
	}

	// 3. Try Base64 decoding
	decodedBody := string(body)
	if b64, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(body))); err == nil {
		decodedBody = string(b64)
	}

	// 4. Process as plain text list of links
	return processLinkList(decodedBody, urlStr, name)
}

func processStructuredData(data SubscriptionResponse, url string, name string) ([]Subscription, []Node, error) {
	var subs []Subscription
	var nodes []Node
	
	for i, sd := range data.Subscriptions {
		subID := fmt.Sprintf("sub-%d", i)
		
		var filteredNodes []Node
		for j, nd := range sd.Nodes {
			if isValidVPNNode(nd.Payload) {
				filteredNodes = append(filteredNodes, Node{
					ID:             fmt.Sprintf("node-%d-%d", i, j),
					SubscriptionID: subID,
					Name:           nd.Name,
					Protocol:       nd.Protocol,
					ConfigPayload:  nd.Payload,
				})
			}
		}
		
		if len(filteredNodes) == 0 {
			continue
		}

		subName := sd.Name
		if name != "" {
			subName = name
		}

		subs = append(subs, Subscription{
			ID:   subID,
			Name: subName,
			URL:  url,
		})
		nodes = append(nodes, filteredNodes...)
	}
	
	if len(subs) == 0 {
		return nil, nil, fmt.Errorf("no valid VPN nodes found in structured data")
	}

	return subs, nodes, nil
}


func isValidVPNNode(payload string) bool {
	payload = strings.ToLower(payload)
	// Protocols that should be present in a valid VPN config
	protocols := []string{"hysteria2://", "vless://", "vmess://", "trojan://", "shadowsocks://", "ss://"}
	for _, p := range protocols {
		if strings.Contains(payload, p) {
			return true
		}
	}
	// Fallback for structured data where protocol might be in a separate field but payload is provided
	// We can also check for common config patterns
	if strings.Contains(payload, "server=") && strings.Contains(payload, "port=") {
		return true
	}
	return false
}

func processLinkList(content, subURL string, name string) ([]Subscription, []Node, error) {
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

		if !isValidVPNNode(line) {
			continue
		}
		
		nodeName := u.Fragment
		if nodeName == "" {
			nodeName = fmt.Sprintf("Node %d", i+1)
		}
		
		nodes = append(nodes, Node{
			ID:             fmt.Sprintf("node-gen-%d", i),
			SubscriptionID: subID,
			Name:           nodeName,
			Protocol:       u.Scheme,
			ConfigPayload:  line,
		})
	}
	
	if len(nodes) == 0 {
		return nil, nil, fmt.Errorf("no valid VPN links found in response")
	}

	subName := "Generic Subscription"
	if name != "" {
		subName = name
	}

	subs := []Subscription{
		{
			ID:   subID,
			Name: subName,
			URL:  subURL,
		},
	}
	
	return subs, nodes, nil
}

