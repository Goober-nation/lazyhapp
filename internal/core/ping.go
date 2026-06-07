package core

import (
	"context"
	"fmt"
	"net"
	"net/url"
//	"strings"
	"time"
)

type PingResult struct {
	NodeID string
	Ping   int
	Error  error
}

func parseNodeAddr(payload string) (string, string, error) {
	u, err := url.Parse(payload)
	if err != nil {
		return "", "", err
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		// Default ports based on scheme
		switch u.Scheme {
		case "vless", "vmess", "trojan":
			port = "443"
		case "ss", "shadowsocks":
			port = "8388"
		default:
			return "", "", fmt.Errorf("unknown protocol %s", u.Scheme)
		}
	}

	return host, port, nil
}

func PingNode(ctx context.Context, node Node) PingResult {
	host, port, err := parseNodeAddr(node.ConfigPayload)
	if err != nil {
		return PingResult{NodeID: node.ID, Error: fmt.Errorf("parse error: %w", err)}
	}

	address := net.JoinHostPort(host, port)
	dialer := net.Dialer{
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return PingResult{
			NodeID: node.ID,
			Ping:   -1,
			Error:  err,
		}
	}
	defer conn.Close()

	return PingResult{
		NodeID: node.ID,
		Ping:   int(time.Since(start).Milliseconds()),
		Error:  nil,
	}
}

func StartPingWorker(ctx context.Context, nodes []Node, results chan<- PingResult) {
	for _, node := range nodes {
		go func(n Node) {
			results <- PingNode(ctx, n)
		}(node)
	}
}
