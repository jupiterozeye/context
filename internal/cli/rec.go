package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var recCmd = &cobra.Command{
	Use:   "rec",
	Short: "Start a recorded shell session",
	Long:  `Starts a new shell session where all commands and output are recorded.`,
	RunE:  runRec,
}

func init() {
	rootCmd.AddCommand(recCmd)
}

func runRec(cmd *cobra.Command, args []string) error {
	if os.Getenv("CONTEXT_RECORDING") == "1" {
		return fmt.Errorf("already in a recorded session")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find home directory: %w", err)
	}

	sessionDir := filepath.Join(homeDir, ".context", "current-session")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return fmt.Errorf("cannot create session directory: %w", err)
	}

	typescriptPath := filepath.Join(sessionDir, "typescript")
	os.Remove(typescriptPath)
	os.Remove(filepath.Join(sessionDir, "commands.log"))

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	shellBase := filepath.Base(shell)

	// Resolve current binary path so 'context' inside the session uses the same binary
	executable, _ := os.Executable()
	if executable != "" {
		executable, _ = filepath.EvalSymlinks(executable)
	}

	var shellCmd string
	env := append(os.Environ(),
		"CONTEXT_RECORDING=1",
		"CONTEXT_SESSION_DIR="+sessionDir,
		"CONTEXT_BIN="+executable,
	)

	switch shellBase {
	case "zsh":
		shellCmd, env, err = setupZsh(shell, homeDir, env)
	case "bash":
		shellCmd, err = setupBash(shell)
	case "fish":
		shellCmd, err = setupFish(shell)
	default:
		shellCmd = shell + " -i"
	}
	if err != nil {
		return err
	}

	fmt.Println("🎥 Starting recorded shell session...")
	fmt.Println("   Type 'exit' to stop recording")
	fmt.Println()

	scriptCmd := exec.Command("script", "-q", "-f", "-c", shellCmd, typescriptPath)
	scriptCmd.Stdin = os.Stdin
	scriptCmd.Stdout = os.Stdout
	scriptCmd.Stderr = os.Stderr
	scriptCmd.Env = env

	if err := scriptCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}

	return nil
}

// setupZsh creates a temporary ZDOTDIR with custom .zshenv/.zshrc that source
// the user's real RC files and add recording hooks.
func setupZsh(shell, homeDir string, env []string) (string, []string, error) {
	tempDir, err := os.MkdirTemp("", "context-zsh-*")
	if err != nil {
		return "", nil, fmt.Errorf("cannot create temp dir: %w", err)
	}
	// Note: tempDir is cleaned up when the shell exits since script blocks.
	// In abnormal termination, /tmp is cleaned by the OS.

	realZdotdir := os.Getenv("ZDOTDIR")
	if realZdotdir == "" {
		realZdotdir = homeDir
	}

	// .zshenv: source the real .zshenv, then force ZDOTDIR back to our temp
	// dir so zsh loads our .zshrc next.
	zshenv := fmt.Sprintf(`if [[ -f "%s/.zshenv" ]]; then
    source "%s/.zshenv"
fi
ZDOTDIR="%s"
`, realZdotdir, realZdotdir, tempDir)

	if err := os.WriteFile(filepath.Join(tempDir, ".zshenv"), []byte(zshenv), 0644); err != nil {
		return "", nil, err
	}

	// .zshrc: reset ZDOTDIR, source the real .zshrc, then add our hooks.
	// Hooks are added AFTER the real .zshrc so they aren't overwritten.
	zshrc := fmt.Sprintf(`ZDOTDIR="%s"
if [[ -f "%s/.zshrc" ]]; then
    source "%s/.zshrc"
fi

# Alias context to the binary that started the session
[[ -n "$CONTEXT_BIN" ]] && alias context="$CONTEXT_BIN"

# Context recording hooks — emit OSC markers into the terminal stream.
# These are invisible to the user but captured by script in the typescript.
__context_preexec() {
    local cmd="${1//$'\n'/\\n}"
    printf '\e]7337;C;%%s\a' "$cmd"
    echo "C:$cmd" >> "$CONTEXT_SESSION_DIR/commands.log"
}
__context_precmd() {
    local ec=$?
    printf '\e]7337;D;%%d\a' "$ec"
    echo "D:$ec" >> "$CONTEXT_SESSION_DIR/commands.log"
}
autoload -Uz add-zsh-hook
add-zsh-hook preexec __context_preexec
add-zsh-hook precmd __context_precmd
`, realZdotdir, realZdotdir, realZdotdir)

	if err := os.WriteFile(filepath.Join(tempDir, ".zshrc"), []byte(zshrc), 0644); err != nil {
		return "", nil, err
	}

	env = append(env,
		"ZDOTDIR="+tempDir,
		"CONTEXT_REAL_ZDOTDIR="+realZdotdir,
	)
	return shell + " -i", env, nil
}

// setupBash creates a temporary rcfile that sources ~/.bashrc and adds hooks.
func setupBash(shell string) (string, error) {
	f, err := os.CreateTemp("", "context-bash-*.bash")
	if err != nil {
		return "", fmt.Errorf("cannot create temp file: %w", err)
	}

	bashrc := `[[ -f "$HOME/.bashrc" ]] && source "$HOME/.bashrc"

# Alias context to the binary that started the session
[[ -n "$CONTEXT_BIN" ]] && alias context="$CONTEXT_BIN"

# Context recording hooks
__context_cmd_recorded=0
__context_preexec() {
    if [[ "$__context_cmd_recorded" == 0 ]]; then
        __context_cmd_recorded=1
        printf '\e]7337;C;%s\a' "$BASH_COMMAND"
        echo "C:$BASH_COMMAND" >> "$CONTEXT_SESSION_DIR/commands.log"
    fi
}
trap '__context_preexec' DEBUG

__context_precmd() {
    local ec=$?
    printf '\e]7337;D;%d\a' "$ec"
    echo "D:$ec" >> "$CONTEXT_SESSION_DIR/commands.log"
    __context_cmd_recorded=0
}
PROMPT_COMMAND="__context_precmd${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
`
	if _, err := f.WriteString(bashrc); err != nil {
		return "", err
	}
	f.Close()

	return fmt.Sprintf("%s --rcfile %s -i", shell, f.Name()), nil
}

// setupFish creates a temporary init file with recording hooks.
func setupFish(shell string) (string, error) {
	f, err := os.CreateTemp("", "context-fish-*.fish")
	if err != nil {
		return "", fmt.Errorf("cannot create temp file: %w", err)
	}

	fishrc := `# Alias context to the binary that started the session
if set -q CONTEXT_BIN
    alias context="$CONTEXT_BIN"
end

function __context_preexec --on-event fish_preexec
    printf '\e]7337;C;%s\a' "$argv[1]"
    echo "C:$argv[1]" >> "$CONTEXT_SESSION_DIR/commands.log"
end
function __context_postcmd --on-event fish_postexec
    printf '\e]7337;D;%d\a' $status
    echo "D:$status" >> "$CONTEXT_SESSION_DIR/commands.log"
end
`
	if _, err := f.WriteString(fishrc); err != nil {
		return "", err
	}
	f.Close()

	return fmt.Sprintf("%s --init-command 'source %s' -i", shell, f.Name()), nil
}
