package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.lumeweb.com/liblbry/protocol"
	"go.lumeweb.com/liblbry/stream"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/lumeweb/lbry-cli-demo/shared"
)

// Config holds the CLI configuration
type Config struct {
	ReflectorAddress string
	LogLevel         string
	StateFile        string
	PortalURL        string
	WaitMode         bool
}

// State holds the upload state information
type State struct {
	Timestamp         string   `json:"timestamp"`
	OriginalSHA256    string   `json:"original_sha256"`
	SDBlobHash        string   `json:"sd_blob_hash"`
	ContentBlobHashes []string `json:"content_blob_hashes"`
	UploadHash        string   `json:"upload_hash"`
	Status            string   `json:"status"`
	// Account information
	AccountEmail     string `json:"account_email"`
	AccountPassword  string `json:"account_password"`
	AccountFirstName string `json:"account_first_name"`
	AccountLastName  string `json:"account_last_name"`
	// Device information
	DeviceName string `json:"device_name"`
}

// Constants
const (
	blobSize         = 10 * 1024 * 1024 // 10MB
	accountSubdomain = "account"
	lbrySubdomain    = "lbry"
)

// parseFlags parses command-line flags and returns configuration
func parseFlags() *Config {
	config := &Config{}
	flag.StringVar(&config.ReflectorAddress, "reflector", "localhost:5669", "Reflector server address (host:port)")
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.StringVar(&config.PortalURL, "portal-url", "pinner.xyz", "Portal domain for account registration (e.g., pinner.xyz)")
	flag.BoolVar(&config.WaitMode, "wait-mode", false, "Wait for account operations to complete (requires existing state file)")

	// Default state file location
	scriptDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		scriptDir = "."
	}
	defaultStateFile := filepath.Join(scriptDir, "..", "state.json")
	flag.StringVar(&config.StateFile, "state-file", defaultStateFile, "State file path")

	flag.Parse()
	return config
}

// createLogger creates a zap logger with the specified level
func createLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		return nil, fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", level)
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(zapLevel),
		Development: level == "debug",
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}

