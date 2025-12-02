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
# Get the GOBIN path using our helper function
    gobin_path=$(get_gobin_path)
    
    # Move the binary from 'cli' to 'lbry-cli' if it exists
    if [ -f "$gobin_path/cli" ]; then
        echo "Moving binary from 'cli' to 'lbry-cli'..."
        mv "$gobin_path/cli" "$gobin_path/lbry-cli"
    fi

    # Post-install verification
    echo ""
    echo "Verifying installation..."
    
    # Check if lbry-cli is accessible in current PATH
    if command_exists lbry-cli; then
        echo "✓ lbry-cli is accessible in PATH"
        lbry_cli_path=$(which lbry-cli)
        echo "  Location: $lbry_cli_path"
    fi
    
    # If not accessible, check if it exists in GOBIN path
    if ! command_exists lbry-cli; then
        echo "⚠ lbry-cli is not accessible in PATH"
    fi
    
    # If found in GOBIN, add to PATH
    if ! command_exists lbry-cli && [ -f "$gobin_path/lbry-cli" ]; then
        echo "✓ Found lbry-cli at: $gobin_path/lbry-cli"
        echo ""
        echo "Adding GOBIN to PATH for current session..."
        
        # Add GOBIN to PATH for current session
        export PATH="$gobin_path:$PATH"
        echo "✓ Added $gobin_path to PATH"
    fi
    
    # Verify after PATH update
    if ! command_exists lbry-cli && [ -f "$gobin_path/lbry-cli" ]; then
        if command_exists lbry-cli; then
            echo "✓ lbry-cli is now accessible!"
            lbry_cli_path=$(which lbry-cli)
            echo "  Location: $lbry_cli_path"
        else
            echo "✗ Still cannot access lbry-cli after PATH update"
        fi
    fi
    
    # If not found in GOBIN either
    if ! command_exists lbry-cli && [ ! -f "$gobin_path/lbry-cli" ]; then
        echo "✗ lbry-cli not found at expected location: $gobin_path/lbry-cli"
    fi

echo ""
echo "Installation completed successfully!"
echo ""
echo "Next steps:"
if command_exists lbry-cli; then
    echo "✓ lbry-cli is ready to use! Run 'lbry-cli --help' to get started"
else
    echo "1. Add the following to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "   export PATH=\"$gobin_path:\$PATH\""
    echo "2. Source your profile or restart your terminal"
    echo "3. Run 'lbry-cli --help' to verify installation"
fi
echo ""
echo "The lbry-cli has been installed to: $gobin_path/lbry-cli"