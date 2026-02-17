// wrapper.go - provides a command that wraps other commands for capture
package cmdexec

import (
	"fmt"
	"os"
	"strings"

	"github.com/creack/pty"
)

// RunWrappedCommand executes a command with full PTY capture and returns exit code
// This is designed to be called from shell preexec hooks
func RunWrappedCommand(logDir string, command string) int {
	executor := NewExecutor(logDir)

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return 0
	}

	cmd := parts[0]
	args := parts[1:]

	// Run with capture - output goes to terminal via PTY
	result, err := executor.Run(cmd, args...)

	// Print a marker so the shell knows we're done
	// This helps shells distinguish our output from the next prompt
	if result != nil {
		fmt.Fprintf(os.Stderr, "\033]133;C;exit=%d\007", result.ExitCode)
	}

	if err != nil {
		if result != nil {
			return result.ExitCode
		}
		return 1
	}

	return 0
}

// IsAvailable checks if PTY execution is available
func IsAvailable() bool {
	// Check if we can create a PTY by testing with 'true' command
	cmd := exec.Command("true")
	_, err := pty.Start(cmd)
	return err == nil
}