// setupLogger creates and returns a logger with proper error handling
func setupLogger(logLevel string) (*zap.Logger, error) {
	logger, err := createLogger(logLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	return logger, nil
}

// buildSubdomainURL creates a full URL for a subdomain of the given base domain
func buildSubdomainURL(baseDomain, subdomain string) string {
	// Ensure baseDomain doesn't have protocol prefix
	baseDomain = strings.TrimPrefix(baseDomain, "http://")
	baseDomain = strings.TrimPrefix(baseDomain, "https://")

	// Remove trailing slash
	baseDomain = strings.TrimSuffix(baseDomain, "/")

	return fmt.Sprintf("https://%s.%s", subdomain, baseDomain)
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

	// Create manifest creator
	manifestCreator := stream.NewManifestCreator()

	// Create stream creator
	streamCreator := stream.NewStreamCreator(manifestCreator)

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

// saveState saves the current state to a JSON file
func saveState(state *State, stateFile string, logger *zap.Logger) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	err = os.WriteFile(stateFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	logger.Info("State saved", zap.String("file", stateFile))
	return nil
}

// loadState loads state from a JSON file
func loadState(stateFile string, logger *zap.Logger) (*State, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	logger.Info("State loaded", zap.String("file", stateFile))
	return &state, nil
}

// updateState updates specific fields in the state and saves it
func updateState(state *State, stateFile string, logger *zap.Logger, updates map[string]interface{}) error {
	for key, value := range updates {
		switch key {
		case "timestamp":
			if str, ok := value.(string); ok {
				state.Timestamp = str
			} else {
				return fmt.Errorf("invalid type for timestamp: expected string, got %T", value)
			}
		case "original_sha256":
			if str, ok := value.(string); ok {
				state.OriginalSHA256 = str
			} else {
				return fmt.Errorf("invalid type for original_sha256: expected string, got %T", value)
			}
		case "sd_blob_hash":
			if str, ok := value.(string); ok {
				state.SDBlobHash = str
			} else {
				return fmt.Errorf("invalid type for sd_blob_hash: expected string, got %T", value)
			}
		case "content_blob_hashes":
			if slice, ok := value.([]string); ok {
				state.ContentBlobHashes = slice
			} else {
				return fmt.Errorf("invalid type for content_blob_hashes: expected []string, got %T", value)
			}
		case "upload_hash":
			if str, ok := value.(string); ok {
				state.UploadHash = str
			} else {
				return fmt.Errorf("invalid type for upload_hash: expected string, got %T", value)
			}
		case "status":
			if str, ok := value.(string); ok {
				state.Status = str
			} else {
				return fmt.Errorf("invalid type for status: expected string, got %T", value)
			}
		case "account_email":
			if str, ok := value.(string); ok {
				state.AccountEmail = str
			} else {
				return fmt.Errorf("invalid type for account_email: expected string, got %T", value)
			}
		case "account_password":
			if str, ok := value.(string); ok {
				state.AccountPassword = str
			} else {
				return fmt.Errorf("invalid type for account_password: expected string, got %T", value)
			}
		case "account_first_name":
			if str, ok := value.(string); ok {
				state.AccountFirstName = str
			} else {
				return fmt.Errorf("invalid type for account_first_name: expected string, got %T", value)
			}
		case "account_last_name":
			if str, ok := value.(string); ok {
				state.AccountLastName = str
			} else {
				return fmt.Errorf("invalid type for account_last_name: expected string, got %T", value)
			}
		case "device_name":
			if str, ok := value.(string); ok {
				state.DeviceName = str
			} else {
				return fmt.Errorf("invalid type for device_name: expected string, got %T", value)
			}

		default:
			return fmt.Errorf("unknown state field: %s", key)
		}
	}

	return saveState(state, stateFile, logger)
}

// registerAccountAndDevice registers a new account and device, storing info in state
func registerAccountAndDevice(state *State, stateFile string, portalURL string, logger *zap.Logger) error {
	logger.Info("Starting account and device registration")

	// Build portal domain URLs from configured portal URL (using subdomain approach like post-upload)
	accountBaseURL := buildSubdomainURL(portalURL, accountSubdomain)
	lbryBaseURL := buildSubdomainURL(portalURL, lbrySubdomain)

	// Create LBRY portal client
	client, err := shared.NewLBRYPortalClient(shared.LBRYPortalClientConfig{
		AccountBaseURL: accountBaseURL,
		LBRYBaseURL:    lbryBaseURL,
	})
	if err != nil {
		return fmt.Errorf("failed to create LBRY portal client: %w", err)
	}

	// Create and login to account
	fakeAccount := shared.GenerateFakeAccount()
	logger.Info("Creating account",
		zap.String("email", fakeAccount.Email),
		zap.String("first_name", fakeAccount.FirstName),
		zap.String("last_name", fakeAccount.LastName))

	err = client.RegisterUser(fakeAccount.Email, fakeAccount.Password, fakeAccount.FirstName, fakeAccount.LastName)
	if err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}

	logger.Info("Account created successfully, logging in...")
	err = client.Login(fakeAccount.Email, fakeAccount.Password)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}

	logger.Info("Login successful!")

	// Register device
	deviceName := "stream-uploader-device"
	logger.Info("Fetching public IP address for device registration...")
	publicIP, err := shared.GetPublicIP()
	if err != nil {
		logger.Warn("Failed to fetch public IP, using localhost", zap.Error(err))
		publicIP = "127.0.0.1"
	}

	logger.Info("Registering device...",
		zap.String("device_name", deviceName),
		zap.String("ip_address", publicIP))
	err = client.RegisterDevice(deviceName, publicIP)
	if err != nil {
		return fmt.Errorf("failed to register device: %w", err)
	}
	logger.Info("Device registered successfully!")

	// Update state with account and device information
	err = updateState(state, stateFile, logger, map[string]interface{}{
		"account_email":      fakeAccount.Email,
		"account_password":   fakeAccount.Password,
		"account_first_name": fakeAccount.FirstName,
		"account_last_name":  fakeAccount.LastName,
		"device_name":        deviceName,
		"status":             "account_registered",
	})
	if err != nil {
		return fmt.Errorf("failed to update state with account info: %w", err)
	}

	logger.Info("Account and device registration completed successfully")
	return nil
}

