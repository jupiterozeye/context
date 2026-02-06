package cli

import (
	"fmt"
	"strconv"

	"github.com/jupiterozeye/context/internal/clipboard"
	"github.com/jupiterozeye/context/internal/output"
	"github.com/spf13/cobra"
)

var (
	lastFormat string
	lastNoCopy bool
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
	lastCmd.Flags().BoolVarP(&lastNoCopy, "no-copy", "c", false, "Print only, don't copy to clipboard")
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

	entries, err := reader.Read(n)
	if err != nil {
		return fmt.Errorf("failed to read command output logs: %w", err)
	}

	formatted := reader.FormatEntries(entries)
	fmt.Print(formatted)

	if !lastNoCopy {
		if err := clipboard.Copy(formatted); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		fmt.Println("\nCopied to clipboard!")
	}

	return nil
}
