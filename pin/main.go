package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log"

	"github.com/lumeweb/lbry-cli-demo/shared"
	"go.uber.org/zap"
)

// Configuration
const (
	// SD hash for the stream we want to pin/download/unpin
	targetSDHash = "acc6adf8b4f10dcddffc5c2ca87dbd9cb3a2664564695ac7aaab038193ff14a280cc3d4ebae55c71d0b885a7316d0137"

	// Expected SHA256 hash of the downloaded PDF file
	expectedPDFHash = "71ee4373bcdbafbe10e842facdb4546a9f8072ebbf6a03550ae8638bb90a5916"
)

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

	zapLogger.Info("Starting pin demo",
		zap.String("sd_hash", targetSDHash))

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

	// Pin the stream
	zapLogger.Info("Pinning stream", zap.String("sd_hash", targetSDHash))
	err = client.PinStream(targetSDHash)
	if err != nil {
		zapLogger.Fatal("Failed to pin stream:", zap.Error(err))
	}
	zapLogger.Info("Stream pinned successfully!")

	// Wait for all operations to complete
	err = accountManager.WaitForOperations()
	if err != nil {
		zapLogger.Fatal("Failed while waiting for operations:", zap.Error(err))
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

	// Cleanup stream
	framework.CleanupStream(targetSDHash)

	zapLogger.Info("Pin demo completed successfully!")
}
