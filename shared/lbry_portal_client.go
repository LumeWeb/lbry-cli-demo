package shared

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"go.uber.org/zap"
)

// LBRYPortalClient coordinates both account and LBRY services
type LBRYPortalClient struct {
	PortalClient *PortalClient
	LBRYClient   *LBRYClient
	rootDomain   string
}

// LBRYPortalClientConfig holds configuration for both services
type LBRYPortalClientConfig struct {
	AccountBaseURL string
	LBRYBaseURL    string
	RootDomain     string
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
		rootDomain:   config.RootDomain,
	}, nil
}

// SaveAuthTokenToState extracts auth token from cookie jar and saves it to account state
func (c *LBRYPortalClient) SaveAuthTokenToState(stateManager *StateManager, logger *zap.Logger) error {
	// Get auth token from cookie jar
	authToken := c.getAuthTokenFromJar()
	if authToken == "" {
		logger.Warn("No auth token found in cookie jar")
		return nil // Not an error, just no token found
	}

	// Load existing account state
	accountState, err := stateManager.LoadAccountState()
	if err != nil {
		return fmt.Errorf("failed to load account state: %w", err)
	}

	if accountState == nil {
		logger.Warn("No account state found to save auth token")
		return nil
	}

	// Update JWT field
	accountState.JWT = authToken

	// Save updated state
	err = stateManager.SaveJSON(AccountStateFile, accountState)
	if err != nil {
		return fmt.Errorf("failed to save account state with auth token: %w", err)
	}

	logger.Info("Auth token saved to account state")
	return nil
}

// getAuthTokenFromJar extracts auth token from cookie jar
func (c *LBRYPortalClient) getAuthTokenFromJar() string {
	// Get cookie jar from HTTP client
	jar := c.PortalClient.httpClient.client.Jar
	if jar == nil {
		return ""
	}

	// Create URL for root domain using the passed root domain
	rootURL := &url.URL{
		Scheme: "https",
		Host:   c.rootDomain,
		Path:   "/",
	}

	cookies := jar.Cookies(rootURL)
	for _, cookie := range cookies {
		if cookie.Name == "auth_token" {
			return cookie.Value
		}
	}

	return ""
}

// Account operations delegated to PortalClient
func (c *LBRYPortalClient) RegisterUser(email, password, firstName, lastName string) error {
	return c.PortalClient.RegisterUser(email, password, firstName, lastName)
}

func (c *LBRYPortalClient) Login(email, password string) error {
	return c.PortalClient.Login(email, password)
}

func (c *LBRYPortalClient) RegisterDevice(deviceName, ipAddress string) error {
	return c.LBRYClient.RegisterDevice(deviceName, ipAddress)
}

func (c *LBRYPortalClient) ListDevices() (*DeviceResponseResponse, error) {
	return c.LBRYClient.ListDevices()
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
