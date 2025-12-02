#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/../lib.sh"

# =============================================================================
# PIN DEMO RUN SCRIPT
# =============================================================================
#
# This script runs PIN demo which:
# - Starts LBRY SDK daemon
# - Clears existing blobs
# - Pins a specific stream by SD hash
# - Downloads and verifies pinned content
# - Performs blob get operations
# - Saves files using sd_hash via lbry-cli
# - Cleans up resources
#
# =============================================================================

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
DEMO_NAME="pin"
LOG_FILE="${LOG_FILE:-$SCRIPT_DIR/pin.log}"

# Main function
main() {
    log "Starting PIN demo system..."
    log "Log file: $LOG_FILE"
    
    # Setup LBRY SDK first
    if ! setup_lbry_for_demo; then
        exit 1
    fi
    
    # Run demo using common template
    run_demo "$DEMO_NAME" "$SCRIPT_DIR" "$LOG_FILE"
    
    # Perform post-demo LBRY operations
    if perform_post_demo_lbry_operations "PIN" "pin_saved_file.bin" "$SCRIPT_DIR" "pin.json"; then
        log_success "PIN demo system completed successfully!"
    else
        log_warning "PIN demo system completed with some issues"
    fi

    # Clean up local bin files
    if cleanup_demo_bins "$SCRIPT_DIR"; then
        log_success "Local bin files cleaned up successfully"
    else
        log_warning "Failed to clean up local bin files"
    fi

    # Run unpin operation after all pin operations are complete
    log "Starting unpin operation..."
    if run_unpin_operation "$SCRIPT_DIR" "$LOG_FILE"; then
        log_success "Unpin operation completed successfully!"
    else
        log_warning "Unpin operation failed"
    fi

    log_success "PIN demo system with unpin completed successfully!"
}

# Function to run unpin operation
run_unpin_operation() {
    local script_dir="$1"
    local log_file="${2:-$script_dir/pin.log}"
    
    log "Running unpin operation..." "$log_file"
    
    cd "$script_dir" || exit
    
    # Run the unpin operation and capture output
    log "Running unpin operation..." "$log_file"
    if go run main.go -mode unpin 2>&1 | tee -a "$log_file"; then
        log_success "Unpin operation completed successfully" "$log_file"
        return 0
    else
        log_error "Unpin operation failed" "$log_file"
        exit 1
    fi
}

# Run main function with all arguments
main "$@"