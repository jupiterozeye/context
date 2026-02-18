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
	Long:  `Starts a new shell session where all commands and output are recorded.`,
	RunE:  runRec,
}

func init() {
	rootCmd.AddCommand(recCmd)
}

func runRec(cmd *cobra.Command, args []string) error {
	// Check if already recording
	if os.Getenv("CONTEXT_RECORDING") == "1" {
		return fmt.Errorf("already in a recorded session")
	}

	// Remove old typescript if exists
	os.Remove("typescript")

	// Detect shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}

	fmt.Println("🎥 Starting recorded shell session...")
	fmt.Println("   Type 'exit' to stop recording")
	fmt.Println()

	// Start script session
	scriptCmd := exec.Command("script", "-q", "-c", shell+" -i")
	scriptCmd.Stdin = os.Stdin
	scriptCmd.Stdout = os.Stdout
	scriptCmd.Stderr = os.Stderr
	scriptCmd.Env = append(os.Environ(),
		"CONTEXT_RECORDING=1",
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
