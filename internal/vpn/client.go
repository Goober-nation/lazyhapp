package vpn

import (
	"context"
	"fmt"
	"lazyhapp/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type Client struct {
	cmd        *exec.Cmd
	configPath string
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) generateXrayConfig(payload string) (string, error) {
	configDir := filepath.Join(os.TempDir(), "lazyhapp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	configPath := filepath.Join(configDir, "config.json")
	
	configContent, err := ConvertURIToXrayConfig(payload)
	if err != nil {
		return "", fmt.Errorf("failed to convert URI to config: %w", err)
	}

	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		return "", err
	}

	return configPath, nil
}

func (c *Client) Connect(ctx context.Context, configPayload string) (int, error) {
	logger.Info("VPN", "Starting connection process...")
	c.Disconnect(0)

	isRoot := os.Geteuid() == 0

	configPath, err := c.generateXrayConfig(configPayload)
	if err != nil {
		logger.Error("VPN", fmt.Sprintf("Failed to generate config: %v", err))
		return 0, fmt.Errorf("failed to generate config: %w", err)
	}
	c.configPath = configPath

	binaryPath, err := ResolveBinaryPath()
	if err != nil {
		logger.Error("VPN", fmt.Sprintf("Failed to resolve binary: %v", err))
		return 0, fmt.Errorf("failed to resolve xray binary: %w", err)
	}

	args := []string{"run", "-c", configPath}

	var cmd *exec.Cmd
	if isRoot {
		cmd = exec.CommandContext(ctx, binaryPath, args...)
	} else {
		sudoArgs := append([]string{binaryPath}, args...)
		cmd = exec.CommandContext(ctx, "sudo", sudoArgs...)
	}
	
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	
	logFile, err := os.OpenFile("lazyhapp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open log file: %w", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	c.cmd = cmd
	err = c.cmd.Start()
	if err != nil {
		logger.Error("VPN", fmt.Sprintf("Failed to start Xray: %v", err))
		logFile.Close()
		return 0, fmt.Errorf("failed to start Xray core: %w", err)
	}

	logger.Info("VPN", fmt.Sprintf("Xray started successfully with PID %d", c.cmd.Process.Pid))
	return c.cmd.Process.Pid, nil
}

func (c *Client) Disconnect(pid int) error {
	logger.Info("VPN", fmt.Sprintf("Disconnecting PID %d...", pid))
	targetPid := pid
	if targetPid == 0 && c.cmd != nil && c.cmd.Process != nil {
		targetPid = c.cmd.Process.Pid
	}

	if targetPid == 0 {
		return nil
	}

	proc, err := os.FindProcess(targetPid)
	if err != nil {
		return err
	}

	err = proc.Kill()
	if err != nil {
		logger.Error("VPN", fmt.Sprintf("Failed to kill process %d: %v", targetPid, err))
	} else {
		logger.Info("VPN", fmt.Sprintf("Process %d killed", targetPid))
	}
	c.cmd = nil
	return err
}

func (c *Client) Cleanup(pid int) {
	c.Disconnect(pid)
	exec.Command("ip", "link", "delete", "tun0").Run()
	exec.Command("ip", "route", "flush", "cache").Run()
	if c.configPath != "" {
		os.Remove(c.configPath)
	}
}
