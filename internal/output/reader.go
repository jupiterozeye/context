package output

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
	opts   Options
	logDir string
}

// NewReader creates a new log reader
func NewReader(opts Options) *Reader {
	homeDir, _ := os.UserHomeDir()
	return &Reader{
		opts:   opts,
		logDir: filepath.Join(homeDir, ".context", "logs"),
	}
}

// Read retrieves the last n log entries
func (r *Reader) Read(n int) ([]LogEntry, error) {
	// Get all log files
	files, err := os.ReadDir(r.logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no log directory found. Have you enabled shell integration?")
		}
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	// Filter and sort log files by modification time (newest first)
	var logFiles []os.FileInfo
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".log") {
			info, err := file.Info()
			if err == nil {
				logFiles = append(logFiles, info)
			}
		}
	}

	if len(logFiles) == 0 {
		return nil, fmt.Errorf("no log files found. Have you run any commands?")
	}

	// Sort by modification time (newest first)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].ModTime().After(logFiles[j].ModTime())
	})

	// Read the last n entries
	var entries []LogEntry
	for i := 0; i < n && i < len(logFiles); i++ {
		entry, err := r.parseLogFile(filepath.Join(r.logDir, logFiles[i].Name()))
		if err == nil && entry != nil {
			entries = append(entries, *entry)
		}
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

	// Parse headers
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
			fmt.Sscanf(durStr, "%d", &entry.Duration)
		} else if strings.HasPrefix(line, "=== EXIT_CODE: ") {
			codeStr := strings.TrimPrefix(line, "=== EXIT_CODE: ")
			fmt.Sscanf(codeStr, "%d", &entry.ExitCode)
		} else if strings.HasPrefix(line, "=== WORKING_DIR: ") {
			entry.WorkingDir = strings.TrimPrefix(line, "=== WORKING_DIR: ")
		}
	}

	entry.Output = strings.Join(outputLines, "\n")

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entry, nil
}

// FormatEntries formats log entries according to the specified format
func (r *Reader) FormatEntries(entries []LogEntry) string {
	var result strings.Builder

	for i, entry := range entries {
		switch r.opts.Format {
		case "markdown":
			r.formatMarkdown(&result, entry, i+1)
		case "detailed":
			r.formatDetailed(&result, entry, i+1)
		default: // raw
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
	result.WriteString(fmt.Sprintf("**Command:** `%s`\n\n", entry.Command))
	result.WriteString(fmt.Sprintf("**Working Directory:** `%s`\n\n", entry.WorkingDir))
	result.WriteString(fmt.Sprintf("**Exit Code:** %d\n\n", entry.ExitCode))
	result.WriteString(fmt.Sprintf("**Duration:** %s\n\n", entry.Duration))

	if entry.Output != "" {
		result.WriteString("**Output:**\n\n```\n")
		result.WriteString(entry.Output)
		result.WriteString("\n```\n\n")
	}
}

func (r *Reader) formatDetailed(result *strings.Builder, entry LogEntry, num int) {
	result.WriteString(fmt.Sprintf("Command %d: %s\n", num, entry.Command))
	result.WriteString(fmt.Sprintf("  Working Directory: %s\n", entry.WorkingDir))
	result.WriteString(fmt.Sprintf("  Start Time: %s\n", entry.StartTime.Format("2006-01-02 15:04:05")))
	result.WriteString(fmt.Sprintf("  Exit Code: %d\n", entry.ExitCode))
	result.WriteString(fmt.Sprintf("  Duration: %s\n", entry.Duration))

	if entry.Output != "" {
		result.WriteString("  Output:\n")
		// Indent output
		output := strings.ReplaceAll(entry.Output, "\n", "\n    ")
		result.WriteString("    ")
		result.WriteString(output)
		result.WriteString("\n")
	}
	result.WriteString("\n")
}

// CleanupOldLogs removes logs older than the specified number of days
func CleanupOldLogs(days int) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	logDir := filepath.Join(homeDir, ".context", "logs")
	cutoff := time.Now().AddDate(0, 0, -days)

	files, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".log") {
			info, err := file.Info()
			if err == nil && info.ModTime().Before(cutoff) {
				os.Remove(filepath.Join(logDir, file.Name()))
			}
		}
	}

	return nil
}
