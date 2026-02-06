# Context Shell Integration for Fish
# Add this to your ~/.config/fish/config.fish

# Configuration
set -g CONTEXT_LOG_DIR "$HOME/.context/logs"
set -g CONTEXT_LOG_ENABLED 1
set -g CONTEXT_MAX_LOG_AGE_DAYS 30
set -g CONTEXT_MAX_LOG_SIZE_MB 100

# Create log directory
mkdir -p $CONTEXT_LOG_DIR

# Function to sanitize command for filename
function _context_sanitize_cmd
    echo $argv | tr -cd '[:alnum:]._-' | cut -c1-50
end

# Function to clean old logs
function _context_clean_logs
    find $CONTEXT_LOG_DIR -type f -name "*.log" -mtime +$CONTEXT_MAX_LOG_AGE_DAYS -delete 2>/dev/null
    
    # Check total size
    set total_size (du -sm $CONTEXT_LOG_DIR 2>/dev/null | cut -f1)
    if test $total_size -gt $CONTEXT_MAX_LOG_SIZE_MB
        while test $total_size -gt $CONTEXT_MAX_LOG_SIZE_MB
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

# Function called before command execution
function _context_preexec --on-event fish_preexec
    set -g CONTEXT_CURRENT_CMD $argv[1]
    
    # Skip context commands and empty commands
    if string match -q "context*" $CONTEXT_CURRENT_CMD; or test -z "$CONTEXT_CURRENT_CMD"
        set -g CONTEXT_SKIP_LOGGING 1
        return
    end
    
    set -g CONTEXT_SKIP_LOGGING 0
    set timestamp (date +%Y%m%d_%H%M%S)
    set sanitized_cmd (_context_sanitize_cmd $CONTEXT_CURRENT_CMD)
    
    set -g CONTEXT_CURRENT_LOG "$CONTEXT_LOG_DIR/$timestamp-$sanitized_cmd.log"
    set -g CONTEXT_CMD_START_TIME (date +%s)
    set -g CONTEXT_OUTPUT_FILE (mktemp)
end

# Function called after command execution  
function _context_postexec --on-event fish_postexec
    set exit_code $status
    
    # Skip if we shouldn't log
    if test "$CONTEXT_SKIP_LOGGING" = "1"; or test -z "$CONTEXT_CURRENT_LOG"
        set -g CONTEXT_SKIP_LOGGING 0
        set -g CONTEXT_CURRENT_LOG ""
        set -g CONTEXT_CURRENT_CMD ""
        return
    end
    
    set end_time (date +%s)
    set duration (math $end_time - $CONTEXT_CMD_START_TIME)
    
    # Write log entry
    echo "=== COMMAND: $CONTEXT_CURRENT_CMD" > $CONTEXT_CURRENT_LOG
    echo "=== START_TIME: "(date -d @$CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null; or date -r $CONTEXT_CMD_START_TIME '+%Y-%m-%d %H:%M:%S' 2>/dev/null; or date) >> $CONTEXT_CURRENT_LOG
    echo "=== END_TIME: "(date '+%Y-%m-%d %H:%M:%S') >> $CONTEXT_CURRENT_LOG
    echo "=== DURATION: {$duration}s" >> $CONTEXT_CURRENT_LOG
    echo "=== EXIT_CODE: $exit_code" >> $CONTEXT_CURRENT_LOG
    echo "=== WORKING_DIR: "(pwd) >> $CONTEXT_CURRENT_LOG
    
    # Clean old logs periodically (1% chance)
    if test (random) -lt 327
        _context_clean_logs
    end
    
    # Reset
    set -g CONTEXT_SKIP_LOGGING 0
    set -g CONTEXT_CURRENT_LOG ""
    set -g CONTEXT_CURRENT_CMD ""
end

# Enable logging if configured
if test $CONTEXT_LOG_ENABLED -eq 1
    # Functions are already registered via --on-event
end
