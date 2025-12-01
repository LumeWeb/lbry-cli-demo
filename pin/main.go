package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"log"
	"time"

	"github.com/lumeweb/lbry-cli-demo/shared"
	"go.uber.org/zap"
)

// Configuration
const (
	// SD hash for the stream we want to pin/download/unpin
	targetSDHash = "acc6adf8b4f10dcddffc5c2ca87dbd9cb3a2664564695ac7aaab038193ff14a280cc3d4ebae55c71d0b885a7316d0137"

	// Expected SHA256 hash of the downloaded PDF file
	expectedPDFHash = "71ee4373bcdbafbe10e842facdb4546a9f8072ebbf6a03550ae8638bb90a5916"

	// State file for cleanup
	stateFile = "pin.json"
)

// Mode variables
var (
	mode = flag.String("mode", "pin", "Operation mode: pin or unpin")
)

// StreamState holds the pin state information
type StreamState struct {
	Timestamp      string   `json:"timestamp"`
	SDHash         string   `json:"sd_hash"`
	OriginalSHA256 string   `json:"original_sha256"`
	Status         string   `json:"status"`
	ContentHashes  []string `json:"content_hashes,omitempty"`
	SDBlobSize     int      `json:"sd_blob_size,omitempty"`
}

// validatePDFHash calculates and validates the SHA256 hash of the downloaded PDF data
func validatePDFHash(data []byte, logger *log.Logger, zapLogger *zap.Logger) bool {
	hash := sha256.Sum256(data)
	calculatedHash := hex.EncodeToString(hash[:])

	zapLogger.Info("Validating PDF hash",
		zap.String("expected", expectedPDFHash),
		zap.String("calculated", calculatedHash))

	if calculatedHash == expectedPDFHash {
		zapLogger.Info("SUCCESS: PDF hash validation passed!")
		logger.Printf("SUCCESS: PDF hash validation passed!")
		return true
	} else {
		zapLogger.Error("FAILURE: PDF hash validation failed!",
			zap.String("expected", expectedPDFHash),
			zap.String("calculated", calculatedHash))
		logger.Printf("FAILURE: PDF hash validation failed!")
		logger.Printf("Expected: %s", expectedPDFHash)
		logger.Printf("Calculated: %s", calculatedHash)
		return false
	}
}

func main() {
	// Parse command line flags
	flag.Parse()

	ctx := context.Background()

	// Create demo framework
	framework, err := shared.NewDemoFrameworkFromFlags("PIN")
	if err != nil {
		panic(err)
	}
	defer framework.Cleanup()

	logger := framework.GetStdLogger()
	zapLogger := framework.GetLogger()
	client := framework.GetClient()
	accountManager := framework.GetAccountManager()
	stateManager := framework.GetStateManager()

	// Execute based on mode
	switch *mode {
	case "unpin", "pin":
		executeMode(ctx, framework, logger, zapLogger, client, accountManager, stateManager)
	default:
		zapLogger.Fatal("Invalid mode. Use 'pin' or 'unpin'")
	}
}

// executeMode handles both pin and unpin operations with shared setup
func executeMode(ctx context.Context, framework *shared.DemoFramework, logger *log.Logger, zapLogger *zap.Logger, client *shared.LBRYPortalClient, accountManager *shared.AccountManager, stateManager *shared.StateManager) {
	// Common setup for both modes
	zapLogger.Info("Starting operation", zap.String("mode", *mode))

	// Login or create account
	_, err := accountManager.LoginOrCreateAccount()
	if err != nil {
		zapLogger.Fatal("Failed to login or create account", zap.Error(err))
	}

	// Execute mode-specific operations
	switch *mode {
	case "unpin":
		executeUnpinOperations(ctx, framework, logger, zapLogger, client, accountManager, stateManager)
	case "pin":
		executePinOperations(ctx, framework, logger, zapLogger, client, accountManager, stateManager)
	}

	// Wait for all operations to complete
	err = accountManager.WaitForOperations()
	if err != nil {
		zapLogger.Fatal("Failed while waiting for operations:", zap.Error(err))
	}

	zapLogger.Info("Operation completed successfully!", zap.String("mode", *mode))
}

