# Context Shell Integration for Fish
# Simple command capture using preexec/postexec hooks

set -gx CONTEXT_LOG_DIR "$HOME/.context/logs"
set -gx CONTEXT_LOG_ENABLED (test -n "$CONTEXT_LOG_ENABLED"; and echo "$CONTEXT_LOG_ENABLED"; or echo 1)
set -gx CONTEXT_MAX_SIZE_MB (test -n "$CONTEXT_MAX_SIZE_MB"; and echo "$CONTEXT_MAX_SIZE_MB"; or echo 50)

# Skip if disabled
if test "$CONTEXT_LOG_ENABLED" != "1"
    exit 0
end

# Create directories
mkdir -p $CONTEXT_LOG_DIR

# Generate session ID if not set
if test -z "$CONTEXT_SESSION_ID"
    set -gx CONTEXT_SESSION_ID (date +%s%N | sha256sum | cut -c1-16)
end

# Session-specific log directory
set -gx CONTEXT_SESSION_DIR "$CONTEXT_LOG_DIR/$CONTEXT_SESSION_ID"
mkdir -p $CONTEXT_SESSION_DIR

# Preexec: runs before each command
function _context_preexec --on-event fish_preexec
    set -g CONTEXT_CURRENT_CMD $argv[1]
    
    # Skip empty commands and context commands
    if string match -q "context*" $CONTEXT_CURRENT_CMD; or test -z "$CONTEXT_CURRENT_CMD"
        return
    end
    
    set -g CONTEXT_CMD_START_TIME (date +%s)
    set -g CONTEXT_CMD_TIMESTAMP (date +%Y%m%d_%H%M%S)
    set -g CONTEXT_OUTPUT_FILE (mktemp)
end

# Postexec: runs after each command
function _context_postexec --on-event fish_postexec
    set exit_code $status
    
    # Skip if no command was captured
    test -z "$CONTEXT_CURRENT_CMD"; and return
    
    set log_file "$CONTEXT_SESSION_DIR/$CONTEXT_CMD_TIMESTAMP.log"
    set duration (math (date +%s) - $CONTEXT_CMD_START_TIME)
    
    # Write log entry
    echo "=== COMMAND: $CONTEXT_CURRENT_CMD" > $log_file
    echo "=== START_TIME: "(date -r $CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null; or date '+%Y-%m-%d %H:%M:%S') >> $log_file
    echo "=== END_TIME: "(date '+%Y-%m-%d %H:%M:%S') >> $log_file
    echo "=== DURATION: {$duration}s" >> $log_file
    echo "=== EXIT_CODE: $exit_code" >> $log_file
    echo "=== WORKING_DIR: "(pwd) >> $log_file
    if test -f "$CONTEXT_OUTPUT_FILE"
        echo "=== OUTPUT:" >> $log_file
        cat $CONTEXT_OUTPUT_FILE >> $log_file
    end
    
    # Cleanup
    rm -f $CONTEXT_OUTPUT_FILE
    set -e CONTEXT_CURRENT_CMD
    set -e CONTEXT_CMD_START_TIME
    set -e CONTEXT_CMD_TIMESTAMP
    set -e CONTEXT_OUTPUT_FILE
end

# Show indicator if in recorded session
if test -n "$CONTEXT_RECORDING"
    function fish_prompt
        echo -n "[rec] "
        command fish_prompt
    end
end
