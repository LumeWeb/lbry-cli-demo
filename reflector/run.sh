#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/../lib.sh"

# =============================================================================
# REFLECTOR DEMO RUN SCRIPT
# =============================================================================
#
# This script runs the reflector demo which:
# - Starts a local reflector server
# - Uploads streams to the reflector
# - Downloads and verifies uploaded content
# - Reflects blobs to external services
# - Waits for operations to complete and performs final verification
# - Manages the complete reflector workflow
#
# =============================================================================
#
# REFLECTOR RUN SCRIPT - Environment Variable Configuration
# =============================================================================
#
# This script can be configured using the following environment variables:
#
# Port Configuration:
#   REFLECTOR_PORT         - Primary reflector server port (default: 5669)
#   REFLECTOR_PEER_PORT    - Primary peer server port (default: 5570)
#
# External Reflector:
#   REFLECTOR_SERVER       - External reflector server address (default: lbry.pinner.xyz:5566)
#
# Standard Demo Configuration:
#   PORTAL                 - Portal domain (default: pinner.xyz)
#   LOG_LEVEL              - Log level (debug, info, warn, error) (default: info)
#   LOG_FILE               - Optional log file path (default: reflector.log)
#
# Reflector-Specific Configuration:
#   REFLECTOR_PORT         - Primary reflector server port (default: 5669)
#   REFLECTOR_PEER_PORT    - Primary peer server port (default: 5570)
#   REFLECTOR_SERVER       - External reflector server address (default: lbry.pinner.xyz:5566)
#
# Usage Examples:
#   # Use default configuration
#   ./run.sh
#
#   # Use custom portal domain
#   PORTAL=my-portal.example.com ./run.sh
#
#   # Use custom log level
#   LOG_LEVEL=debug ./run.sh
#
#   # Use custom ports
#   REFLECTOR_PORT=6669 REFLECTOR_PEER_PORT=6570 ./run.sh
#
#   # Use custom external reflector
#   REFLECTOR_SERVER=my-reflector.example.com:5566 ./run.sh
#
#   # Full custom configuration
#   PORTAL=my-portal.example.com \
#   LOG_LEVEL=debug \
#   LOG_FILE=custom.log \
#   REFLECTOR_PORT=6669 \
#   REFLECTOR_PEER_PORT=6570 \
#   REFLECTOR_SERVER=my-reflector.example.com:5566 \
#   ./run.sh
#
# =============================================================================

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# State file for tracking blob information (using state manager)
STATE_FILE="reflector.json"
# Support standard LOG_FILE environment variable
LOG_FILE="${LOG_FILE:-$SCRIPT_DIR/reflector.log}"
REFLECTOR_LOG_FILE="$LOG_FILE"
UPLOADER_LOG_FILE="$SCRIPT_DIR/reflector-uploader.log"
PID_FILE="$SCRIPT_DIR/reflector.pid"

# Cleanup function
cleanup() {
    cleanup_process "reflector server" "$PID_FILE" "$REFLECTOR_LOG_FILE"
}

# Signal handlers
trap cleanup EXIT
trap cleanup INT
trap cleanup TERM

# Check dependencies
check_dependencies() {
    log "Checking dependencies..."
    
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed"
        exit 1
    fi
    
    if ! command -v lbry-cli >/dev/null 2>&1; then
        log_error "lbry-cli is not installed or not in PATH"
        exit 1
    fi
    
    if [ ! -f "$PROJECT_ROOT/cleanup-blobs.sh" ]; then
        log_error "cleanup-blobs.sh not found at $PROJECT_ROOT/cleanup-blobs.sh"
        exit 1
    fi
    
    log_success "All dependencies available"
}





# Clean up blobs and restart services
cleanup_and_restart_services() {
    log "Cleaning up blobs and restarting LBRY services..."
    
    cd "$PROJECT_ROOT"
    if ./cleanup-blobs.sh; then
        log_success "Blob cleanup and service restart completed successfully"
    else
        log_error "Failed to cleanup blobs and restart services"
        exit 1
    fi
    
    cd "$SCRIPT_DIR"
}

# Port configuration with environment variable overrides
get_port_config() {
    # Primary ports with environment variable defaults
    REFLECTOR_PORT=${REFLECTOR_PORT:-5669}
    REFLECTOR_PEER_PORT=${REFLECTOR_PEER_PORT:-5570}
    
    # External reflector server (already configurable)
    REFLECTOR_SERVER=${REFLECTOR_SERVER:-"lbry.pinner.xyz:5566"}
    
    # Portal domain configuration
    PORTAL=${PORTAL:-"pinner.xyz"}
    
    log "Port configuration:"
    log "  Reflector Port: $REFLECTOR_PORT"
    log "  Peer Port: $REFLECTOR_PEER_PORT"
    log "  External Reflector: $REFLECTOR_SERVER"
    log "  Portal Domain: $PORTAL"
}

