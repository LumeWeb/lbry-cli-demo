#!/bin/bash

# Common business logic for LBRY CLI scripts

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
    echo "Checking dependencies..."
    check_dependency "docker" "Docker" "curl https://get.docker.com | bash"
    check_docker_compose
    check_dependency "git" "Git" "sudo apt-get install git"
    check_dependency "go" "Go" "sudo apt-get install golang-go"
    check_dependency "jq" "jq" "sudo apt-get install jq"
}

# Function to stop all Docker Compose services
stop_docker_services() {
    echo "Stopping all Docker Compose services..."
    if docker-compose down; then
        echo "Services stopped successfully"
        return 0
    else
        echo "Failed to stop services"
        return 1
    fi
}

# Function to check runtime dependencies (for start.sh)
check_runtime_dependencies() {
    echo "Checking runtime dependencies..."
    
    if ! command_exists "docker"; then
        echo "Error: Docker is not installed"
        exit 1
    fi
    
    if ! docker compose version >/dev/null 2>&1; then
        echo "Error: Docker Compose is not available"
        exit 1
    fi
    
    if ! command_exists "lbry-cli"; then
        echo "Error: lbry-cli is not installed or not in PATH"
        exit 1
    fi
    
    if ! command_exists "jq"; then
        echo "Error: jq is not installed"
        echo "Please install jq by running: sudo apt-get install jq"
        exit 1
    fi
    
    echo "All runtime dependencies are available"
}

# Function to cleanup blob data only (lbrynet.sqlite* and blobfiles)
cleanup_blobs() {
    echo "Cleaning up blob data (lbrynet.sqlite* and blobfiles)..."
    if docker compose run --rm cleanup-blobs; then
        echo "Blob data cleaned up successfully"
        return 0
    else
        echo "Failed to cleanup blob data"
        return 1
    fi
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

# Read and display JSON state file contents
read_json_state() {
    local state_file="$1"
    local log_file="${2:-}"
    
    if [ ! -f "$state_file" ]; then
        log_error "State file not found: $state_file" "$log_file"
        return 1
    fi
    
    if [ -n "$log_file" ] && [ -f "$log_file" ]; then
        cat "$state_file" | tee -a "$log_file"
    else
        cat "$state_file"
    fi
}

# State management functions (aligned with shared/state_manager.go)

# Get state directory path (equivalent to GetStateDir in shared)
get_state_dir() {
    # Look for .lbry-demo marker file to find shared root
    local current_dir="$(pwd)"
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
    
    # Check dependencies
    check_dependencies() {
        log "Checking dependencies..." "$log_file"
        
        if ! command_exists "go"; then
            log_error "Go is not installed" "$log_file"
            exit 1
        fi
        
        if ! command_exists "lbry-cli"; then
            log_error "lbry-cli is not installed or not in PATH" "$log_file"
            exit 1
        fi
        
        log_success "All dependencies available" "$log_file"
    }
    
    # Run the demo
    run_demo_app() {
        log "Starting $demo_name demo..." "$log_file"
        
        cd "$script_dir" || exit
        
        # Run the demo and capture output
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
        check_dependencies
        
        # Run the demo
        run_demo_app
        
        log_success "$demo_name demo system completed successfully!" "$log_file"
    }
    
    # Run main function with all arguments
    main "$@"
}

