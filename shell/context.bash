# Context Shell Integration for Bash
# Add this to your .bashrc to enable command output logging

# Configuration
CONTEXT_LOG_DIR="${HOME}/.context/logs"
CONTEXT_LOG_ENABLED=1
CONTEXT_MAX_LOG_AGE_DAYS=30
CONTEXT_MAX_LOG_SIZE_MB=100

# Create log directory if it doesn't exist
mkdir -p "$CONTEXT_LOG_DIR"

# Function to sanitize command for filename
_context_sanitize_cmd() {
    echo "$1" | tr -cd '[:alnum:]._-' | cut -c1-50
}

# Function to clean old logs
_context_clean_logs() {
    find "$CONTEXT_LOG_DIR" -type f -name "*.log" -mtime +$CONTEXT_MAX_LOG_AGE_DAYS -delete 2>/dev/null
    
    # Also check total size and delete oldest if over limit
    local total_size=$(du -sm "$CONTEXT_LOG_DIR" 2>/dev/null | cut -f1)
    if [[ $total_size -gt $CONTEXT_MAX_LOG_SIZE_MB ]]; then
        # Delete oldest files until under limit
        while [[ $total_size -gt $CONTEXT_MAX_LOG_SIZE_MB ]]; do
            local oldest=$(find "$CONTEXT_LOG_DIR" -type f -name "*.log" -printf '%T+ %p\n' 2>/dev/null | sort | head -1 | cut -d' ' -f2)
            if [[ -n "$oldest" ]]; then
                rm -f "$oldest"
                total_size=$(du -sm "$CONTEXT_LOG_DIR" 2>/dev/null | cut -f1)
            else
                break
            fi
        done
    fi
}

# Debug trap - called before each command
_context_debug_trap() {
    local cmd="$BASH_COMMAND"
    
    # Skip context commands, empty commands, and non-interactive commands
    if [[ "$cmd" == context* ]] || [[ -z "$cmd" ]] || [[ "$cmd" =~ ^[[:space:]]*$ ]]; then
        CONTEXT_SKIP_LOGGING=1
        return 0
    fi
    
    CONTEXT_SKIP_LOGGING=0
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local sanitized_cmd=$(_context_sanitize_cmd "$cmd")
    
    CONTEXT_CURRENT_LOG="${CONTEXT_LOG_DIR}/${timestamp}_${sanitized_cmd}.log"
    CONTEXT_CURRENT_CMD="$cmd"
    CONTEXT_CMD_START_TIME=$(date +%s)
    CONTEXT_OUTPUT_FILE=$(mktemp)
    
    # Redirect output to temp file and terminal
    exec 3>&1 4>&2
    exec 1> >(tee "$CONTEXT_OUTPUT_FILE")
    exec 2> >(tee -a "$CONTEXT_OUTPUT_FILE" >&2)
}

# Called after command execution
_context_prompt_command() {
    local exit_code=$?
    
    # Restore stdout/stderr
    if [[ -n "$CONTEXT_OUTPUT_FILE" && -f "$CONTEXT_OUTPUT_FILE" ]]; then
        exec 1>&3 2>&4
    fi
    
    # Skip if we shouldn't log this command
    if [[ ${CONTEXT_SKIP_LOGGING:-0} -eq 1 ]] || [[ -z "$CONTEXT_CURRENT_LOG" ]]; then
        CONTEXT_SKIP_LOGGING=0
        CONTEXT_CURRENT_LOG=""
        CONTEXT_CURRENT_CMD=""
        CONTEXT_OUTPUT_FILE=""
        return 0
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - CONTEXT_CMD_START_TIME))
    
    # Write log entry
    cat > "$CONTEXT_CURRENT_LOG" <<EOF
=== COMMAND: ${CONTEXT_CURRENT_CMD}
=== START_TIME: $(date -d @$CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date -r $CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null || echo "$(date)")
=== END_TIME: $(date '+%Y-%m-%d %H:%M:%S')
=== DURATION: ${duration}s
=== EXIT_CODE: ${exit_code}
=== WORKING_DIR: $(pwd)
EOF
    
    # Add output if available
    if [[ -f "$CONTEXT_OUTPUT_FILE" && -s "$CONTEXT_OUTPUT_FILE" ]]; then
        echo "=== OUTPUT:" >> "$CONTEXT_CURRENT_LOG"
        cat "$CONTEXT_OUTPUT_FILE" >> "$CONTEXT_CURRENT_LOG"
        rm -f "$CONTEXT_OUTPUT_FILE"
    fi
    
    # Clean old logs periodically (1% chance)
    if [[ $((RANDOM % 100)) -eq 0 ]]; then
        _context_clean_logs
    fi
    
    # Reset
    CONTEXT_SKIP_LOGGING=0
    CONTEXT_CURRENT_LOG=""
    CONTEXT_CURRENT_CMD=""
    CONTEXT_OUTPUT_FILE=""
}

# Set up hooks
if [[ $CONTEXT_LOG_ENABLED -eq 1 ]]; then
    trap '_context_debug_trap' DEBUG
    PROMPT_COMMAND="_context_prompt_command${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
fi