// setupBlobDownloader creates and configures a blob downloader for stream verification
func setupBlobDownloader(portalURL string, logger *zap.Logger) (*shared.BlobDownloader, error) {
	logger.Info("Setting up blob downloader for stream verification...")

	// Get free ports for DHT and peer
	dhtPort, err := shared.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get free DHT port: %w", err)
	}

	peerPort, err := shared.GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get free peer port: %w", err)
	}

	dhtAddress := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", dhtPort))
	peerAddress := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", peerPort))

	// Default fixed peers to portal domain at port 5567
	fixedPeer := net.JoinHostPort(portalURL, "5567")
	// Seed nodes use port 4444
	seedNode := net.JoinHostPort(portalURL, "4444")

	downloaderConfig := shared.BlobDownloaderConfig{
		DHTAddress:  dhtAddress,
		PeerAddress: peerAddress,
		FixedPeers:  []string{fixedPeer},
		Logger:      logger,
		SeedNodes:   []string{seedNode},
		MaxPeers:    5,
		Timeout:     30 * time.Second,
	}

	downloader, err := shared.NewBlobDownloader(downloaderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob downloader: %w", err)
	}

	return downloader, nil
}

// downloadAndVerifyStream downloads a stream and verifies its SHA256 hash against the original
func downloadAndVerifyStream(downloader *shared.BlobDownloader, sdHash, originalSHA256 string, logger *zap.Logger) error {
	logger.Info("Downloading stream for verification", zap.String("sd_hash", sdHash))

	downloadedData, err := downloader.DownloadStream(context.Background(), sdHash)
	if err != nil {
		return fmt.Errorf("failed to download stream: %w", err)
	}

	// Calculate SHA256 of downloaded data
	downloadedHash := sha256.Sum256(downloadedData)
	downloadedHashStr := hex.EncodeToString(downloadedHash[:])

	logger.Info("Stream downloaded successfully",
		zap.Int("downloaded_size_bytes", len(downloadedData)),
		zap.String("downloaded_sha256", downloadedHashStr))

	// Verify SHA256 matches original
	if originalSHA256 == "" {
		logger.Warn("No original SHA256 found in state for verification")
		return nil
	}

	if downloadedHashStr == originalSHA256 {
		logger.Info("SUCCESS: Downloaded data SHA256 matches original!",
			zap.String("original_sha256", originalSHA256),
			zap.String("downloaded_sha256", downloadedHashStr))
		fmt.Printf("=== BLOB VERIFICATION SUCCESS ===\n")
		fmt.Printf("Original SHA256: %s\n", originalSHA256)
		fmt.Printf("Downloaded SHA256: %s\n", downloadedHashStr)
		fmt.Printf("Data Size: %d bytes\n", len(downloadedData))
		fmt.Printf("================================\n")
	} else {
		logger.Error("FAILURE: Downloaded data SHA256 does not match original!",
			zap.String("original_sha256", originalSHA256),
			zap.String("downloaded_sha256", downloadedHashStr))
		fmt.Printf("=== BLOB VERIFICATION FAILURE ===\n")
		fmt.Printf("Original SHA256: %s\n", originalSHA256)
		fmt.Printf("Downloaded SHA256: %s\n", downloadedHashStr)
		fmt.Printf("Data Size: %d bytes\n", len(downloadedData))
		fmt.Printf("=================================\n")
	}

	return nil
}

