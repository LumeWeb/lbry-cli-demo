package shared

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// HTTPClient wraps an HTTP client with basic HTTP operations
type HTTPClient struct {
	client          *http.Client
	baseURL         string
	responseHandler *ResponseHandler
}

// AccountHTTPClient wraps an HTTP client for account service operations
type AccountHTTPClient struct {
	*HTTPClient
}

// LBRYHTTPClient wraps an HTTP client for LBRY service operations
type LBRYHTTPClient struct {
	*HTTPClient
}

// NewHTTPClient creates a new HTTP client with cookie jar support
func NewHTTPClient(baseURL string) (*HTTPClient, error) {
	jar := NewCookieJar()

	return &HTTPClient{
		client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		baseURL:         strings.TrimSuffix(baseURL, "/"),
		responseHandler: NewResponseHandler(),
	}, nil
}

// NewHTTPClientWithJar creates a new HTTP client with a specific cookie jar
func NewHTTPClientWithJar(baseURL string, jar http.CookieJar) (*HTTPClient, error) {
	return &HTTPClient{
		client: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		baseURL:         strings.TrimSuffix(baseURL, "/"),
		responseHandler: NewResponseHandler(),
	}, nil
}

// NewAccountHTTPClient creates a new HTTP client for account service
func NewAccountHTTPClient(baseURL string) (*AccountHTTPClient, error) {
	httpClient, err := NewHTTPClient(baseURL)
	if err != nil {
		return nil, err
	}

	return &AccountHTTPClient{
		HTTPClient: httpClient,
	}, nil
}

// NewAccountHTTPClientWithJar creates a new HTTP client for account service with specific cookie jar
func NewAccountHTTPClientWithJar(baseURL string, jar http.CookieJar) (*AccountHTTPClient, error) {
	httpClient, err := NewHTTPClientWithJar(baseURL, jar)
	if err != nil {
		return nil, err
	}

	return &AccountHTTPClient{
		HTTPClient: httpClient,
	}, nil
}

// NewLBRYHTTPClient creates a new HTTP client for LBRY service
func NewLBRYHTTPClient(baseURL string) (*LBRYHTTPClient, error) {
	httpClient, err := NewHTTPClient(baseURL)
	if err != nil {
		return nil, err
	}

	return &LBRYHTTPClient{
		HTTPClient: httpClient,
	}, nil
}

// NewLBRYHTTPClientWithJar creates a new HTTP client for LBRY service with specific cookie jar
func NewLBRYHTTPClientWithJar(baseURL string, jar http.CookieJar) (*LBRYHTTPClient, error) {
	httpClient, err := NewHTTPClientWithJar(baseURL, jar)
	if err != nil {
		return nil, err
	}

	return &LBRYHTTPClient{
		HTTPClient: httpClient,
	}, nil
}

// PostJSON makes a POST request with JSON data
func (c *HTTPClient) PostJSON(path string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + path

	// Create request with proper headers
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return c.responseHandler.ExecuteRequest(c.client, req)
}

// PostJSONWithAuth makes a POST request with JSON data and authentication
func (c *HTTPClient) PostJSONWithAuth(path string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	url := c.baseURL + path
	return c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
}

// Get makes a GET request
func (c *HTTPClient) Get(path string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.responseHandler.ExecuteRequest(c.client, req)
}

// SetTimeout sets the timeout for the HTTP client
func (c *HTTPClient) SetTimeout(timeout time.Duration) {
	c.client.Timeout = timeout
}

// GetBaseURL returns the base URL of the client
func (c *HTTPClient) GetBaseURL() string {
	return c.baseURL
}

// ExecuteRequest executes an HTTP request with common error handling
func (c *HTTPClient) ExecuteRequest(req *http.Request) (*http.Response, error) {
	return c.responseHandler.ExecuteRequest(c.client, req)
}

// ExecuteAndHandle combines request execution with response status checking
func (c *HTTPClient) ExecuteAndHandle(req *http.Request, expectedStatus int, operation string) (*http.Response, error) {
	return c.responseHandler.ExecuteAndHandle(c.client, req, expectedStatus, operation)
}

// HandleResponse checks response status code and returns appropriate error
func (c *HTTPClient) HandleResponse(resp *http.Response, expectedStatus int, operation string) error {
	return c.responseHandler.HandleResponse(resp, expectedStatus, operation)
}

// HandleAndDecode combines status checking and JSON decoding
func (c *HTTPClient) HandleAndDecode(resp *http.Response, expectedStatus int, target interface{}, operation string) error {
	return c.responseHandler.HandleAndDecode(resp, expectedStatus, target, operation)
}
