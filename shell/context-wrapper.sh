#!/bin/bash
# Context command wrapper - captures output of a single command
# Usage: eval "$(context-wrapper.sh)"

context_run() {
    local cmd="$@"
    local log_dir="${HOME}/.context/logs/${CONTEXT_SESSION_ID:-default}"
    
    mkdir -p "$log_dir"
    
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local log_file="${log_dir}/${timestamp}.log"
    local start_time=$(date '+%Y-%m-%d %H:%M:%S')
    local start_epoch=$(date +%s)
    
    # Write header
    cat > "$log_file" <<EOF
=== COMMAND: $cmd
=== START_TIME: $start_time
=== WORKING_DIR: $(pwd)
=== OUTPUT:
EOF

    # Run command and capture output
    echo "$ $cmd" >> "$log_file"
    eval "$cmd" 2>&1 | tee -a "$log_file"
    local exit_code=${PIPESTATUS[0]}
    
    # Write trailer
    cat >> "$log_file" <<EOF

=== END_TIME: $(date '+%Y-%m-%d %H:%M:%S')
=== DURATION: $(($(date +%s) - start_epoch))s
=== EXIT_CODE: $exit_code
EOF

    return $exit_code
}

export -f context_run
