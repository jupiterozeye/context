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

// OSC markers emitted by the shell hooks in context rec.
var (
	cmdMarkerRe = regexp.MustCompile("\x1b\\]7337;C;([^\x07]*)\x07")
	endMarkerRe = regexp.MustCompile("\x1b\\]7337;D;([0-9]+)\x07")
)

// NewReader creates a new log reader
func NewReader(opts Options) *Reader {
	homeDir, _ := os.UserHomeDir()
	isRecording := os.Getenv("CONTEXT_RECORDING") == "1"

	sessionDir := os.Getenv("CONTEXT_SESSION_DIR")
	if sessionDir == "" {
		sessionDir = filepath.Join(homeDir, ".context", "current-session")
	}

	// Prefer session dir typescript
	typescriptPath := filepath.Join(sessionDir, "typescript")
	if _, err := os.Stat(typescriptPath); err == nil {
		isRecording = true
	} else if _, err := os.Stat(filepath.Join(sessionDir, "commands.log")); err == nil {
		isRecording = true
	} else {
		// Fall back to typescript in cwd (legacy)
		if _, err := os.Stat("typescript"); err == nil {
			typescriptPath = "typescript"
			if isRecording {
				// Only use cwd typescript if env says we're recording
			} else if data, err := os.ReadFile("typescript"); err == nil {
				if strings.Contains(string(data), "CONTEXT_SESSION") {
					isRecording = true
				}
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

	var entries []LogEntry

	// Try typescript with OSC markers (has both commands and output)
	if content, err := os.ReadFile(r.typescriptPath); err == nil {
		entries = parseMarkers(string(content))

		// Fall back to prompt-regex parsing (legacy typescript files)
		if len(entries) == 0 {
			entries = parsePrompts(string(content))
		}
	}

	// Fall back to commands.log (has commands but not output)
	if len(entries) == 0 {
		logPath := filepath.Join(filepath.Dir(r.typescriptPath), "commands.log")
		entries = parseCommandsLog(logPath)
	}

	// Filter out context last/rec commands (but keep context dir, etc.)
	var filtered []LogEntry
	for _, e := range entries {
		if isContextMetaCommand(e.Command) {
			continue
		}
		filtered = append(filtered, e)
	}
	entries = filtered

	if len(entries) == 0 {
		return nil, fmt.Errorf("no commands recorded yet")
	}

	if n > len(entries) {
		n = len(entries)
	}
	return entries[len(entries)-n:], nil
}

// parseMarkers extracts commands by finding OSC 7337 markers in the raw typescript.
// The shell hooks emit C;command before execution and D;exitcode after.
func parseMarkers(raw string) []LogEntry {
	cmdMatches := cmdMarkerRe.FindAllStringSubmatchIndex(raw, -1)
	if len(cmdMatches) == 0 {
		return nil
	}

	endMatches := endMarkerRe.FindAllStringSubmatchIndex(raw, -1)

	var entries []LogEntry
	for _, cmdMatch := range cmdMatches {
		cmdText := raw[cmdMatch[2]:cmdMatch[3]]
		cmdText = strings.ReplaceAll(cmdText, "\\n", "\n")
		cmdEnd := cmdMatch[1] // end of the C marker

		// Find the next D marker after this C marker
		var exitCode int
		outputEnd := len(raw)
		for _, endMatch := range endMatches {
			if endMatch[0] > cmdEnd {
				outputEnd = endMatch[0]
				fmt.Sscanf(raw[endMatch[2]:endMatch[3]], "%d", &exitCode)
				break
			}
		}

		output := raw[cmdEnd:outputEnd]
		output = cleanOutput(output)

		entries = append(entries, LogEntry{
			Command:  cmdText,
			Output:   output,
			ExitCode: exitCode,
		})
	}

	return entries
}

// parseCommandsLog reads from the structured commands.log file written directly
// by shell hooks. This is a reliable fallback when the typescript hasn't been
// flushed yet — it provides command text and exit codes but not output.
func parseCommandsLog(logPath string) []LogEntry {
	data, err := os.ReadFile(logPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	var entries []LogEntry
	var currentCmd string

	for _, line := range lines {
		if strings.HasPrefix(line, "C:") {
			currentCmd = strings.TrimPrefix(line, "C:")
		} else if strings.HasPrefix(line, "D:") && currentCmd != "" {
			var exitCode int
			fmt.Sscanf(strings.TrimPrefix(line, "D:"), "%d", &exitCode)
			entries = append(entries, LogEntry{
				Command:  currentCmd,
				ExitCode: exitCode,
			})
			currentCmd = ""
		}
	}

	return entries
}

// parsePrompts is the legacy parser that tries to detect commands by matching
// common shell prompt patterns (❯, $, %) in the escape-stripped typescript.
func parsePrompts(raw string) []LogEntry {
	lines := strings.Split(raw, "\n")

	var entries []LogEntry
	var currentCmd string
	var outputLines []string
	inOutput := false

	cmdPattern := regexp.MustCompile(`^[\s]*[~\/]?[^❯]*❯\s+(.+)$|^\$\s+(.+)$|^%\s+(.+)$`)

	for _, line := range lines {
		cleanLine := stripEscapeSequences(line)
		trimmed := strings.TrimSpace(cleanLine)

		if trimmed == "" || strings.HasPrefix(trimmed, "Script ") {
			continue
		}

		if matches := cmdPattern.FindStringSubmatch(cleanLine); matches != nil {
			if currentCmd != "" {
				output := strings.TrimSpace(strings.Join(outputLines, "\n"))
				outLines := strings.Split(output, "\n")
				for len(outLines) > 0 && isPromptLine(outLines[len(outLines)-1]) {
					outLines = outLines[:len(outLines)-1]
				}

				entries = append(entries, LogEntry{
					Command: currentCmd,
					Output:  strings.TrimSpace(strings.Join(outLines, "\n")),
				})
			}

			for i := 1; i < len(matches); i++ {
				if matches[i] != "" {
					currentCmd = strings.TrimSpace(matches[i])
					break
				}
			}

			if isContextMetaCommand(currentCmd) {
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

	if currentCmd != "" && len(outputLines) > 0 {
		output := strings.TrimSpace(strings.Join(outputLines, "\n"))
		outLines := strings.Split(output, "\n")
		for len(outLines) > 0 && isPromptLine(outLines[len(outLines)-1]) {
			outLines = outLines[:len(outLines)-1]
		}

		entries = append(entries, LogEntry{
			Command: currentCmd,
			Output:  strings.TrimSpace(strings.Join(outLines, "\n")),
		})
	}

	return entries
}

func stripEscapeSequences(s string) string {
	// CSI sequences: ESC [ (params)(intermediates)(final byte)
	// Handles sequences like ESC[0m, ESC[?2004h, ESC[0 q (cursor shape with space)
	s = regexp.MustCompile("\x1b\\[[\x20-\x3f]*[\x40-\x7e]").ReplaceAllString(s, "")
	// OSC sequences: ESC ] ... (BEL or ST terminator)
	s = regexp.MustCompile("\x1b][^\x07\x1b]*(?:\x07|\x1b\\\\)").ReplaceAllString(s, "")
	// DCS/PM/APC sequences: ESC P/^/_ ... ST
	s = regexp.MustCompile("\x1b[P^_][^\x1b]*\x1b\\\\").ReplaceAllString(s, "")
	// Remove orphaned ESC (and any single char after it)
	s = regexp.MustCompile("\x1b.?").ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// cleanOutput strips escape sequences and removes trailing shell prompt artifacts
// like zsh's partial line indicator (PROMPT_SP: % or # on an otherwise empty line).
func cleanOutput(s string) string {
	s = stripEscapeSequences(s)
	s = strings.TrimSpace(s)

	lines := strings.Split(s, "\n")
	for len(lines) > 0 {
		last := strings.TrimSpace(lines[len(lines)-1])
		if last == "%" || last == "#" || last == "" {
			lines = lines[:len(lines)-1]
		} else {
			break
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// isContextMetaCommand returns true for context subcommands that should be
// filtered from output (last, rec) but NOT for useful commands like dir.
func isContextMetaCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		return false
	}
	base := filepath.Base(parts[0])
	if base != "context" {
		return false
	}
	switch parts[1] {
	case "last", "rec", "flush":
		return true
	}
	return false
}

func isPromptLine(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
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
