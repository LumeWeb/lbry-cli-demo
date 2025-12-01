# LBRY CLI Demo

An interactive demonstration of LBRY network operations using Go. This project showcases various LBRY functionalities including file uploads, downloads, pinning, and network operations through multiple demo applications.

## Quick Start

Get up and running quickly:

```bash
# 1. Install dependencies
./install.sh

# 2. Start LBRY services
./start.sh

# 3. Run your first demo (POST upload)
bash ./post-upload/run.sh

# 4. Stop services when done
./stop.sh

# Clean up blob data (run independently, not while demos are running)
./cleanup-blobs.sh
```

## Prerequisites

- **Go 1.21+** - For building and running demos
- **Docker & Docker Compose** - For LBRY SDK services
- **Git** - For cloning the repository
- **jq** - For JSON processing and state management
- **curl** - For HTTP requests and API calls
- **lbry-cli** - For LBRY SDK interactions (installed via install.sh)

### Installation

```bash
# Clone the repository
git clone https://github.com/LumeWeb/lbry-cli-demo
cd lbry-cli-demo

# Install dependencies and tools
./install.sh
```

## Setup

### 1. Start LBRY Services

The LBRY SDK service provides the core LBRY functionality:

```bash
./start.sh
```

This starts:
- LBRY SDK service (Docker container)
- Required networking components
- State persistence in `state/` directory

### 2. Verify Services

Check that services are running:

```bash
docker ps | grep lbry
```

You should see the LBRY SDK container running.

## Important Notices

**Before running the demos, please read these important notices:**

### DHT Logging Messages
You may see errors in DHT logs about `this bucket range does not cover this peer` - these can be safely ignored. This is normal LBRY DHT logging behavior and not a demo failure.

### Custom Components
- **Forked DHT**: We forked the LBRY DHT at https://github.com/LumeWeb/lbry-dht and improved it for numerous technical reasons
- **Patched SDK**: Applied 20+ patches to the LBRY server daemon to enable blockchain-free file access, fix DHT bugs, and implement required RPC methods. See: https://github.com/LBRYFoundation/lbry-sdk/compare/master...LumeWeb:lbry-sdk:master
- **Custom Docker Image**: We run our own Docker image with these changes

### Performance Expectations
- **Slow Performance**: Demos may run slowly as they attempt DHT communication before falling back to fixed peers
- **Fixed Peer Fallback**: For poorly seeded files, demos typically acquire files from the portal via the fixed peer list
- **Timing**: Each demo may take up to 5 minutes. Running all demos may take roughly 30 minutes total

### Account State Management
- **CRITICAL**: If `state/account.json` is lost after creating an account and registering a device, you will need to contact the portal operator to remove the device
- **Device Uniqueness**: Device registrations are globally unique by IP address, not per-account
- **State Backup**: Consider backing up the `state/` directory to prevent loss of account credentials

### Demo Behavior
- **State Reset**: Every demo resets pinned state and cleans up downloads
- **Single Demo Only**: Do not run multiple demos simultaneously - this will break the system
- **Chain Sync**: The LBRY daemon requires syncing chain headers (SPV), which happens automatically but takes time. The Python daemon is not coded in a way that allows us to reasonably disable this functionality without making large changes to the codebase. It's too involved to bypass this requirement even though we don't need it for the demos.

### Platform Requirements
- **Linux Only**: Tested and guaranteed to work on Linux only
- **Ubuntu 24.04**: Explicitly tested on Ubuntu 24.04
- **Debian-based**: Recent Debian-based distributions should work
- **Untested**: Fedora/Redhat distributions are untested

### Isolation
Everything is isolated to Docker Compose and individual demo folders.

## Demo Guide

Each demo demonstrates different aspects of the LBRY network. All demos share common account management and state persistence.

### Demo 1: POST Upload (Small Files)

**Purpose**: Traditional POST-based upload for smaller files (10MB)

```bash
bash ./post-upload/run.sh
```

