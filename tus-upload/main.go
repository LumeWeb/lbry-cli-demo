package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"time"

	"github.com/docker/go-units"
	"github.com/lumeweb/lbry-cli-demo/shared"
	"go.uber.org/zap"
)

// Configuration
const (
	blobSize     = units.MiB * 100
	uploadFile   = "upload_blob.bin"
	downloadFile = "download_blob.bin"
	stateFile    = "tus_upload.json"
)

// StreamState holds the upload state information
type StreamState struct {
	Timestamp      string   `json:"timestamp"`
	SDHash         string   `json:"sd_hash"`
	OriginalSHA256 string   `json:"original_sha256"`
	Status         string   `json:"status"`
	ContentHashes  []string `json:"content_hashes,omitempty"`
	SDBlobSize     int      `json:"sd_blob_size,omitempty"`
}

func main() {
	ctx := context.Background()

	// Create demo framework
	framework, err := shared.NewDemoFrameworkFromFlags("TUS-UPLOAD")
	if err != nil {
		panic(err)
	}
	defer framework.Cleanup()

	logger := framework.GetStdLogger()
	zapLogger := framework.GetLogger()
	client := framework.GetClient()
	accountManager := framework.GetAccountManager()

	// Login or create account and cleanup existing streams
	_, err = accountManager.LoginOrCreateAccount()
	if err != nil {
		zapLogger.Fatal("Failed to login or create account", zap.Error(err))
	}

	// Cleanup all existing streams before starting
	err = framework.CleanupAllStreams()
	if err != nil {
		zapLogger.Warn("Failed to cleanup existing streams", zap.Error(err))
	}

	// Generate 100MB random data blob
	data, _, err := framework.GenerateRandomBlob(blobSize)
	if err != nil {
		logger.Fatalf("Failed to generate random data: %v", err)
	}

	// Calculate and store original SHA256 hash
	hash := sha256.Sum256(data)
	originalSHA256 := hex.EncodeToString(hash[:])
	logger.Printf("Original data SHA256: %s", originalSHA256)

	// Write to disk
	err = framework.WriteBlobToFile(data, uploadFile)
	if err != nil {
		logger.Fatalf("Failed to write upload file: %v", err)
	}

	// Upload the stream using TUS
	logger.Printf("Uploading stream with TUS...")
	file, err := os.Open(uploadFile)
	if err != nil {
		logger.Fatalf("Failed to open upload file: %v", err)
	}
	defer file.Close()

	// Prepare metadata for the stream
	metadata := shared.StreamMetadataRequest{
		StreamName:        "test_stream",
		SuggestedFileName: "test_stream.bin",
	}

	err = client.UploadStreamWithTUS(file, metadata)
	if err != nil {
		logger.Fatalf("Failed to upload stream with TUS: %v", err)
	}

	logger.Printf("Stream uploaded successfully with TUS!")

	// Wait for all operations to complete
	err = accountManager.WaitForOperations()
	if err != nil {
		logger.Fatalf("Failed while waiting for operations: %v", err)
	}

	// List streams to get the stream hash
	streams, err := accountManager.ListAndLogStreams()
	if err != nil {
		logger.Fatalf("Failed to list streams: %v", err)
	}

	var uploadedStream *shared.StreamResponse
	if len(streams) > 0 {
		uploadedStream = &streams[0]
	}

	if uploadedStream == nil {
		logger.Fatalf("No streams available")
	}

	logger.Printf("Found stream: ID=%d, SDHash=%s, StreamHash=%s",
		uploadedStream.ID, uploadedStream.SDHash, uploadedStream.StreamHash)

	// Get SD blob and extract content hashes using unified method
	logger.Printf("Getting SD blob to extract content hashes...")
	sdBlob, contentHashes, err := framework.GetSDBlobAndExtractHashes(ctx, uploadedStream.SDHash)
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
	stateManager := framework.GetStateManager()
	var sdBlobSize int
	if sdBlob != nil {
		sdBlobSize = len(sdBlob.BlobInfos)
	}
	streamState := StreamState{
		Timestamp:      time.Now().Format(time.RFC3339),
		SDHash:         uploadedStream.SDHash,
		OriginalSHA256: originalSHA256,
		Status:         "uploaded",
		SDBlobSize:     sdBlobSize,
		ContentHashes:  contentHashes,
	}

	err = stateManager.SaveJSON(stateFile, streamState)
	if err != nil {
		zapLogger.Warn("Failed to save stream state for cleanup", zap.Error(err))
	} else {
		zapLogger.Info("Stream state saved for cleanup",
			zap.String("state_file", stateFile),
			zap.String("sd_hash", uploadedStream.SDHash),
			zap.Int("sd_blob_size", sdBlobSize),
			zap.Int("content_hashes_count", len(contentHashes)))
	}

	// Download and verify stream using framework
	downloadedData, err := framework.DownloadAndVerifyStream(ctx, uploadedStream.SDHash, data)
	if err != nil {
		logger.Fatalf("Failed to download and verify stream: %v", err)
	}

	// Save downloaded data
	err = framework.WriteBlobToFile(downloadedData, downloadFile)
	if err != nil {
		logger.Fatalf("Failed to write download file: %v", err)
	}

	logger.Printf("Program completed successfully!")
}
