#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/lib.sh"

show_script_header "LBRY Stop Script"

echo "Stopping LBRY services..."
echo ""

# Use the unified service stop function
if stop_docker_services; then
    show_script_footer "LBRY Stop" "success" "LBRY services stopped successfully!"
    echo "Your data is preserved. To start services again, run './start.sh'"
    echo "To completely reset and wipe data, run './reset.sh'"
else
    show_script_footer "LBRY Stop" "error" "Failed to stop LBRY services"
    exit 1
fi