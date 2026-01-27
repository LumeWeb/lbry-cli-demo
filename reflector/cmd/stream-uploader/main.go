package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/go-units"
	"go.lumeweb.com/liblbry/protocol"
	"go.lumeweb.com/liblbry/stream"
	"go.uber.org/zap"

	"github.com/lumeweb/lbry-cli-demo/shared"
)

// Config holds the CLI configuration
type Config struct {
	ReflectorAddress string
	WaitMode         bool
	StateFile        string
}

// StreamState holds the upload state information specific to stream-uploader
type StreamState struct {
	Timestamp         string   `json:"timestamp"`
	OriginalSHA256    string   `json:"original_sha256"`
	SDBlobHash        string   `json:"sd_blob_hash"`
	ContentBlobHashes []string `json:"content_blob_hashes"`
	UploadHash        string   `json:"upload_hash"`
	Status            string   `json:"status"`
}

// Constants
const (
	blobSize = units.MiB * 10
)

// parseFlags parses command-line flags and returns configuration
func parseFlags() *Config {
	config := &Config{}
	flag.StringVar(&config.ReflectorAddress, "reflector", "localhost:5669", "Reflector server address (host:port)")
	flag.BoolVar(&config.WaitMode, "wait-mode", false, "Wait for account operations to complete (requires existing state file)")

	// Default state file location
	defaultStateFile := "reflector.json"
	flag.StringVar(&config.StateFile, "state-file", defaultStateFile, "Reflector state file path")

	return config
}

// generateCryptoRandomBuffer creates a crypto-random buffer of specified size
func generateCryptoRandomBuffer(size int) (io.Reader, string, error) {
	data := make([]byte, size)
	_, err := rand.Read(data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate crypto-random data: %w", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hash[:])

	return bytes.NewReader(data), hashHex, nil
}

// createStreamFromReader creates a stream from an io.Reader using liblbry stream creator
func createStreamFromReader(reader io.Reader, size int64, logger *zap.Logger) (*stream.StreamResult, error) {
	logger.Info("Creating stream from reader", zap.Int64("size_bytes", size))

	// Create stream creator
	streamCreator := stream.NewStreamCreator()

	// Create stream from reader
	result, err := streamCreator.CreateStream(reader, size)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	logger.Info("Stream created successfully",
		zap.String("stream_hash", result.StreamHash),
		zap.String("sd_blob_hash", result.SDBlobHash),
		zap.Int("total_chunks", result.TotalChunks))

	return result, nil
}

// uploadStreamToReflector uploads a stream to the reflector server
func uploadStreamToReflector(result *stream.StreamResult, reflectorAddr string, logger *zap.Logger) error {
	logger.Info("Connecting to reflector server", zap.String("address", reflectorAddr))

	// Create reflector client
	reflectorClient := protocol.NewReflectorClient(
		protocol.WithReflectorClientLogger(logger),
		protocol.WithReflectorClientTimeout(30*time.Second),
	)

	// Connect to reflector
	err := reflectorClient.Connect(reflectorAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to reflector at %s: %w", reflectorAddr, err)
	}
	defer func() {
		if closeErr := reflectorClient.Close(); closeErr != nil {
			logger.Warn("Failed to close reflector connection", zap.Error(closeErr))
		}
	}()

	logger.Info("Connected to reflector server successfully")

	// Upload SD blob first
	logger.Info("Uploading SD blob", zap.String("hash", result.SDBlobHash))
	err = reflectorClient.SendSDBlob(result.SDBlobHash, result.SDBlobData)
	if err != nil {
		return fmt.Errorf("failed to upload SD blob: %w", err)
	}
	logger.Info("SD blob uploaded successfully")

	// Upload content blobs
	for i, blobData := range result.ContentBlobs {
		hash := result.ContentHashes[i]
		logger.Info("Uploading content blob",
			zap.String("hash", hash),
			zap.Int("blob_index", i),
			zap.Int("blob_size", len(blobData)))

		err = reflectorClient.SendBlob(hash, blobData)
		if err != nil {
			return fmt.Errorf("failed to upload content blob %d: %w", i, err)
		}
	}

	logger.Info("All blobs uploaded successfully to reflector")
	return nil
}

// setupSignalHandling creates a context and signal channel for graceful interruption
func setupSignalHandling() (context.Context, context.CancelFunc, chan os.Signal) {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	return ctx, cancel, sigChan
}

// registerAccountAndDevice registers a new account and device using framework
func registerAccountAndDevice(framework *shared.DemoFramework) error {
	logger := framework.GetLogger()
	accountManager := framework.GetAccountManager()
	client := framework.GetClient()

	logger.Info("Starting account and device registration")

	// Login or create account
	_, err := accountManager.LoginOrCreateAccount()
	if err != nil {
		return fmt.Errorf("failed to login or create account: %w", err)
	}

	// Cleanup all existing streams before starting
	err = accountManager.CleanupAllStreams()
	if err != nil {
		logger.Warn("Failed to cleanup existing streams", zap.Error(err))
	}

	// Check existing devices before registering
	deviceName := "stream-uploader-device"
	logger.Info("Fetching public IP address for device registration...")
	publicIP, err := shared.GetPublicIP()
	if err != nil {
		logger.Warn("Failed to fetch public IP, using localhost", zap.Error(err))
		publicIP = "127.0.0.1"
	}

	// List existing devices
	logger.Info("Checking existing devices...")
	devices, err := client.ListDevices()
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}

	// Check if any device already exists with our IP
	var deviceWithSameIP *shared.DeviceResponse
	for _, device := range devices.Data {
		if device.IPAddress == publicIP {
			deviceWithSameIP = &device
			break
		}
	}

	if deviceWithSameIP != nil {
		logger.Info("Device already exists with this IP address",
			zap.String("device_name", deviceWithSameIP.Name),
			zap.String("ip_address", deviceWithSameIP.IPAddress),
			zap.Int("device_id", deviceWithSameIP.ID))
	} else {
		logger.Info("No device found with this IP, registering new device...",
			zap.String("device_name", deviceName),
			zap.String("ip_address", publicIP))
		err = client.RegisterDevice(deviceName, publicIP)
		if err != nil {
			return fmt.Errorf("failed to register device: %w", err)
		}
		logger.Info("Device registered successfully!")
	}

	logger.Info("Account and device registration completed successfully")
	return nil
}

