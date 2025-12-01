#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/lib.sh"

show_script_header "LBRY Blob Cleanup Script" "This script will:
- Stop the LBRY SDK
- Clean up blob data only (lbrynet.sqlite* and blobfiles)
- Restart the LBRY SDK

Wallet and other data will be preserved."

echo "Starting blob cleanup and restart process..."
echo ""

# Step 1: Stop services using existing stop.sh
echo "Step 1: Stopping LBRY services..."
if ./stop.sh; then
    echo "Services stopped successfully"
else
    echo "Failed to stop services"
    exit 1
fi

echo ""

# Step 2: Clean up blob data using shared function
if clear_lbry_blobs; then
    echo "Blob data cleaned up successfully"
else
    echo "Failed to cleanup blob data"
    exit 1
fi

echo ""

# Step 3: Restart services using existing start.sh
echo "Step 3: Restarting LBRY services..."
if ./start.sh; then
    echo "Services started successfully"
else
    echo "Failed to start services"
    exit 1
fi

show_script_footer "Blob Cleanup and Restart" "success" "LBRY SDK stopped, blob data cleaned up, and LBRY SDK restarted"
echo "Your wallet and other data are preserved."
echo "Services are now running with fresh blob data."