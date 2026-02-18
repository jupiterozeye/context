# Context Shell Integration for Bash
# Simple command capture using debug trap and PROMPT_COMMAND

export CONTEXT_LOG_DIR="${HOME}/.context/logs"
export CONTEXT_LOG_ENABLED=${CONTEXT_LOG_ENABLED:-1}
export CONTEXT_MAX_SIZE_MB=${CONTEXT_MAX_SIZE_MB:-50}

# Skip if disabled or non-interactive
[[ ${CONTEXT_LOG_ENABLED} -ne 1 ]] && return 0
[[ $- != *i* ]] && return 0

# Create directories
mkdir -p "$CONTEXT_LOG_DIR"

# Generate session ID if not set
if [[ -z "$CONTEXT_SESSION_ID" ]]; then
    export CONTEXT_SESSION_ID=$(date +%s%N | sha256sum | cut -c1-16)
fi

# Session-specific log directory
export CONTEXT_SESSION_DIR="${CONTEXT_LOG_DIR}/${CONTEXT_SESSION_ID}"
mkdir -p "$CONTEXT_SESSION_DIR"

# Debug trap: runs before each command
_context_debug_trap() {
    local cmd="$BASH_COMMAND"
    
    # Skip empty commands, context commands, and internal commands
    [[ -z "$cmd" ]] && return
    [[ "$cmd" == context* ]] && return
    [[ "$cmd" == exit* ]] && return
    [[ "$cmd" == *_context_* ]] && return
    
    CONTEXT_CURRENT_CMD="$cmd"
    CONTEXT_CMD_START_TIME=$(date +%s)
    CONTEXT_CMD_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    CONTEXT_OUTPUT_FILE=$(mktemp)
    
    # Redirect output to temp file AND terminal
    exec 5>&1 6>&2
    exec 1> >(tee "$CONTEXT_OUTPUT_FILE" >&5)
    exec 2> >(tee "$CONTEXT_OUTPUT_FILE" >&6)
}

# Prompt command: runs after each command
_context_prompt_command() {
    local exit_code=$?
    
    # Restore stdout/stderr
    if [[ -n "$CONTEXT_OUTPUT_FILE" ]]; then
        exec 1>&5 2>&6 2>/dev/null || true
    fi
    
    # Skip if no command was captured
    [[ -z "$CONTEXT_CURRENT_CMD" ]] && return
    
    local log_file="${CONTEXT_SESSION_DIR}/${CONTEXT_CMD_TIMESTAMP}.log"
    local duration=$(($(date +%s) - CONTEXT_CMD_START_TIME))
    
    # Write log entry
    {
        echo "=== COMMAND: ${CONTEXT_CURRENT_CMD}"
        echo "=== START_TIME: $(date -d @$CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date '+%Y-%m-%d %H:%M:%S')"
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

# Set up hooks
trap '_context_debug_trap' DEBUG
if [[ -n "$PROMPT_COMMAND" ]]; then
    PROMPT_COMMAND="_context_prompt_command; ${PROMPT_COMMAND}"
else
    PROMPT_COMMAND="_context_prompt_command"
fi

# Show indicator if in recorded session
if [[ -n "$CONTEXT_RECORDING" ]]; then
    PS1="[rec] $PS1"
fi
