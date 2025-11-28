#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/lib.sh"

echo "=== LBRY Stop Script ==="
echo ""

# Check dependencies
check_runtime_dependencies

echo "Stopping LBRY services..."
echo ""

# Stop all services using shared function
if stop_docker_services; then
    echo ""
    echo "LBRY services stopped successfully!"
    echo ""
    echo "Your data is preserved. To start services again, run './start.sh'"
    echo "To completely reset and wipe data, run './reset.sh'"
    echo ""
else
    echo ""
    echo "Failed to stop LBRY services"
    exit 1
fi