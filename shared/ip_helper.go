package shared

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// IPResponse represents the response from IPify API
type IPResponse struct {
	IP string `json:"ip"`
}

// GetPublicIP fetches the public IP address from IPify API
func GetPublicIP() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://api.ipify.org?format=json")
	if err != nil {
		return "", fmt.Errorf("failed to fetch public IP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IPify API returned status: %d", resp.StatusCode)
	}

	var ipResp IPResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipResp); err != nil {
		return "", fmt.Errorf("failed to decode IP response: %w", err)
	}

	if ipResp.IP == "" {
		return "", fmt.Errorf("empty IP address received from IPify API")
	}

	return ipResp.IP, nil
}