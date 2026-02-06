# Context Shell Integration for Zsh
# Add this to your .zshrc to enable command output logging

# Configuration
CONTEXT_LOG_DIR="${HOME}/.context/logs"
CONTEXT_LOG_ENABLED=1
CONTEXT_MAX_LOG_AGE_DAYS=30

# Create log directory if it doesn't exist
mkdir -p "$CONTEXT_LOG_DIR"

# Function to sanitize command for filename
_context_sanitize_cmd() {
    echo "$1" | tr -cd '[:alnum:]._-' | cut -c1-50
}

# Function to clean old logs
_context_clean_logs() {
    find "$CONTEXT_LOG_DIR" -type f -name "*.log" -mtime +$CONTEXT_MAX_LOG_AGE_DAYS -delete 2>/dev/null
}

# Function called before command execution
_context_preexec() {
    local cmd="$1"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local sanitized_cmd=$(_context_sanitize_cmd "$cmd")
    
    # Skip context commands and empty commands
    if [[ "$cmd" == context* ]] || [[ -z "$cmd" ]] || [[ "$cmd" =~ ^[[:space:]]*$ ]]; then
        CONTEXT_CURRENT_LOG=""
        return
    fi
    
    # Generate unique log filename
    CONTEXT_CURRENT_LOG="${CONTEXT_LOG_DIR}/${timestamp}_${sanitized_cmd}.log"
    CONTEXT_CURRENT_CMD="$cmd"
    CONTEXT_CMD_START_TIME=$(date +%s)
    
    # Start logging output using script
    # We use a temp file approach to avoid interfering with terminal
    CONTEXT_OUTPUT_FILE=$(mktemp)
}

# Function called before prompt (after command execution)
_context_precmd() {
    local exit_code=$?
    
    # If we were logging this command
    if [[ -n "$CONTEXT_CURRENT_LOG" && -n "$CONTEXT_CURRENT_CMD" ]]; then
        local end_time=$(date +%s)
        local duration=$((end_time - CONTEXT_CMD_START_TIME))
        
        # Write log entry
        cat > "$CONTEXT_CURRENT_LOG" <<EOF
=== COMMAND: ${CONTEXT_CURRENT_CMD}
=== START_TIME: $(date -r $CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S')
=== END_TIME: $(date '+%Y-%m-%d %H:%M:%S')
=== DURATION: ${duration}s
=== EXIT_CODE: ${exit_code}
=== WORKING_DIR: $(pwd)
EOF
        
        # If we have output to add
        if [[ -f "$CONTEXT_OUTPUT_FILE" && -s "$CONTEXT_OUTPUT_FILE" ]]; then
            echo "=== OUTPUT:" >> "$CONTEXT_CURRENT_LOG"
            cat "$CONTEXT_OUTPUT_FILE" >> "$CONTEXT_CURRENT_LOG"
            rm -f "$CONTEXT_OUTPUT_FILE"
        fi
        
        # Clean old logs periodically (1% chance to avoid slowdown)
        if [[ $((RANDOM % 100)) -eq 0 ]]; then
            _context_clean_logs
        fi
    fi
    
    # Reset
    CONTEXT_CURRENT_LOG=""
    CONTEXT_CURRENT_CMD=""
    CONTEXT_OUTPUT_FILE=""
}

# Alternative approach: log via PROMPT_COMMAND for simple output capture
# This captures stderr and stdout by redirecting to a file
_context_start_logging() {
    if [[ -n "$CONTEXT_CURRENT_LOG" ]]; then
        exec 1> >(tee "$CONTEXT_OUTPUT_FILE")
        exec 2> >(tee -a "$CONTEXT_OUTPUT_FILE" >&2)
    fi
}

_context_stop_logging() {
    exec 1>&3 2>&4
}

# Set up hooks
if [[ $CONTEXT_LOG_ENABLED -eq 1 ]]; then
    add-zsh-hook preexec _context_preexec
    add-zsh-hook precmd _context_precmd
fi
