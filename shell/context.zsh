# Context Shell Integration for Zsh
# Automatically wraps shell in script session for full output capture

CONTEXT_LOG_DIR="${HOME}/.context"
CONTEXT_TYPESCRIPT="${CONTEXT_LOG_DIR}/typescript"
CONTEXT_LOG_ENABLED=${CONTEXT_LOG_ENABLED:-1}
CONTEXT_MAX_SIZE_MB=${CONTEXT_MAX_SIZE_MB:-50}

# Create directories
mkdir -p "$CONTEXT_LOG_DIR"

# Early exit if already in script session or if disabled
if [[ -n "$CONTEXT_RECORDING" ]] || [[ ${CONTEXT_LOG_ENABLED} -ne 1 ]]; then
    return 0 2>/dev/null || true
fi

# Only run in interactive shells
[[ ! -o interactive ]] && return 0 2>/dev/null || true

# Don't run in subshells (like command substitution)
[[ -n "$ZSH_SUBSHELL" ]] && return 0 2>/dev/null || true

# Check if parent is already script
if command -v pstree >/dev/null 2>&1; then
    if pstree -s $$ 2>/dev/null | grep -q "script"; then
        return 0 2>/dev/null || true
    fi
fi

# Rotate typescript if too large
if [[ -f "$CONTEXT_TYPESCRIPT" ]]; then
    local size=$(stat -c%s "$CONTEXT_TYPESCRIPT" 2>/dev/null || echo 0)
    local max=$((CONTEXT_MAX_SIZE_MB * 1024 * 1024))
    if [[ $size -gt $max ]]; then
        # Keep last 10MB
        tail -c 10485760 "$CONTEXT_TYPESCRIPT" > "${CONTEXT_TYPESCRIPT}.new" 2>/dev/null
        mv "${CONTEXT_TYPESCRIPT}.new" "$CONTEXT_TYPESCRIPT" 2>/dev/null
    fi
fi

# Start script session
export CONTEXT_RECORDING=1
exec script -q -a "$CONTEXT_TYPESCRIPT" -c "CONTEXT_RECORDING=1 exec zsh -i"