# Validate port number
validate_port() {
    local port=$1
    local port_name=$2
    
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        log_error "Invalid $port_name: $port (must be 1-65535)"
        return 1
    fi
    return 0
}

# Validate all port configurations
validate_port_config() {
    validate_port "$REFLECTOR_PORT" "REFLECTOR_PORT" || return 1
    validate_port "$REFLECTOR_PEER_PORT" "REFLECTOR_PEER_PORT" || return 1
    
    log_success "All port configurations are valid"
    return 0
}

# Kill process using a specific port
kill_process_on_port() {
    local port=$1
    local port_name=$2
    
    if lsof -i:$port >/dev/null 2>&1; then
        log_warning "Port $port is already in use, killing process..."
        
        # Get PID of process using the port
        local pid
        pid=$(lsof -ti:$port)
        
        if [ -n "$pid" ]; then
            log "Killing process $pid using port $port ($port_name)"
            kill -9 "$pid" 2>/dev/null || true
            
            # Wait a moment for the process to die
            sleep 2
            
            # Verify the port is now free
            if lsof -i:$port >/dev/null 2>&1; then
                log_error "Failed to kill process on port $port"
                return 1
            else
                log_success "Successfully killed process on port $port"
            fi
        fi
    fi
    return 0
}

# Start reflector server in background
start_reflector_server() {
    log "Starting reflector server in background..."
    
    # Get port configuration
    get_port_config
    
    # Validate port configuration
    if ! validate_port_config; then
        log_error "Port configuration validation failed"
        exit 1
    fi
    
    # Use configured ports
    local port=$REFLECTOR_PORT
    local peer_port=$REFLECTOR_PEER_PORT
    
    # Kill processes using the ports if they're in use
    if ! kill_process_on_port "$port" "reflector"; then
        log_error "Failed to free up port $port"
        exit 1
    fi
    
    if ! kill_process_on_port "$peer_port" "peer"; then
        log_error "Failed to free up port $peer_port"
        exit 1
    fi
    
    log "Using ports: $port (reflector), $peer_port (peer)"
    
    # Start reflector server in background
    cd "$SCRIPT_DIR"
    nohup go run ./cmd/reflector -port="$port" -peer-port="$peer_port" -log-level=info \
        > "$REFLECTOR_LOG_FILE" 2>&1 &
    
    local pid=$!
    echo "$pid" > "$PID_FILE"
    
    log "Reflector server started with PID: $pid"
    
    # Wait a moment for server to start
    sleep 3
    
    # Check if server is still running
    if ! kill -0 "$pid" 2>/dev/null; then
        log_error "Reflector server failed to start"
        exit 1
    fi
    
    log_success "Reflector server is running"
}

# Run stream uploader
run_stream_uploader() {
    log "Running stream uploader..."
    
    cd "$SCRIPT_DIR"

    # Run stream uploader and capture output
    local reflector_address="127.0.0.1:$REFLECTOR_PORT"
    log "Using reflector address: $reflector_address"
    log "Using portal domain: $PORTAL"
    if PORTAL="$PORTAL" LOG_LEVEL=info go run ./cmd/stream-uploader -reflector="$reflector_address" -state-file="$STATE_FILE" 2>&1 | tee -a "$UPLOADER_LOG_FILE"; then
        log_success "Stream uploader completed successfully"
    else
        log_error "Stream uploader failed"
        exit 1
    fi
}

# Extract blob information from state file
extract_blob_info() {
    log "Reading blob information from state file..."
    local state_content
    state_content=$(load_json_state "$STATE_FILE" "$REFLECTOR_LOG_FILE")
    if [[ $? -eq 0 ]]; then
        echo "$state_content" | tee -a "$REFLECTOR_LOG_FILE"
    else
        log_error "Failed to load state file: $STATE_FILE" "$REFLECTOR_LOG_FILE"
        return 1
    fi
}

