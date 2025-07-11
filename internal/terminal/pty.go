package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/creack/pty"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"
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

	// Configure PTY terminal attributes for web terminal use
	if err := configurePTYTerminalAttributes(ptty); err != nil {
		logrus.WithError(err).Warn("Failed to configure PTY terminal attributes, continuing anyway")
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
		// Ensure interactive mode for the specified shell
		return config.Shell, getInteractiveArgs(config.Shell)
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
			return shell, getInteractiveArgs(shell)
		}

		// Default fallbacks for Unix
		shells := []string{"/bin/bash", "/bin/sh", "/bin/zsh"}
		for _, shell := range shells {
			if _, err := os.Stat(shell); err == nil {
				return shell, getInteractiveArgs(shell)
			}
		}

		// Last resort
		return "/bin/sh", getInteractiveArgs("/bin/sh")
	}
}

// getInteractiveArgs returns the arguments needed to start a shell in interactive mode
func getInteractiveArgs(shell string) []string {
	switch filepath.Base(shell) {
	case "bash":
		return []string{"-i"}
	case "zsh":
		return []string{"-i"}
	case "sh":
		return []string{"-i"}
	case "fish":
		return []string{"-i"}
	default:
		// For unknown shells, try with -i flag
		return []string{"-i"}
	}
}

// resolveWorkingDirectory determines the working directory for the session
func resolveWorkingDirectory(workingDir string) string {
	if workingDir != "" {
		// Verify the directory exists and is a directory
		if stat, err := os.Stat(workingDir); err == nil && stat.IsDir() {
			return workingDir
		}

		logrus.WithField("working_dir", workingDir).Warn("Specified working directory does not exist, using home directory")
	}

	// Try user home directory
	if currentUser, err := user.Current(); err == nil {
		if stat, err := os.Stat(currentUser.HomeDir); err == nil && stat.IsDir() {
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

	// Ensure essential environment variables are set for interactive shells
	essentialVars := map[string]string{
		"TERM":    "xterm-256color",
		"COLUMNS": "80",
		"LINES":   "24",
		"PS1":     "$ ",
		"PS2":     "> ",
		"PS3":     "#? ",
		"PS4":     "+ ",
	}

	// Check which essential variables are already set
	existingVars := make(map[string]bool)
	for _, envVar := range env {
		if idx := strings.Index(envVar, "="); idx > 0 {
			existingVars[envVar[:idx]] = true
		}
	}

	// Add missing essential variables
	for key, value := range essentialVars {
		if !existingVars[key] {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
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

// configurePTYTerminalAttributes configures the PTY for web terminal use
func configurePTYTerminalAttributes(ptty *os.File) error {
	// For web terminals, we need to configure the PTY properly
	// to ensure the shell stays interactive and doesn't exit immediately

	// Set the PTY to raw mode to handle terminal control sequences properly
	// This is essential for interactive shells to work correctly
	if _, err := term.MakeRaw(int(ptty.Fd())); err != nil {
		return fmt.Errorf("failed to set pty to raw mode: %w", err)
	}

	if err := pty.Setsize(ptty, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	}); err != nil {
		return fmt.Errorf("failed to set initial PTY size: %w", err)
	}

	logrus.Debug("PTY terminal attributes configured for web terminal use with proper sizing")
	return nil
}
