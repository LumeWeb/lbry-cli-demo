package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"go.lumeweb.com/liblbry/protocol"
	"go.lumeweb.com/liblbry/stream"
	"go.uber.org/zap"

	"github.com/lumeweb/lbry-cli-demo/shared"
)

// Config holds the CLI configuration
type Config struct {
	Email     string
	Password  string
	FilePath  string
	Reflector string
}

// parseFlags parses command-line flags and returns configuration
func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.Email, "email", "", "Email for login (required)")
	flag.StringVar(&config.Password, "password", "", "Password for login (required)")
	flag.StringVar(&config.FilePath, "file", "", "File path to upload (required)")
	flag.StringVar(&config.Reflector, "reflector", "", "Reflector server address (default: lbry.<portal>.xyz:5566)")

	return config
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.Email == "" {
		return fmt.Errorf("email is required")
	}
	if config.Password == "" {
		return fmt.Errorf("password is required")
	}
	if config.FilePath == "" {
		return fmt.Errorf("file path is required")
	}

	// Check if file exists
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", config.FilePath)
	}

	return nil
}

// buildReflectorAddress builds the reflector address from portal domain
func buildReflectorAddress(portalDomain, reflectorAddr string) string {
	if reflectorAddr != "" {
		return reflectorAddr
	}
	return fmt.Sprintf("lbry.%s:5566", portalDomain)
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

func main() {
	// Parse common flags (portal, log-level)
	portalDomain, logLevel := shared.ParseCommonFlags()

	// Parse demo-specific flags
	config := parseFlags()

	// Parse all flags after they are defined
	flag.Parse()

	// Validate configuration
	if err := validateConfig(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create demo framework for logging
	framework, err := shared.NewDemoFramework(*portalDomain, *logLevel, "LBRY-UPLOAD")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create framework: %v\n", err)
		os.Exit(1)
	}
	defer framework.Cleanup()

	logger := framework.GetLogger()

	logger.Info("Starting LBRY upload demo",
		zap.String("portal", *portalDomain),
		zap.String("reflector", config.Reflector),
		zap.String("file", config.FilePath),
		zap.String("email", config.Email),
	)

	// Login to existing account
	logger.Info("Logging in to account...")
	accountManager := framework.GetAccountManager()

	// Use LoginWithCredentials() to login to existing account (not create new one)
	err = accountManager.LoginWithCredentials(config.Email, config.Password)
	if err != nil {
		logger.Fatal("Failed to login", zap.Error(err))
	}

	logger.Info("Login successful")

	// Create stream from file
	logger.Info("Creating stream from file", zap.String("file", config.FilePath))

	streamCreator := stream.NewStreamCreator()

	streamResult, err := streamCreator.CreateStreamFromPath(config.FilePath)
	if err != nil {
		logger.Fatal("Failed to create stream", zap.Error(err))
	}

	logger.Info("Stream created successfully",
		zap.String("stream_hash", streamResult.StreamHash),
		zap.String("sd_blob_hash", streamResult.SDBlobHash),
		zap.Int("total_blobs", streamResult.TotalChunks),
	)

	// Build reflector address from portal domain
	reflectorAddr := buildReflectorAddress(*portalDomain, config.Reflector)

	// Upload stream to reflector
	logger.Info("Starting upload to reflector")
	err = uploadStreamToReflector(streamResult, reflectorAddr, logger)
	if err != nil {
		logger.Fatal("Failed to upload stream to reflector", zap.Error(err))
	}

	logger.Info("Stream uploaded successfully to reflector",
		zap.String("stream_hash", streamResult.StreamHash),
		zap.String("sd_blob_hash", streamResult.SDBlobHash),
		zap.Int("total_blobs", streamResult.TotalChunks),
	)

	// Display results
	fmt.Println("\n=== Upload Complete ===")
	fmt.Printf("Stream Hash: %s\n", streamResult.StreamHash)
	fmt.Printf("SD Blob Hash: %s\n", streamResult.SDBlobHash)
	fmt.Printf("Total Blobs: %d\n", streamResult.TotalChunks)
	fmt.Printf("Reflector: %s\n", reflectorAddr)
	fmt.Println("======================\n")

	logger.Info("LBRY upload demo completed successfully")
}
