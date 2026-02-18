package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var flushCmd = &cobra.Command{
	Use:   "flush",
	Short: "Clear all context logs",
	Long:  `Removes all log files and typescript recordings for debugging.`,
	RunE:  runFlush,
}

func init() {
	rootCmd.AddCommand(flushCmd)
}

func runFlush(cmd *cobra.Command, args []string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot get home directory: %w", err)
	}

	logDir := filepath.Join(homeDir, ".context", "logs")
	typescriptPath := filepath.Join(homeDir, ".context", "typescript")

	// Remove typescript file
	if err := os.Remove(typescriptPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: could not remove typescript: %v\n", err)
	}

	// Remove all log files
	files, err := os.ReadDir(logDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read log directory: %w", err)
	}

	count := 0
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		path := filepath.Join(logDir, f.Name())
		if err := os.Remove(path); err == nil {
			count++
		}
	}

	fmt.Printf("✓ Flushed %d log files\n", count)
	return nil
}
