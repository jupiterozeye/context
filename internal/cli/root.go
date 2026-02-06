package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "context",
	Short: "Terminal context capture tool for AI-assisted debugging",
	Long: `Context is a CLI tool that simplifies sharing terminal context with AI.

It can generate directory trees and show your recent shell history,
automatically copying them to your clipboard for easy sharing.

Usage:
  context dir [path]     - Generate directory tree and copy to clipboard
  context last [n]       - Show last n commands from shell history`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() error {
	return rootCmd.Execute()
}
