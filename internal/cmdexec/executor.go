package cmdexec

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// Result contains the execution result
type Result struct {
	Command    string
	Args       []string
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	ExitCode   int
	WorkingDir string
	Output     string
}

// Executor runs commands in a PTY and captures output
type Executor struct {
	LogDir string
}

// NewExecutor creates a new executor
func NewExecutor(logDir string) *Executor {
	return &Executor{LogDir: logDir}
}

// Run executes a command in a PTY and captures all output
func (e *Executor) Run(cmdStr string, args ...string) (*Result, error) {
	result := &Result{
		Command:    cmdStr,
		Args:       args,
		StartTime:  time.Now(),
		WorkingDir: mustGetwd(),
	}

	// Create log directory if needed
	if err := os.MkdirAll(e.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log dir: %w", err)
	}

	// Build the full command
	fullArgs := append([]string{cmdStr}, args...)
	cmd := exec.Command("sh", "-c", strings.Join(fullArgs, " "))
	cmd.Dir = result.WorkingDir

	// Start PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}
	defer ptmx.Close()

	// Set terminal size (reasonable default)
	pty.Setsize(ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})

	// Capture output
	var outputBuilder strings.Builder
	done := make(chan error, 1)

	go func() {
		reader := bufio.NewReader(ptmx)
		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				outputBuilder.Write(buf[:n])
				// Also write to terminal so user sees it
				os.Stdout.Write(buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					done <- err
				} else {
					done <- nil
				}
				return
			}
		}
	}()

	// Wait for command to finish
	cmdErr := cmd.Wait()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Get exit code
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
			}
		} else {
			result.ExitCode = 1
		}
	}

	// Small delay to let output channel finish
	time.Sleep(50 * time.Millisecond)

	// Get captured output
	result.Output = outputBuilder.String()

	// Write log file
	if err := e.writeLog(result); err != nil {
		// Log error but don't fail the command
		fmt.Fprintf(os.Stderr, "context: failed to write log: %v\n", err)
	}

	return result, cmdErr
}

// RunRaw executes without echoing to terminal (for shell integration)
func (e *Executor) RunRaw(cmdStr string, args ...string) (*Result, error) {
	result := &Result{
		Command:    cmdStr,
		Args:       args,
		StartTime:  time.Now(),
		WorkingDir: mustGetwd(),
	}

	// Create log directory if needed
	if err := os.MkdirAll(e.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log dir: %w", err)
	}

	// Build the full command
	fullArgs := append([]string{cmdStr}, args...)
	cmd := exec.Command("sh", "-c", strings.Join(fullArgs, " "))
	cmd.Dir = result.WorkingDir

	// Start PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty: %w", err)
	}
	defer ptmx.Close()

	// Set terminal size
	pty.Setsize(ptmx, &pty.Winsize{
		Rows: 24,
		Cols: 80,
	})

	// Capture output (don't echo to stdout - shell handles that)
	output, err := io.ReadAll(ptmx)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read output: %w", err)
	}

	// Wait for command
	cmdErr := cmd.Wait()
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Get exit code
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
			}
		} else {
			result.ExitCode = 1
		}
	}

	result.Output = string(output)

	// Write log file
	if err := e.writeLog(result); err != nil {
		fmt.Fprintf(os.Stderr, "context: failed to write log: %v\n", err)
	}

	return result, cmdErr
}

func (e *Executor) writeLog(result *Result) error {
	timestamp := result.StartTime.Format("20060102_150405")
	sanitizedCmd := sanitizeFilename(result.Command)
	filename := filepath.Join(e.LogDir, fmt.Sprintf("%s_%s.log", timestamp, sanitizedCmd))

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "=== COMMAND: %s\n", result.Command)
	fmt.Fprintf(file, "=== START_TIME: %s\n", result.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "=== END_TIME: %s\n", result.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "=== DURATION: %ds\n", int(result.Duration.Seconds()))
	fmt.Fprintf(file, "=== EXIT_CODE: %d\n", result.ExitCode)
	fmt.Fprintf(file, "=== WORKING_DIR: %s\n", result.WorkingDir)

	// Write output
	if result.Output != "" {
		fmt.Fprintf(file, "=== OUTPUT:\n%s\n", result.Output)
	}

	return nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func sanitizeFilename(s string) string {
	// Remove or replace problematic characters
	s = strings.TrimSpace(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			result.WriteRune(r)
		}
	}
	s = result.String()
	if len(s) > 50 {
		s = s[:50]
	}
	if s == "" {
		s = "cmd"
	}
	return s
}
