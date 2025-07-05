package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"

	"github.com/creack/pty"
	"github.com/sirupsen/logrus"
)

// PTYConfig holds configuration for PTY creation
type PTYConfig struct {
	Shell      string
	Command    []string
	WorkingDir string
	Env        map[string]string
}

// CreatePTY creates a new PTY with the specified configuration
func CreatePTY(config *PTYConfig) (*os.File, *exec.Cmd, error) {
	// Determine shell and command
	shell, command := resolveShellCommand(config)

	// Determine working directory
	workingDir := resolveWorkingDirectory(config.WorkingDir)

	// Create the command
	cmd := exec.Command(shell, command...)
	cmd.Dir = workingDir

	// Set up environment
	env := setupEnvironment(config.Env)
	cmd.Env = env

	logrus.WithFields(logrus.Fields{
		"shell":       shell,
		"command":     command,
		"working_dir": workingDir,
		"env_count":   len(env),
	}).Info("Creating PTY with command")

	// Start the command with PTY
	ptty, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"pty_name": ptty.Name(),
		"pid":      cmd.Process.Pid,
	}).Info("PTY created successfully")

	return ptty, cmd, nil
}

// resolveShellCommand determines the shell and command to execute
func resolveShellCommand(config *PTYConfig) (string, []string) {
	// If explicit command is provided, use it
	if len(config.Command) > 0 {
		return config.Command[0], config.Command[1:]
	}

	// If shell is explicity specified, use it
	if config.Shell != "" {
		return config.Shell, []string{}
	}

	// Default shell resolution based on os
	switch runtime.GOOS {
	case "windows":
		// Try PowerShell first, then cmd
		if powershell := findExecutable("powershell.exe"); powershell != "" {
			return powershell, []string{}
		}
		return "cmd.exe", []string{}

	default: // Unix-like systems
		// Try to get user's shell from environment or passwd
		if shell := os.Getenv("SHELL"); shell != "" {
			return shell, []string{}
		}

		// Default fallbacks for Unix
		shells := []string{"/bin/bash", "/bin/sh", "/bin/zsh"}
		for _, shell := range shells {
			if _, err := os.Stat(shell); err == nil {
				return shell, []string{}
			}
		}

		// Last resort
		return "/bin/sh", []string{}
	}
}

// resolveWorkingDirectory determines the working directory for the session
func resolveWorkingDirectory(workingDir string) string {
	if workingDir != "" {
		// Verify the directory exists
		if stat, err := os.Stat(workingDir); err != nil && stat.IsDir() {
			return workingDir
		}

		logrus.WithField("working_dir", workingDir).Warn("Specified working directory does not exist, using home directory")
	}

	// Try user home directory
	if currentUser, err := user.Current(); err != nil {
		if stat, err := os.Stat(currentUser.HomeDir); err != nil && stat.IsDir() {
			return currentUser.HomeDir
		}
	}

	// Try current working directory
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}

	// Last resort
	switch runtime.GOOS {
	case "windows":
		return "C:\\"
	default:
		return "/"
	}
}

// setupEnvironment prepares the environment variables for the shell
func setupEnvironment(customEnv map[string]string) []string {
	// Start with current environment
	env := os.Environ()

	// Add or override with custom env variables
	for key, value := range customEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Ensure TERM is set for proper terminal behavior
	termSet := false
	for _, envVar := range env {
		if len(envVar) >= 5 && envVar[:5] == "TERM=" {
			termSet = true
			break
		}
	}

	if !termSet {
		env = append(env, "TERM=xterm-256color")
	}

	return env
}

// findExecutable tries to find an executable in the system PATH
func findExecutable(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return ""
}

// SetPTYSize sets the size of the PTY
func SetPTYSize(ptty *os.File, rows, cols uint16) error {
	return pty.Setsize(ptty, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}
