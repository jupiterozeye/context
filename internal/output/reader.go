package output

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// LogEntry represents a single logged command with its output
type LogEntry struct {
	Command    string
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	ExitCode   int
	WorkingDir string
	Output     string
}

// Options for reading log files
type Options struct {
	Format string // raw, markdown, detailed
}

// Reader handles reading and parsing log files
type Reader struct {
	opts           Options
	logDir         string
	typescriptPath string
}

// NewReader creates a new log reader
func NewReader(opts Options) *Reader {
	homeDir, _ := os.UserHomeDir()
	return &Reader{
		opts:           opts,
		logDir:         filepath.Join(homeDir, ".context", "logs"),
		typescriptPath: filepath.Join(homeDir, ".context", "typescript"),
	}
}

// Read retrieves the last n log entries
func (r *Reader) Read(n int) ([]LogEntry, error) {
	// First, try to read from typescript if it exists (has actual output)
	if entries, err := r.readFromTypescript(n); err == nil && len(entries) > 0 {
		return entries, nil
	}

	// Fall back to log files (command metadata only)
	return r.readFromLogFiles(n)
}

// readFromLogFiles reads from individual log files
func (r *Reader) readFromLogFiles(n int) ([]LogEntry, error) {
	files, err := os.ReadDir(r.logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no log directory found. Enable shell integration first.")
		}
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	// Filter and sort log files by name (which includes timestamp)
	var logFiles []os.DirEntry
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".log") {
			logFiles = append(logFiles, file)
		}
	}

	if len(logFiles) == 0 {
		return nil, fmt.Errorf("no log files found. Run some commands first.")
	}

	// Sort by name descending (newest first based on timestamp in filename)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].Name() > logFiles[j].Name()
	})

	// Read the last n entries
	var entries []LogEntry
	for i := 0; i < n && i < len(logFiles); i++ {
		entry, err := r.parseLogFile(filepath.Join(r.logDir, logFiles[i].Name()))
		if err == nil && entry != nil {
			entries = append(entries, *entry)
		}
	}

	// Reverse so oldest is first (natural reading order)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

// parseLogFile parses a single log file into a LogEntry
func (r *Reader) parseLogFile(path string) (*LogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entry := &LogEntry{}
	scanner := bufio.NewScanner(file)

	inOutput := false
	var outputLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "=== OUTPUT:") {
			inOutput = true
			continue
		}

		if inOutput {
			outputLines = append(outputLines, line)
			continue
		}

		// Parse header fields
		if strings.HasPrefix(line, "=== COMMAND: ") {
			entry.Command = strings.TrimPrefix(line, "=== COMMAND: ")
		} else if strings.HasPrefix(line, "=== START_TIME: ") {
			timeStr := strings.TrimPrefix(line, "=== START_TIME: ")
			entry.StartTime, _ = time.Parse("2006-01-02 15:04:05", timeStr)
		} else if strings.HasPrefix(line, "=== END_TIME: ") {
			timeStr := strings.TrimPrefix(line, "=== END_TIME: ")
			entry.EndTime, _ = time.Parse("2006-01-02 15:04:05", timeStr)
		} else if strings.HasPrefix(line, "=== DURATION: ") {
			durStr := strings.TrimPrefix(line, "=== DURATION: ")
			durStr = strings.TrimSuffix(durStr, "s")
			var secs int
			fmt.Sscanf(durStr, "%d", &secs)
			entry.Duration = time.Duration(secs) * time.Second
		} else if strings.HasPrefix(line, "=== EXIT_CODE: ") {
			codeStr := strings.TrimPrefix(line, "=== EXIT_CODE: ")
			fmt.Sscanf(codeStr, "%d", &entry.ExitCode)
		} else if strings.HasPrefix(line, "=== WORKING_DIR: ") {
			entry.WorkingDir = strings.TrimPrefix(line, "=== WORKING_DIR: ")
		}
	}

	// Clean output - remove ANSI codes and bat warnings
	output := strings.Join(outputLines, "\n")
	output = stripANSI(output)
	output = strings.TrimSpace(output)

	// Filter out bat warnings
	if strings.Contains(output, "[bat warning]") {
		lines := strings.Split(output, "\n")
		var filtered []string
		for _, line := range lines {
			if !strings.Contains(line, "[bat warning]") {
				filtered = append(filtered, line)
			}
		}
		output = strings.Join(filtered, "\n")
	}

	entry.Output = output

	return entry, nil
}

