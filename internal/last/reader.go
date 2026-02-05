package last

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Options struct {
	Format string
	Raw    bool
}

type Reader struct {
	opts Options
}

func NewReader(opts Options) *Reader {
	return &Reader{opts: opts}
}

func (r *Reader) Read(n int) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot get home directory: %w", err)
	}

	// Try to read from shell history
	var commands []string

	// Try zsh history first
	zshHistory := filepath.Join(homeDir, ".zsh_history")
	if cmds, err := r.readZshHistory(zshHistory, n); err == nil && len(cmds) > 0 {
		commands = cmds
	} else {
		// Try bash history
		bashHistory := filepath.Join(homeDir, ".bash_history")
		if cmds, err := r.readBashHistory(bashHistory, n); err == nil && len(cmds) > 0 {
			commands = cmds
		} else {
			return "", fmt.Errorf("no shell history found. Searched for ~/.zsh_history and ~/.bash_history")
		}
	}

	if len(commands) == 0 {
		return "", fmt.Errorf("no history entries found")
	}

	return r.formatOutput(commands), nil
}

func (r *Reader) readZshHistory(path string, n int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Zsh history format can be "command" or ": timestamp:0;command"
		if strings.Contains(line, ";") {
			parts := strings.SplitN(line, ";", 2)
			if len(parts) == 2 {
				line = parts[1]
			}
		}
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "context") {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last n entries
	start := len(lines) - n
	if start < 0 {
		start = 0
	}
	return lines[start:], nil
}

func (r *Reader) readBashHistory(path string, n int) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "context") {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last n entries
	start := len(lines) - n
	if start < 0 {
		start = 0
	}
	return lines[start:], nil
}

func (r *Reader) formatOutput(commands []string) string {
	var result strings.Builder

	for i, cmd := range commands {
		if r.opts.Raw {
			result.WriteString(cmd)
			if i < len(commands)-1 {
				result.WriteString("\n")
			}
		} else if r.opts.Format == "markdown" {
			result.WriteString(fmt.Sprintf("### Command %d\n\n```bash\n%s\n```\n\n", i+1, cmd))
		} else {
			result.WriteString(fmt.Sprintf("Command %d: %s\n", i+1, cmd))
		}
	}

	return result.String()
}
