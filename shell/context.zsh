# Context Shell Integration for Zsh
# Captures command output using preexec/precmd hooks

export CONTEXT_LOG_DIR="${HOME}/.context/logs"
export CONTEXT_LOG_ENABLED=${CONTEXT_LOG_ENABLED:-1}
export CONTEXT_MAX_SIZE_MB=${CONTEXT_MAX_SIZE_MB:-50}

# Skip if disabled
[[ ${CONTEXT_LOG_ENABLED} -ne 1 ]] && return 0

# Create directories
mkdir -p "$CONTEXT_LOG_DIR"

# Preexec: runs before each command
_context_preexec() {
    local cmd="$1"
    
    # Skip context commands and empty commands
    [[ "$cmd" == context* ]] && return
    [[ -z "$cmd" ]] && return
    
    CONTEXT_CURRENT_CMD="$cmd"
    CONTEXT_CMD_START_TIME=$(date +%s)
    CONTEXT_CMD_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    CONTEXT_OUTPUT_FILE=$(mktemp)
    
    # Redirect output to temp file and terminal
    exec 3>&1 4>&2
    exec 1> >(tee "$CONTEXT_OUTPUT_FILE")
    exec 2> >(tee -a "$CONTEXT_OUTPUT_FILE" >&2)
}

# Precmd: runs before each prompt (after command completes)
_context_precmd() {
    local exit_code=$?
    
    # Restore stdout/stderr if we were capturing
    if [[ -n "$CONTEXT_OUTPUT_FILE" && -f "$CONTEXT_OUTPUT_FILE" ]]; then
        exec 1>&3 2>&4
    fi
    
    # Skip if no command was captured
    [[ -z "$CONTEXT_CURRENT_CMD" ]] && return
    [[ -z "$CONTEXT_CMD_TIMESTAMP" ]] && return
    
    local end_time=$(date +%s)
    local duration=$((end_time - CONTEXT_CMD_START_TIME))
    local sanitized_cmd=$(echo "$CONTEXT_CURRENT_CMD" | tr -cd '[:alnum:]._-' | cut -c1-50)
    local log_file="${CONTEXT_LOG_DIR}/${CONTEXT_CMD_TIMESTAMP}_${sanitized_cmd}.log"
    
    # Write log entry
    {
        echo "=== COMMAND: ${CONTEXT_CURRENT_CMD}"
        echo "=== START_TIME: $(date -r $CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date '+%Y-%m-%d %H:%M:%S')"
        echo "=== END_TIME: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "=== DURATION: ${duration}s"
        echo "=== EXIT_CODE: ${exit_code}"
        echo "=== WORKING_DIR: $(pwd)"
        if [[ -f "$CONTEXT_OUTPUT_FILE" && -s "$CONTEXT_OUTPUT_FILE" ]]; then
            echo "=== OUTPUT:"
            cat "$CONTEXT_OUTPUT_FILE"
        fi
    } > "$log_file"
    
    # Cleanup
    rm -f "$CONTEXT_OUTPUT_FILE"
    CONTEXT_CURRENT_CMD=""
    CONTEXT_CMD_TIMESTAMP=""
    CONTEXT_OUTPUT_FILE=""
    
    # Clean old logs occasionally (1% chance)
    if [[ $((RANDOM % 100)) -eq 0 ]]; then
        _context_clean_logs
    fi
}

# Clean old log files
_context_clean_logs() {
    # Delete logs older than 30 days
    find "$CONTEXT_LOG_DIR" -type f -name "*.log" -mtime +30 -delete 2>/dev/null
    
    # Check total size
    local total_size=$(du -sm "$CONTEXT_LOG_DIR" 2>/dev/null | cut -f1)
    if [[ $total_size -gt $CONTEXT_MAX_SIZE_MB ]]; then
        # Delete oldest files until under limit
        while [[ $total_size -gt $CONTEXT_MAX_SIZE_MB ]]; do
            local oldest=$(find "$CONTEXT_LOG_DIR" -type f -name "*.log" -printf '%T+ %p\n' 2>/dev/null | sort | head -1 | cut -d' ' -f2)
            [[ -z "$oldest" ]] && break
            rm -f "$oldest"
            total_size=$(du -sm "$CONTEXT_LOG_DIR" 2>/dev/null | cut -f1)
        done
    fi
}

# Register hooks
autoload -Uz add-zsh-hook
add-zsh-hook preexec _context_preexec
add-zsh-hook precmd _context_precmd
