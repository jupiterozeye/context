package cli

import (
	"fmt"
	"os"
	"os/exec"

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

	// Create log directory
	logDir := homeDir + "/.context/logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Detect shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	fmt.Println("🎥 Starting recorded shell session...")
	fmt.Println("   Commands and output will be saved for 'context last'")
	fmt.Println("   Type 'exit' to stop recording")
	fmt.Println()

	// Start script session
	typescriptPath := homeDir + "/.context/typescript"
	scriptCmd := exec.Command("script", "-q", "-a", typescriptPath, "-c", shell+" -i")
	scriptCmd.Stdin = os.Stdin
	scriptCmd.Stdout = os.Stdout
	scriptCmd.Stderr = os.Stderr
	scriptCmd.Env = append(os.Environ(), "CONTEXT_RECORDING=1")

	if err := scriptCmd.Run(); err != nil {
		// Don't error on normal exit
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}
