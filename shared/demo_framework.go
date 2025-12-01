package shared

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	"go.lumeweb.com/liblbry/stream"
	"go.uber.org/zap"
)

// DemoFramework provides common demo workflow patterns
type DemoFramework struct {
	config         *AppConfig
	portals        PortalURLs
	client         *LBRYPortalClient
	accountManager *AccountManager
	stateManager   *StateManager
	logger         *zap.Logger
}

// NewDemoFrameworkFromFlags creates a new demo framework instance by parsing command-line flags
func NewDemoFrameworkFromFlags(appName string) (*DemoFramework, error) {
	portalDomain, logLevel := ParseAccountFlags()
	flag.Parse()
	return NewDemoFramework(*portalDomain, *logLevel, appName)
}

// NewDemoFramework creates a new demo framework instance
func NewDemoFramework(portalDomain, logLevelStr, appName string) (*DemoFramework, error) {
	// Setup app configuration
	config, err := SetupApp(portalDomain, logLevelStr, appName)
	if err != nil {
		return nil, fmt.Errorf("failed to setup app: %w", err)
	}

	// Build portal URLs
	portals := BuildPortalURLs(portalDomain)

	// Create LBRY portal client
	client, err := NewLBRYPortalClient(LBRYPortalClientConfig{
		AccountBaseURL: portals.AccountBaseURL,
		LBRYBaseURL:    portals.LBRYBaseURL,
		RootDomain:     portals.RootDomain,
	})
	if err != nil {
		CleanupApp(config)
		return nil, fmt.Errorf("failed to create LBRY portal client: %w", err)
	}

	// Create account manager
	accountManager, err := NewAccountManager(client, config.Logger)
	if err != nil {
		CleanupApp(config)
		return nil, fmt.Errorf("failed to create account manager: %w", err)
	}

	// Create state manager
	stateManager, err := NewStateManager(config.Logger)
	if err != nil {
		CleanupApp(config)
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	return &DemoFramework{
		config:         config,
		portals:        portals,
		client:         client,
		accountManager: accountManager,
		stateManager:   stateManager,
		logger:         config.Logger,
	}, nil
}

// GetClient returns the LBRY portal client
func (df *DemoFramework) GetClient() *LBRYPortalClient {
	return df.client
}

// GetAccountManager returns the account manager
func (df *DemoFramework) GetAccountManager() *AccountManager {
	return df.accountManager
}

// GetStateManager returns the state manager
func (df *DemoFramework) GetStateManager() *StateManager {
	return df.stateManager
}

// GetStateDir returns the state directory path
func (df *DemoFramework) GetStateDir() string {
	return df.stateManager.GetStateDir()
}

// LoginOrCreateAccount attempts to login with existing account or creates a new one
func (df *DemoFramework) LoginOrCreateAccount() (*FakeAccountData, error) {
	return df.accountManager.LoginOrCreateAccount()
}

// CleanupAllStreams removes all streams for the current account
func (df *DemoFramework) CleanupAllStreams() error {
	return df.accountManager.CleanupAllStreams()
}

// GetLogger returns the zap logger
func (df *DemoFramework) GetLogger() *zap.Logger {
	return df.logger
}

// GetStdLogger returns the standard logger
func (df *DemoFramework) GetStdLogger() *log.Logger {
	return df.config.StdLogger
}

// GetConfig returns the app configuration
func (df *DemoFramework) GetConfig() *AppConfig {
	return df.config
}

// Cleanup performs cleanup of framework resources
func (df *DemoFramework) Cleanup() {
	CleanupApp(df.config)
}

// GenerateRandomBlob generates a random data blob of specified size
func (df *DemoFramework) GenerateRandomBlob(size int) ([]byte, string, error) {
	df.logger.Info("Generating random data blob", zap.Int("size_bytes", size))

	data := make([]byte, size)
	_, err := rand.Read(data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate random data: %w", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	df.logger.Info("Generated blob", zap.String("sha256", hashStr))
	return data, hashStr, nil
}

// WriteBlobToFile writes blob data to a file
func (df *DemoFramework) WriteBlobToFile(data []byte, filename string) error {
	err := os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filename, err)
	}

	df.config.StdLogger.Printf("Blob saved to: %s", filename)
	return nil
}

// VerifyBlobIntegrity verifies that two blobs match
func (df *DemoFramework) VerifyBlobIntegrity(original, downloaded []byte) bool {
	df.logger.Info("Verifying blob integrity",
		zap.Int("original_size", len(original)),
		zap.Int("downloaded_size", len(downloaded)))

	matches := bytes.Equal(original, downloaded)

	if matches {
		df.logger.Info("SUCCESS: Data matches exactly!")
		df.config.StdLogger.Printf("SUCCESS: Data matches exactly!")
	} else {
		df.logger.Error("FAILURE: Data does not match!")
		df.config.StdLogger.Printf("FAILURE: Data does not match!")
	}

	return matches
}

