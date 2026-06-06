package vpn

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type Client struct {
	cmd *exec.Cmd
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Connect(ctx context.Context, configPayload string) error {
	if c.cmd != nil && c.cmd.Process != nil {
		c.Disconnect()
	}

	// Check if we are running as root
	isRoot := os.Geteuid() == 0

	// Mocking the core binary execution (e.g., sing-box or xray)
	cmdName := "sleep"
	args := []string{"1000"}

	var cmd *exec.Cmd
	if isRoot {
		cmd = exec.CommandContext(ctx, cmdName, args...)
	} else {
		// Use sudo to invoke the command
		sudoArgs := append([]string{cmdName}, args...)
		cmd = exec.CommandContext(ctx, "sudo", sudoArgs...)
	}
	
	fmt.Printf("Connecting with payload: %s (root: %v)\n", configPayload, isRoot)
	c.cmd = cmd
	
	err := c.cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start VPN core: %w", err)
	}

	return nil
}

func (c *Client) Disconnect() error {
	if c.cmd != nil && c.cmd.Process != nil {
		err := c.cmd.Process.Kill()
		c.cmd = nil
		return err
	}
	return nil
}
