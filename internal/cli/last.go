package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jupiterozeye/context/internal/clipboard"
	"github.com/jupiterozeye/context/internal/output"
	"github.com/spf13/cobra"
)

var (
	lastFormat string
	lastNoCopy bool
	lastPrint  bool
)

var lastCmd = &cobra.Command{
	Use:   "last [n]",
	Short: "Show last n commands with their output",
	Long:  `Show the last n commands with their output from the command logs and copy to clipboard.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLast,
}

func init() {
	rootCmd.AddCommand(lastCmd)
	lastCmd.Flags().StringVarP(&lastFormat, "format", "f", "raw", "Output format: raw|markdown|detailed")
	lastCmd.Flags().BoolVarP(&lastNoCopy, "no-copy", "c", false, "Don't copy to clipboard")
	lastCmd.Flags().BoolVarP(&lastPrint, "print", "p", false, "Print output to terminal (default: just confirmation)")
}

func runLast(cmd *cobra.Command, args []string) error {
	n := 1
	if len(args) > 0 {
		var err error
		n, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid number: %s", args[0])
		}
	}

	if n <= 0 {
		return fmt.Errorf("number must be positive")
	}

	reader := output.NewReader(output.Options{
		Format: lastFormat,
	})

	// Check if in recording session
	if !reader.IsRecording() {
		return fmt.Errorf("not in a recorded session. run 'context rec' first to start recording")
	}

	entries, err := reader.Read(n)
	if err != nil {
		return err
	}

	formatted := reader.FormatEntries(entries)

	// Copy to clipboard (unless --no-copy)
	if !lastNoCopy {
		if err := clipboard.Copy(formatted); err != nil {
			// Clipboard might not work in script session, just warn
			fmt.Fprintf(os.Stderr, "Warning: could not copy to clipboard: %v\n", err)
		} else {
			if n == 1 {
				fmt.Println("✓ Last command copied to clipboard")
			} else {
				fmt.Printf("✓ Last %d commands copied to clipboard\n", n)
			}
		}
	}

	// Print output only if --print flag is set
	if lastPrint {
		fmt.Print(formatted)
	}

	return nil
}
