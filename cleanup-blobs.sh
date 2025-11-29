#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/lib.sh"

echo "=== LBRY Blob Cleanup Script ==="
echo ""
echo "This script will:"
echo "- Stop the LBRY SDK"
echo "- Clean up blob data only (lbrynet.sqlite* and blobfiles)"
echo "- Restart the LBRY SDK"
echo ""
echo "Wallet and other data will be preserved."
echo ""

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
if cleanup_blobs; then
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

echo ""
echo "=== Blob Cleanup and Restart Complete ==="
echo ""
echo "✓ LBRY SDK stopped"
echo "✓ Blob data cleaned up (lbrynet.sqlite* and blobfiles)"
echo "✓ LBRY SDK restarted"
echo ""
echo "Your wallet and other data are preserved."
echo "Services are now running with fresh blob data."