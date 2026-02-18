# Context Shell Integration for Zsh
# Simple approach: use 'script' command for recording

export CONTEXT_LOG_DIR="${HOME}/.context/logs"
export CONTEXT_LOG_ENABLED=${CONTEXT_LOG_ENABLED:-1}

# Skip if disabled
[[ ${CONTEXT_LOG_ENABLED} -ne 1 ]] && return 0

# Show indicator if in recorded session
if [[ -n "$CONTEXT_RECORDING" ]]; then
    PS1="[rec] $PS1"
fi

# NOTE: Actual capture happens via 'context rec' which uses 'script'
# This file just shows the indicator and sets up the environment
