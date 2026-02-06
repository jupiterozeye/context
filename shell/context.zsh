# Context Shell Integration for Zsh
# Captures commands and their output to ~/.context/logs/

# Configuration
CONTEXT_LOG_DIR="${HOME}/.context/logs"
CONTEXT_LOG_ENABLED=${CONTEXT_LOG_ENABLED:-1}
CONTEXT_MAX_LOG_AGE_DAYS=30
CONTEXT_MAX_LOG_SIZE_MB=100

# Create log directory
mkdir -p "$CONTEXT_LOG_DIR"

# Function to sanitize command for filename
_context_sanitize_cmd() {
    echo "$1" | tr -cd '[:alnum:]._-' | cut -c1-50
}

# Format timestamp for GNU date (Linux)
_context_format_time() {
    local epoch="$1"
    date -d "@$epoch" '+%Y-%m-%d %H:%M:%S' 2>/dev/null || date '+%Y-%m-%d %H:%M:%S'
}

# Clean old logs
_context_clean_logs() {
    find "$CONTEXT_LOG_DIR" -type f -name "*.log" -mtime +$CONTEXT_MAX_LOG_AGE_DAYS -delete 2>/dev/null
    
    local total_size=$(du -sm "$CONTEXT_LOG_DIR" 2>/dev/null | cut -f1)
    if [[ $total_size -gt $CONTEXT_MAX_LOG_SIZE_MB ]]; then
        while [[ $total_size -gt $CONTEXT_MAX_LOG_SIZE_MB ]]; do
            local oldest=$(find "$CONTEXT_LOG_DIR" -type f -name "*.log" -printf '%T+ %p\n' 2>/dev/null | sort | head -1 | cut -d' ' -f2)
            [[ -n "$oldest" ]] && rm -f "$oldest"
            total_size=$(du -sm "$CONTEXT_LOG_DIR" 2>/dev/null | cut -f1)
        done
    fi
}

# State variables
typeset -g _CTX_CMD=""
typeset -g _CTX_LOG=""
typeset -g _CTX_START=""
typeset -g _CTX_OUT=""
typeset -g _CTX_SKIP=0

# Called before command execution
_context_preexec() {
    local cmd="$1"
    
    # Skip context commands, empty commands, and interactive shells
    if [[ "$cmd" == context* ]] || [[ -z "$cmd" ]] || [[ "$cmd" =~ ^[[:space:]]*$ ]] || [[ -n "$ZSH_EXECUTION_STRING" ]]; then
        _CTX_SKIP=1
        _CTX_CMD=""
        return
    fi
    
    _CTX_SKIP=0
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local sanitized=$(_context_sanitize_cmd "$cmd")
    
    _CTX_LOG="${CONTEXT_LOG_DIR}/${timestamp}_${sanitized}.log"
    _CTX_CMD="$cmd"
    _CTX_START=$(date +%s)
    _CTX_OUT=$(mktemp)
    
    # Save original file descriptors and redirect to both terminal and file
    exec 5>&1 6>&2
    exec 1> >(tee "$_CTX_OUT" >&5)
    exec 2> >(tee "$_CTX_OUT" >&6)
}

# Called after command execution  
_context_precmd() {
    local exit_code=$?
    
    # Restore file descriptors
    if [[ -n "$_CTX_OUT" ]]; then
        exec 1>&5 2>&6
        exec 5>&- 6>&-
    fi
    
    # Only process if we logged a command
    if [[ $_CTX_SKIP -eq 0 && -n "$_CTX_CMD" && -n "$_CTX_LOG" ]]; then
        local end_time=$(date +%s)
        local duration=$((end_time - _CTX_START))
        
        # Small delay to ensure output is flushed
        sleep 0.05
        
        # Write log
        cat > "$_CTX_LOG" <<EOF
=== COMMAND: ${_CTX_CMD}
=== START_TIME: $(_context_format_time $_CTX_START)
=== END_TIME: $(_context_format_time $end_time)
=== DURATION: ${duration}s
=== EXIT_CODE: ${exit_code}
=== WORKING_DIR: $(pwd)
EOF
        
        # Add output if we have any
        if [[ -f "$_CTX_OUT" ]]; then
            local output=$(cat "$_CTX_OUT" 2>/dev/null)
            if [[ -n "$output" ]]; then
                echo "" >> "$_CTX_LOG"
                echo "=== OUTPUT:" >> "$_CTX_LOG"
                echo "$output" >> "$_CTX_LOG"
            fi
            rm -f "$_CTX_OUT"
        fi
        
        # Clean old logs occasionally
        [[ $((RANDOM % 100)) -eq 0 ]] && _context_clean_logs
    fi
    
    # Reset
    _CTX_CMD=""
    _CTX_LOG=""
    _CTX_OUT=""
    _CTX_SKIP=0
}

# Setup
_context_setup() {
    autoload -Uz add-zsh-hook 2>/dev/null || return
    add-zsh-hook preexec _context_preexec
    add-zsh-hook precmd _context_precmd
}

if [[ ${CONTEXT_LOG_ENABLED} -eq 1 ]]; then
    _context_setup
fi
