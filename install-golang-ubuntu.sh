#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/lib.sh"

show_script_header "Go Installation Script for Ubuntu"

# =============================================================================
# GO INSTALLATION SCRIPT - Environment Variable Configuration
# =============================================================================
#
# This script installs Go 1.25 on Ubuntu systems using the longsleep PPA.
# It will abort if:
#   - Not running on Ubuntu
#   - Go is already installed
#
# Usage:
#   ./install-golang-ubuntu.sh
#
# Environment Variables:
#   None required
#
# =============================================================================

# Check if running on Ubuntu
check_ubuntu() {
    log "Checking if running on Ubuntu..."
    
    if [ ! -f /etc/os-release ]; then
        log_error "Cannot determine OS: /etc/os-release not found"
        exit 1
    fi
    
    # Source the OS release file
    source /etc/os-release
    
    if [ "$ID" != "ubuntu" ]; then
        log_error "This script is designed for Ubuntu only. Detected OS: $ID"
        exit 1
    fi
    
    log_success "Ubuntu detected: $PRETTY_NAME"
}

# Check if Go is already installed
check_go_installed() {
    log "Checking if Go is already installed..."
    
    if command -v go >/dev/null 2>&1; then
        local go_version
        go_version=$(go version 2>/dev/null || echo "unknown")
        log_error "Go is already installed: $go_version"
        log_error "Please uninstall existing Go installation before running this script"
        log_error "You can remove it with: sudo apt remove golang-*"
        exit 1
    fi
    
    log_success "Go is not installed, proceeding with installation"
}

# Add PPA repository
add_ppa() {
    log "Adding longsleep/golang-backports PPA..."
    
    # Check if PPA is already added
    if grep -r "longsleep/golang-backports" /etc/apt/sources.list* >/dev/null 2>&1; then
        log "PPA longsleep/golang-backports is already added"
    else
        log "Adding PPA repository..."
        if sudo add-apt-repository -y ppa:longsleep/golang-backports; then
            log_success "PPA added successfully"
        else
            log_error "Failed to add PPA repository"
            exit 1
        fi
    fi
    
    # Update package list
    log "Updating package list..."
    if sudo apt update; then
        log_success "Package list updated successfully"
    else
        log_error "Failed to update package list"
        exit 1
    fi
}

# Install Go 1.25
install_go() {
    log "Installing golang-1.25..."
    
    if sudo apt install -y golang-1.25; then
        log_success "golang-1.25 installed successfully"
    else
        log_error "Failed to install golang-1.25"
        exit 1
    fi
}

# Configure update-alternatives
configure_alternatives() {
    log "Configuring update-alternatives for Go..."
    
    local go_binary="/usr/lib/go-1.25/bin/go"
    
    # Check if the Go binary exists
    if [ ! -f "$go_binary" ]; then
        log_error "Go binary not found at: $go_binary"
        exit 1
    fi
    
    # Configure update-alternatives
    if sudo update-alternatives --install /usr/bin/go go "$go_binary" 10; then
        log_success "update-alternatives configured successfully"
    else
        log_error "Failed to configure update-alternatives"
        exit 1
    fi
    
    # Set the alternative
    if sudo update-alternatives --set go "$go_binary"; then
        log_success "Go alternative set successfully"
    else
        log_error "Failed to set Go alternative"
        exit 1
    fi
}

# Verify installation
verify_installation() {
    log "Verifying Go installation..."
    
    # Wait a moment for alternatives to update
    sleep 2
    
    if command -v go >/dev/null 2>&1; then
        local go_version
        go_version=$(go version 2>/dev/null)
        log_success "Go installation verified: $go_version"
        
        # Show Go version and environment
        log "Go environment information:"
        go version
        echo "Go location: $(which go)"
        echo "GOPATH: ${GOPATH:-default (not set)}"
        echo "GOROOT: ${GOROOT:-default (not set)}"
    else
        log_error "Go installation verification failed"
        exit 1
    fi
}

# Main execution
main() {
    # Check prerequisites
    check_ubuntu
    check_go_installed
    
    # Installation steps
    add_ppa
    install_go
    configure_alternatives
    verify_installation
    
    show_script_footer "Go Installation" "success" "Go 1.25 has been successfully installed on Ubuntu!"
    
    echo ""
    echo "Next steps:"
    echo "1. Verify installation: go version"
    echo "2. Set up your Go workspace if needed"
    echo "3. Start building Go applications"
    echo ""
    echo "To manage Go versions in the future:"
    echo "  sudo update-alternatives --config go"
}

# Run main function
main "$@"