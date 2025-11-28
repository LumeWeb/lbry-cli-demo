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