// reloginAndWait relogs in using framework and waits for operations
func reloginAndWait(framework *shared.DemoFramework, streamState *StreamState) error {
	logger := framework.GetLogger()
	accountManager := framework.GetAccountManager()

	logger.Info("Starting relogin and wait mode")

	// Relogin using existing account state or create new account
	_, err := accountManager.LoginOrCreateAccount()
	if err != nil {
		return fmt.Errorf("failed to relogin: %w", err)
	}

	// Wait 5 seconds before monitoring operations
	logger.Info("Waiting 5 seconds before monitoring operations...")
	time.Sleep(5 * time.Second)

	// Wait for all operations to complete
	err = accountManager.WaitForOperations()
	if err != nil {
		return fmt.Errorf("failed to wait for operations: %w", err)
	}

	// List streams and output the first found stream
	streams, err := accountManager.ListAndLogStreams()
	if err != nil {
		return fmt.Errorf("failed to list streams: %w", err)
	}

	if len(streams) == 0 {
		logger.Warn("No streams found")
	} else {
		firstStream := streams[0]
		logger.Info("First found stream",
			zap.Int("id", firstStream.ID),
			zap.String("sd_hash", firstStream.SDHash),
			zap.String("stream_hash", firstStream.StreamHash))

		// Verify SD hash matches our uploaded stream
		if streamState.SDBlobHash != "" {
			if firstStream.SDHash == streamState.SDBlobHash {
				logger.Info("SD hash matches our uploaded stream!",
					zap.String("expected_sd_hash", streamState.SDBlobHash),
					zap.String("found_sd_hash", firstStream.SDHash))
			} else {
				logger.Warn("SD hash does not match our uploaded stream",
					zap.String("expected_sd_hash", streamState.SDBlobHash),
					zap.String("found_sd_hash", firstStream.SDHash))
			}
		}

		// Output the first stream details to console
		fmt.Printf("=== FIRST FOUND STREAM ===\n")
		fmt.Printf("ID: %d\n", firstStream.ID)
		fmt.Printf("SD Hash: %s\n", firstStream.SDHash)
		fmt.Printf("Stream Hash: %s\n", firstStream.StreamHash)
		fmt.Printf("============================\n")

		// Fetch the blob and verify using framework
		downloadedData, err := framework.DownloadAndVerifyStream(context.Background(), firstStream.SDHash, nil)
		if err != nil {
			logger.Warn("Failed to download and verify stream", zap.Error(err))
			return nil
		}

		logger.Info("Stream downloaded and verified successfully",
			zap.Int("downloaded_size_bytes", len(downloadedData)))
	}

	logger.Info("Relogin and wait mode completed successfully")
	return nil
}

