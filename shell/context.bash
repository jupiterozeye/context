# Context Shell Integration for Bash
# Captures command output using named pipes for synchronous capture

export CONTEXT_LOG_DIR="${HOME}/.context/logs"
export CONTEXT_LOG_ENABLED=${CONTEXT_LOG_ENABLED:-1}

# Skip if disabled or non-interactive
[[ ${CONTEXT_LOG_ENABLED} -ne 1 ]] && return 0
[[ $- != *i* ]] && return 0

# Create directories
mkdir -p "$CONTEXT_LOG_DIR"

# Generate session ID
if [[ -z "$CONTEXT_SESSION_ID" ]]; then
    export CONTEXT_SESSION_ID=$(date +%s%N | sha256sum | cut -c1-16)
fi
export CONTEXT_SESSION_DIR="${CONTEXT_LOG_DIR}/${CONTEXT_SESSION_ID}"
mkdir -p "$CONTEXT_SESSION_DIR"

# Setup FIFO for output capture
_context_setup_fifo() {
    export CONTEXT_FIFO="${CONTEXT_SESSION_DIR}/fifo.$$"
    [[ -p "$CONTEXT_FIFO" ]] || mkfifo "$CONTEXT_FIFO" 2>/dev/null
}

# Cleanup function
_context_cleanup() {
    [[ -n "$CONTEXT_FIFO" && -p "$CONTEXT_FIFO" ]] && rm -f "$CONTEXT_FIFO"
    [[ -n "$CONTEXT_READER_PID" ]] && kill "$CONTEXT_READER_PID" 2>/dev/null
}

# Output reader - reads from FIFO and writes to both terminal and log
_context_output_reader() {
    local log_file="$1"
    local fifo="$2"
    
    while IFS= read -r line; do
        echo "$line"                    # To terminal
        echo "$line" >> "$log_file"     # To log
    done < "$fifo"
}

# Debug trap: runs before each command
_context_debug_trap() {
    local cmd="$BASH_COMMAND"
    
    # Skip empty commands and internal stuff
    [[ -z "$cmd" ]] && return
    [[ "$cmd" == context* ]] && return
    [[ "$cmd" == exit* ]] && return
    [[ "$cmd" == *_context_* ]] && return
    
    # Setup
    _context_setup_fifo
    CONTEXT_CURRENT_CMD="$cmd"
    CONTEXT_CMD_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    CONTEXT_CMD_START_TIME=$(date +%s)
    CONTEXT_LOG_FILE="${CONTEXT_SESSION_DIR}/${CONTEXT_CMD_TIMESTAMP}.log"
    
    # Write header
    {
        echo "=== COMMAND: $cmd"
        echo "=== START_TIME: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "=== WORKING_DIR: $(pwd)"
        echo "=== OUTPUT:"
    } > "$CONTEXT_LOG_FILE"
    
    # Start output reader in background
    _context_output_reader "$CONTEXT_LOG_FILE" "$CONTEXT_FIFO" &
    CONTEXT_READER_PID=$!
    
    # Redirect stdout/stderr to FIFO
    exec 5>&1 6>&2
    exec 1>"$CONTEXT_FIFO"
    exec 2>"$CONTEXT_FIFO"
}

# Prompt command: runs after each command
_context_prompt_command() {
    local exit_code=$?
    
    # Restore stdout/stderr
    exec 1>&5 2>&6 2>/dev/null || true
    
    # Wait for reader to finish
    if [[ -n "$CONTEXT_READER_PID" ]]; then
        # Close FIFO to signal EOF to reader
        [[ -p "$CONTEXT_FIFO" ]] && rm -f "$CONTEXT_FIFO"
        wait "$CONTEXT_READER_PID" 2>/dev/null
    fi
    
    # Skip if no command was captured
    [[ -z "$CONTEXT_CURRENT_CMD" ]] && return
    
    # Write trailer
    local duration=$(($(date +%s) - CONTEXT_CMD_START_TIME))
    {
        echo ""
        echo "=== END_TIME: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "=== DURATION: ${duration}s"
        echo "=== EXIT_CODE: ${exit_code}"
    } >> "$CONTEXT_LOG_FILE"
    
    # Cleanup
    _context_cleanup
    unset CONTEXT_CURRENT_CMD CONTEXT_CMD_TIMESTAMP CONTEXT_CMD_START_TIME CONTEXT_LOG_FILE CONTEXT_READER_PID
}

# Set up hooks
trap _context_cleanup EXIT
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
