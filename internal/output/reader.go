package output

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	Format string
}

// Reader handles reading and parsing log files
type Reader struct {
	opts           Options
	typescriptPath string
	isRecording    bool
}

// NewReader creates a new log reader
func NewReader(opts Options) *Reader {
	homeDir, _ := os.UserHomeDir()
	isRecording := os.Getenv("CONTEXT_RECORDING") == "1"
	
	// Use typescript from current directory if recording
	typescriptPath := filepath.Join(homeDir, ".context", "typescript")
	if _, err := os.Stat("typescript"); err == nil {
		// Check if this is our recording
		if data, err := os.ReadFile("typescript"); err == nil {
			if strings.Contains(string(data), "CONTEXT_SESSION") || isRecording {
				typescriptPath = "typescript"
				isRecording = true
			}
		}
	}
	
	return &Reader{
		opts:           opts,
		typescriptPath: typescriptPath,
		isRecording:    isRecording,
	}
}

// IsRecording returns true if in a recorded session
func (r *Reader) IsRecording() bool {
	return r.isRecording
}

// Read retrieves the last n log entries
func (r *Reader) Read(n int) ([]LogEntry, error) {
	if !r.isRecording {
		return nil, fmt.Errorf("not in a recorded session. run 'context rec' first")
	}

	// Try to read from typescript
	if _, err := os.Stat(r.typescriptPath); err == nil {
		entries, err := r.readFromTypescript(n)
		if err == nil && len(entries) > 0 {
			return entries, nil
		}
	}

	return nil, fmt.Errorf("no commands recorded yet")
}

// readFromTypescript reads commands from script typescript
func (r *Reader) readFromTypescript(n int) ([]LogEntry, error) {
	content, err := os.ReadFile(r.typescriptPath)
	if err != nil {
		return nil, err
	}

	// Simple parsing: look for command lines (lines starting with prompt patterns)
	lines := strings.Split(string(content), "\n")
	
	var entries []LogEntry
	var currentCmd string
	var outputLines []string
	inOutput := false
	
	// Simple prompt patterns
	cmdPattern := regexp.MustCompile(`^[\s]*[~\/]?[^❯]*❯\s+(.+)$|^\$\s+(.+)$|^%\s+(.+)$`)
	
	for _, line := range lines {
		// Clean escape sequences
		cleanLine := stripEscapeSequences(line)
		trimmed := strings.TrimSpace(cleanLine)
		
		// Skip noise
		if trimmed == "" || strings.HasPrefix(trimmed, "Script ") {
			continue
		}
		
		// Check for command
		if matches := cmdPattern.FindStringSubmatch(cleanLine); matches != nil {
			// Save previous command
			if currentCmd != "" && len(outputLines) > 0 {
				output := strings.TrimSpace(strings.Join(outputLines, "\n"))
				// Filter out the next prompt from output
				lines := strings.Split(output, "\n")
				for len(lines) > 0 && isPromptLine(lines[len(lines)-1]) {
					lines = lines[:len(lines)-1]
				}
				
				entries = append(entries, LogEntry{
					Command: currentCmd,
					Output:  strings.TrimSpace(strings.Join(lines, "\n")),
				})
			}
			
			// Extract command
			for i := 1; i < len(matches); i++ {
				if matches[i] != "" {
					currentCmd = strings.TrimSpace(matches[i])
					break
				}
			}
			
			// Skip context commands
			if strings.HasPrefix(currentCmd, "context ") && currentCmd != "context rec" {
				currentCmd = ""
				continue
			}
			
			outputLines = nil
			inOutput = true
			continue
		}
		
		if inOutput && currentCmd != "" {
			outputLines = append(outputLines, cleanLine)
		}
	}
	
	// Save last command
	if currentCmd != "" && len(outputLines) > 0 {
		output := strings.TrimSpace(strings.Join(outputLines, "\n"))
		lines := strings.Split(output, "\n")
		for len(lines) > 0 && isPromptLine(lines[len(lines)-1]) {
			lines = lines[:len(lines)-1]
		}
		
		entries = append(entries, LogEntry{
			Command: currentCmd,
			Output:  strings.TrimSpace(strings.Join(lines, "\n")),
		})
	}
	
	if len(entries) == 0 {
		return nil, fmt.Errorf("no commands found")
	}
	
	// Return last n
	if n > len(entries) {
		n = len(entries)
	}
	return entries[len(entries)-n:], nil
}

func stripEscapeSequences(s string) string {
	// Remove ANSI escape codes
	// CSI sequences: ESC [ ... letter
	s = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]").ReplaceAllString(s, "")
	// OSC sequences with BEL terminator: ESC ] ... BEL
	s = regexp.MustCompile("\x1b][^\x07]*\x07").ReplaceAllString(s, "")
	// OSC sequences with ST terminator: ESC ] ... ESC \
	// Need 4 backslashes in Go string to match literal \ in regex
	s = regexp.MustCompile("\x1b][^\x07\x1b]*(?:\x07|\x1b\\\\)").ReplaceAllString(s, "")
	// Remove orphaned ESC
	s = strings.ReplaceAll(s, "\x1b", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func isPromptLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	// Common prompt endings
	if regexp.MustCompile(`[~\/][^❯]*❯\s*$`).MatchString(line) {
		return true
	}
	if regexp.MustCompile(`^\$\s*$`).MatchString(line) {
		return true
	}
	if regexp.MustCompile(`^%\s*$`).MatchString(line) {
		return true
	}
	return false
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
