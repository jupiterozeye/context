package dir

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Options struct {
	MaxDepth      int
	Exclude       string
	IncludeHidden bool
	Format        string
}

type Generator struct {
	opts   Options
	excludes []string
}

func NewGenerator(opts Options) *Generator {
	excludes := []string{}
	if opts.Exclude != "" {
		excludes = strings.Split(opts.Exclude, ",")
		for i := range excludes {
			excludes[i] = strings.TrimSpace(excludes[i])
		}
	}
	return &Generator{
		opts:     opts,
		excludes: excludes,
	}
}

func (g *Generator) Generate(rootPath string) (string, error) {
	info, err := os.Stat(rootPath)
	if err != nil {
		return "", fmt.Errorf("cannot access %s: %w", rootPath, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", rootPath)
	}

	rootName := filepath.Base(rootPath)
	if rootName == "." {
		cwd, _ := os.Getwd()
		rootName = filepath.Base(cwd)
	}

	output := rootName + "/\n"
	entries, err := g.readDir(rootPath, 1)
	if err != nil {
		return "", err
	}

	output += g.formatTree(entries, "")
	return output, nil
}

type entry struct {
	name     string
	path     string
	isDir    bool
	children []entry
}

func (g *Generator) readDir(path string, depth int) ([]entry, error) {
	if g.opts.MaxDepth > 0 && depth > g.opts.MaxDepth {
		return nil, nil
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, nil
	}

	var entries []entry
	for _, file := range files {
		name := file.Name()

		if !g.opts.IncludeHidden && strings.HasPrefix(name, ".") {
			continue
		}

		if g.isExcluded(name) {
			continue
		}

		isDir := file.IsDir()
		e := entry{
			name:  name,
			path:  filepath.Join(path, name),
			isDir: isDir,
		}

		if isDir {
			children, _ := g.readDir(e.path, depth+1)
			e.children = children
		}

		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return entries[i].name < entries[j].name
	})

	return entries, nil
}

func (g *Generator) isExcluded(name string) bool {
	for _, pattern := range g.excludes {
		if pattern == name {
			return true
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

func (g *Generator) formatTree(entries []entry, prefix string) string {
	var result strings.Builder

	for i, e := range entries {
		isLast := i == len(entries)-1

		connector := "├── "
		if isLast {
			connector = "└── "
		}

		result.WriteString(prefix + connector + e.name)
		if e.isDir {
			result.WriteString("/")
		}
		result.WriteString("\n")

		if len(e.children) > 0 {
			extension := "│   "
			if isLast {
				extension = "    "
			}
			result.WriteString(g.formatTree(e.children, prefix+extension))
		}
	}

	return result.String()
}