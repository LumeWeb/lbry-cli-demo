#!/bin/bash

set -e

# Source common functions
source "$(dirname "$0")/lib.sh"

echo "Starting LBRY service..."

# Check runtime dependencies using common function
check_runtime_dependencies

# Ensure data directory exists with proper permissions
if [ ! -d "./data" ]; then
    echo "Creating data directory..."
    mkdir -p ./data
fi

echo "Setting data directory permissions..."
chmod 777 ./data

# Start Docker Compose services
echo "Starting Docker Compose services..."
docker compose up -d

# Wait a moment for services to start
echo "Waiting for services to initialize..."
sleep 10

# Function to check startup status progress
check_startup_progress() {
    local status_output="$1"
    
    # Check if status_output is valid JSON
    if ! echo "$status_output" | jq . >/dev/null 2>&1; then
        echo "0"
        return
    fi
    
    # Check if startup_status exists
    if ! echo "$status_output" | jq -e '.startup_status' >/dev/null 2>&1; then
        echo "0"
        return
    fi
    
    # Count total and ready items
    local total_items
    local ready_items
    total_items=$(echo "$status_output" | jq -r '.startup_status | keys | length')
    ready_items=$(echo "$status_output" | jq -r '.startup_status | to_entries[] | select(.value == true) | .key' | wc -l)
    
    if [ "$total_items" -eq 0 ]; then
        echo "0"
        return
    fi
    
    echo "$ready_items"
}

# Function to get total startup items
get_startup_total() {
    local status_output="$1"
    
    # Check if status_output is valid JSON
    if ! echo "$status_output" | jq . >/dev/null 2>&1; then
        echo "1"
        return
    fi
    
    # Check if startup_status exists
    if ! echo "$status_output" | jq -e '.startup_status' >/dev/null 2>&1; then
        echo "1"
        return
    fi
    
    local total_items
    total_items=$(echo "$status_output" | jq -r '.startup_status | keys | length')
    
    if [ "$total_items" -eq 0 ]; then
        echo "1"
    else
        echo "$total_items"
    fi
}

# Function to get current blockchain height from API
get_blockchain_height() {
    local height=0
    
    # Try to fetch from LBRY explorer API
    if command -v curl >/dev/null 2>&1; then
        local api_response
        api_response=$(curl -s "https://explorer.lbry.org/api/v1/status" 2>/dev/null)
        
        if [ $? -eq 0 ] && echo "$api_response" | jq . >/dev/null 2>&1; then
            height=$(echo "$api_response" | jq -r '.status.height // 0')
            if [ "$height" = "null" ] || [ "$height" = "" ]; then
                height=0
            fi
        fi
    fi
    
    echo "$height"
}

# Function to check headers synchronization progress
check_headers_progress() {
    local status_output="$1"
    local progress=0
    
    # Check if status_output is valid JSON
    if ! echo "$status_output" | jq . >/dev/null 2>&1; then
        echo "0"
        return
    fi
    
    # Check if wallet.blocks and wallet.blocks_behind exist
    if ! echo "$status_output" | jq -e '.wallet.blocks' >/dev/null 2>&1 || ! echo "$status_output" | jq -e '.wallet.blocks_behind' >/dev/null 2>&1; then
        echo "0"
        return
    fi
    
    local blocks
    local blocks_behind
    blocks=$(echo "$status_output" | jq -r '.wallet.blocks // 0')
    blocks_behind=$(echo "$status_output" | jq -r '.wallet.blocks_behind // 0')
    
    if [ "$blocks" = "null" ] || [ "$blocks" = "" ]; then
        blocks=0
    fi
    
    if [ "$blocks_behind" = "null" ] || [ "$blocks_behind" = "" ]; then
        blocks_behind=0
    fi
    
    # Get actual blockchain height from API for accurate calculation
    local blockchain_height
    blockchain_height=$(get_blockchain_height)
    
    # If API call failed, fall back to local calculation
    if [ "$blockchain_height" -eq 0 ]; then
        # Fallback: use blocks + blocks_behind as total height
        blockchain_height=$((blocks + blocks_behind))
    fi
    
    # Calculate progress based on actual blockchain height
    if [ "$blockchain_height" -eq 0 ]; then
        progress=0
    elif [ "$blocks_behind" -eq 0 ]; then
        progress=100
    else
        # Calculate: (current_height / blockchain_height) * 100
        progress=$((blocks * 100 / blockchain_height))
    fi
    
    echo "$progress"
}