func main() {
	// Parse command-line flags
	config := parseFlags()

	// Create demo framework
	framework, err := shared.NewDemoFrameworkFromFlags("STREAM-UPLOADER")
	if err != nil {
		panic(err)
	}
	defer framework.Cleanup()

	logger := framework.GetLogger()
	zapLogger := framework.GetLogger()

	logger.Info("Starting LBRY stream uploader",
		zap.String("reflector_address", config.ReflectorAddress),
		zap.Bool("wait_mode", config.WaitMode),
		zap.Int("blob_size_mb", blobSize/(1024*1024)),
	)

	// Handle wait mode
	if config.WaitMode {
		logger.Info("Running in wait mode")

		// Load existing stream state
		stateManager := framework.GetStateManager()
		var streamState StreamState
		err = stateManager.LoadJSON(config.StateFile, &streamState)
		if err != nil {
			logger.Fatal("Failed to load stream state file", zap.Error(err))
		}

		// Relogin and wait for operations
		err = reloginAndWait(framework, &streamState)
		if err != nil {
			zapLogger.Fatal("Failed to relogin and wait", zap.Error(err))
		}

		logger.Info("Wait mode completed successfully")
		return
	}

	// Upload mode (original behavior)
	logger.Info("Running in upload mode")

	// Initialize stream state
	streamState := StreamState{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    "initializing",
	}

	// Save initial state using framework's StateManager
	stateManager := framework.GetStateManager()
	err = stateManager.SaveJSON(config.StateFile, streamState)
	if err != nil {
		zapLogger.Fatal("Failed to save initial stream state", zap.Error(err))
	}

	// Register account and device
	err = registerAccountAndDevice(framework)
	if err != nil {
		zapLogger.Fatal("Failed to register account and device", zap.Error(err))
	}

	// Setup signal handling for graceful interruption during upload
	ctx, cancel, sigChan := setupSignalHandling()
	defer cancel()

	// Start signal handler in background for interruption during upload
	go func() {
		sig := <-sigChan
		logger.Info("Received interruption signal", zap.String("signal", sig.String()))
		cancel()
	}()

	// Generate crypto-random buffer
	logger.Info("Generating crypto-random buffer", zap.Int("size_bytes", blobSize))

	// Check for interruption before starting expensive operations
	select {
	case <-ctx.Done():
		logger.Info("Upload cancelled by user")
		return
	default:
	}

	reader, blobHash, err := generateCryptoRandomBuffer(blobSize)
	if err != nil {
		zapLogger.Fatal("Failed to generate crypto-random buffer", zap.Error(err))
	}

	// Update state with original SHA256
	streamState.OriginalSHA256 = blobHash
	streamState.Status = "generated_data"
	err = stateManager.SaveJSON(config.StateFile, streamState)
	if err != nil {
		zapLogger.Fatal("Failed to update stream state with SHA256", zap.Error(err))
	}

	// Print the SHA256 hash of the generated blob
	fmt.Printf("Generated blob SHA256: %s\n", blobHash)

	// Create stream from the buffer
	// Check for interruption before stream creation
	select {
	case <-ctx.Done():
		logger.Info("Upload cancelled by user")
		return
	default:
	}

	streamResult, err := createStreamFromReader(reader, int64(blobSize), logger)
	if err != nil {
		zapLogger.Fatal("Failed to create stream", zap.Error(err))
	}

	// Update state with stream information
	streamState.SDBlobHash = streamResult.SDBlobHash
	streamState.ContentBlobHashes = streamResult.ContentHashes
	streamState.UploadHash = streamResult.StreamHash
	streamState.Status = "stream_created"
	err = stateManager.SaveJSON(config.StateFile, streamState)
	if err != nil {
		zapLogger.Fatal("Failed to update stream state with stream info", zap.Error(err))
	}

	// Upload stream to reflector
	// Check for interruption before upload
	select {
	case <-ctx.Done():
		logger.Info("Upload cancelled by user")
		return
	default:
	}

	logger.Info("Starting upload to reflector")
	err = uploadStreamToReflector(streamResult, config.ReflectorAddress, logger)
	if err != nil {
		// Update state with error
		streamState.Status = "upload_failed"
		saveErr := stateManager.SaveJSON(config.StateFile, streamState)
		if saveErr != nil {
			logger.Warn("Failed to update stream state with upload failed status", zap.Error(saveErr))
		}
		zapLogger.Fatal("Failed to upload stream to reflector", zap.Error(err))
	}

	// Update state with successful upload
	streamState.Status = "uploaded"
	err = stateManager.SaveJSON(config.StateFile, streamState)
	if err != nil {
		logger.Warn("Failed to update stream state with upload status", zap.Error(err))
	}

	logger.Info("Stream uploaded successfully to reflector",
		zap.String("stream_hash", streamResult.StreamHash),
		zap.String("sd_blob_hash", streamResult.SDBlobHash),
		zap.Int("total_blobs", streamResult.TotalChunks))

	logger.Info("Stream uploader completed successfully")
}