// executeUnpinOperations handles unpin-specific logic
func executeUnpinOperations(ctx context.Context, framework *shared.DemoFramework, logger *log.Logger, zapLogger *zap.Logger, client *shared.LBRYPortalClient, accountManager *shared.AccountManager, stateManager *shared.StateManager) {
	// Load existing stream state
	var streamState StreamState
	err := stateManager.LoadJSON(stateFile, &streamState)
	if err != nil {
		zapLogger.Fatal("Failed to load stream state file", zap.Error(err))
	}

	zapLogger.Info("Unpinning stream",
		zap.String("sd_hash", streamState.SDHash),
		zap.String("state_file", stateFile))

	// Unpin the stream using framework's CleanupStream method
	framework.CleanupStream(streamState.SDHash)

	// Delete the state file
	err = stateManager.DeleteStateFile(stateFile)
	if err != nil {
		zapLogger.Warn("Failed to delete state file", zap.Error(err))
	} else {
		zapLogger.Info("State file deleted successfully", zap.String("state_file", stateFile))
	}
}

// executePinOperations handles pin-specific logic
func executePinOperations(ctx context.Context, framework *shared.DemoFramework, logger *log.Logger, zapLogger *zap.Logger, client *shared.LBRYPortalClient, accountManager *shared.AccountManager, stateManager *shared.StateManager) {
	zapLogger.Info("Starting pin demo", zap.String("sd_hash", targetSDHash))

	// Cleanup all existing streams before starting
	err := framework.CleanupAllStreams()
	if err != nil {
		zapLogger.Warn("Failed to cleanup existing streams", zap.Error(err))
	}

	// Pin the stream
	zapLogger.Info("Pinning stream", zap.String("sd_hash", targetSDHash))
	err = client.PinStream(targetSDHash)
	if err != nil {
		zapLogger.Fatal("Failed to pin stream:", zap.Error(err))
	}
	zapLogger.Info("Stream pinned successfully!")

	// Get SD blob and extract content hashes using unified method
	logger.Printf("Getting SD blob to extract content hashes...")
	sdBlob, contentHashes, err := framework.GetSDBlobAndExtractHashes(ctx, targetSDHash)
	if err != nil {
		logger.Printf("Failed to get SD blob or extract hashes: %v", err)
		contentHashes = []string{} // Ensure it's initialized
	} else {
		logger.Printf("Successfully extracted %d content hashes", len(contentHashes))
		for i, hash := range contentHashes {
			logger.Printf("Content hash %d: %s", i+1, hash)
		}
	}

	// Save stream state for cleanup
	var sdBlobSize int
	if sdBlob != nil {
		sdBlobSize = len(sdBlob.BlobInfos)
	}
	streamState := StreamState{
		Timestamp:      time.Now().Format(time.RFC3339),
		SDHash:         targetSDHash,
		OriginalSHA256: expectedPDFHash,
		Status:         "pinned",
		SDBlobSize:     sdBlobSize,
		ContentHashes:  contentHashes,
	}

	err = stateManager.SaveJSON(stateFile, streamState)
	if err != nil {
		zapLogger.Warn("Failed to save stream state for cleanup", zap.Error(err))
	} else {
		zapLogger.Info("Stream state saved for cleanup",
			zap.String("state_file", stateFile),
			zap.String("sd_hash", targetSDHash),
			zap.Int("sd_blob_size", sdBlobSize),
			zap.Int("content_hashes_count", len(contentHashes)))
	}

	// Download the stream using the predefined SD hash
	zapLogger.Info("Downloading stream", zap.String("sd_hash", targetSDHash))
	downloadedData, err := framework.DownloadAndVerifyStream(ctx, targetSDHash, nil)
	if err != nil {
		logger.Fatalf("Failed to download stream: %v", err)
	}

	// Save downloaded data (will be a PDF)
	downloadFile := "downloaded_stream.pdf"
	err = framework.WriteBlobToFile(downloadedData, downloadFile)
	if err != nil {
		logger.Fatalf("Failed to write download file: %v", err)
	}
	zapLogger.Info("Downloaded stream saved", zap.String("file", downloadFile))

	// Verify the downloaded data size
	zapLogger.Info("Downloaded data size", zap.Int("bytes", len(downloadedData)))

	// Validate the PDF hash
	if !validatePDFHash(downloadedData, logger, zapLogger) {
		logger.Fatalf("PDF hash validation failed")
	}
}