# Monitor status until ready
echo "Monitoring LBRY status..."
max_attempts=300  # 5 minutes with 1-second intervals
attempt=0

# Start timer
start_time=$(date +%s)

# Get total startup items on first successful status check
startup_total=1
headers_total=100

while [ $attempt -lt $max_attempts ]; do
    attempt=$((attempt + 1))
    
    # Get current status
    status_output=$(lbry-cli status 2>/dev/null)
    if [ $? -eq 0 ]; then
        # Get startup total on first successful check
        if [ "$startup_total" -eq 1 ]; then
            startup_total=$(get_startup_total "$status_output")
        fi
        
        # Check startup progress
        startup_ready=$(check_startup_progress "$status_output")
        
        # Check headers synchronization progress
        headers_progress=$(check_headers_progress "$status_output")
        
        # Calculate elapsed time
        current_time=$(date +%s)
        elapsed_time=$((current_time - start_time))
        minutes=$((elapsed_time / 60))
        seconds=$((elapsed_time % 60))
        formatted_time=$(printf "%02d:%02d" $minutes $seconds)
        
        # Display progress bars with timer
        clear_line
        show_progress_bar "$startup_ready" "$startup_total" "Startup" "$formatted_time"
        echo ""
        show_progress_bar "$headers_progress" "$headers_total" "Blockchain Headers Sync" "$formatted_time"
        
        # Check if startup is complete
        if [ "$startup_ready" -eq "$startup_total" ]; then
            # Check headers synchronization progress from the API field as backup
            api_headers_progress=0
            if echo "$status_output" | jq -e '.wallet.headers_synchronization_progress' >/dev/null 2>&1; then
                api_headers_progress=$(echo "$status_output" | jq -r '.wallet.headers_synchronization_progress // 0')
                if [ "$api_headers_progress" = "null" ] || [ "$api_headers_progress" = "" ]; then
                    api_headers_progress=0
                fi
            fi
            
            # Only consider fully ready if both calculated progress AND API progress are at 100%
            if [ "$headers_progress" -eq 100 ] && [ "$api_headers_progress" -eq 100 ]; then
                # Calculate startup time
                end_time=$(date +%s)
                startup_duration=$((end_time - start_time))
                
                # Format duration as MM:SS
                minutes=$((startup_duration / 60))
                seconds=$((startup_duration % 60))
                formatted_time=$(printf "%02d:%02d" $minutes $seconds)
                
                echo ""
                echo ""
                echo "LBRY service is fully ready."
                echo "All startup components are running and headers are synchronized."
                echo "Startup completed in: $formatted_time"
                echo ""
                echo "Service status summary:"
                echo "$status_output" | jq -r '
                    "• Is Running: \(.is_running)",
                    "• DHT Peers: \(.dht.peers_in_routing_table)",
                    "• Wallet Connected: \(.wallet.connected)",
                    "• Headers Progress: \(.wallet.headers_synchronization_progress)%",
                    "• Blocks Behind: \(.wallet.blocks_behind // 0)"
                '
                echo ""
                echo "LBRY service is ready to use!"
                exit 0
            else
                # Startup complete but headers not fully synced, continue monitoring
                if [ "$api_headers_progress" -gt 0 ]; then
                    echo ""
                    echo "Startup complete, waiting for headers sync: $api_headers_progress% (API) vs $headers_progress% (calculated)"
                fi
            fi
        fi
    else
        clear_line
        echo "Attempt $attempt: Unable to get status (service may still be starting)"
    fi
    
    sleep 1
done

echo ""
echo "Timeout: LBRY service did not become fully ready within 5 minutes."
echo ""
echo "Current status:"
if lbry-cli status 2>/dev/null; then
    echo "Service is responding but not fully ready"
else
    echo "Service is not responding"
fi
echo ""
echo "Please check the Docker logs with: docker-compose logs lbry"
exit 1