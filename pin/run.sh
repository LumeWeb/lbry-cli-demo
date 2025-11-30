#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/../lib.sh"

# =============================================================================
# PIN DEMO RUN SCRIPT
# =============================================================================
#
# This script runs the pin demo which:
# - Pins a specific stream by SD hash
# - Downloads the pinned stream
# - Verifies and saves the downloaded content
# - Cleans up the stream
#
# Usage Examples:
#   # Use default configuration
#   ./run.sh
#
#   # Use custom portal domain
#   PORTAL=my-portal.example.com ./run.sh
#
#   # Use custom log level
#   LOG_LEVEL=debug ./run.sh
#
#   # Full custom configuration
#   PORTAL=my-portal.example.com \
#   LOG_LEVEL=debug \
#   LOG_FILE=custom.log \
#   ./run.sh
#
# Environment Variables:
#   PORTAL - Portal domain (default: pinner.xyz)
#   LOG_LEVEL     - Log level (debug, info, warn, error) (default: info)
#   LOG_FILE      - Optional log file path (default: pin.log)
#
# =============================================================================

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Configuration
DEMO_NAME="pin"
LOG_FILE="${LOG_FILE:-$SCRIPT_DIR/pin.log}"

# Run the demo using the common template
run_demo "$DEMO_NAME" "$SCRIPT_DIR" "$LOG_FILE"