**What happens**:
1. Reuses existing account or creates new one
2. Generates a 10MB test file
3. Uploads via HTTP POST
4. Returns stream information

### Demo 2: TUS Upload (Large Files)

**Purpose**: Demonstrates resumable upload protocol for large files (100MB)

```bash
bash ./tus-upload/run.sh
```

**What happens**:
1. Creates or loads account from `state/account.json`
2. Generates a 100MB test file
3. Uploads using TUS protocol (resumable)
4. Returns stream URL and metadata

**Expected output**:
```
INFO: Account loaded from state
INFO: Generated 100MB test file
INFO: Starting TUS upload...
INFO: Upload completed successfully
INFO: Stream URL: https://.../...
```

### Demo 3: Pin/Unpin Streams

**Purpose**: Demonstrate stream pinning and unpinning by SD hash

**Test SD hash**: `acc6adf8b4f10dcddffc5c2ca87dbd9cb3a2664564695ac7aaab038193ff14a280cc3d4ebae55c71d0b885a7316d0137` (the hash this demo uses for demonstration)

```bash
bash ./pin/run.sh
```

**What happens**:
1. Validates SD hash format
2. Sends pin/unpin request to portal
3. Confirms operation success

### Demo 4: Reflector Server

**Purpose**: Run your own reflector server for blob storage and stream uploads

This requires a local reflector and peer server because LBRY cannot create blobs without a claim at present. We workaround this limitation by generating an SD blob ourselves and having lbry-cli pull from us locally as a peer before reflecting it to the portal.

```bash
bash ./reflector/run.sh
```

**What happens**:
1. Starts reflector on port 5669
2. Peer service on port 5570
3. Uploads streams directly to reflector (bypassing portal)
4. Ready to receive blob uploads
5. Performs stream upload verification and completion status
6. Allows lbry-cli to pull blobs locally as a peer before reflecting to portal



## Architecture Overview

### Core Components

- **Shared Library** (`shared/`): Common functionality across all demos
  - `DemoFramework`: Main orchestrator
  - `LBRYPortalClient`: Unified portal client
  - `AccountManager`: Account creation and login
  - `StateManager`: Persistent state management
  - `BlobDownloader`: Network blob downloading
  - `ReflectorUploadClient`: Reflector uploads

### Key Patterns

**Account Management**: All demos automatically create or load accounts from `state/account.json`

**State Persistence**: Uses `.lbry-demo` marker file to locate shared state directory

**Logging**: Structured logging with configurable levels (debug/info/warn/error)

**Network Operations**: Dynamic port allocation with configurable peers and seed nodes

## Configuration

### Command Line Options

All demos support common flags:

```bash
-portal string     Portal domain (default "pinner.xyz")
-log-level string  Log level: debug, info, warn, error (default "info")
```

### Portal Configuration

Change the default portal domain:

```bash
go run ./tus-upload -portal=dev.lbry.com
```

### LBRY SDK Configuration

Edit `webconf.yaml` to modify:
- Fixed peers
- Reflector servers
- Network settings

## Troubleshooting

### Common Issues

**Services won't start**:
```bash
# Check Docker status
docker ps
docker logs lbry-sdk

# Reset services
./stop.sh
./start.sh
```

**Account creation fails**:
```bash
# Clear state and retry
./reset.sh
./start.sh
```

**Upload fails with timeout**:
```bash
# Check network connectivity
ping pinner.xyz
```

**Permission denied errors**:
```bash
# Fix permissions on state directory
sudo chown -R $USER:$USER state/
```

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
go run ./tus-upload -log-level=debug
```

### Reset Everything

If you encounter persistent issues:

```bash
# WARNING: This will delete all Docker data for the demo
./reset.sh
```

## Learn More

- [LBRY Documentation](https://lbry.tech/)
- [Go LBRY Library](https://github.com/lbryio/lbry.go)
- [TUS Protocol](https://tus.io/)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Need help?** Open an issue or reach out through [lumeweb.com](https://lumeweb.com) contact channels.