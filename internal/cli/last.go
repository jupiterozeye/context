package cli

import (
	"fmt"
	"strconv"

	"github.com/jupiterozeye/context/internal/clipboard"
	"github.com/jupiterozeye/context/internal/last"
	"github.com/spf13/cobra"
)

var (
	lastRaw    bool
	lastFormat string
	lastNoCopy bool
)

var lastCmd = &cobra.Command{
	Use:   "last [n]",
	Short: "Copy last n terminal outputs to clipboard",
	Long:  `Copy the previous terminal command outputs to the clipboard. Requires shell integration to be set up.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLast,
}

func init() {
	rootCmd.AddCommand(lastCmd)
	lastCmd.Flags().BoolVarP(&lastRaw, "raw", "r", false, "Raw output without formatting")
	lastCmd.Flags().StringVarP(&lastFormat, "format", "f", "raw", "Output format: raw|command|markdown")
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

	reader := last.NewReader(last.Options{
		Format: lastFormat,
		Raw:    lastRaw,
	})

	output, err := reader.Read(n)
	if err != nil {
		return fmt.Errorf("failed to read history: %w", err)
	}

	fmt.Print(output)

	if !lastNoCopy {
		if err := clipboard.Copy(output); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		fmt.Println("\nCopied to clipboard!")
	}

	return nil
}