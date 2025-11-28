package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ResponseHandler provides common HTTP response handling patterns
type ResponseHandler struct{}

// NewResponseHandler creates a new response handler
func NewResponseHandler() *ResponseHandler {
	return &ResponseHandler{}
}

// HandleResponse checks response status code and returns appropriate error
func (r *ResponseHandler) HandleResponse(resp *http.Response, expectedStatus int, operation string) error {
	if resp.StatusCode != expectedStatus {
		// Read response body to include in error message
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("%s failed with status: %d (failed to read response body: %w)", operation, resp.StatusCode, err)
		}

		// Restore the body for subsequent reads
		resp.Body = io.NopCloser(bytes.NewBuffer(body))

		bodyStr := string(body)
		if bodyStr == "" {
			bodyStr = "(empty body)"
		}

		return fmt.Errorf("%s failed with status: %d, body: %s", operation, resp.StatusCode, bodyStr)
	}
	return nil
}

// DecodeResponse decodes JSON response into target struct with error wrapping
func (r *ResponseHandler) DecodeResponse(resp *http.Response, target interface{}, operation string) error {
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode %s response: %w", operation, err)
	}
	return nil
}

// HandleAndDecode combines status checking and JSON decoding
func (r *ResponseHandler) HandleAndDecode(resp *http.Response, expectedStatus int, target interface{}, operation string) error {
	if err := r.HandleResponse(resp, expectedStatus, operation); err != nil {
		return err
	}

	return r.DecodeResponse(resp, target, operation)
}

// ExecuteRequest executes an HTTP request with common error handling
func (r *ResponseHandler) ExecuteRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	return resp, nil
}

// ExecuteAndHandle combines request execution with response status checking
func (r *ResponseHandler) ExecuteAndHandle(client *http.Client, req *http.Request, expectedStatus int, operation string) (*http.Response, error) {
	resp, err := r.ExecuteRequest(client, req)
	if err != nil {
		return nil, err
	}

	if err := r.HandleResponse(resp, expectedStatus, operation); err != nil {
		resp.Body.Close()
		return nil, err
	}

	return resp, nil
}

// ExecuteHandleAndDecode combines request execution, status checking, and JSON decoding
func (r *ResponseHandler) ExecuteHandleAndDecode(client *http.Client, req *http.Request, expectedStatus int, target interface{}, operation string) error {
	resp, err := r.ExecuteAndHandle(client, req, expectedStatus, operation)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return r.DecodeResponse(resp, target, operation)
}

// Global response handler instance for convenience
var DefaultResponseHandler = NewResponseHandler()
