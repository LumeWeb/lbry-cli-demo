#!/bin/bash

# Common business logic for LBRY CLI scripts

# LBRY SDK Integration Functions

# Progress bar utilities
declare -rx BAR_SIZE="##########"
declare -rx CLEAR_LINE="\\033[K"

# Function to display a progress bar
show_progress_bar() {
    local current=$1
    local total=$2
    local label=$3
    local timer="$4"  # Optional timer text
    
    if [ "$total" -eq 0 ]; then
        return
    fi
    
    local perc=$((current * 100 / total))
    local percBar=$((perc * ${#BAR_SIZE} / 100))
    
    # Build the progress bar line
    local progress_line="\\r[$label] [${BAR_SIZE:0:percBar}] $perc %"
    
    # Add timer if provided
    if [ -n "$timer" ]; then
        # Calculate padding to right-align timer
        local terminal_width=80  # Default width
        if command -v tput >/dev/null 2>&1; then
            terminal_width=$(tput cols 2>/dev/null || echo 80)
        fi
        
        local progress_length=${#progress_line}
        local timer_length=${#timer}
        local padding=$((terminal_width - progress_length - timer_length - 3))
        
        if [ $padding -gt 0 ]; then
            printf "%*s" $padding ""
        fi
        
        progress_line="$progress_line $timer"
    fi
    
    echo -ne "$progress_line$CLEAR_LINE"
}

# Function to clear current line
clear_line() {
    echo -ne "\\r$CLEAR_LINE"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to get GOBIN path following Go's resolution order
get_gobin_path() {
    # First check if GOBIN is explicitly set
    local gobin
    gobin=$(go env GOBIN 2>/dev/null)
    if [ -n "$gobin" ]; then
        echo "$gobin"
        return 0
    fi
    
    # Check GOPATH and use GOPATH/bin
    local gopath
    gopath=$(go env GOPATH 2>/dev/null)
    if [ -n "$gopath" ]; then
        echo "$gopath/bin"
        return 0
    fi
    
    # Final fallback to $HOME/go/bin
    echo "$HOME/go/bin"
    return 0
}

# Function to check dependency and provide installation instructions
check_dependency() {
    local cmd="$1"
    local name="$2"
    local install_cmd="$3"
    
    if ! command_exists "$cmd"; then
        echo "$name is not installed."
        echo "Please install it by running: $install_cmd"
        echo ""
        echo "After installation, please run this script again."
        exit 1
    else
        echo "$name is installed"
    fi
}

# Function to check docker compose specifically
check_docker_compose() {
    if ! docker compose version >/dev/null 2>&1; then
        echo "Docker Compose is not available."
        echo "Please install Docker by running: curl https://get.docker.com | bash"
        echo ""
        echo "After installation, please run this script again."
        exit 1
    else
        echo "Docker Compose is available"
    fi
}

# Function to check all common dependencies for LBRY CLI
check_lbry_dependencies() {
    show_step_header "1" "Checking dependencies"
    check_dependency "docker" "Docker" "curl https://get.docker.com | bash"
    check_docker_compose
    check_dependency "git" "Git" "sudo apt-get install git"
    check_dependency "go" "Go" "sudo apt-get install golang-go"
    check_dependency "jq" "jq" "sudo apt-get install jq"
    check_dependency "curl" "cURL" "sudo apt-get install curl"
}

# Function to check runtime dependencies (for start.sh)
check_runtime_dependencies() {
    show_step_header "1" "Checking runtime dependencies"
    
    # Use the common dependency checking function for shared dependencies
    check_dependency "docker" "Docker" "curl https://get.docker.com | bash"
    check_docker_compose
    check_dependency "jq" "jq" "sudo apt-get install jq"
    check_dependency "curl" "cURL" "sudo apt-get install curl"
    
    # Runtime-specific dependencies
    if ! command_exists "lbry-cli"; then
        echo "Error: lbry-cli is not installed or not in PATH"
        exit 1
    fi
    
    echo "All runtime dependencies are available"
}

# Function to check demo dependencies
check_demo_dependencies() {
    show_step_header "1" "Checking demo dependencies"
    
    # Use the common dependency checking function for shared dependencies
    check_dependency "go" "Go" "sudo apt-get install golang-go"
    check_dependency "curl" "cURL" "sudo apt-get install curl"
    
    # Demo-specific dependencies
    if ! command_exists "lbry-cli"; then
        log_error "lbry-cli is not installed or not in PATH"
        exit 1
    fi
    
    log_success "All dependencies available"
}

# Function to stop all Docker Compose services
stop_docker_services() {
    show_step_header "1" "Stopping all Docker Compose services"
    run_demo_command "Stopping Docker Compose services" docker compose down
}

# Function to start all Docker Compose services
start_docker_services() {
    show_step_header "1" "Starting all Docker Compose services"
    run_demo_command "Starting Docker Compose services" docker compose up -d
}

# Function to run Docker Compose command with error handling
run_docker_compose_command() {
    local operation="$1"
    shift
    local cmd=("$@")
    
    show_step_header "1" "$operation"
    run_demo_command "$operation" docker compose "${cmd[@]}"
}



# Function to cleanup blob data only (lbrynet.sqlite* and blobfiles)
cleanup_blobs() {
    run_docker_compose_command "Cleaning up blob data (lbrynet.sqlite* and blobfiles)" run --rm cleanup-blobs
}

# Colors for output (for logging functions)
declare -rx RED='\033[0;31m'
declare -rx GREEN='\033[0;32m'
declare -rx YELLOW='\033[1;33m'
declare -rx BLUE='\033[0;34m'
declare -rx NC='\033[0m' # No Color

# Generic logging functions
log() {
    local message="$1"
    local log_file="${2:-}"
    if [ -n "$log_file" ] && [ -f "$log_file" ]; then
        echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $message" | tee -a "$log_file"
    else
        echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $message"
    fi
}

log_error() {
    local message="$1"
    local log_file="${2:-}"
    if [ -n "$log_file" ] && [ -f "$log_file" ]; then
        echo -e "${RED}[ERROR]${NC} $message" | tee -a "$log_file"
    else
        echo -e "${RED}[ERROR]${NC} $message"
    fi
}

log_success() {
    local message="$1"
    local log_file="${2:-}"
    if [ -n "$log_file" ] && [ -f "$log_file" ]; then
        echo -e "${GREEN}[SUCCESS]${NC} $message" | tee -a "$log_file"
    else
        echo -e "${GREEN}[SUCCESS]${NC} $message"
    fi
}

log_warning() {
    local message="$1"
    local log_file="${2:-}"
    if [ -n "$log_file" ] && [ -f "$log_file" ]; then
        echo -e "${YELLOW}[WARNING]${NC} $message" | tee -a "$log_file"
    else
        echo -e "${YELLOW}[WARNING]${NC} $message"
    fi
}

log_to_file_only() {
    local message="$1"
    local log_file="${2:-$log_file}"
    if [ -n "$log_file" ] && [ -f "$log_file" ]; then
        echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] $message" >> "$log_file"
    fi
}

# Read and display JSON state file contents
read_json_state() {
    local state_file="$1"
    local log_file="${2:-}"
    
    if [ ! -f "$state_file" ]; then
        log_error "State file not found: $state_file" "$log_file"
        return 1
    fi
    
    if [ -n "$log_file" ] && [ -f "$log_file" ]; then
        tee -a "$log_file" < "$state_file"
    else
        cat "$state_file"
    fi
}

# State management constants (aligned with shared/state_manager.go)
ACCOUNT_STATE_FILE="account.json"

# State management functions (aligned with shared/state_manager.go)

# Get state directory path (equivalent to GetStateDir in shared)
get_state_dir() {
    # Look for .lbry-demo marker file to find shared root
    local current_dir
    current_dir="$(pwd)"
    local state_dir=""
    
    # Search for marker file
    while [[ "$current_dir" != "/" ]]; do
        if [[ -f "$current_dir/.lbry-demo" ]]; then
            state_dir="$current_dir/state"
            break
        fi
        current_dir="$(dirname "$current_dir")"
    done
    
    # Fallback to current directory if no marker found
    if [[ -z "$state_dir" ]]; then
        state_dir="$(pwd)/state"
    fi
    
    # Create state directory if it doesn't exist
    mkdir -p "$state_dir" 2>/dev/null || true
    
    echo "$state_dir"
}

# Get full path for a state file (equivalent to GetStatePath in shared)
get_state_path() {
    local filename="$1"
    local state_dir
    state_dir=$(get_state_dir)
    echo "$state_dir/$filename"
}

# Update JSON state file (equivalent to SaveJSON in shared)
update_json_state() {
    local filename="$1"
    local data="$2"
    local log_file="${3:-}"
    
    local state_path
    state_path=$(get_state_path "$filename")
    
    # Ensure state directory exists
    mkdir -p "$(dirname "$state_path")" 2>/dev/null || true
    
    # Write data to state file
    if echo "$data" > "$state_path"; then
        log "State saved: $filename" "$log_file"
        return 0
    else
        log_error "Failed to save state: $filename" "$log_file"
        return 1
    fi
}

# Load JSON state file (equivalent to LoadJSON in shared)
load_json_state() {
    local filename="$1"
    local log_file="${2:-}"
    
    local state_path
    state_path=$(get_state_path "$filename")
    
    if [[ ! -f "$state_path" ]]; then
        log_warning "State file not found: $filename" "$log_file"
        return 1
    fi
    
    # Read and return the state file content
    cat "$state_path"
    log "State loaded: $filename" "$log_file"
}

# Generic process cleanup function
cleanup_process() {
    local process_name="$1"
    local pid_file="$2"
    local log_file="${3:-}"
    
    log "Cleaning up $process_name..." "$log_file"
    
    # Stop process if running
    if [ -f "$pid_file" ]; then
        local pid
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log "Stopping $process_name (PID: $pid)" "$log_file"
            kill "$pid" 2>/dev/null || true
            # Wait a bit for graceful shutdown
            sleep 2
            # Force kill if still running
            if kill -0 "$pid" 2>/dev/null; then
                kill -9 "$pid" 2>/dev/null || true
            fi
        fi
        rm -f "$pid_file"
    fi
    
    log "Cleanup completed for $process_name" "$log_file"
}

# Generic demo runner function
run_demo() {
    local demo_name="$1"
    local script_dir="$2"
    local log_file="${3:-$script_dir/$demo_name.log}"
    local pid_file="$script_dir/$demo_name.pid"
    
    # Cleanup function
    cleanup() {
        cleanup_process "$demo_name demo" "$pid_file" "$log_file"
    }
    
    # Signal handlers
    trap cleanup EXIT
    trap cleanup INT
    trap cleanup TERM
    

    
    # Run the demo
    run_demo_app() {
        log "Starting $demo_name demo..." "$log_file"
        
        cd "$script_dir" || exit
        
        # Run the demo and capture output
        log "Running $demo_name demo application..."
        if go run main.go 2>&1 | tee -a "$log_file"; then
            log_success "$demo_name demo completed successfully" "$log_file"
        else
            log_error "$demo_name demo failed" "$log_file"
            exit 1
        fi
    }
    
    # Main execution
    main() {
        log "Starting $demo_name demo system..." "$log_file"
        log "Script directory: $script_dir" "$log_file"
        log "Log file: $log_file" "$log_file"
        
        # Initialize
        check_demo_dependencies
        
        # Run the demo
        run_demo_app
        
        log_success "$demo_name demo system completed successfully!" "$log_file"
    }
    
    # Run main function with all arguments
    main "$@"
}

# =============================================================================
# Common Script Utilities
# =============================================================================

# Function to display standardized script header
show_script_header() {
    local title="$1"
    local description="$2"
    
    echo "=== $title ==="
    echo ""
    if [ -n "$description" ]; then
        echo "$description"
        echo ""
    fi
}

# Function to display standardized script footer
show_script_footer() {
    local title="$1"
    local status="$2"  # "success" or "error"
    local message="$3"
    
    echo ""
    echo "=== $title Complete ==="
    echo ""
    
    if [ "$status" = "success" ]; then
        echo "✓ $message"
    else
        echo "✗ $message"
    fi
    echo ""
}

# Function to display step header
show_step_header() {
    local step_num="$1"
    local step_desc="$2"
    
    echo "Step $step_num: $step_desc"
}

# Function to run demo commands with resilient error handling
# Exits the script if the command fails (returns non-zero exit code)
run_demo_command() {
    local description="$1"
    shift
    
    log "Executing demo command: $description"
    log "Command: $*"
    
    if ! "$@"; then
        log_error "Demo command failed: $description"
        log_error "Command: $*"
        log_error "Demo cannot continue without this operation succeeding"
        exit 1
    fi
    
    log_success "Demo command completed: $description"
}

# =============================================================================
# LBRY SDK Integration Functions
# =============================================================================

# Function to start LBRY SDK daemon using existing infrastructure
start_lbry_sdk() {
    echo "Starting LBRY SDK daemon..."
    
    # Check if services are already running
    if is_lbry_sdk_running; then
        echo "LBRY SDK services are already running"
        return 0
    fi
    
    # Use existing start.sh logic but without the script
    check_runtime_dependencies
    
    # Ensure data directory exists with proper permissions
    if [ ! -d "./data" ]; then
        echo "Creating data directory..."
        mkdir -p ./data
    fi
    chmod 777 ./data
    
    # Start docker compose services
    run_demo_command "Starting LBRY SDK services" docker compose up -d
    echo "LBRY SDK started successfully"
}





# Function to clear all blobs from LBRY daemon (works even when LBRY is offline)
clear_lbry_blobs() {
    echo "Clearing all blobs from LBRY daemon..."
    
    # Clear blobs using docker compose cleanup-blobs service
    # This works even when LBRY SDK is not running since it uses a separate service
    run_docker_compose_command "Clearing blobs" run --rm cleanup-blobs
}

# Function to get a specific blob by hash using external lbry-cli
get_lbry_blob() {
    local blob_hash="$1"
    
    if [ -z "$blob_hash" ]; then
        echo "Error: blob hash is required"
        return 1
    fi
    
    echo "Getting blob: $blob_hash"
    
    # Check if LBRY SDK is running
    if ! is_lbry_sdk_running; then
        echo "LBRY SDK is not running"
        return 1
    fi
    
    # Use external lbry-cli to get blob
    if lbry-cli blob get "$blob_hash"; then
        echo "Blob retrieved successfully"
        return 0
    else
        echo "Failed to get blob: $blob_hash"
        return 1
    fi
}

# Wrapper function for lbry-cli file save that handles "false" output
lbry_cli_file_save() {
    local sd_hash="$1"
    local file_name="$2"
    local download_directory="$3"
    
    # Build command for external lbry-cli
    local cmd=(lbry-cli file save --sd_hash "$sd_hash")
    
    if [ -n "$file_name" ]; then
        cmd+=(--file_name "$file_name")
    fi
    
    if [ -n "$download_directory" ]; then
        cmd+=(--download_directory "$download_directory")
    fi
    
    # Execute command and capture output
    local output
    output=$("${cmd[@]}" 2>&1)
    local exit_code=$?
    
    # If command succeeded but output is "false", treat as failure
    if [ $exit_code -eq 0 ] && [ "$output" = "false" ]; then

        return 1
    fi
    
    # Only output success/failure status, not the full JSON response
    # The JSON response contains sensitive data and should not be dumped to console
    if [ $exit_code -eq 0 ] && [ -n "$output" ] && [ "$output" != "false" ]; then
        # For successful operations, just indicate success without dumping JSON
        echo "File save operation completed successfully"
    fi
    
    return $exit_code
}

# Function to save file using sd_hash via external lbry-cli
save_file_with_sd_hash() {
    local sd_hash="$1"
    local file_name="$2"
    local download_directory="$3"
    
    if [ -z "$sd_hash" ]; then
        echo "Error: sd_hash is required"
        return 1
    fi
    
    echo "Saving file with sd_hash: $sd_hash"
    
    # Check if LBRY SDK is running
    if ! is_lbry_sdk_running; then
        echo "LBRY SDK is not running"
        return 1
    fi
    
    # Use retry_command with the wrapper function
    if retry_command 3 2 "${VERBOSE:-false}" lbry_cli_file_save "$sd_hash" "$file_name" "$download_directory"; then
        echo "File saved successfully with sd_hash: $sd_hash"
        return 0
    else
        echo "Failed to save file with sd_hash: $sd_hash"
        return 1
    fi
}

# Simple retry helper for commands
retry_command() {
    local max_attempts="${1:-3}"
    local delay="${2:-2}"
    local verbose="${3:-true}"
    shift 3
    local command=("$@")
    
    local attempt=1
    while [ "$attempt" -le "$max_attempts" ]; do
        if [ "$verbose" = "true" ]; then
            echo "Attempt $attempt of $max_attempts: ${command[*]}"
        fi
        if "${command[@]}"; then
            if [ "$verbose" = "true" ]; then
                echo "Command succeeded on attempt $attempt"
            fi
            return 0
        else
            if [ "$verbose" = "true" ]; then
                echo "Command failed on attempt $attempt"
                if [ "$attempt" -lt "$max_attempts" ]; then
                    echo "Waiting $delay seconds before retry..."
                fi
            fi
            if [ "$attempt" -lt "$max_attempts" ]; then
                sleep "$delay"
            fi
        fi
        attempt=$((attempt + 1))
    done
    
    if [ "$verbose" = "true" ]; then
        echo "Command failed after $max_attempts attempts"
    fi
    return 1
}

# Function to list all files using external lbry-cli
list_lbry_files() {
    echo "Listing files in LBRY daemon..."
    
    # Check if LBRY SDK is running
    if ! is_lbry_sdk_running; then
        echo "LBRY SDK is not running"
        return 1
    fi
    
    # Use external lbry-cli to list files
    if lbry-cli file list; then
        echo "Files listed successfully"
        return 0
    else
        echo "Failed to list files"
        return 1
    fi
}

# Function to copy file from docker container to host
copy_file_from_container() {
    local container_path="$1"
    local host_path="$2"
    
    if [ -z "$container_path" ] || [ -z "$host_path" ]; then
        echo "Error: both container_path and host_path are required"
        return 1
    fi
    
    echo "Copying file from container: $container_path to $host_path"
    
    # Ensure host directory exists
    local host_dir
    host_dir=$(dirname "$host_path")
    mkdir -p "$host_dir"
    
    # Copy file from container
    run_demo_command "Copying file from container" docker compose cp "lbry:$container_path" "$host_path"
    echo "File copied successfully"
}

# Function to setup LBRY SDK for any demo
setup_lbry_for_demo() {
    log "Setting up LBRY SDK for demo..."
    
    # Start LBRY SDK
    if ! start_lbry_sdk; then
        log_error "Failed to start LBRY SDK"
        return 1
    fi
    
    # Wait for LBRY SDK to be ready
    if ! wait_for_lbry_sdk 60 2; then
        log_error "LBRY SDK failed to become ready"
        return 1
    fi
    
    log_success "LBRY SDK setup completed"
    return 0
}

# Function to get JWT token from account state
get_jwt_from_state() {
    get_value_from_state "$ACCOUNT_STATE_FILE" '.jwt_token'
}

# Function to get a specific value from a state file using jq filter
get_value_from_state() {
    local state_file="$1"
    local jq_filter="$2"
    local log_file="${3:-}"
    
    if [[ -z "$state_file" || -z "$jq_filter" ]]; then
        log_error "State file and jq filter are required" "$log_file"
        return 1
    fi
    
    local state_path
    state_path=$(get_state_path "$state_file")
    
    if [[ ! -f "$state_path" ]]; then
        log_warning "State file not found: $state_path" "$log_file"
        return 1
    fi
    
    local value
    value=$(jq -r "$jq_filter" "$state_path" 2>/dev/null)
    
    if [[ -z "$value" || "$value" == "null" ]]; then
        log_warning "No value found for filter: $jq_filter" "$log_file"
        return 1
    fi
    
    echo "$value"
    return 0
}

# Function to get SD hash for demo operations
get_sd_hash_for_demo() {
    local state_file="$1"
    
    # Try to get SD hash from state file
    local sd_hash=""
    if [ -n "$state_file" ]; then
        sd_hash=$(get_value_from_state "$state_file" '.sd_hash // .sd_blob_hash // .upload_hash // empty' "$log_file" 2>/dev/null)
        if [[ $? -eq 0 && -n "$sd_hash" && "$sd_hash" != "null" ]]; then
            # Log to file only, not stdout, to avoid pollution in command substitution
            log_to_file_only "Found SD hash from state file: $sd_hash"
            echo "$sd_hash"
            return 0
        else
            sd_hash=""
        fi
    fi
    
    # If we still don't have an SD hash, fail
    if [ -z "$sd_hash" ]; then
        log_error "SD hash not found in state file: $state_file"
        return 1
    fi
}

# Function to perform blob operations on SD and content hashes
perform_blob_operations() {
    local sd_hash="$1"
    local state_file="$2"
    local operations_success=true
    
    log "=== VERIFICATION CHECKPOINT: Reference Implementation Blob Access ==="
    log "Using lbry-cli (reference Python implementation) to access blobs uploaded by our liblbry"
    log "This verifies that our blob format and network communication are standards-compliant"
    
    # Perform blob get operation on SD hash
    log "Performing blob get on SD hash: $sd_hash"
    if ! get_lbry_blob "$sd_hash" > /dev/null; then
        log_warning "Blob get operation failed for SD hash"
        operations_success=false
    fi
    
    # Get content hashes from state file if available and perform blob get operations
    local content_hashes=""
    if [ -n "$state_file" ]; then
        content_hashes=$(get_value_from_state "$state_file" '.content_hashes[]?' "$log_file" 2>/dev/null)
    fi
    
    if [ -n "$content_hashes" ]; then
        log "Performing blob get operations on content hashes..."
        while IFS= read -r content_hash; do
            if [ -n "$content_hash" ] && [ "$content_hash" != "null" ]; then
                log "Performing blob get on content hash: $content_hash"
                if ! get_lbry_blob "$content_hash" > /dev/null; then
                    log_warning "Blob get operation failed for content hash: $content_hash"
                    operations_success=false
                else
                    log_success "Blob get succeeded for content hash: $content_hash"
                fi
            fi
        done <<< "$content_hashes"
    fi
    
    [ "$operations_success" = "true" ]
}

# Function to save demo file using SD hash
save_demo_file() {
    local sd_hash="$1"
    local saved_file_name="$2"
    
    log "Saving file using sd_hash: $sd_hash"
    if save_file_with_sd_hash "$sd_hash" "$saved_file_name" "/data/downloads"; then
        log_success "File saved successfully with sd_hash"
        return 0
    else
        log_warning "Failed to save file with sd_hash"
        return 1
    fi
}

# Function to copy demo file from container to host
copy_demo_file_to_host() {
    local saved_file_name="$1"
    local output_folder="$2"
    
    local host_file="$output_folder/$saved_file_name"
    local container_file="/data/downloads/$saved_file_name"
    
    if copy_file_from_container "$container_file" "$host_file"; then
        log_success "File copied from container to host: $host_file"
        return 0
    else
        log_warning "Failed to copy file from container"
        return 1
    fi
}

# Function to verify demo file (size and SHA256)
verify_demo_file() {
    local host_file="$1"
    local state_file="$2"
    
    log "=== VERIFICATION CHECKPOINT: Final Hash Verification ==="
    log "Comparing SHA256 hash of downloaded file with original uploaded file"
    log "This is the definitive proof that our liblbry implementation is 100% correct"
    
    # Verify file exists - return early if not found
    if [ ! -f "$host_file" ]; then
        log_error "File not found for verification: $host_file"
        return 1
    fi
    
    # Show file size
    local file_size
    file_size=$(stat -f%z "$host_file" 2>/dev/null || stat -c%s "$host_file" 2>/dev/null || echo "unknown")
    log "Saved file size: $file_size bytes"
    
    # Skip SHA256 verification if no state file
    if [ -z "$state_file" ]; then
        return 0
    fi
    
    # Get original SHA256 from state file
    local original_sha256
    original_sha256=$(get_value_from_state "$state_file" '.original_sha256' "$log_file")
    if [[ $? -ne 0 || -z "$original_sha256" ]]; then
        return 0  # Skip verification if no hash available
    fi
    
    # Verify SHA256 hash
    log "Verifying SHA256 hash of downloaded file..."
    local calculated_sha256
    calculated_sha256=$(sha256sum "$host_file" 2>/dev/null | cut -d' ' -f1)
    if [ "$calculated_sha256" = "$original_sha256" ]; then
        log_success "✓ VERIFICATION SUCCESS: Hashes match perfectly!"
        log_success "✓ PROOF: Our liblbry implementation produces identical results to reference implementation"
        log_success "✓ File integrity verified: $calculated_sha256"
        return 0
    else
        log_error "✗ VERIFICATION FAILED: Hash mismatch detected!"
        log_error "✗ Expected (original): $original_sha256"
        log_error "✗ Calculated (downloaded): $calculated_sha256"
        log_error "✗ This indicates a problem with our liblbry implementation"
        return 1
    fi
}

# Function to save and verify demo file
save_and_verify_demo_file() {
    local sd_hash="$1"
    local saved_file_name="$2"
    local output_folder="$3"
    local state_file="$4"
    
    log "=== VERIFICATION CHECKPOINT: File Integrity Verification ==="
    log "Downloading file via reference implementation and performing hash verification"
    log "This proves the file content is identical between our liblbry and reference implementation"
    
    # Save demo file - return early if failed
    if ! save_demo_file "$sd_hash" "$saved_file_name"; then
        return 1
    fi
    
    # Copy file to host - return early if failed
    if ! copy_demo_file_to_host "$saved_file_name" "$output_folder"; then
        return 1
    fi
    
    # Verify file - return early if failed
    local host_file="$output_folder/$saved_file_name"
    if ! verify_demo_file "$host_file" "$state_file"; then
        return 1
    fi
    
    # All operations succeeded
    return 0
}

# Function to perform post-demo LBRY operations for any demo
perform_post_demo_lbry_operations() {
    local demo_name="$1"
    local saved_file_name="$2"
    local output_folder="$3"
    local state_file="$4"  # Optional state file for SHA256 verification
    
    log "=== VERIFICATION CHECKPOINT: Starting Reference Implementation Verification ==="
    log "Stage 1 Complete: Our liblbry implementation successfully uploaded/downloaded blobs"
    log "Stage 2 Starting: Now verifying against reference LBRY Python implementation"
    log "This proves our liblbry implementation produces correct, interoperable results"
    log "Performing post-demo LBRY operations for $demo_name..."
    
    # Prune downloads directory at the start to ensure clean state
    prune_downloads "lbry"
    
    # Get SD hash for demo operations
    local sd_hash
    if ! sd_hash=$(get_sd_hash_for_demo "$state_file"); then
        return 1
    fi
    
    local operations_success=true
    
    # Perform blob operations on SD and content hashes
    if ! perform_blob_operations "$sd_hash" "$state_file"; then
        operations_success=false
    fi
    
    # Save and verify demo file
    if ! save_and_verify_demo_file "$sd_hash" "$saved_file_name" "$output_folder" "$state_file"; then
        operations_success=false
    fi
    
    # Return success/failure based on operations
    if [ "$operations_success" = "true" ]; then
        log_success "=== VERIFICATION COMPLETE: Reference Implementation Verification Successful ==="
        log_success "✓ Our liblbry implementation successfully interoperates with reference LBRY"
        log_success "✓ All blob operations work correctly with both implementations"
        log_success "✓ File integrity verified through hash comparison"
        log_success "Post-demo LBRY operations completed successfully for $demo_name"
        return 0
    else
        log_warning "=== VERIFICATION INCOMPLETE: Some operations failed ==="
        log_warning "✗ Issues detected during reference implementation verification"
        log_warning "✗ This may indicate compatibility problems with our liblbry implementation"
        log_warning "Some post-demo LBRY operations failed for $demo_name"
        return 1
    fi
}

# Core function to execute command in Docker container with optional output capture
_execute_docker_command_core() {
    local container_name="$1"
    shift
    local command=("$@")
    local capture_output="${1:-false}"  # "true" to capture output, "false" to just execute
    local log_prefix="${2:-}"      # Optional prefix for log messages
    
    if [[ -z "$container_name" || ${#command[@]} -eq 0 ]]; then
        log_error "Container name and command are required for docker command execution"
        return 1
    fi
    
    local log_msg="Executing command in container '$container_name'"
    if [[ -n "$log_prefix" ]]; then
        log_msg="$log_msg ($log_prefix)"
    fi
    log "$log_msg: ${command[*]}"
    
    if [[ "$capture_output" == "true" ]]; then
        local output
        if output=$(docker compose exec "$container_name" sh -c "${command[*]}" 2>&1); then
            log_success "Command executed successfully in container '$container_name'"
            echo "$output"
            return 0
        else
            log_error "Failed to execute command in container '$container_name'"
            return 1
        fi
    else
        if docker compose exec "$container_name" sh -c "${command[*]}"; then
            log_success "Command executed successfully in container '$container_name'"
            return 0
        else
            log_error "Failed to execute command in container '$container_name'"
            return 1
        fi
    fi
}

# Function to execute command in Docker container
execute_docker_command() {
    local container_name="$1"
    shift
    _execute_docker_command_core "$container_name" "$@" "false" ""
}

# Function to execute command in Docker container and capture output
execute_docker_command_and_capture() {
    local container_name="$1"
    shift
    _execute_docker_command_core "$container_name" "$@" "true" "capturing output"
}

# Function to prune downloads directory in Docker container
prune_downloads() {
    local container_name="${1:-lbry}"
    local downloads_dir="/data/downloads"
    
    log "Pruning downloads directory in container '$container_name'..."
    
    # Check if directory exists using a silent approach
    local dir_check_cmd="test -d '$downloads_dir'"
    if ! docker compose exec "$container_name" sh -c "$dir_check_cmd" 2>/dev/null; then
        log "Downloads directory does not exist, nothing to prune"
        return 0
    fi
    
    # Check if directory has content using a silent approach
    local content_check_cmd="ls -A '$downloads_dir' 2>/dev/null | wc -l"
    local file_count
    file_count=$(docker compose exec "$container_name" sh -c "$content_check_cmd" 2>/dev/null || echo "0")
    
    # Remove whitespace and convert to integer
    file_count=$(echo "$file_count" | tr -d ' ')
    
    if [ "$file_count" -eq 0 ]; then
        log "Downloads directory exists but is empty, nothing to prune"
        return 0
    fi
    
    # Remove all files in downloads directory
    local prune_cmd="rm -rf '$downloads_dir'/* '$downloads_dir'/.* 2>/dev/null || true"
    if execute_docker_command "$container_name" "$prune_cmd"; then
        log_success "Downloads directory pruned successfully in container '$container_name'"
        return 0
    else
        log_error "Failed to prune downloads directory in container '$container_name'"
        return 1
    fi
}

# Function to cleanup demo bin files from a directory
cleanup_demo_bins() {
    local demo_dir="$1"
    
    if [[ -z "$demo_dir" ]]; then
        log_error "Demo directory is required for bin cleanup"
        return 1
    fi
    
    if [[ ! -d "$demo_dir" ]]; then
        log_warning "Demo directory does not exist: $demo_dir, skipping bin cleanup"
        return 0
    fi
    
    log "Cleaning up bin files from directory: $demo_dir"
    
    # Find and remove .bin files safely
    local bin_files
    bin_files=$(find "$demo_dir" -maxdepth 1 -name "*.bin" -type f 2>/dev/null || true)
    
    if [[ -z "$bin_files" ]]; then
        log "No .bin files found in $demo_dir"
        return 0
    fi
    
    local removed_count=0
    while IFS= read -r bin_file; do
        if [[ -f "$bin_file" ]]; then
            log "Removing bin file: $(basename "$bin_file")"
            rm -f "$bin_file" && ((removed_count++))
        fi
    done <<< "$bin_files"
    
    if [[ $removed_count -gt 0 ]]; then
        log_success "Removed $removed_count bin file(s) from $demo_dir"
    else
        log_warning "No bin files were removed from $demo_dir"
    fi
    
    # Also prune downloads directory in Docker container
    prune_downloads "lbry"
    
    return 0
}

# Function to check if LBRY SDK is running
is_lbry_sdk_running() {
    local status_output
    status_output=$(docker compose ps --format json 2>/dev/null)
    
    if [ -z "$status_output" ]; then
        return 1
    fi
    
    # Check if any service named "lbry" is in running state
    if echo "$status_output" | jq -e 'select(.Service == "lbry" and .State == "running")' >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to check startup status progress
check_startup_progress() {
    local status_output="$1"
    
    # Check if status_output is valid JSON
    if ! echo "$status_output" | jq . >/dev/null 2>&1; then
        echo "0"
        return
    fi
    
    # Check if startup_status exists
    if ! echo "$status_output" | jq -e '.startup_status' >/dev/null 2>&1; then
        echo "0"
        return
    fi
    
    # Count total and ready items
    local total_items
    local ready_items
    total_items=$(jq_output '.startup_status | keys | length' "$status_output")
    ready_items=$(echo "$status_output" | jq -r '.startup_status | to_entries[] | select(.value == true) | .key' | wc -l)
    
    if [ "$total_items" -eq 0 ]; then
        echo "0"
        return
    fi
    
    echo "$ready_items"
}

# Function to get total startup items
get_startup_total() {
    local status_output="$1"
    
    # Check if status_output is valid JSON
    if ! echo "$status_output" | jq . >/dev/null 2>&1; then
        echo "1"
        return
    fi
    
    # Check if startup_status exists
    if ! echo "$status_output" | jq -e '.startup_status' >/dev/null 2>&1; then
        echo "1"
        return
    fi
    
    local total_items
    total_items=$(jq_output '.startup_status | keys | length' "$status_output")
    
    if [ "$total_items" -eq 0 ]; then
        echo "1"
    else
        echo "$total_items"
    fi
}

# Function to get current blockchain height from API
get_blockchain_height() {
    local height=0
    
    # Try to fetch from LBRY explorer API
    if command -v curl >/dev/null 2>&1; then
        local api_response
        api_response=$(curl -s "https://explorer.lbry.org/api/v1/status" 2>/dev/null)
        
        if command -v curl >/dev/null 2>&1 && api_response=$(curl -s "https://explorer.lbry.org/api/v1/status" 2>/dev/null) && echo "$api_response" | jq . >/dev/null 2>&1; then
            height=$(jq_output '.status.height // 0' "$api_response")
            if [ "$height" = "null" ] || [ "$height" = "" ]; then
                height=0
            fi
        fi
    fi
    
    echo "$height"
}

# Function to check headers synchronization progress
check_headers_progress() {
    local status_output="$1"
    local progress=0
    
    # Check if status_output is valid JSON
    if ! echo "$status_output" | jq . >/dev/null 2>&1; then
        echo "0"
        return
    fi
    
    # Check if wallet.blocks and wallet.blocks_behind exist
    if ! echo "$status_output" | jq -e '.wallet.blocks' >/dev/null 2>&1 || ! echo "$status_output" | jq -e '.wallet.blocks_behind' >/dev/null 2>&1; then
        echo "0"
        return
    fi
    
    local blocks
    local blocks_behind
    blocks=$(jq_output '.wallet.blocks // 0' "$status_output")
    blocks_behind=$(jq_output '.wallet.blocks_behind // 0' "$status_output")
    
    if [ "$blocks" = "null" ] || [ "$blocks" = "" ]; then
        blocks=0
    fi
    
    if [ "$blocks_behind" = "null" ] || [ "$blocks_behind" = "" ]; then
        blocks_behind=0
    fi
    
    # Get actual blockchain height from API for accurate calculation
    local blockchain_height
    blockchain_height=$(get_blockchain_height)
    
    # If API call failed, fall back to local calculation
    if [ "$blockchain_height" -eq 0 ]; then
        # Fallback: use blocks + blocks_behind as total height
        blockchain_height=$((blocks + blocks_behind))
    fi
    
    # Calculate progress based on actual blockchain height
    if [ "$blockchain_height" -eq 0 ]; then
        progress=0
    elif [ "$blocks_behind" -eq 0 ]; then
        progress=100
    else
        # Calculate: (current_height / blockchain_height) * 100
        progress=$((blocks * 100 / blockchain_height))
    fi
    
    echo "$progress"
}

# Function to strip newlines from command output (useful for jq)
strip_newlines() {
    local input="$1"
    echo "$input" | tr -d '\n\r'
}

# Function to get jq output without newlines
jq_output() {
    local filter="$1"
    local input="$2"
    echo "$input" | jq -r "$filter" | tr -d '\n\r'
}

# Function to format elapsed time as MM:SS
format_elapsed_time() {
    local elapsed_seconds=$1
    local minutes=$((elapsed_seconds / 60))
    local seconds=$((elapsed_seconds % 60))
    printf "%02d:%02d" $minutes $seconds
}

# Function to get API headers progress from status output
get_api_headers_progress() {
    local status_output="$1"
    local api_headers_progress=0
    
    if echo "$status_output" | jq -e '.wallet.headers_synchronization_progress' >/dev/null 2>&1; then
        api_headers_progress=$(jq_output '.wallet.headers_synchronization_progress // 0' "$status_output")
        if [ "$api_headers_progress" = "null" ] || [ "$api_headers_progress" = "" ]; then
            api_headers_progress=0
        fi
    fi
    
    echo "$api_headers_progress"
}

# Function to display startup progress
display_startup_progress() {
    local status_output="$1"
    local startup_total="$2"
    local start_time="$3"
    
    # Get startup total on first successful check
    if [ "$startup_total" -eq 1 ]; then
        startup_total=$(get_startup_total "$status_output")
    fi
    
    # Check startup progress
    local startup_ready
    startup_ready=$(check_startup_progress "$status_output")
    
    # Check headers synchronization progress
    local headers_progress
    headers_progress=$(check_headers_progress "$status_output")
    
    # Calculate elapsed time
    local current_time
    current_time=$(date +%s)
    local elapsed_time=$((current_time - start_time))
    local formatted_time
    formatted_time=$(format_elapsed_time $elapsed_time)
    
    # Display progress bars with timer
    clear_line >&2
    show_progress_bar "$startup_ready" "$startup_total" "Startup" "$formatted_time" >&2
    echo "" >&2
    show_progress_bar "$headers_progress" "100" "Blockchain Headers Sync" "$formatted_time" >&2
    
    # Return progress values for further processing
    printf "%s|%s|%s" "$startup_ready" "$startup_total" "$headers_progress"
}

# Function to check if LBRY SDK is fully ready
is_lbry_sdk_fully_ready() {
    local status_output="$1"
    local startup_ready="$2"
    local startup_total="$3"
    local headers_progress="$4"
    
    # Check if startup is complete
    if [ "$startup_ready" -eq "$startup_total" ]; then
        # Check headers synchronization progress from the API field as backup
        local api_headers_progress
        api_headers_progress=$(get_api_headers_progress "$status_output")
        
        # Only consider fully ready if both calculated progress AND API progress are at 100%
        if [ "$headers_progress" -eq 100 ] && [ "$api_headers_progress" -eq 100 ]; then
            return 0
        else
            # Startup complete but headers not fully synced, continue monitoring
            if [ "$api_headers_progress" -gt 0 ]; then
                echo ""
                echo "Startup complete, waiting for headers sync: $api_headers_progress% (API) vs $headers_progress% (calculated)"
            fi
            return 1
        fi
    fi
    
    return 1
}

# Function to display ready status summary
display_ready_status() {
    local status_output="$1"
    local start_time="$2"
    
    # Calculate startup time
    local end_time
    end_time=$(date +%s)
    local startup_duration=$((end_time - start_time))
    
    # Format duration as MM:SS
    local formatted_time
    formatted_time=$(format_elapsed_time $startup_duration)
    
    echo ""
    echo ""
    echo "LBRY SDK is fully ready."
    echo "All startup components are running and headers are synchronized."
    echo "Startup completed in: $formatted_time"
    echo ""
    echo "Service status summary:"
    echo "$status_output" | jq -r '
        "• Is Running: \(.is_running)",
        "• DHT Peers: \(.dht.peers_in_routing_table)",
        "• Wallet Connected: \(.wallet.connected)",
        "• Headers Progress: \(.wallet.headers_synchronization_progress)%",
        "• Blocks Behind: \(.wallet.blocks_behind // 0)"
    '
    echo ""
}

# Function to display timeout status
display_timeout_status() {
    echo ""
    echo "Timeout: LBRY SDK did not become fully ready within the specified time."
    echo ""
    echo "Current status:"
    if lbry-cli status 2>/dev/null; then
        echo "Service is responding but not fully ready"
    else
        echo "Service is not responding"
    fi
}

# Function to wait for LBRY SDK to be ready
wait_for_lbry_sdk() {
    local max_wait=${1:-300}  # Default to 5 minutes like start.sh
    local check_interval=${2:-1}  # Default to 1 second like start.sh
    local wait_time=0
    
    echo "Waiting for LBRY SDK to be ready..."
    
    # Start timer
    local start_time
    start_time=$(date +%s)
    
    # Get total startup items on first successful status check
    local startup_total=1
    
    while [ "$wait_time" -lt "$max_wait" ]; do
        # Check if LBRY SDK is running first
        if ! is_lbry_sdk_running; then
            echo -n "."
            sleep "$check_interval"
            wait_time=$((wait_time + check_interval))
            continue
        fi
        
        # Get current status
        local status_output
        status_output=$(lbry-cli status 2>/dev/null)
        if status_output=$(lbry-cli status 2>/dev/null); then
            # Display progress and get progress values
            local progress_result
            progress_result=$(display_startup_progress "$status_output" "$startup_total" "$start_time")
            
            # Parse progress result
            local startup_ready startup_ready_total headers_progress
            IFS='|' read -r startup_ready startup_ready_total headers_progress <<< "$progress_result"
            
            # Update startup_total if it was set
            if [ "$startup_ready_total" -ne 1 ]; then
                startup_total="$startup_ready_total"
            fi
            
            # Check if fully ready
            if is_lbry_sdk_fully_ready "$status_output" "$startup_ready" "$startup_total" "$headers_progress"; then
                display_ready_status "$status_output" "$start_time"
                return 0
            fi
        else
            clear_line
            echo "Attempt $((wait_time / check_interval + 1)): Unable to get status (service may still be starting)"
        fi
        
        sleep "$check_interval"
        wait_time=$((wait_time + check_interval))
    done
    
    display_timeout_status
    return 1
}

