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
	// Check which source is more recent
	tsTime := r.getModTime(r.typescriptPath)
	logTime := r.getLatestLogTime()

	// Use typescript if it exists and is newer than logs
	if !tsTime.IsZero() && (logTime.IsZero() || tsTime.After(logTime)) {
		if entries, err := r.readFromTypescript(n); err == nil && len(entries) > 0 {
			return entries, nil
		}
	}

	// Fall back to log files
	return r.readFromLogFiles(n)
}

func (r *Reader) getModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func (r *Reader) getLatestLogTime() time.Time {
	files, err := os.ReadDir(r.logDir)
	if err != nil {
		return time.Time{}
	}

	var latest time.Time
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".log") {
			info, err := f.Info()
			if err == nil && info.ModTime().After(latest) {
				latest = info.ModTime()
			}
		}
	}
	return latest
}

// readFromLogFiles reads from individual log files
func (r *Reader) readFromLogFiles(n int) ([]LogEntry, error) {
	files, err := os.ReadDir(r.logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no log directory found. enable shell integration first")
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
		return nil, fmt.Errorf("no log files found. run some commands first")
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
		entry, err := r.parseLogFile(filepath.Join(r.logDir, logFiles[i].Name()))
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

// readFromTypescript reads from the script typescript file
func (r *Reader) readFromTypescript(n int) ([]LogEntry, error) {
	content, err := os.ReadFile(r.typescriptPath)
	if err != nil {
		return nil, err
	}

	entries := r.parseTypescript(string(content))
	if len(entries) == 0 {
		return nil, fmt.Errorf("no commands found in typescript")
	}

	// Return last n entries
	if n > len(entries) {
		n = len(entries)
	}
	return entries[len(entries)-n:], nil
}

// parseTypescript parses the script typescript format
func (r *Reader) parseTypescript(content string) []LogEntry {
	// First, completely strip all escape sequences
	content = stripAllEscapes(content)

	var entries []LogEntry
	lines := strings.Split(content, "\n")

	// Find commands from window titles (]2;) which are now clean
	titlePattern := regexp.MustCompile(`\]2;(.+)$`)

	var currentEntry *LogEntry
	var outputLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Skip script headers
		if strings.HasPrefix(trimmed, "Script ") || strings.HasPrefix(trimmed, "Copied to clipboard") {
			continue
		}

		// Look for command in title sequence
		if matches := titlePattern.FindStringSubmatch(line); len(matches) > 1 {
			cmd := strings.TrimSpace(matches[1])

			// Skip prompt-only titles
			if cmd == "~" || cmd == "/" || regexp.MustCompile(`^[~/][^ ]*$`).MatchString(cmd) {
				continue
			}

			// Skip context commands
			if strings.HasPrefix(cmd, "context ") || cmd == "context" {
				continue
			}

			// Skip exit
			if cmd == "exit" || strings.HasPrefix(cmd, "exit ") {
				continue
			}

			// Save previous entry
			if currentEntry != nil && len(outputLines) > 0 {
				currentEntry.Output = cleanOutput(strings.Join(outputLines, "\n"))
				entries = append(entries, *currentEntry)
			}

			// Start new entry
			currentEntry = &LogEntry{Command: cmd}
			outputLines = nil
			continue
		}

		// Collect output
		if currentEntry != nil {
			// Skip prompt lines
			if isPromptLine(trimmed) {
				continue
			}
			outputLines = append(outputLines, line)
		}
	}

	// Save last entry
	if currentEntry != nil && len(outputLines) > 0 {
		currentEntry.Output = cleanOutput(strings.Join(outputLines, "\n"))
		entries = append(entries, *currentEntry)
	}

	return entries
}

// stripAllEscapes removes ALL escape sequences
func stripAllEscapes(content string) string {
	// OSC sequences: ESC ] <params> BEL or ESC \ 
	// Params can include digits, semicolons, letters
	// Keep removing until stable
	for {
		// Match OSC with BEL terminator
		newContent := regexp.MustCompile(`\x1b\][0-9;:A-Za-z]+[^\x07\x1b]*\x07`).ReplaceAllString(content, "")
		// Match OSC with ST terminator (ESC \)
		newContent = regexp.MustCompile(`\x1b\][0-9;:A-Za-z]+[^\x07\x1b]*\x1b\\`).ReplaceAllString(newContent, "")
		// Catch any remaining OSC-like sequences
		newContent = regexp.MustCompile(`\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)?`).ReplaceAllString(newContent, "")
		if newContent == content {
			break
		}
		content = newContent
	}

	// Remove CSI sequences (ESC [ ... letter)  
	content = regexp.MustCompile(`\x1b\[[0-9;:!?<>]*[a-zA-Z@]`).ReplaceAllString(content, "")

	// Remove other escape sequences
	content = regexp.MustCompile(`\x1b[()][0-9A-Za-z~]`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`\x1b[#%\*\+\-\\\.\/:]`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`\x1b[NOc\|\}^G_@=]`).ReplaceAllString(content, "")

	// Remove orphaned ESC characters
	content = strings.ReplaceAll(content, "\x1b", "")

	// Remove backspace characters
	var result []rune
	for _, r := range content {
		if r != '\b' && r != 0x7f {
			result = append(result, r)
		}
	}

	// Normalize line endings
	content = strings.ReplaceAll(string(result), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	return content
}

// isPromptLine checks if line looks like a shell prompt
func isPromptLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}

	// Match common prompt patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^[~\/][^❯]*❯`),
		regexp.MustCompile(`^\[[^\]]+\][#\$]`),
		regexp.MustCompile(`^\$\s`),
		regexp.MustCompile(`^%\s`),
		regexp.MustCompile(`^>`),
	}

	for _, p := range patterns {
		if p.MatchString(line) {
			return true
		}
	}

	// Single char prompt (just ~ or $)
	if regexp.MustCompile(`^[~\$%#]\s*$`).MatchString(line) {
		return true
	}

	return false
}

// stripANSI removes basic ANSI codes (for log files)
func stripANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(s, "")
}

// cleanOutput final cleanup
func cleanOutput(s string) string {
	s = strings.TrimSpace(s)

	// Collapse multiple blank lines
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
