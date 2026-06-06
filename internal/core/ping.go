package core

import (
	"context"
	"time"
)

type PingResult struct {
	NodeID string
	Ping   int
	Error  error
}

func PingNode(ctx context.Context, node Node) PingResult {
	// In a real app, we'd use the actual node address/port from the config payload
	// Mocking a ping by trying to connect to a random port or using ICMP
	start := time.Now()
	
	// Mocking network latency
	select {
	case <-time.After(time.Duration(10 + (time.Now().UnixNano()%100)) * time.Millisecond):
		return PingResult{
			NodeID: node.ID,
			Ping:   int(time.Since(start).Milliseconds()),
			Error:  nil,
		}
	case <-ctx.Done():
		return PingResult{NodeID: node.ID, Error: ctx.Err()}
	}
}

func StartPingWorker(ctx context.Context, nodes []Node, results chan<- PingResult) {
	for _, node := range nodes {
		go func(n Node) {
			results <- PingNode(ctx, n)
		}(node)
	}
}
