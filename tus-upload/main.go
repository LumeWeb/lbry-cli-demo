package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/lumeweb/lbry-cli-demo/shared"
	"go.uber.org/zap"
)

// Configuration
const (
	blobSize     = 100 * 1024 * 1024 // 100MB
	uploadFile   = "upload_blob.bin"
	downloadFile = "download_blob.bin"
)

// Build URLs from portal domain
const (
	accountSubdomain = "account"
	lbrySubdomain    = "lbry"
)

// buildSubdomainURL creates a full URL for a subdomain of the given base domain
func buildSubdomainURL(baseDomain, subdomain string) string {
	// Ensure baseDomain doesn't have protocol prefix
	baseDomain = strings.TrimPrefix(baseDomain, "http://")
	baseDomain = strings.TrimPrefix(baseDomain, "https://")

	// Remove trailing slash
	baseDomain = strings.TrimSuffix(baseDomain, "/")

	return fmt.Sprintf("https://%s.%s", subdomain, baseDomain)
}

func main() {
	// Parse command-line flags
	portalDomain := flag.String("portal", "pinner.xyz", "Base portal domain (e.g., pinner.xyz)")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	accountBaseURL := buildSubdomainURL(*portalDomain, accountSubdomain)
	lbryBaseURL := buildSubdomainURL(*portalDomain, lbrySubdomain)

	ctx := context.Background()

	// Parse log level and create zap logger
	parsedLogLevel := shared.ParseLogLevel(*logLevel)
	zapLogger, err := shared.CreateZapLogger(parsedLogLevel)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer zapLogger.Sync()

	// Create a standard logger for backward compatibility with existing code
	logger := log.New(os.Stdout, "[TUS-UPLOAD] ", log.LstdFlags)

	zapLogger.Info("Starting tus-upload demo",
		zap.String("portal_domain", *portalDomain),
		zap.String("log_level", parsedLogLevel.String()))

	// Create LBRY Portal Client
	config := shared.LBRYPortalClientConfig{
		AccountBaseURL: accountBaseURL,
		LBRYBaseURL:    lbryBaseURL,
	}

	client, err := shared.NewLBRYPortalClient(config)
	if err != nil {
		logger.Fatalf("Failed to create LBRY portal client: %v", err)
	}

	// Create and login to account
	fakeAccount := shared.GenerateFakeAccount()
	zapLogger.Info("Creating account",
		zap.String("email", fakeAccount.Email),
		zap.String("first_name", fakeAccount.FirstName),
		zap.String("last_name", fakeAccount.LastName))

	err = client.RegisterUser(fakeAccount.Email, fakeAccount.Password, fakeAccount.FirstName, fakeAccount.LastName)
	if err != nil {
		zapLogger.Fatal("Failed to register user", zap.Error(err))
	}

	zapLogger.Info("Account created successfully, logging in...")
	err = client.Login(fakeAccount.Email, fakeAccount.Password)
	if err != nil {
		zapLogger.Fatal("Failed to login", zap.Error(err))
	}

	zapLogger.Info("Login successful!")

	// Generate 100MB random data blob
	zapLogger.Info("Generating random data blob", zap.Int("size_bytes", blobSize))
	data := make([]byte, blobSize)

	// Fill with cryptographically secure random data
	_, err = rand.Read(data)
	if err != nil {
		logger.Fatalf("Failed to generate random data: %v", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])
	zapLogger.Info("Generated blob", zap.String("sha256", hashStr))

	// Write to disk
	err = os.WriteFile(uploadFile, data, 0644)
	if err != nil {
		logger.Fatalf("Failed to write upload file: %v", err)
	}
	logger.Printf("Blob saved to: %s", uploadFile)

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
	logger.Printf("Waiting for operations to complete...")
	err = client.WaitForAllOperations()
	if err != nil {
		logger.Fatalf("Failed while waiting for operations: %v", err)
	}

	logger.Printf("All operations completed successfully!")

	// List streams to get the stream hash
	logger.Printf("Listing streams to find uploaded stream...")
	streams, err := client.ListStreams()
	if err != nil {
		logger.Fatalf("Failed to list streams: %v", err)
	}

	var uploadedStream *shared.StreamResponse
	if len(streams.Data) > 0 {
		uploadedStream = &streams.Data[0]
	}

	if uploadedStream == nil {
		logger.Fatalf("No streams available")
	}

	logger.Printf("Found stream: ID=%d, SDHash=%s, StreamHash=%s",
		uploadedStream.ID, uploadedStream.SDHash, uploadedStream.StreamHash)

	// Set up blob downloader and fetch the stream
	logger.Printf("Setting up blob downloader...")

	// Get free ports for DHT and peer
	dhtPort, err := shared.GetFreePort()
	if err != nil {
		logger.Fatalf("Failed to get free DHT port: %v", err)
	}

	peerPort, err := shared.GetFreePort()
	if err != nil {
		logger.Fatalf("Failed to get free peer port: %v", err)
	}

	dhtAddress := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", dhtPort))
	peerAddress := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", peerPort))

	// Default fixed peers to portal domain at port 56667
	fixedPeer := net.JoinHostPort(*portalDomain, "5567")
	// Seed nodes use port 4444
	seedNode := net.JoinHostPort(*portalDomain, "4444")

	downloaderConfig := shared.BlobDownloaderConfig{
		DHTAddress:  dhtAddress,
		PeerAddress: peerAddress,
		FixedPeers:  []string{fixedPeer},
		Logger:      zapLogger,
		SeedNodes:   []string{seedNode},
		MaxPeers:    5,
		Timeout:     30 * time.Second,
	}

	downloader, err := shared.NewBlobDownloader(downloaderConfig)
	if err != nil {
		logger.Fatalf("Failed to create blob downloader: %v", err)
	}
	defer downloader.Close()

	logger.Printf("Downloading stream with SD hash: %s", uploadedStream.SDHash)
	downloadedData, err := downloader.DownloadStream(ctx, uploadedStream.SDHash)
	if err != nil {
		logger.Fatalf("Failed to download stream: %v", err)
	}

	// Save downloaded data and verify SHA256
	err = os.WriteFile(downloadFile, downloadedData, 0644)
	if err != nil {
		logger.Fatalf("Failed to write download file: %v", err)
	}
	logger.Printf("Downloaded blob saved to: %s", downloadFile)

	// Verify using bytes.Equal for direct comparison
	logger.Printf("Original data size: %d bytes", len(data))
	logger.Printf("Downloaded data size: %d bytes", len(downloadedData))

	if len(data) == len(downloadedData) && string(data) == string(downloadedData) {
		logger.Printf("SUCCESS: Data matches exactly!")
	} else {
		logger.Printf("FAILURE: Data does not match!")
	}

	// Unpin the stream after downloading it
	// In LBRY network, deleting a stream effectively unpins it
	logger.Printf("Unpinning stream by deleting it...")
	err = client.DeleteStream(uploadedStream.SDHash)
	if err != nil {
		logger.Printf("Warning: Failed to delete/unpin stream: %v", err)
	} else {
		logger.Printf("Stream successfully deleted/unpinned")
	}

	logger.Printf("Program completed successfully!")
}
