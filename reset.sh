#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/lib.sh"

show_script_header "LBRY Data Reset Script" "WARNING: This script will completely wipe all LBRY data!
This includes:
- Wallet data
- Downloaded content
- Blockchain data
- All configuration and state

This action is IRREVERSIBLE!"

# Check dependencies
check_runtime_dependencies

echo "Starting reset process..."
echo ""

# Step 1: Stop all services
echo "Step 1: Stopping all Docker Compose services..."
if stop_docker_services; then
    echo "Services stopped successfully"
else
    echo "Failed to stop services"
    exit 1
fi

echo ""

# Step 2: Run cleanup to wipe data directory
echo "Step 2: Wiping data directory..."
echo "This will remove all contents in the ./data directory..."
echo ""

if docker compose run --rm cleanup; then
    echo "Data directory wiped successfully"
else
    echo "Failed to wipe data directory"
    exit 1
fi

show_script_footer "LBRY Data Reset" "success" "The LBRY data directory has been completely wiped."
echo "You can now run './start.sh' to start fresh with a clean installation."