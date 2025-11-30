package shared

import (
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
)

// NetworkConfig holds network configuration for blob downloader
type NetworkConfig struct {
	DHTAddress  string
	PeerAddress string
	FixedPeers  []string
	SeedNodes   []string
	MaxPeers    int
	Timeout     time.Duration
}

// SetupBlobDownloaderConfig creates a standard blob downloader configuration
func SetupBlobDownloaderConfig(portalDomain string, logger *zap.Logger) (*NetworkConfig, error) {
	// Get free ports for DHT and peer
	dhtPort, err := GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get free DHT port: %w", err)
	}

	peerPort, err := GetFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get free peer port: %w", err)
	}

	dhtAddress := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", dhtPort))
	peerAddress := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", peerPort))

	// Default fixed peers to portal domain at port 5567
	fixedPeer := net.JoinHostPort(portalDomain, "5567")
	// Seed nodes use port 4444
	seedNode := net.JoinHostPort(portalDomain, "4444")

	config := &NetworkConfig{
		DHTAddress:  dhtAddress,
		PeerAddress: peerAddress,
		FixedPeers:  []string{fixedPeer},
		SeedNodes:   []string{seedNode},
		MaxPeers:    5,
		Timeout:     30 * time.Second,
	}

	logger.Info("Network configuration created",
		zap.String("dht_address", dhtAddress),
		zap.String("peer_address", peerAddress),
		zap.Strings("fixed_peers", config.FixedPeers),
		zap.Strings("seed_nodes", config.SeedNodes))

	return config, nil
}

// CreateBlobDownloader creates and configures a blob downloader
func CreateBlobDownloader(config *NetworkConfig, logger *zap.Logger) (*BlobDownloader, error) {
	downloaderConfig := BlobDownloaderConfig{
		DHTAddress:  config.DHTAddress,
		PeerAddress: config.PeerAddress,
		FixedPeers:  config.FixedPeers,
		Logger:      logger,
		SeedNodes:   config.SeedNodes,
		MaxPeers:    config.MaxPeers,
		Timeout:     config.Timeout,
	}

	downloader, err := NewBlobDownloader(downloaderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob downloader: %w", err)
	}

	logger.Info("Blob downloader created successfully")
	return downloader, nil
}