package main

import (
	"context"
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
	// SD hash for the stream we want to pin/download/unpin
	targetSDHash = "acc6adf8b4f10dcddffc5c2ca87dbd9cb3a2664564695ac7aaab038193ff14a280cc3d4ebae55c71d0b885a7316d0137"
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
	logger := log.New(os.Stdout, "[PIN] ", log.LstdFlags)

	zapLogger.Info("Starting pin demo",
		zap.String("portal_domain", *portalDomain),
		zap.String("log_level", parsedLogLevel.String()),
		zap.String("sd_hash", targetSDHash))

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

	// Pin the stream
	zapLogger.Info("Pinning stream", zap.String("sd_hash", targetSDHash))
	err = client.PinStream(targetSDHash)
	if err != nil {
		zapLogger.Fatal("Failed to pin stream:", zap.Error(err))
	}
	zapLogger.Info("Stream pinned successfully!")

	// Wait for all operations to complete
	zapLogger.Info("Waiting for pin operations to complete...")
	err = client.WaitForAllOperations()
	if err != nil {
		zapLogger.Fatal("Failed while waiting for operations:", zap.Error(err))
	}

	zapLogger.Info("All pin operations completed successfully!")

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

	// Download the stream using the predefined SD hash
	zapLogger.Info("Downloading stream", zap.String("sd_hash", targetSDHash))
	downloadedData, err := downloader.DownloadStream(ctx, targetSDHash)
	if err != nil {
		logger.Fatalf("Failed to download stream: %v", err)
	}

	// Save downloaded data (will be a PDF)
	downloadFile := "downloaded_stream.pdf"
	err = os.WriteFile(downloadFile, downloadedData, 0644)
	if err != nil {
		logger.Fatalf("Failed to write download file: %v", err)
	}
	zapLogger.Info("Downloaded stream saved", zap.String("file", downloadFile))

	// Verify the downloaded data size
	zapLogger.Info("Downloaded data size", zap.Int("bytes", len(downloadedData)))

	// Unpin the stream by deleting it (as mentioned in the instructions)
	zapLogger.Info("Unpinning stream by deleting it", zap.String("sd_hash", targetSDHash))
	err = client.DeleteStream(targetSDHash)
	if err != nil {
		zapLogger.Warn("Failed to delete/unpin stream", zap.Error(err))
	} else {
		zapLogger.Info("Stream successfully deleted/unpinned")
	}

	zapLogger.Info("Pin demo completed successfully!")
}
