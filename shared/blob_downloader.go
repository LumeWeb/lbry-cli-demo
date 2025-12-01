package shared

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"go.lumeweb.com/liblbry"
	"go.lumeweb.com/liblbry/blob/transfer"
	"go.lumeweb.com/liblbry/blob/transfer/peer_transfer"
	"go.lumeweb.com/liblbry/client"
	"go.lumeweb.com/liblbry/protocol"
	"go.lumeweb.com/liblbry/storage/memory"
	"go.lumeweb.com/liblbry/stream"
	"go.uber.org/zap"
)

// BlobDownloader handles downloading blobs from the LBRY network
type BlobDownloader struct {
	streamAcquirer client.StreamAcquirer
	dhtNode        protocol.DHTNode
	peerTransfer   *peer_transfer.PeerTransfer
	logger         *zap.Logger // Store the logger for later use
}

// BlobDownloaderConfig holds configuration for the blob downloader
type BlobDownloaderConfig struct {
	DHTAddress  string
	PeerAddress string
	FixedPeers  []string
	Logger      *zap.Logger // Zap logger for structured logging
	SeedNodes   []string
	MaxPeers    int
	Timeout     time.Duration
}

// NewBlobDownloader creates a new blob downloader with DHT, peer transfer, and memory storage
func NewBlobDownloader(config BlobDownloaderConfig) (*BlobDownloader, error) {
	// Use the provided zap logger
	zapLogger := config.Logger
	if zapLogger == nil {
		// Fallback to development logger if none provided
		var err error
		zapLogger, err = zap.NewDevelopment()
		if err != nil {
			return nil, fmt.Errorf("failed to create fallback zap logger: %w", err)
		}
	}

	// Create memory storage for caching blobs
	memoryStore := memory.NewMemoryStore()
	zapLogger.Debug("Created memory store for blob caching")

	// Create DHT node for peer discovery
	zapLogger.Debug("Creating DHT node",
		zap.String("address", config.DHTAddress),
		zap.Strings("seed_nodes", config.SeedNodes))

	dhtNode, err := protocol.NewDHTNodeWithDefaults(
		protocol.WithDHTLogger(zapLogger),
		protocol.WithDHTAddress(config.DHTAddress),
		protocol.WithDHTSeedNodes(config.SeedNodes),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DHT node: %w", err)
	}

	// Create peer transfer with DHT discovery using DefaultPeerClientFactory
	fixedPeers := config.FixedPeers
	if len(fixedPeers) == 0 && config.PeerAddress != "" {
		fixedPeers = []string{config.PeerAddress}
	}

	zapLogger.Debug("Creating peer transfer",
		zap.Strings("fixed_peers", fixedPeers),
		zap.Int("max_peers", config.MaxPeers),
		zap.Duration("timeout", config.Timeout))

	peerTransfer, err := peer_transfer.NewPeerTransfer(
		dhtNode,
		protocol.DefaultPeerClientFactory(),
		peer_transfer.WithPeerTransferLogger(zapLogger),
		peer_transfer.WithPeerTransferTimeout(config.Timeout),
		peer_transfer.WithPeerTransferMaxPeers(config.MaxPeers),
		peer_transfer.WithPeerTransferFixedPeers(fixedPeers),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer transfer: %w", err)
	}

	// Start the components
	zapLogger.Info("Starting DHT node")
	err = dhtNode.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start dht: %w", err)
	}

	zapLogger.Info("Starting peer transfer")
	peerTransfer.Start()

	// Create blob acquirer with peer transfer and memory storage
	transfers := []transfer.Transfer{peerTransfer}
	acquirer, err := liblbry.NewBlobAcquirer(transfers, memoryStore)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob acquirer: %w", err)
	}

	streamAcquirer, err := client.NewStreamAcquirerBuilder(zapLogger).WithAcquirer(acquirer).WithStore(memoryStore).WithFactory(client.NewStreamAcquirerFactory(zapLogger)).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create stream acquirer: %w", err)
	}

	return &BlobDownloader{
		streamAcquirer: streamAcquirer,
		dhtNode:        dhtNode,
		peerTransfer:   peerTransfer,
		logger:         zapLogger,
	}, nil
}

// DownloadStream downloads a stream by its hash and returns the data as a byte slice
func (bd *BlobDownloader) DownloadStream(ctx context.Context, streamHash string) ([]byte, error) {
	if streamHash == "" {
		return nil, fmt.Errorf("stream hash cannot be empty")
	}

	// Get the stream reader
	bd.logger.Info("Downloading stream", zap.String("stream_hash", streamHash))

	reader, err := bd.streamAcquirer.GetStream(ctx, streamHash, client.WithAcquireVerification(true))
	if err != nil {
		return nil, fmt.Errorf("failed to acquire stream %s: %w", streamHash, err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			// Log warning but don't fail the operation
			fmt.Printf("Warning: failed to close reader for stream %s: %v\n", streamHash, closeErr)
		}
	}()

	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from stream %s: %w", streamHash, err)
	}

	bd.logger.Info("Successfully downloaded stream",
		zap.String("stream_hash", streamHash),
		zap.Int("data_size_bytes", len(data)))

	return data, nil
}

// DownloadStreamToFile downloads a stream by its hash and saves it to a file
func (bd *BlobDownloader) DownloadStreamToFile(ctx context.Context, streamHash, filename string) error {
	data, err := bd.DownloadStream(ctx, streamHash)
	if err != nil {
		return err
	}

	// Write data to file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write data to file %s: %w", filename, err)
	}

	return nil
}

// GetSDBlob retrieves just the SD blob content for a given SD hash
func (bd *BlobDownloader) GetSDBlob(ctx context.Context, sdHash string) (*stream.SDBlob, error) {
	if sdHash == "" {
		return nil, fmt.Errorf("SD hash cannot be empty")
	}

	bd.logger.Info("Getting SD blob", zap.String("sd_hash", sdHash))

	// Get the SD blob using the stream acquirer's GetSDBlob method
	sdBlob, _, err := bd.streamAcquirer.GetSDBlob(ctx, sdHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get SD blob %s: %w", sdHash, err)
	}

	bd.logger.Info("Successfully retrieved SD blob",
		zap.String("sd_hash", sdHash))

	return sdBlob, nil
}

// Close shuts down the blob downloader components
func (bd *BlobDownloader) Close() {
	if bd.peerTransfer != nil {
		bd.logger.Info("Stopping peer transfer")
		bd.peerTransfer.Stop()
	}
	if bd.dhtNode != nil {
		bd.logger.Info("Shutting down DHT node")
		bd.dhtNode.Shutdown()
	}
}

// GetFreePort returns a free port on the local machine
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