// readFromTypescript reads from the script typescript file
func (r *Reader) readFromTypescript(n int) ([]LogEntry, error) {
	if _, err := os.Stat(r.typescriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no typescript file")
	}

	content, err := os.ReadFile(r.typescriptPath)
	if err != nil {
		return nil, err
	}

	entries := r.parseTypescript(string(content))
	if len(entries) == 0 {
		return nil, fmt.Errorf("no commands in typescript")
	}

	// Return last n entries
	if n > len(entries) {
		n = len(entries)
	}
	return entries[len(entries)-n:], nil
}

// parseTypescript parses the script typescript format
func (r *Reader) parseTypescript(content string) []LogEntry {
	var entries []LogEntry

	// Clean ANSI codes
	content = stripANSI(content)

	lines := strings.Split(content, "\n")

	// Prompt patterns
	promptPatterns := []*regexp.Regexp{
		regexp.MustCompile(`[~\/][^\s]*\s*❯\s*(.+)$`),
		regexp.MustCompile(`[~\/][^\s]*\s*>\s+(.+)$`),
		regexp.MustCompile(`\$\s+(.+)$`),
		regexp.MustCompile(`%\s+(.+)$`),
	}

	var currentEntry *LogEntry
	var outputLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip noise
		if len(line) == 0 || strings.HasPrefix(line, "Script ") {
			continue
		}

		// Check for command prompt
		var command string
		for _, pattern := range promptPatterns {
			if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
				cmd := strings.TrimSpace(matches[1])
				if len(cmd) > 1 && !strings.HasPrefix(cmd, "context") {
					command = cmd
					break
				}
			}
		}

		if command != "" {
			if currentEntry != nil {
				currentEntry.Output = cleanOutput(strings.Join(outputLines, "\n"))
				entries = append(entries, *currentEntry)
			}
			currentEntry = &LogEntry{Command: command}
			outputLines = []string{}
		} else if currentEntry != nil && !strings.HasPrefix(line, "Copied to clipboard") {
			outputLines = append(outputLines, line)
		}
	}

	if currentEntry != nil {
		currentEntry.Output = cleanOutput(strings.Join(outputLines, "\n"))
		entries = append(entries, *currentEntry)
	}

	return entries
}

// stripANSI removes ANSI escape codes
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[PX^_][^\x1b]*\x1b\\|\x1b\[[0-9;]*[mKHJPq]`)
	return ansiRegex.ReplaceAllString(s, "")
}

// cleanOutput cleans output text
func cleanOutput(s string) string {
	s = strings.TrimSpace(s)
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}

// FormatEntries formats log entries
func (r *Reader) FormatEntries(entries []LogEntry) string {
	var result strings.Builder

	for i, entry := range entries {
		switch r.opts.Format {
		case "markdown":
			r.formatMarkdown(&result, entry, i+1)
		case "detailed":
			r.formatDetailed(&result, entry, i+1)
		default:
			r.formatRaw(&result, entry)
		}
	}

	return result.String()
}

func (r *Reader) formatRaw(result *strings.Builder, entry LogEntry) {
	result.WriteString(fmt.Sprintf("$ %s\n", entry.Command))
	if entry.Output != "" {
		result.WriteString(entry.Output)
		result.WriteString("\n")
	}
}

func (r *Reader) formatMarkdown(result *strings.Builder, entry LogEntry, num int) {
	result.WriteString(fmt.Sprintf("### Command %d\n\n", num))
	result.WriteString(fmt.Sprintf("```bash\n$ %s\n", entry.Command))
	if entry.Output != "" {
		result.WriteString(entry.Output)
		result.WriteString("\n")
	}
	result.WriteString("```\n\n")
}

func (r *Reader) formatDetailed(result *strings.Builder, entry LogEntry, num int) {
	result.WriteString(fmt.Sprintf("Command %d: %s\n", num, entry.Command))
	if entry.WorkingDir != "" {
		result.WriteString(fmt.Sprintf("  Directory: %s\n", entry.WorkingDir))
	}
	if entry.ExitCode != 0 {
		result.WriteString(fmt.Sprintf("  Exit Code: %d\n", entry.ExitCode))
	}
	if entry.Output != "" {
		result.WriteString("  Output:\n")
		for _, line := range strings.Split(entry.Output, "\n") {
			result.WriteString("    ")
			result.WriteString(line)
			result.WriteString("\n")
		}
	}
	result.WriteString("\n")
}
