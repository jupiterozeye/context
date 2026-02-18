# Context Command Capture
# Source this in your shell to enable command logging
# Usage: source /path/to/context-capture.sh

export CONTEXT_LOG_DIR="${HOME}/.context/logs"
export CONTEXT_CAPTURE_ENABLED=1

# Create log directory
mkdir -p "$CONTEXT_LOG_DIR"

# Function to capture command output
_context_capture() {
    local cmd="$1"
    shift
    local args="$@"
    
    # Skip empty commands and context commands
    [[ -z "$cmd" ]] && return
    [[ "$cmd" == "context" ]] && { command context $args; return; }
    
    # Generate timestamp
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local log_file="${CONTEXT_LOG_DIR}/${timestamp}.log"
    
    # Record start time
    local start_time=$(date '+%Y-%m-%d %H:%M:%S')
    local start_epoch=$(date +%s)
    local pwd=$(pwd)
    
    # Write command header
    {
        echo "=== COMMAND: $cmd $args"
        echo "=== START_TIME: $start_time"
        echo "=== WORKING_DIR: $pwd"
        echo "=== OUTPUT:"
    } > "$log_file"
    
    # Run the command and capture output to both terminal and log
    {
        echo "$ $cmd $args"
        command "$cmd" "$@" 2>&1
    } | tee -a "$log_file"
    
    local exit_code=${PIPESTATUS[1]}
    
    # Record end time
    local end_time=$(date '+%Y-%m-%d %H:%M:%S')
    local end_epoch=$(date +%s)
    local duration=$((end_epoch - start_epoch))
    
    # Append trailer
    {
        echo ""
        echo "=== END_TIME: $end_time"
        echo "=== DURATION: ${duration}s"
        echo "=== EXIT_CODE: $exit_code"
    } >> "$log_file"
    
    return $exit_code
}

# Override command_not_found_handle for bash
command_not_found_handle() {
    _context_capture "$@"
}
