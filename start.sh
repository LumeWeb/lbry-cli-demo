#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/lib.sh"

# Ensure Go binaries are accessible in PATH
setup_go_path

echo "Starting LBRY service..."

# Use the new LBRY SDK start function
start_lbry_sdk

# Wait a moment for services to start
echo "Waiting for services to initialize..."
sleep 10



# Wait for LBRY SDK to be ready using the DRY'ed function from lib.sh
if wait_for_lbry_sdk 300 1; then
    echo "LBRY service is ready to use!"
    exit 0
else
    echo ""
    echo "Please check the Docker logs with: docker compose logs lbry"
    exit 1
fi