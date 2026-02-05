package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Print shell integration setup instructions",
	Long:  `Print instructions for setting up shell integration to enable the 'context last' command.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(`Shell Integration Setup
========================

To enable 'context last', add one of the following to your shell config:

Bash (~/.bashrc):
  source /usr/local/share/context/shell/context.bash

Zsh (~/.zshrc):
  source /usr/local/share/context/shell/context.zsh

Fish (~/.config/fish/config.fish):
  source /usr/local/share/context/shell/context.fish

Or use the install script:
  ./scripts/install.sh

After setup, restart your terminal or run:
  source ~/.bashrc  # or ~/.zshrc, etc.
`)
	},
}