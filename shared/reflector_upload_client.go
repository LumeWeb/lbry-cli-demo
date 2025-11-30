package shared

import (
	"fmt"
	"time"

	"go.lumeweb.com/liblbry/protocol"
	"go.uber.org/zap"
)

// ReflectorUploadClient handles uploading blobs to a reflector server
type ReflectorUploadClient struct {
	reflectorClient protocol.ReflectorClient
	serverAddress   string
	logger          *zap.Logger
}

// NewReflectorUploadClient creates a new reflector upload client
func NewReflectorUploadClient(serverAddress string, logger *zap.Logger) *ReflectorUploadClient {
	return &ReflectorUploadClient{
		serverAddress: serverAddress,
		logger:        logger,
	}
}

// Connect establishes a connection to the reflector server
func (c *ReflectorUploadClient) Connect() error {
	c.logger.Info("Connecting to reflector server", zap.String("address", c.serverAddress))

	// Create reflector client
	c.reflectorClient = protocol.NewReflectorClient(
		protocol.WithReflectorClientLogger(c.logger),
		protocol.WithReflectorClientTimeout(30*time.Second),
	)

	// Connect to reflector
	err := c.reflectorClient.Connect(c.serverAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to reflector at %s: %w", c.serverAddress, err)
	}

	c.logger.Info("Connected to reflector server successfully")
	return nil
}

// Close closes the connection to the reflector server
func (c *ReflectorUploadClient) Close() error {
	if c.reflectorClient != nil {
		err := c.reflectorClient.Close()
		if err != nil {
			c.logger.Warn("Failed to close reflector connection", zap.Error(err))
			return err
		}
		c.logger.Info("Reflector connection closed successfully")
	}
	return nil
}

// IsConnected returns true if the client is connected to the reflector
func (c *ReflectorUploadClient) IsConnected() bool {
	return c.reflectorClient != nil
}