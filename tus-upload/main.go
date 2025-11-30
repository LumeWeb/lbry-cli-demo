package main

import (
	"context"
	"os"

	"github.com/docker/go-units"
	"github.com/lumeweb/lbry-cli-demo/shared"
	"go.uber.org/zap"
)

// Configuration
const (
	blobSize     = units.MiB * 100
	uploadFile   = "upload_blob.bin"
	downloadFile = "download_blob.bin"
)

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

	// Cleanup stream
	framework.CleanupStream(uploadedStream.SDHash)

	logger.Printf("Program completed successfully!")
}
