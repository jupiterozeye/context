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
	sessionID      string
	isRecording    bool
}

// NewReader creates a new log reader
func NewReader(opts Options) *Reader {
	homeDir, _ := os.UserHomeDir()
	
	// Check if we're in a recorded session
	sessionID := os.Getenv("CONTEXT_SESSION_ID")
	isRecording := os.Getenv("CONTEXT_RECORDING") == "1"
	
	var logDir, typescriptPath string
	if sessionID != "" {
		// Use session-specific directory
		logDir = filepath.Join(homeDir, ".context", "logs", sessionID)
		typescriptPath = filepath.Join(logDir, "typescript")
		isRecording = true
	} else {
		// Use main logs directory
		logDir = filepath.Join(homeDir, ".context", "logs")
		typescriptPath = filepath.Join(homeDir, ".context", "typescript")
	}
	
	return &Reader{
		opts:           opts,
		logDir:         logDir,
		typescriptPath: typescriptPath,
		sessionID:      sessionID,
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

	// Use typescript if it exists
	if _, err := os.Stat(r.typescriptPath); err == nil {
		entries, err := r.readFromTypescript(n)
		if err == nil && len(entries) > 0 {
			return entries, nil
		}
	}

	// Fall back to log files
	return r.readFromLogFiles(n)
}

// readFromLogFiles reads from individual log files
func (r *Reader) readFromLogFiles(n int) ([]LogEntry, error) {
	files, err := os.ReadDir(r.logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no log directory found")
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
		return nil, fmt.Errorf("no commands recorded yet. run some commands first")
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

	// Extract commands and their positions from OSC sequences
	commands := extractCommandsFromOSC(content)
	if len(commands) == 0 {
		return entries
	}

	// Build entries from commands
	for i, cmdInfo := range commands {
		cmd := cmdInfo.command
		if cmd == "" || cmd == "exit" || strings.HasPrefix(cmd, "exit ") {
			continue
		}
		// Skip context last to avoid recursion
		if cmd == "context last" || strings.HasPrefix(cmd, "context last ") {
			continue
		}

		// Extract output between this command and the next
		var nextPos int
		if i+1 < len(commands) {
			nextPos = commands[i+1].position
		} else {
			nextPos = len(content)
		}

		// Extract output from original content between positions
		section := content[cmdInfo.position:nextPos]
		output := extractCleanOutput(section)

		entries = append(entries, LogEntry{
			Command: cmd,
			Output:  output,
		})
	}

	return entries
}

// commandInfo holds a command and its position in the original content
type commandInfo struct {
	command  string
	position int
}

// extractCommandsFromOSC extracts commands from OSC window title sequences
func extractCommandsFromOSC(content string) []commandInfo {
	var commands []commandInfo
	seen := make(map[string]bool)

	// Match OSC 2 or 0 sequences: ESC ] 0/2 ; title BEL or ESC \ 
	oscPattern := regexp.MustCompile("\x1b][02];([^\x07\x1b]+)")

	// Find all matches with their positions
	matches := oscPattern.FindAllStringSubmatchIndex(content, -1)
	for _, m := range matches {
		if len(m) >= 4 {
			// m[0], m[1] = full match start/end
			// m[2], m[3] = group 1 (command) start/end
			cmd := strings.TrimSpace(content[m[2]:m[3]])
			
			// Filter out prompt-only titles
			// Filter out prompt-only and unwanted commands
			if isPromptOnly(cmd) {
				continue
			}
			if cmd == "context last" || strings.HasPrefix(cmd, "context last ") {
				continue
			}
			// Skip duplicates
			if seen[cmd] {
				continue
			}
			seen[cmd] = true
			commands = append(commands, commandInfo{
				command:  cmd,
				position: m[1], // End of full match (after BEL or ST)
			})
		}
	}

	return commands
}

// isPromptOnly checks if a title is just a path/prompt without a command
func isPromptOnly(title string) bool {
	if title == "" || title == "~" || title == "/" {
		return true
	}
	// Just a path
	if regexp.MustCompile(`^[~/][^ ]*$`).MatchString(title) {
		return true
	}
	return false
}

// extractCleanOutput extracts clean output from a section of typescript
func extractCleanOutput(section string) string {
	// Strip all escapes
	clean := stripAllEscapes(section)

	// Split into lines
	lines := strings.Split(clean, "\n")

	// Find where actual output starts (skip prompt lines)
	var startIdx int
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Skip prompt-like lines at the start
		if isPromptLine(trimmed) {
			startIdx = i + 1
			continue
		}
		// Found first non-prompt line
		break
	}
	lines = lines[startIdx:]

	// Remove trailing prompt/empty lines
	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "" || isPromptLine(last) || isNoiseLine(last) {
			lines = lines[:len(lines)-1]
		} else {
			break
		}
	}

	// Join remaining lines
	result := strings.Join(lines, "\n")

	// Collapse multiple blank lines
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(result)
}

// stripAllEscapes removes terminal escape sequences
func stripAllEscapes(content string) string {
	// OSC sequences
	content = regexp.MustCompile("\x1b][0-9;:A-Za-z]*[^\x07\x1b]*(?:\x07|\x1b\\\\)?").ReplaceAllString(content, "")

	// CSI sequences
	content = regexp.MustCompile("\x1b\\[[0-9;:!?<>]*[a-zA-Z@]").ReplaceAllString(content, "")

	// Other escape sequences
	content = regexp.MustCompile("\x1b[()][0-9A-Za-z~]").ReplaceAllString(content, "")
	content = regexp.MustCompile("\x1b[#%*+\\-\\\\.\\/:]").ReplaceAllString(content, "")
	content = regexp.MustCompile("\x1b[NOc\\|\\}^G_@=]").ReplaceAllString(content, "")

	// Remove orphaned ESC
	content = strings.ReplaceAll(content, "\x1b", "")

	// Remove backspaces
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

// isPromptLine checks if line is a shell prompt
func isPromptLine(line string) bool {
	if line == "" {
		return false
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`[~\/][^❯]*❯\s*$`),
		regexp.MustCompile(`\[[^\]]+\][#\$]\s*$`),
		regexp.MustCompile(`^\$\s*$`),
		regexp.MustCompile(`^%\s*$`),
	}
	for _, p := range patterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

// isNoiseLine checks for noise lines
func isNoiseLine(line string) bool {
	noise := []string{
		"Script done",
		"Script started",
		"Copied to clipboard",
		"Starting recorded",
		"Commands and output",
		"Type 'exit'",
	}
	for _, n := range noise {
		if strings.Contains(line, n) {
			return true
		}
	}
	return false
}

// stripANSI removes basic ANSI codes
func stripANSI(s string) string {
	return regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]").ReplaceAllString(s, "")
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