# Verify blobs using lbry-cli
verify_blobs() {
    log "Verifying blobs using lbry-cli..."

    # Verify SD blob
    local state_path
    state_path=$(get_state_path "$STATE_FILE")
    local sd_hash
    sd_hash=$(jq -r '.sd_blob_hash' "$state_path")
    if [ -n "$sd_hash" ] && [ "$sd_hash" != "null" ]; then
        log "Verifying SD blob: $sd_hash"
        if lbry-cli blob get "$sd_hash"; then
            log_success "SD blob verified: $sd_hash"
        else
            log_error "Failed to verify SD blob: $sd_hash"
        fi
    fi
    
    # Verify content blobs
    mapfile -t content_hashes < <(jq -r '.content_blob_hashes[]?' "$state_path")
    for hash in "${content_hashes[@]}"; do
        if [ -n "$hash" ] && [ "$hash" != "null" ]; then
            log "Verifying content blob: $hash"
            if lbry-cli blob get "$hash"; then
                log_success "Content blob verified: $hash"
            else
                log_error "Failed to verify content blob: $hash"
            fi
        fi
    done
}

# Reflect blobs to external reflector
reflect_blobs() {
    log "Reflecting blobs to external reflector..."
    
    # Reflect SD blob
    local state_path
    state_path=$(get_state_path "$STATE_FILE")
    local sd_hash
    sd_hash=$(jq -r '.sd_blob_hash' "$state_path")
    if [ -n "$sd_hash" ] && [ "$sd_hash" != "null" ]; then
        log "Reflecting SD blob: $sd_hash"
        if lbry-cli blob reflect "$sd_hash"; then
            log_success "SD blob reflected: $sd_hash"
        else
            log_error "Failed to reflect SD blob: $sd_hash"
        fi
    fi
    
    # Reflect content blobs
    mapfile -t content_hashes < <(jq -r '.content_blob_hashes[]?' "$state_path")
    for hash in "${content_hashes[@]}"; do
        if [ -n "$hash" ] && [ "$hash" != "null" ]; then
            log "Reflecting content blob: $hash"
            if lbry-cli blob reflect "$hash"; then
                log_success "Content blob reflected: $hash"
            else
                log_error "Failed to reflect content blob: $hash"
            fi
        fi
    done
}

# Wait for operations to complete
wait_for_operations() {
    log "Waiting for account operations to complete..."
    
    cd "$SCRIPT_DIR"
    
    # Run stream uploader in wait mode
    local reflector_address="localhost:$REFLECTOR_PORT"
    log "Using reflector address: $reflector_address"
    log "Using portal domain: $PORTAL"
    if PORTAL="$PORTAL" LOG_LEVEL=info go run ./cmd/stream-uploader -reflector="$reflector_address" -state-file="$STATE_FILE" -wait-mode 2>&1 | tee -a "$UPLOADER_LOG_FILE"; then
        log_success "Wait operations completed successfully"
    else
        log_error "Wait operations failed"
        exit 1
    fi
}

# Main execution
main() {
    log "Starting reflector system..."
    log "Script directory: $SCRIPT_DIR"
    log "Project root: $PROJECT_ROOT"
    log "State file: $STATE_FILE"
    log "Reflector log file: $REFLECTOR_LOG_FILE"
    log "Uploader log file: $UPLOADER_LOG_FILE"
    
    # Display environment variable configuration
    log "Environment Variables:"
    log "  PORTAL: ${PORTAL:-pinner.xyz (default)}"
    log "  LOG_LEVEL: ${LOG_LEVEL:-info (default)}"
    log "  LOG_FILE: ${LOG_FILE:-$SCRIPT_DIR/reflector.log (default)}"
    log "  REFLECTOR_PORT: ${REFLECTOR_PORT:-5669 (default)}"
    log "  REFLECTOR_PEER_PORT: ${REFLECTOR_PEER_PORT:-5570 (default)}"
    log "  REFLECTOR_SERVER: ${REFLECTOR_SERVER:-lbry.pinner.xyz:5566 (default)}"
    
    # Initialize
    check_dependencies
    
    # Clean up and restart services
    cleanup_and_restart_services
    start_reflector_server
    
    # Upload and process
    run_stream_uploader
    extract_blob_info
    verify_blobs
    reflect_blobs
    wait_for_operations
    
    log_success "Reflector system completed successfully!"
    log "Final state:"
    local state_path
    state_path=$(get_state_path "$STATE_FILE")
    if [[ -f "$state_path" ]]; then
        cat "$state_path" | tee -a "$REFLECTOR_LOG_FILE"
    else
        log_warning "State file not found: $state_path" "$REFLECTOR_LOG_FILE"
    fi
}

# Run main function
main "$@"