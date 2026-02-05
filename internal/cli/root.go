package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "context",
	Short: "Terminal context capture tool for AI-assisted debugging",
	Long: `Context is a CLI tool that simplifies sharing terminal context with AI.

It can generate directory trees and capture previous terminal outputs,
automatically copying them to your clipboard for easy sharing.

Usage:
  context dir [path]     - Generate directory tree and copy to clipboard
  context last [n]       - Copy last n terminal outputs to clipboard

Setup:
  To use 'context last', you need to source the shell integration:
    Bash:  source /path/to/context/shell/context.bash
    Zsh:   source /path/to/context/shell/context.zsh
    Fish:  source /path/to/context/shell/context.fish`,
}

func Execute() error {
	return rootCmd.Execute()
}