// waitForOperations waits for all account operations to complete
func waitForOperations(client *shared.LBRYPortalClient, logger *zap.Logger) error {
	logger.Info("Waiting for all operations to complete...")

	err := client.WaitForAllOperations()
	if err != nil {
		return fmt.Errorf("failed while waiting for operations: %w", err)
	}

	logger.Info("All operations completed successfully!")

	// List streams to verify completion
	logger.Info("Listing streams to verify completion...")
	streams, err := client.ListStreams()
	if err != nil {
		return fmt.Errorf("failed to list streams: %w", err)
	}

	logger.Info("Found streams", zap.Int("count", len(streams.Data)))
	for i, _stream := range streams.Data {
		logger.Info("Stream",
			zap.Int("index", i),
			zap.Int("id", _stream.ID),
			zap.String("sd_hash", _stream.SDHash),
			zap.String("stream_hash", _stream.StreamHash))
	}

	return nil
}

// reloginAndWait relogs in using saved credentials and waits for operations
func reloginAndWait(state *State, portalURL string, logger *zap.Logger) error {
	logger.Info("Starting relogin and wait mode")

	// Build portal domain URLs
	accountBaseURL := buildSubdomainURL(portalURL, accountSubdomain)
	lbryBaseURL := buildSubdomainURL(portalURL, lbrySubdomain)

	// Create LBRY portal client
	client, err := shared.NewLBRYPortalClient(shared.LBRYPortalClientConfig{
		AccountBaseURL: accountBaseURL,
		LBRYBaseURL:    lbryBaseURL,
	})
	if err != nil {
		return fmt.Errorf("failed to create LBRY portal client: %w", err)
	}

	// Validate that we have account credentials
	if state.AccountEmail == "" || state.AccountPassword == "" {
		return fmt.Errorf("account credentials not found in state file")
	}

	logger.Info("Relogging in with saved credentials",
		zap.String("email", state.AccountEmail))

	err = client.Login(state.AccountEmail, state.AccountPassword)
	if err != nil {
		return fmt.Errorf("failed to relogin: %w", err)
	}

	logger.Info("Relogin successful!")

	// Wait 30 seconds before monitoring operations
	logger.Info("Waiting 5 seconds before monitoring operations...")
	time.Sleep(5 * time.Second)

	// Wait for all operations to complete
	err = waitForOperations(client, logger)
	if err != nil {
		return fmt.Errorf("failed to wait for operations: %w", err)
	}

	// List streams and output the first found stream
	logger.Info("Listing streams to find first stream...")
	streams, err := client.ListStreams()
	if err != nil {
		return fmt.Errorf("failed to list streams: %w", err)
	}

	if len(streams.Data) == 0 {
		logger.Warn("No streams found")
	} else {
		firstStream := streams.Data[0]
		logger.Info("First found stream",
			zap.Int("id", firstStream.ID),
			zap.String("sd_hash", firstStream.SDHash),
			zap.String("stream_hash", firstStream.StreamHash))

		// Verify SD hash matches our uploaded stream
		if state.SDBlobHash != "" {
			if firstStream.SDHash == state.SDBlobHash {
				logger.Info("SD hash matches our uploaded stream!",
					zap.String("expected_sd_hash", state.SDBlobHash),
					zap.String("found_sd_hash", firstStream.SDHash))
			} else {
				logger.Warn("SD hash does not match our uploaded stream",
					zap.String("expected_sd_hash", state.SDBlobHash),
					zap.String("found_sd_hash", firstStream.SDHash))
			}
		}

		// Output the first stream details to console
		fmt.Printf("=== FIRST FOUND STREAM ===\n")
		fmt.Printf("ID: %d\n", firstStream.ID)
		fmt.Printf("SD Hash: %s\n", firstStream.SDHash)
		fmt.Printf("Stream Hash: %s\n", firstStream.StreamHash)
		fmt.Printf("============================\n")

		// Fetch the blob and verify SHA hash like post-upload does
		downloader, err := setupBlobDownloader(portalURL, logger)
		if err != nil {
			logger.Warn("Failed to setup blob downloader, skipping blob verification", zap.Error(err))
			return nil
		}
		defer downloader.Close()

		err = downloadAndVerifyStream(downloader, firstStream.SDHash, state.OriginalSHA256, logger)
		if err != nil {
			logger.Warn("Failed to download and verify stream", zap.Error(err))
			return nil
		}
	}

	logger.Info("Relogin and wait mode completed successfully")
	return nil
}

