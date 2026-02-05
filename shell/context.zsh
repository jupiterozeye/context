#!/bin/zsh
# Context shell integration for Zsh
# Source this file in your ~/.zshrc

# Configuration
CONTEXT_HISTORY_FILE="${HOME}/.local/share/context/history.jsonl"
CONTEXT_MAX_HISTORY=1000

# Create directory if needed
mkdir -p "$(dirname "$CONTEXT_HISTORY_FILE")"

# Initialize temp file for current command output
CONTEXT_TEMP_FILE=$(mktemp)

# Function to run before command execution
__context_preexec() {
    local cmd="$1"
    
    # Don't log context commands themselves
    if [[ "$cmd" == context* ]]; then
        return
    fi
    
    CONTEXT_LAST_COMMAND="$cmd"
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
            output=$(base64 < "$CONTEXT_TEMP_FILE" 2>/dev/null || true)
            > "$CONTEXT_TEMP_FILE"
        fi
        
        # Escape command for JSON
        local escaped_cmd=$(echo "$CONTEXT_LAST_COMMAND" | sed 's/\\/\\\\/g; s/"/\\"/g')
        local escaped_pwd=$(echo "$CONTEXT_PWD" | sed 's/\\/\\\\/g; s/"/\\"/g')
        
        # Create JSON entry
        printf '{"timestamp":%s,"command":"%s","output":"%s","exit_code":%s,"pwd":"%s"}\n' \
            "$timestamp" \
            "$escaped_cmd" \
            "$output" \
            "$exit_code" \
            "$escaped_pwd" >> "$CONTEXT_HISTORY_FILE"
        
        # Rotate history if needed
        local line_count=$(wc -l < "$CONTEXT_HISTORY_FILE" 2>/dev/null || echo 0)
        if [[ $line_count -gt $CONTEXT_MAX_HISTORY ]]; then
            tail -n "$CONTEXT_MAX_HISTORY" "$CONTEXT_HISTORY_FILE" > "${CONTEXT_HISTORY_FILE}.tmp"
            mv "${CONTEXT_HISTORY_FILE}.tmp" "$CONTEXT_HISTORY_FILE"
        fi
    fi
    
    CONTEXT_LAST_COMMAND=""
}

# Use zsh hooks
autoload -U add-zsh-hook
add-zsh-hook preexec __context_preexec
add-zsh-hook precmd __context_precmd