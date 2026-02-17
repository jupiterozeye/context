# Context Shell Integration for Fish
# Captures command output using typescript (script) for reliable logging

set -gx CONTEXT_LOG_DIR "$HOME/.context/logs"
set -gx CONTEXT_TYPESCRIPT "$HOME/.context/typescript"
set -gx CONTEXT_LOG_ENABLED (test -n "$CONTEXT_LOG_ENABLED"; and echo "$CONTEXT_LOG_ENABLED"; or echo 1)
set -gx CONTEXT_MAX_SIZE_MB (test -n "$CONTEXT_MAX_SIZE_MB"; and echo "$CONTEXT_MAX_SIZE_MB"; or echo 50)

# Prevent nested recording
if test -n "$CONTEXT_RECORDING"; or test "$CONTEXT_LOG_ENABLED" != "1"
    exit 0
end

# Only in interactive shells
if not status is-interactive
    exit 0
end

# Check if already in script session
if test (ps -o comm= -p $fish_pid 2>/dev/null) = "script"
    exit 0
end

# Create directories
mkdir -p "$CONTEXT_LOG_DIR"
mkdir -p (dirname "$CONTEXT_TYPESCRIPT")

# Rotate typescript if too large
if test -f "$CONTEXT_TYPESCRIPT"
    set size (stat -c%s "$CONTEXT_TYPESCRIPT" 2>/dev/null; or echo 0)
    set max (math $CONTEXT_MAX_SIZE_MB * 1024 * 1024)
    if test $size -gt $max
        tail -c 10485760 "$CONTEXT_TYPESCRIPT" > "$CONTEXT_TYPESCRIPT.tmp" 2>/dev/null
        mv "$CONTEXT_TYPESCRIPT.tmp" "$CONTEXT_TYPESCRIPT" 2>/dev/null
    end
end

# Start recording session
set -gx CONTEXT_RECORDING 1
exec script -q -a "$CONTEXT_TYPESCRIPT" -c "fish -i"
