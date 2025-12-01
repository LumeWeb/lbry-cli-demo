package shared

import (
	"fmt"
	"time"

	"go.lumeweb.com/queryutil"
)

// PortalClient wraps an HTTP client with business logic methods for account operations
type PortalClient struct {
	httpClient *AccountHTTPClient
}

// NewPortalClient creates a new portal client with the given account HTTP client
func NewPortalClient(httpClient *AccountHTTPClient) *PortalClient {
	return &PortalClient{
		httpClient: httpClient,
	}
}

// RegisterUser registers a new user
func (p *PortalClient) RegisterUser(email, password, firstName, lastName string) error {
	registerData := map[string]interface{}{
		"email":      email,
		"password":   password,
		"first_name": firstName,
		"last_name":  lastName,
	}

	resp, err := p.httpClient.PostJSON(EndpointAuthRegister, registerData)
	if err != nil {
		return fmt.Errorf("failed to make registration request: %w", err)
	}
	defer resp.Body.Close()

	return p.httpClient.HandleResponse(resp, 200, "registration")
}

// Login logs in a user
func (p *PortalClient) Login(email, password string) error {
	loginData := map[string]interface{}{
		"email":    email,
		"password": password,
		"remember": true,
	}

	resp, err := p.httpClient.PostJSON(EndpointAuthLogin, loginData)
	if err != nil {
		return fmt.Errorf("failed to make login request: %w", err)
	}
	defer resp.Body.Close()

	err = p.httpClient.responseHandler.HandleResponse(resp, 200, "login")
	if err != nil {
		return err
	}

	return p.httpClient.responseHandler.HandleResponse(resp, 200, "login")
}

// WaitForAllOperations waits until all operations are completed
// by polling the operations endpoint until no pending operations remain
func (p *PortalClient) WaitForAllOperations() error {
	// Poll every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Build URL to filter for non-completed operations
			url, err := queryutil.BuildURL(
				EndpointOperations,
				nil, // no sorts
				nil, // default pagination
				queryutil.Filters(
					queryutil.NotEqual("status", "completed"),
				)...,
			)
			if err != nil {
				return fmt.Errorf("failed to build operations URL: %w", err)
			}

			// Make request to check operations
			resp, err := p.httpClient.Get(url)
			if err != nil {
				return fmt.Errorf("failed to check operations: %w", err)
			}
			defer resp.Body.Close()

			var operationsResp struct {
				Data  []interface{} `json:"data"`
				Total int           `json:"total"`
			}

			if err := p.httpClient.HandleAndDecode(resp, 200, &operationsResp, "check operations"); err != nil {
				return err
			}

			// If no pending operations, we're done
			if operationsResp.Total == 0 {
				return nil
			}
		}
	}
}
