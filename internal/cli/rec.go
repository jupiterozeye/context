package cli

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var recCmd = &cobra.Command{
	Use:   "rec",
	Short: "Start a recorded shell session",
	Long:  `Starts a new shell session where all commands and output are recorded for context last.`,
	RunE:  runRec,
}

func init() {
	rootCmd.AddCommand(recCmd)
}

func runRec(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot get home directory: %w", err)
	}

	// Generate unique session ID
	sessionID := generateSessionID()

	// Create session log directory
	logDir := filepath.Join(homeDir, ".context", "logs", sessionID)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Detect shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	fmt.Println("🎥 Starting recorded shell session...")
	fmt.Printf("   Session ID: %s\n", sessionID)
	fmt.Println("   Commands and output will be saved for 'context last'")
	fmt.Println("   Type 'exit' to stop recording")
	fmt.Println()

	// Start script session with session ID in environment
	typescriptPath := filepath.Join(logDir, "typescript")
	scriptCmd := exec.Command("script", "-q", "-a", typescriptPath, "-c", shell+" -i")
	scriptCmd.Stdin = os.Stdin
	scriptCmd.Stdout = os.Stdout
	scriptCmd.Stderr = os.Stderr
	scriptCmd.Env = append(os.Environ(), 
		"CONTEXT_RECORDING=1",
		"CONTEXT_SESSION_ID="+sessionID,
	)

	if err := scriptCmd.Run(); err != nil {
		// Don't error on normal exit
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}

func generateSessionID() string {
	// Generate 8 random bytes
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp
		return fmt.Sprintf("%d", time.Now().Unix())
	}
	// Convert to hex
	return fmt.Sprintf("%x", b)
}
