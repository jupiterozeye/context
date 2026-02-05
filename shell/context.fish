#!/usr/bin/env fish
# Context shell integration for Fish
# Source this file in your ~/.config/fish/config.fish

set -gx CONTEXT_HISTORY_FILE "$HOME/.local/share/context/history.jsonl"
set -gx CONTEXT_MAX_HISTORY 1000
set -gx CONTEXT_TEMP_FILE (mktemp)

# Create directory if needed
mkdir -p (dirname $CONTEXT_HISTORY_FILE)

# Function to run before command execution
function __context_preexec --on-event fish_preexec
    set -g CONTEXT_LAST_COMMAND $argv[1]
    set -g CONTEXT_START_TIME (date +%s)
    set -g CONTEXT_PWD $PWD
    
    # Clear temp file
    echo -n "" > $CONTEXT_TEMP_FILE
end

# Function to run after command completion
function __context_precmd --on-event fish_prompt
    set -l exit_code $status
    
    # Only log if we have a command
    if test -n "$CONTEXT_LAST_COMMAND"
        set -l timestamp (date +%s)
        set -l output ""
        
        # Read output from temp file
        if test -f $CONTEXT_TEMP_FILE
            set output (base64 < $CONTEXT_TEMP_FILE 2>/dev/null)
            echo -n "" > $CONTEXT_TEMP_FILE
        end
        
        # Escape command for JSON
        set -l escaped_cmd (echo $CONTEXT_LAST_COMMAND | string replace -a '\\' '\\\\' | string replace -a '"' '\\"')
        set -l escaped_pwd (echo $CONTEXT_PWD | string replace -a '\\' '\\\\' | string replace -a '"' '\\"')
        
        # Create JSON entry
        printf '{"timestamp":%s,"command":"%s","output":"%s","exit_code":%s,"pwd":"%s"}\n' \
            $timestamp \
            $escaped_cmd \
            $output \
            $exit_code \
            $escaped_pwd >> $CONTEXT_HISTORY_FILE
        
        # Rotate history if needed
        set -l line_count (wc -l < $CONTEXT_HISTORY_FILE 2>/dev/null)
        if test $line_count -gt $CONTEXT_MAX_HISTORY
            tail -n $CONTEXT_MAX_HISTORY $CONTEXT_HISTORY_FILE > $CONTEXT_HISTORY_FILE.tmp
            mv $CONTEXT_HISTORY_FILE.tmp $CONTEXT_HISTORY_FILE
        end
    end
    
    set -e CONTEXT_LAST_COMMAND
end