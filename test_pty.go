package main

import (
	"fmt"
	"time"

	"github.com/piyushgupta53/webterm/internal/terminal"
	"github.com/piyushgupta53/webterm/internal/types"
)

func main() {
	fmt.Println("Testing PTY creation...")

	config := &terminal.PTYConfig{
		Shell:      "", // Use default shell
		WorkingDir: "", // Use default working directory
	}

	ptty, cmd, err := terminal.CreatePTY(config)
	if err != nil {
		fmt.Printf("Error creating PTY: %v\n", err)
		return
	}
	defer ptty.Close()

	fmt.Printf("PTY created successfully!\n")
	fmt.Printf("PTY name: %s\n", ptty.Name())
	fmt.Printf("Process PID: %d\n", cmd.Process.Pid)

	// Test pipes
	pipeManager := terminal.NewPipeManager("/tmp/webterm-test-pipes")
	inputPipe, outputFile, err := pipeManager.CreateSessionPipes("test-session")
	if err != nil {
		fmt.Printf("Error creating pipes: %v\n", err)
		return
	}

	fmt.Printf("Input pipe: %s\n", inputPipe)
	fmt.Printf("Output file: %s\n", outputFile)

	// Cleanup
	time.Sleep(2 * time.Second)
	cleanupManager := terminal.NewCleanupManager(pipeManager)
	session := &types.Session{
		ID:         "test-session",
		PTY:        ptty,
		Process:    cmd,
		InputPipe:  inputPipe,
		OutputFile: outputFile,
	}

	cleanupManager.CleanupSession(session)
	fmt.Println("Cleanup completed!")
}
