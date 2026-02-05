#!/bin/bash
# Context shell integration for Bash
# Source this file in your ~/.bashrc

# Configuration
CONTEXT_HISTORY_FILE="${HOME}/.local/share/context/history.jsonl"
CONTEXT_MAX_HISTORY=1000

# Create directory if needed
mkdir -p "$(dirname "$CONTEXT_HISTORY_FILE")"

# Initialize temp file for current command output
CONTEXT_TEMP_FILE=$(mktemp)

# Function to run before command execution
__context_preexec() {
    CONTEXT_LAST_COMMAND="$1"
    CONTEXT_START_TIME=$(date +%s)
    CONTEXT_PWD="$PWD"
    
    # Clear temp file
    > "$CONTEXT_TEMP_FILE"
}

# Function to run after command completion
__context_precmd() {
    local exit_code=$?
    
    # Only log if we have a command
    if [[ -n "$CONTEXT_LAST_COMMAND" ]]; then
        local timestamp=$(date +%s)
        local output=""
        
        # Read output from temp file
        if [[ -f "$CONTEXT_TEMP_FILE" ]]; then
            output=$(cat "$CONTEXT_TEMP_FILE" | base64 -w 0 2>/dev/null || cat "$CONTEXT_TEMP_FILE" | base64)
            > "$CONTEXT_TEMP_FILE"
        fi
        
        # Create JSON entry
        local json_entry=$(printf '{"timestamp":%s,"command":"%s","output":"%s","exit_code":%s,"pwd":"%s"}' \
            "$timestamp" \
            "$(echo "$CONTEXT_LAST_COMMAND" | sed 's/"/\\"/g')" \
            "$output" \
            "$exit_code" \
            "$CONTEXT_PWD")
        
        # Append to history file
        echo "$json_entry" >> "$CONTEXT_HISTORY_FILE"
        
        # Rotate history if needed
        local line_count=$(wc -l < "$CONTEXT_HISTORY_FILE" 2>/dev/null || echo 0)
        if [[ $line_count -gt $CONTEXT_MAX_HISTORY ]]; then
            tail -n "$CONTEXT_MAX_HISTORY" "$CONTEXT_HISTORY_FILE" > "${CONTEXT_HISTORY_FILE}.tmp"
            mv "${CONTEXT_HISTORY_FILE}.tmp" "$CONTEXT_HISTORY_FILE"
        fi
    fi
    
    CONTEXT_LAST_COMMAND=""
}

# Hook into bash using DEBUG trap
__context_debug_trap() {
    local cmd="$BASH_COMMAND"
    
    # Don't log context commands themselves
    if [[ "$cmd" == context* ]]; then
        return
    fi
    
    __context_preexec "$cmd"
}

trap '__context_debug_trap' DEBUG

# Hook into PROMPT_COMMAND
if [[ -z "$PROMPT_COMMAND" ]]; then
    PROMPT_COMMAND='__context_precmd'
else
    PROMPT_COMMAND="__context_precmd; $PROMPT_COMMAND"
fi