#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/lib.sh"

echo "Starting LBRY CLI installation script..."

# Check all dependencies using common function
check_lbry_dependencies

echo ""
echo "All dependencies found. Installing lbry-cli..."

(
    # Clone lbry-cli repository
    if [ -d "lbry-cli" ]; then
        echo "lbry-cli directory already exists. Removing it..."
        rm -rf lbry-cli
    fi

    echo "Cloning lbry-cli repository..."
    git clone https://github.com/LBRYFoundation/lbry-cli.git

    # Change to lbry-cli directory and install
    echo "Installing lbry-cli..."
    cd lbry-cli
    go install
)

echo ""
echo "Installation completed successfully!"
echo ""
echo "Next steps:"
echo "1. Make sure your Go bin directory is in your PATH"
echo "2. Run 'lbry-cli --help' to verify installation"
echo ""
echo "The lbry-cli has been installed to: $(go env GOPATH)/bin/lbry-cli"