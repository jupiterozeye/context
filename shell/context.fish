# Context Shell Integration for Fish
# Captures command output using preexec/postexec hooks

set -gx CONTEXT_LOG_DIR "$HOME/.context/logs"
set -gx CONTEXT_LOG_ENABLED (test -n "$CONTEXT_LOG_ENABLED"; and echo "$CONTEXT_LOG_ENABLED"; or echo 1)
set -gx CONTEXT_MAX_SIZE_MB (test -n "$CONTEXT_MAX_SIZE_MB"; and echo "$CONTEXT_MAX_SIZE_MB"; or echo 50)

# Skip if disabled
if test "$CONTEXT_LOG_ENABLED" != "1"
    exit 0
end

# Create directories
mkdir -p $CONTEXT_LOG_DIR

# Preexec: runs before each command
function _context_preexec --on-event fish_preexec
    set -g CONTEXT_CURRENT_CMD $argv[1]
    
    # Skip context commands and empty commands
    if string match -q "context*" $CONTEXT_CURRENT_CMD; or test -z "$CONTEXT_CURRENT_CMD"
        set -g CONTEXT_SKIP_LOGGING 1
        return
    end
    
    set -g CONTEXT_SKIP_LOGGING 0
    set -g CONTEXT_CMD_START_TIME (date +%s)
    set -g CONTEXT_CMD_TIMESTAMP (date +%Y%m%d_%H%M%S)
    set -g CONTEXT_OUTPUT_FILE (mktemp)
end

# Postexec: runs after each command
function _context_postexec --on-event fish_postexec
    set exit_code $status
    
    # Skip if we shouldn't log
    if test "$CONTEXT_SKIP_LOGGING" = "1"; or test -z "$CONTEXT_CURRENT_CMD"
        set -g CONTEXT_SKIP_LOGGING 0
        set -g CONTEXT_CURRENT_CMD ""
        return
    end
    
    set end_time (date +%s)
    set duration (math $end_time - $CONTEXT_CMD_START_TIME)
    set sanitized_cmd (echo $CONTEXT_CURRENT_CMD | tr -cd '[:alnum:]._-' | cut -c1-50)
    set log_file "$CONTEXT_LOG_DIR/$CONTEXT_CMD_TIMESTAMP-$sanitized_cmd.log"
    
    # Write log entry
    echo "=== COMMAND: $CONTEXT_CURRENT_CMD" > $log_file
    echo "=== START_TIME: "(date -r $CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null; or date -r $CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null; or date '+%Y-%m-%d %H:%M:%S') >> $log_file
    echo "=== END_TIME: "(date '+%Y-%m-%d %H:%M:%S') >> $log_file
    echo "=== DURATION: {$duration}s" >> $log_file
    echo "=== EXIT_CODE: $exit_code" >> $log_file
    echo "=== WORKING_DIR: "(pwd) >> $log_file
    
    # Note: Fish doesn't easily support capturing output with tee like bash/zsh
    # Users should use 'context rec' for full output capture in fish
    
    # Clean old logs occasionally (1% chance)
    if test (random) -lt 327
        _context_clean_logs
    end
    
    # Reset
    set -g CONTEXT_SKIP_LOGGING 0
    set -g CONTEXT_CURRENT_CMD ""
end

# Clean old logs
function _context_clean_logs
    # Delete logs older than 30 days
    find $CONTEXT_LOG_DIR -type f -name "*.log" -mtime +30 -delete 2>/dev/null
    
    # Check total size
    set total_size (du -sm $CONTEXT_LOG_DIR 2>/dev/null | cut -f1)
    if test $total_size -gt $CONTEXT_MAX_SIZE_MB
        while test $total_size -gt $CONTEXT_MAX_SIZE_MB
            set oldest (find $CONTEXT_LOG_DIR -type f -name "*.log" -printf '%T+ %p\n' 2>/dev/null | sort | head -1 | cut -d' ' -f2)
            if test -n "$oldest"
                rm -f $oldest
                set total_size (du -sm $CONTEXT_LOG_DIR 2>/dev/null | cut -f1)
            else
                break
            end
        end
    end
end
