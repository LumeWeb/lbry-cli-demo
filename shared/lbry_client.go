package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// LBRYClient handles LBRY-specific functionality
type LBRYClient struct {
	httpClient *LBRYHTTPClient
}

// NewLBRYClient creates a new LBRY client
func NewLBRYClient(httpClient *LBRYHTTPClient) *LBRYClient {
	return &LBRYClient{
		httpClient: httpClient,
	}
}

// Device and Stream request/response structs

// CreateDeviceRequest represents a device creation request
type CreateDeviceRequest struct {
	Name      string `json:"name"`
	IPAddress string `json:"ip_address"`
}

// DeviceResponse represents a device response
type DeviceResponse struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeviceResponseResponse represents a paginated device response
type DeviceResponseResponse struct {
	Data  []DeviceResponse `json:"data"`
	Total int              `json:"total"`
}

// StreamResponse represents a stream response
type StreamResponse struct {
	ID                int       `json:"id"`
	StreamHash        string    `json:"stream_hash"`
	SDHash            string    `json:"sd_hash"`
	StreamName        string    `json:"stream_name"`
	StreamType        string    `json:"stream_type"`
	SuggestedFileName string    `json:"suggested_file_name"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// StreamResponseResponse represents a paginated stream response
type StreamResponseResponse struct {
	Data  []StreamResponse `json:"data"`
	Total int              `json:"total"`
}

// StreamPinRequest represents a stream pin request
type StreamPinRequest struct {
	SDHash string `json:"sd_hash"`
}

// StreamMetadataRequest represents metadata for stream upload
type StreamMetadataRequest struct {
	// StreamName is the name of the stream in the LBRY network
	StreamName string `json:"stream_name"`

	// SuggestedFileName is the recommended filename when downloading the stream
	SuggestedFileName string `json:"suggested_file_name"`
}

// PostStreamUploadResponse represents a stream upload response
type PostStreamUploadResponse struct {
	UploadHash string `json:"upload_hash"`
}

// AddDevice adds a new device to the whitelist
func (l *LBRYClient) AddDevice(name, ipAddress string) (*DeviceResponse, error) {
	request := CreateDeviceRequest{
		Name:      name,
		IPAddress: ipAddress,
	}

	resp, err := l.httpClient.PostJSON(LBRYEndpointDevices, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}
	defer resp.Body.Close()

	var device DeviceResponse
	if err := l.httpClient.HandleAndDecode(resp, 201, &device, "device creation"); err != nil {
		return nil, err
	}

	return &device, nil
}

// UploadStream uploads a stream from an io.Reader
func (l *LBRYClient) UploadStream(file io.Reader, filename string) (*PostStreamUploadResponse, error) {
	// Create default metadata
	metadata := StreamMetadataRequest{
		StreamName:        filename,
		SuggestedFileName: filename,
	}

	return l.UploadStreamWithMetadata(file, filename, metadata)
}

// UploadStreamWithMetadata uploads a stream from an io.Reader with custom metadata
func (l *LBRYClient) UploadStreamWithMetadata(file io.Reader, filename string, metadata StreamMetadataRequest) (*PostStreamUploadResponse, error) {
	// Create a multipart form
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Create the file field
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// Copy the file content
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add metadata as JSON form field
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	metaField, err := writer.CreateFormField("meta")
	if err != nil {
		return nil, fmt.Errorf("failed to create meta form field: %w", err)
	}

	if _, err := metaField.Write(metadataJSON); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	// Close the multipart writer to set the terminating boundary
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the HTTP request
	url := l.httpClient.GetBaseURL() + LBRYEndpointStreamUpload
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set the content type header from the multipart writer
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Make the request
	resp, err := l.httpClient.ExecuteRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload stream: %w", err)
	}
	defer resp.Body.Close()

	var uploadResp PostStreamUploadResponse
	if err := l.httpClient.HandleAndDecode(resp, 201, &uploadResp, "stream upload"); err != nil {
		return nil, err
	}

	return &uploadResp, nil
}

// ListStreams lists all streams for the authenticated user
func (l *LBRYClient) ListStreams() (*StreamResponseResponse, error) {
	resp, err := l.httpClient.Get(LBRYEndpointStreams)
	if err != nil {
		return nil, fmt.Errorf("failed to list streams: %w", err)
	}
	defer resp.Body.Close()

	var streamsResp StreamResponseResponse
	if err := l.httpClient.HandleAndDecode(resp, 200, &streamsResp, "list streams"); err != nil {
		return nil, err
	}

	return &streamsResp, nil
}

// PinStream pins a stream to keep it available on the LBRY network
func (l *LBRYClient) PinStream(sdHash string) error {
	request := StreamPinRequest{
		SDHash: sdHash,
	}

	resp, err := l.httpClient.PostJSON(LBRYEndpointStreamPin, request)
	if err != nil {
		return fmt.Errorf("failed to pin stream: %w", err)
	}
	defer resp.Body.Close()

	return l.httpClient.HandleResponse(resp, 201, "stream pin")
}

// DeleteStream deletes a stream by its SD hash
func (l *LBRYClient) DeleteStream(sdHash string) error {
	url := l.httpClient.GetBaseURL() + LBRYEndpointStreams + "/" + sdHash

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := l.httpClient.ExecuteRequest(req)
	if err != nil {
		return fmt.Errorf("failed to delete stream: %w", err)
	}
	defer resp.Body.Close()

	return l.httpClient.HandleResponse(resp, 204, "stream deletion")
}
