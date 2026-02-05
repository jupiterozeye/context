package last

import (
	"bufio"
	"encoding/json"
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

type HistoryEntry struct {
	Timestamp int64  `json:"timestamp"`
	Command   string `json:"command"`
	Output    string `json:"output"`
	ExitCode  int    `json:"exit_code"`
	Pwd       string `json:"pwd"`
}

func (r *Reader) Read(n int) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot get home directory: %w", err)
	}

	logPath := filepath.Join(homeDir, ".local", "share", "context", "history.jsonl")
	
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("history file not found. Run 'context init' for setup instructions")
		}
		return "", fmt.Errorf("cannot read history: %w", err)
	}
	defer file.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading history: %w", err)
	}

	if len(entries) == 0 {
		return "", fmt.Errorf("no history entries found")
	}

	start := len(entries) - n
	if start < 0 {
		start = 0
	}

	selected := entries[start:]
	return r.formatOutput(selected), nil
}

func (r *Reader) formatOutput(entries []HistoryEntry) string {
	var result strings.Builder

	for i, entry := range entries {
		if r.opts.Raw {
			result.WriteString(entry.Output)
			if i < len(entries)-1 {
				result.WriteString("\n")
			}
		} else if r.opts.Format == "markdown" {
			result.WriteString(fmt.Sprintf("### Command: %s\n\n```\n%s\n```\n\n", entry.Command, entry.Output))
		} else {
			result.WriteString(fmt.Sprintf("=== Command: %s ===\n%s\n", entry.Command, entry.Output))
		}
	}

	return result.String()
}