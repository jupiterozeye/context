# Context Shell Integration for Zsh
# Captures command output using typescript (script) for reliable logging

export CONTEXT_LOG_DIR="${HOME}/.context/logs"
export CONTEXT_TYPESCRIPT="${HOME}/.context/typescript"
export CONTEXT_LOG_ENABLED=${CONTEXT_LOG_ENABLED:-1}
export CONTEXT_MAX_SIZE_MB=${CONTEXT_MAX_SIZE_MB:-50}

# Prevent nested recording
if [[ -n "$CONTEXT_RECORDING" ]] || [[ ${CONTEXT_LOG_ENABLED} -ne 1 ]]; then
    return 0 2>/dev/null || true
fi

# Only in interactive shells
[[ ! -o interactive ]] && return 0 2>/dev/null || true

# Don't run in subshells
[[ -n "$ZSH_SUBSHELL" ]] && return 0 2>/dev/null || true

# Check if parent is already script
if command -v pstree >/dev/null 2>&1; then
    if pstree -s $$ 2>/dev/null | grep -q "script"; then
        return 0 2>/dev/null || true
    fi
fi

# Create directories
mkdir -p "$CONTEXT_LOG_DIR"
mkdir -p "$(dirname "$CONTEXT_TYPESCRIPT")"

# Rotate typescript if too large
if [[ -f "$CONTEXT_TYPESCRIPT" ]]; then
    local size=$(stat -c%s "$CONTEXT_TYPESCRIPT" 2>/dev/null || echo 0)
    local max=$((CONTEXT_MAX_SIZE_MB * 1024 * 1024))
    if [[ $size -gt $max ]]; then
        tail -c 10485760 "$CONTEXT_TYPESCRIPT" > "${CONTEXT_TYPESCRIPT}.tmp" 2>/dev/null
        mv "${CONTEXT_TYPESCRIPT}.tmp" "$CONTEXT_TYPESCRIPT" 2>/dev/null
    fi
fi

# Start recording session
export CONTEXT_RECORDING=1
exec script -q -a "$CONTEXT_TYPESCRIPT" -c "zsh -i"
