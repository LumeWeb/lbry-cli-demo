#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/../lib.sh"

# =============================================================================
# TUS-UPLOAD DEMO RUN SCRIPT
# =============================================================================
#
# This script runs TUS upload demo which:
# - Starts LBRY SDK daemon
# - Clears existing blobs
# - Uploads streams using TUS protocol
# - Downloads and verifies uploaded content
# - Performs blob get operations
# - Saves files using sd_hash via lbry-cli
# - Cleans up resources
#
# =============================================================================

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
DEMO_NAME="tus-upload"
LOG_FILE="${LOG_FILE:-$SCRIPT_DIR/tus-upload.log}"

# Main function
main() {
    log "=== TUS UPLOAD DEMO: Two-Stage Verification Process ==="
    log "Stage 1: Our liblbry Go implementation will upload/download blobs via TUS protocol"
    log "Stage 2: Reference LBRY Python implementation will verify the same blobs"
    log "This proves our liblbry TUS implementation is fully compatible and correct"
    log ""
    log "Starting TUS upload demo system..."
    log "Log file: $LOG_FILE"
    
    # Setup LBRY SDK first
    if ! setup_lbry_for_demo; then
        exit 1
    fi
    
    # Run demo using common template
    run_demo "$DEMO_NAME" "$SCRIPT_DIR" "$LOG_FILE"
    
    # Perform post-demo LBRY operations
    if perform_post_demo_lbry_operations "TUS upload" "tus_upload_saved_file.bin" "$SCRIPT_DIR" "tus_upload.json"; then
        log_success "TUS upload demo system completed successfully!"
    else
        log_warning "TUS upload demo system completed with some issues"
    fi
    
    # Clean up local bin files
    if cleanup_demo_bins "$SCRIPT_DIR"; then
        log_success "Local bin files cleaned up successfully"
    else
        log_warning "Failed to clean up local bin files"
    fi
}

# Run main function with all arguments
main "$@"