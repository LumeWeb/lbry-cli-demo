#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/../lib.sh"

# =============================================================================
# LBRY UPLOAD DEMO RUN SCRIPT
# =============================================================================
#
# This script runs the lbry-upload demo which:
# - Logs in to an existing LBRY account
# - Uploads a file to the portal's reflector endpoint
# - Uses the native liblbry implementation (no Docker infrastructure required)
# - Displays stream hash and SD blob hash
#
# Environment Variables:
#   PORTAL                 - Portal domain (default: pinner.xyz)
#   LOG_LEVEL              - Log level (debug, info, warn, error) (default: info)
#   REFLECTOR              - Reflector server address (default: lbry.<portal>.xyz:5566)
#
# Usage Examples:
#   # Interactive mode (prompts for email, password, file)
#   ./run.sh
#
#   # Non-interactive mode with flags
#   ./run.sh -email user@example.com -password secret -file /path/to/file.mp4
#
#   # Use custom portal
#   PORTAL=my-portal.example.com ./run.sh
#
#   # Use custom log level
#   LOG_LEVEL=debug ./run.sh
#
#   # Use custom reflector
#   REFLECTOR=my-reflector.example.com:5566 ./run.sh
#
# =============================================================================

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# State file for tracking upload information (using state manager)
STATE_FILE="lbry_upload.json"
LOG_FILE="${LOG_FILE:-$SCRIPT_DIR/lbry-upload.log}"

# Cleanup function
cleanup() {
    # No processes to cleanup for this demo
    true
}

# Signal handlers
trap cleanup EXIT
trap cleanup INT
trap cleanup TERM

# Prompt for user input interactively
prompt_user_input() {
    local email="$1"
    local password="$2"
    local file_path="$3"
    
    # Handle interrupt during prompts
    trap 'echo ""; log_error "Interrupted by user"; exit 130' INT
    
    # Prompt for email if not provided
    if [ -z "$email" ]; then
        echo -n "Email: "
        read -r email || { echo ""; log_error "Interrupted"; exit 130; }
    fi
    
    # Prompt for password if not provided
    if [ -z "$password" ]; then
        echo -n "Password: "
        read -rs password || { echo ""; log_error "Interrupted"; exit 130; }
        echo
    fi
    
    # Prompt for file path if not provided
    if [ -z "$file_path" ]; then
        echo -n "File path: "
        read -r file_path || { echo ""; log_error "Interrupted"; exit 130; }
    fi
    
    # Confirm device registration
    echo ""
    echo "IMPORTANT: Your device must be registered with your portal account."
    echo "If your device is not registered, the upload will fail."
    echo ""
    echo -n "Press Enter to confirm you have your device registered, or Ctrl+C to cancel..."
    read -r || { echo ""; log_error "Interrupted"; exit 130; }
    
    # Reset trap to default
    trap - INT
    
    # Export values for the Go program
    export UPLOAD_EMAIL="$email"
    export UPLOAD_PASSWORD="$password"
    export UPLOAD_FILE="$file_path"
}

# Main execution
main() {
    show_script_header "LBRY Upload Demo" "Upload files to LBRY using native liblbry implementation
    
This demo:
- Logs in to an existing LBRY account
- Uploads a file to the portal's reflector endpoint
- Uses native liblbry (no Docker infrastructure required)
- Displays stream hash and SD blob hash

Interactive Mode:
  Run without flags to be prompted for:
  - Email
  - Password
  - File path

Non-Interactive Mode:
  Use flags to provide all required inputs:
  ./run.sh -email user@example.com -password secret -file /path/to/file.mp4

Environment Variables:
  PORTAL: ${PORTAL:-pinner.xyz (default)}
  LOG_LEVEL: ${LOG_LEVEL:-info (default)}
  REFLECTOR: ${REFLECTOR:-lbry.<portal>.xyz:5566 (default)}"

    # Initialize
    check_minimal_dependencies

    # Ensure Go binaries are accessible in PATH
    setup_go_path

    # Parse flags to get email, password, and file path
    local email=""
    local password=""
    local file_path=""
    
    # Parse command line arguments for -email, -password, -file
    while [[ $# -gt 0 ]]; do
        case $1 in
            -email)
                email="$2"
                shift 2
                ;;
            -password)
                password="$2"
                shift 2
                ;;
            -file)
                file_path="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done

    # Prompt for interactive input if not provided via flags
    prompt_user_input "$email" "$password" "$file_path"
    
    # Validate file exists
    if [ ! -f "$UPLOAD_FILE" ]; then
        log_error "File does not exist: $UPLOAD_FILE"
        exit 1
    fi
    
    # Run the lbry-upload demo
    log "Starting LBRY upload demo..."
    cd "$SCRIPT_DIR"
    
    # Build arguments for Go program
    local go_args=(
        "-email" "$UPLOAD_EMAIL"
        "-password" "$UPLOAD_PASSWORD"
        "-file" "$UPLOAD_FILE"
    )
    
    # Add optional arguments
    if [ -n "${PORTAL:-}" ]; then
        go_args+=("-portal" "$PORTAL")
    fi
    if [ -n "${REFLECTOR:-}" ]; then
        go_args+=("-reflector" "$REFLECTOR")
    fi
    if [ -n "${LOG_LEVEL:-}" ]; then
        go_args+=("-log-level" "$LOG_LEVEL")
    fi
    
    # Build and run the Go program
    if run_go_command run main.go "${go_args[@]}" 2>&1 | tee -a "$LOG_FILE"; then
        log_success "LBRY upload demo completed successfully"
    else
        log_error "LBRY upload demo failed"
        exit 1
    fi

    show_script_footer "LBRY Upload Demo" "success" "Upload completed successfully!"
}

# Run main function
main "$@"