// GetSDBlob retrieves just the SD blob content for a given SD hash
func (df *DemoFramework) GetSDBlob(ctx context.Context, sdHash string) (*stream.SDBlob, error) {
	// Setup network configuration
	networkConfig, err := SetupBlobDownloaderConfig(df.config.PortalDomain, df.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to setup network config: %w", err)
	}

	// Create blob downloader
	downloader, err := CreateBlobDownloader(networkConfig, df.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob downloader: %w", err)
	}
	defer downloader.Close()

	// Get SD blob
	df.logger.Info("Getting SD blob", zap.String("sd_hash", sdHash))
	sdBlob, err := downloader.GetSDBlob(ctx, sdHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get SD blob: %w", err)
	}

	return sdBlob, nil
}

// ParseSDBlobAndExtractHashes parses SD blob struct and extracts content blob hashes
func (df *DemoFramework) ParseSDBlobAndExtractHashes(sdBlob *stream.SDBlob) ([]string, error) {
	df.logger.Info("Parsing SD blob to extract content hashes", zap.Int("blob_count", len(sdBlob.BlobInfos)))

	// Extract content hashes from BlobInfos
	var contentHashes []string
	for i, blobInfo := range sdBlob.BlobInfos {
		if len(blobInfo.BlobHash) > 0 {
			hashHex := hex.EncodeToString(blobInfo.BlobHash)
			contentHashes = append(contentHashes, hashHex)
			df.logger.Debug("Extracted content hash",
				zap.Int("blob_index", i),
				zap.String("blob_hash", hashHex),
				zap.Int("blob_length", blobInfo.Length))
		} else {
			df.logger.Debug("Skipping zero-length blob", zap.Int("blob_index", i))
		}
	}

	df.logger.Info("Successfully extracted content hashes",
		zap.Int("total_blobs", len(sdBlob.BlobInfos)),
		zap.Int("content_hashes", len(contentHashes)))

	return contentHashes, nil
}

// GetSDBlobAndExtractHashes is a convenience method that gets an SD blob and extracts content hashes
func (df *DemoFramework) GetSDBlobAndExtractHashes(ctx context.Context, sdHash string) (*stream.SDBlob, []string, error) {
	df.logger.Info("Getting SD blob to extract content hashes", zap.String("sd_hash", sdHash))

	// Get SD blob
	sdBlob, err := df.GetSDBlob(ctx, sdHash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get SD blob: %w", err)
	}

	df.logger.Info("Successfully retrieved SD blob")

	// Parse SD blob to extract content hashes
	contentHashes, err := df.ParseSDBlobAndExtractHashes(sdBlob)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse SD blob: %w", err)
	}

	df.logger.Info("Successfully extracted content hashes",
		zap.Int("hash_count", len(contentHashes)))

	for i, hash := range contentHashes {
		df.logger.Debug("Content hash",
			zap.Int("index", i+1),
			zap.String("hash", hash))
	}

	return sdBlob, contentHashes, nil
}

// DownloadAndVerifyStream downloads a stream and verifies its integrity
func (df *DemoFramework) DownloadAndVerifyStream(ctx context.Context, sdHash string, originalData []byte) ([]byte, error) {
	// Setup network configuration
	networkConfig, err := SetupBlobDownloaderConfig(df.config.PortalDomain, df.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to setup network config: %w", err)
	}

	// Create blob downloader
	downloader, err := CreateBlobDownloader(networkConfig, df.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob downloader: %w", err)
	}
	defer downloader.Close()

	// Download stream
	df.logger.Info("Downloading stream", zap.String("sd_hash", sdHash))
	downloadedData, err := downloader.DownloadStream(ctx, sdHash)
	if err != nil {
		return nil, fmt.Errorf("failed to download stream: %w", err)
	}

	// Verify integrity if original data is provided
	if originalData != nil {
		df.VerifyBlobIntegrity(originalData, downloadedData)
	}

	return downloadedData, nil
}

// CleanupStream deletes/unpins a stream
func (df *DemoFramework) CleanupStream(sdHash string) {
	df.logger.Info("Unpinning stream by deleting it", zap.String("sd_hash", sdHash))

	err := df.client.DeleteStream(sdHash)
	if err != nil {
		df.logger.Warn("Failed to delete/unpin stream", zap.Error(err))
		df.config.StdLogger.Printf("Warning: Failed to delete/unpin stream: %v", err)
	} else {
		df.logger.Info("Stream successfully deleted/unpinned")
		df.config.StdLogger.Printf("Stream successfully deleted/unpinned")
	}
}
