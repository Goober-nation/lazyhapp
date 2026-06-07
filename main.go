package main

import (
	"fmt"
	"lazyhapp/internal/logger"
	"lazyhapp/internal/tui"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/bubbletea"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Println("lazyhapp requires root privileges for network management.")
		fmt.Println("Relaunching with sudo...")
		
		args := append([]string{"sudo"}, os.Args[0])
		args = append(args, os.Args[1:]...)
		
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error relaunching with sudo: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err := logger.Init(); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	m := tui.InitialModel()
	
	// Handle graceful shutdown to clear NIC traces
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		os.Exit(0)
	}()


	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
