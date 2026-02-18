# Context Shell Integration for Bash
# Logs commands for context last (use 'context rec' for full output capture)

export CONTEXT_LOG_DIR="${HOME}/.context/logs"
export CONTEXT_LOG_ENABLED=${CONTEXT_LOG_ENABLED:-1}
export CONTEXT_MAX_SIZE_MB=${CONTEXT_MAX_SIZE_MB:-50}

# Skip if disabled or non-interactive
[[ ${CONTEXT_LOG_ENABLED} -ne 1 ]] && return 0
[[ $- != *i* ]] && return 0

# Create directories
mkdir -p "$CONTEXT_LOG_DIR"

# Debug trap: runs before each command
_context_debug_trap() {
    local cmd="$BASH_COMMAND"
    
    # Skip context commands, empty commands, and our own internals
    [[ "$cmd" == context* ]] && return
    [[ -z "$cmd" ]] && return
    [[ "$cmd" == *_context_* ]] && return
    
    CONTEXT_CURRENT_CMD="$cmd"
    CONTEXT_CMD_START_TIME=$(date +%s)
    CONTEXT_CMD_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
}

# Prompt command: runs after each command
_context_prompt_command() {
    local exit_code=$?
    
    # Skip if no command was captured
    [[ -z "$CONTEXT_CURRENT_CMD" ]] && return
    [[ -z "$CONTEXT_CMD_TIMESTAMP" ]] && return
    
    local end_time=$(date +%s)
    local duration=$((end_time - CONTEXT_CMD_START_TIME))
    local sanitized_cmd=$(echo "$CONTEXT_CURRENT_CMD" | tr -cd '[:alnum:]._-' | cut -c1-50)
    local log_file="${CONTEXT_LOG_DIR}/${CONTEXT_CMD_TIMESTAMP}_${sanitized_cmd}.log"
    
    # Write log entry (metadata only)
    {
        echo "=== COMMAND: ${CONTEXT_CURRENT_CMD}"
        echo "=== START_TIME: $(date -d @$CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r $CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date '+%Y-%m-%d %H:%M:%S')"
        echo "=== END_TIME: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "=== DURATION: ${duration}s"
        echo "=== EXIT_CODE: ${exit_code}"
        echo "=== WORKING_DIR: $(pwd)"
        echo "=== OUTPUT:"
        echo "[Use 'context rec' to capture command output]"
    } > "$log_file"
    
    # Reset
    CONTEXT_CURRENT_CMD=""
    CONTEXT_CMD_TIMESTAMP=""
    
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
        while [[ $total_size -gt $CONTEXT_MAX_SIZE_MB ]]; do
            local oldest=$(find "$CONTEXT_LOG_DIR" -type f -name "*.log" -printf '%T+ %p\n' 2>/dev/null | sort | head -1 | cut -d' ' -f2)
            [[ -z "$oldest" ]] && break
            rm -f "$oldest"
            total_size=$(du -sm "$CONTEXT_LOG_DIR" 2>/dev/null | cut -f1)
        done
    fi
}

# Set up hooks
trap '_context_debug_trap' DEBUG
if [[ -n "$PROMPT_COMMAND" ]]; then
    PROMPT_COMMAND="_context_prompt_command; ${PROMPT_COMMAND}"
else
    PROMPT_COMMAND="_context_prompt_command"
fi
