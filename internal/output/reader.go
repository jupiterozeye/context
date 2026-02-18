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
	sessionDir     string
	isRecording    bool
}

// NewReader creates a new log reader
func NewReader(opts Options) *Reader {
	homeDir, _ := os.UserHomeDir()
	
	// Check if we're in a recorded session
	sessionID := os.Getenv("CONTEXT_SESSION_ID")
	isRecording := os.Getenv("CONTEXT_RECORDING") == "1"
	
	var sessionDir string
	if sessionID != "" {
		// Use session-specific directory
		sessionDir = filepath.Join(homeDir, ".context", "logs", sessionID)
		isRecording = true
	} else {
		// Fall back to main logs directory (for backwards compatibility)
		sessionDir = filepath.Join(homeDir, ".context", "logs")
	}
	
	return &Reader{
		opts:           opts,
		sessionDir:     sessionDir,
		isRecording:    isRecording,
	}
}

// IsRecording returns true if in a recorded session
func (r *Reader) IsRecording() bool {
	return r.isRecording
}

// Read retrieves the last n log entries
func (r *Reader) Read(n int) ([]LogEntry, error) {
	// Must be in a recording session
	if !r.isRecording {
		return nil, fmt.Errorf("not in a recorded session. run 'context rec' first to start recording")
	}

	// Read from session log files
	return r.readFromLogFiles(n)
}

// readFromLogFiles reads from individual log files
func (r *Reader) readFromLogFiles(n int) ([]LogEntry, error) {
	files, err := os.ReadDir(r.sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no commands recorded yet. run some commands first")
		}
		return nil, fmt.Errorf("failed to read log directory: %w", err)
	}

	// Filter and sort log files by modification time (newest first)
	var logFiles []os.DirEntry
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".log") {
			logFiles = append(logFiles, file)
		}
	}

	if len(logFiles) == 0 {
		return nil, fmt.Errorf("no commands recorded yet. run some commands first")
	}

	// Sort by mod time descending
	sort.Slice(logFiles, func(i, j int) bool {
		fi, _ := logFiles[i].Info()
		fj, _ := logFiles[j].Info()
		return fi.ModTime().After(fj.ModTime())
	})

	// Read the last n entries
	var entries []LogEntry
	for i := 0; i < n && i < len(logFiles); i++ {
		entry, err := r.parseLogFile(filepath.Join(r.sessionDir, logFiles[i].Name()))
		if err == nil && entry != nil {
			entries = append(entries, *entry)
		}
	}

	// Reverse so oldest is first
	reverseEntries(entries)

	return entries, nil
}

func reverseEntries(entries []LogEntry) {
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
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

	// Clean output
	output := strings.Join(outputLines, "\n")
	output = stripANSI(output)
	output = strings.TrimSpace(output)
	
	// Remove the "$ command" line from output if present
	lines := strings.Split(output, "\n")
	if len(lines) > 0 && strings.HasPrefix(lines[0], "$ ") {
		lines = lines[1:]
	}
	output = strings.Join(lines, "\n")
	
	output = filterBatWarnings(output)

	entry.Output = output

	return entry, nil
}

func filterBatWarnings(output string) string {
	if !strings.Contains(output, "[bat warning]") {
		return output
	}
	lines := strings.Split(output, "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, "[bat warning]") {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}

// stripANSI removes ANSI escape codes
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")
	return ansiRegex.ReplaceAllString(s, "")
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
