package shared

import (
	"fmt"
	"io"
	"os"
)

// LBRYPortalClient coordinates both account and LBRY services
type LBRYPortalClient struct {
	PortalClient *PortalClient
	LBRYClient   *LBRYClient
}

// LBRYPortalClientConfig holds configuration for both services
type LBRYPortalClientConfig struct {
	AccountBaseURL string
	LBRYBaseURL    string
}

// NewLBRYPortalClient creates a new LBRY portal client with shared cookie jar
func NewLBRYPortalClient(config LBRYPortalClientConfig) (*LBRYPortalClient, error) {
	// Create a shared cookie jar for both services
	sharedJar := NewCookieJar()

	// Create account HTTP client with shared jar
	accountHTTPClient, err := NewAccountHTTPClientWithJar(config.AccountBaseURL, sharedJar)
	if err != nil {
		return nil, fmt.Errorf("failed to create account HTTP client: %w", err)
	}

	// Create LBRY HTTP client with shared jar
	lbryHTTPClient, err := NewLBRYHTTPClientWithJar(config.LBRYBaseURL, sharedJar)
	if err != nil {
		return nil, fmt.Errorf("failed to create LBRY HTTP client: %w", err)
	}

	// Create service clients
	portalClient := NewPortalClient(accountHTTPClient)
	lbryClient := NewLBRYClient(lbryHTTPClient)

	return &LBRYPortalClient{
		PortalClient: portalClient,
		LBRYClient:   lbryClient,
	}, nil
}

// Account operations delegated to PortalClient
func (c *LBRYPortalClient) RegisterUser(email, password, firstName, lastName string) error {
	return c.PortalClient.RegisterUser(email, password, firstName, lastName)
}

func (c *LBRYPortalClient) Login(email, password string) error {
	return c.PortalClient.Login(email, password)
}

func (c *LBRYPortalClient) ListDevices() (*DeviceResponseResponse, error) {
	return c.PortalClient.ListDevices()
}

// LBRY operations delegated to LBRYClient
func (c *LBRYPortalClient) AddDevice(name, ipAddress string) (*DeviceResponse, error) {
	return c.LBRYClient.AddDevice(name, ipAddress)
}

func (c *LBRYPortalClient) UploadStream(file io.Reader, filename string) (*PostStreamUploadResponse, error) {
	return c.LBRYClient.UploadStream(file, filename)
}

func (c *LBRYPortalClient) ListStreams() (*StreamResponseResponse, error) {
	return c.LBRYClient.ListStreams()
}

func (c *LBRYPortalClient) PinStream(sdHash string) error {
	return c.LBRYClient.PinStream(sdHash)
}

func (c *LBRYPortalClient) DeleteStream(sdHash string) error {
	return c.LBRYClient.DeleteStream(sdHash)
}

// WaitForAllOperations waits until all operations are completed
func (c *LBRYPortalClient) WaitForAllOperations() error {
	return c.PortalClient.WaitForAllOperations()
}

// UploadStreamWithTUS uploads a stream using TUS protocol
func (c *LBRYPortalClient) UploadStreamWithTUS(file *os.File, metadata StreamMetadataRequest) error {
	return c.LBRYClient.UploadStreamWithTUS(file, metadata)
}
