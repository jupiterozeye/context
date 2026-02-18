# Context Shell Integration for Zsh
# Simple command capture using preexec/precmd hooks

export CONTEXT_LOG_DIR="${HOME}/.context/logs"
export CONTEXT_LOG_ENABLED=${CONTEXT_LOG_ENABLED:-1}
export CONTEXT_MAX_SIZE_MB=${CONTEXT_MAX_SIZE_MB:-50}

# Skip if disabled
[[ ${CONTEXT_LOG_ENABLED} -ne 1 ]] && return 0

# Create directories
mkdir -p "$CONTEXT_LOG_DIR"

# Generate session ID if not set (for isolation)
if [[ -z "$CONTEXT_SESSION_ID" ]]; then
    export CONTEXT_SESSION_ID=$(date +%s%N | sha256sum | cut -c1-16)
fi

# Session-specific log directory
export CONTEXT_SESSION_DIR="${CONTEXT_LOG_DIR}/${CONTEXT_SESSION_ID}"
mkdir -p "$CONTEXT_SESSION_DIR"

# Preexec: runs before each command
_context_preexec() {
    local cmd="$1"
    
    # Skip empty commands, context commands, and internal commands
    [[ -z "$cmd" ]] && return
    [[ "$cmd" == context* ]] && return
    [[ "$cmd" == exit* ]] && return
    
    CONTEXT_CURRENT_CMD="$cmd"
    CONTEXT_CMD_START_TIME=$(date +%s)
    CONTEXT_CMD_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    CONTEXT_OUTPUT_FILE=$(mktemp)
    
    # Redirect output to temp file AND terminal using tee
    exec 3>&1 4>&2
    exec 1> >(tee "$CONTEXT_OUTPUT_FILE" >&3)
    exec 2> >(tee "$CONTEXT_OUTPUT_FILE" >&4)
}

# Precmd: runs after each command
_context_precmd() {
    local exit_code=$?
    
    # Restore stdout/stderr
    if [[ -n "$CONTEXT_OUTPUT_FILE" ]]; then
        exec 1>&3 2>&4 2>/dev/null || true
    fi
    
    # Skip if no command was captured
    [[ -z "$CONTEXT_CURRENT_CMD" ]] && return
    
    local log_file="${CONTEXT_SESSION_DIR}/${CONTEXT_CMD_TIMESTAMP}.log"
    local duration=$(($(date +%s) - CONTEXT_CMD_START_TIME))
    
    # Write log entry
    {
        echo "=== COMMAND: ${CONTEXT_CURRENT_CMD}"
        echo "=== START_TIME: $(date -r $CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date '+%Y-%m-%d %H:%M:%S')"
        echo "=== END_TIME: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "=== DURATION: ${duration}s"
        echo "=== EXIT_CODE: ${exit_code}"
        echo "=== WORKING_DIR: $(pwd)"
        if [[ -f "$CONTEXT_OUTPUT_FILE" ]]; then
            echo "=== OUTPUT:"
            cat "$CONTEXT_OUTPUT_FILE"
        fi
    } > "$log_file"
    
    # Cleanup
    rm -f "$CONTEXT_OUTPUT_FILE"
    unset CONTEXT_CURRENT_CMD CONTEXT_CMD_START_TIME CONTEXT_CMD_TIMESTAMP CONTEXT_OUTPUT_FILE
}

# Register hooks
autoload -Uz add-zsh-hook
add-zsh-hook preexec _context_preexec
add-zsh-hook precmd _context_precmd

# Show indicator if in recorded session
if [[ -n "$CONTEXT_RECORDING" ]]; then
    PS1="[rec] $PS1"
fi