func main() {
	// Parse command-line flags
	config := parseFlags()

	// Setup logger with proper error handling
	logger, err := setupLogger(config.LogLevel)
	if err != nil {
		log.Fatalf("Failed to setup logger: %v", err)
	}

	logger.Info("Starting LBRY stream uploader",
		zap.String("reflector_address", config.ReflectorAddress),
		zap.String("log_level", config.LogLevel),
		zap.String("state_file", config.StateFile),
		zap.Bool("wait_mode", config.WaitMode),
		zap.Int("blob_size_mb", blobSize/(1024*1024)),
	)

	// Handle wait mode
	if config.WaitMode {
		logger.Info("Running in wait mode")

		// Load existing state
		state, err := loadState(config.StateFile, logger)
		if err != nil {
			logger.Fatal("Failed to load state file", zap.Error(err))
		}

		// Relogin and wait for operations
		err = reloginAndWait(state, config.PortalURL, logger)
		if err != nil {
			logger.Fatal("Failed to relogin and wait", zap.Error(err))
		}

		logger.Info("Wait mode completed successfully")
		return
	}

	// Upload mode (original behavior)
	logger.Info("Running in upload mode")

	// Initialize state
	state := &State{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status:    "initializing",
	}

	// Save initial state
	err = saveState(state, config.StateFile, logger)
	if err != nil {
		logger.Fatal("Failed to save initial state", zap.Error(err))
	}

	// Register account and device
	err = registerAccountAndDevice(state, config.StateFile, config.PortalURL, logger)
	if err != nil {
		logger.Fatal("Failed to register account and device", zap.Error(err))
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
		logger.Fatal("Failed to generate crypto-random buffer", zap.Error(err))
	}

	// Update state with original SHA256
	err = updateState(state, config.StateFile, logger, map[string]interface{}{
		"original_sha256": blobHash,
		"status":          "generated_data",
	})
	if err != nil {
		logger.Fatal("Failed to update state with SHA256", zap.Error(err))
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
		logger.Fatal("Failed to create stream", zap.Error(err))
	}

	// Update state with stream information
	err = updateState(state, config.StateFile, logger, map[string]interface{}{
		"sd_blob_hash":        streamResult.SDBlobHash,
		"content_blob_hashes": streamResult.ContentHashes,
		"upload_hash":         streamResult.StreamHash,
		"status":              "stream_created",
	})
	if err != nil {
		logger.Fatal("Failed to update state with stream info", zap.Error(err))
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
		updateErr := updateState(state, config.StateFile, logger, map[string]interface{}{
			"status": "upload_failed",
		})
		if updateErr != nil {
			logger.Warn("Failed to update state with upload failed status", zap.Error(updateErr))
		}
		logger.Fatal("Failed to upload stream to reflector", zap.Error(err))
	}

	// Update state with successful upload
	err = updateState(state, config.StateFile, logger, map[string]interface{}{
		"status": "uploaded",
	})
	if err != nil {
		logger.Warn("Failed to update state with upload status", zap.Error(err))
	}

	logger.Info("Stream uploaded successfully to reflector",
		zap.String("stream_hash", streamResult.StreamHash),
		zap.String("sd_blob_hash", streamResult.SDBlobHash),
		zap.Int("total_blobs", streamResult.TotalChunks))

	logger.Info("Stream uploader completed successfully")
}
