package cli

import (
	"fmt"

	"github.com/jupiterozeye/context/internal/clipboard"
	"github.com/jupiterozeye/context/internal/dir"
	"github.com/spf13/cobra"
)

var (
	dirDepth    int
	dirExclude  string
	dirHidden   bool
	dirFormat   string
	dirNoCopy   bool
)

var dirCmd = &cobra.Command{
	Use:   "dir [path]",
	Short: "Generate directory tree and copy to clipboard",
	Long:  `Generate a file structure tree of the specified directory (or current directory if not specified) and copy it to the clipboard.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDir,
}

func init() {
	rootCmd.AddCommand(dirCmd)
	dirCmd.Flags().IntVarP(&dirDepth, "depth", "d", 0, "Max depth (0 = unlimited)")
	dirCmd.Flags().StringVarP(&dirExclude, "exclude", "e", "", "Comma-separated patterns to exclude (e.g., 'node_modules,.git')")
	dirCmd.Flags().BoolVarP(&dirHidden, "hidden", "H", false, "Include hidden files")
	dirCmd.Flags().StringVarP(&dirFormat, "format", "f", "tree", "Output format: tree|json|markdown")
	dirCmd.Flags().BoolVarP(&dirNoCopy, "no-copy", "c", false, "Print only, don't copy to clipboard")
}

func runDir(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	generator := dir.NewGenerator(dir.Options{
		MaxDepth:     dirDepth,
		Exclude:      dirExclude,
		IncludeHidden: dirHidden,
		Format:       dirFormat,
	})

	output, err := generator.Generate(path)
	if err != nil {
		return fmt.Errorf("failed to generate tree: %w", err)
	}

	fmt.Print(output)

	if !dirNoCopy {
		if err := clipboard.Copy(output); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		fmt.Println("\nCopied to clipboard!")
	}

	return nil
}