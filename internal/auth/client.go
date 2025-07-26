package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient is an HTTP client that automatically handles authentication
type APIClient struct {
	config       *Config
	tokenManager *TokenManager
	httpClient   *http.Client
}

// NewAPIClient creates a new authenticated API client
func NewAPIClient(config *Config) *APIClient {
	return &APIClient{
		config:       config,
		tokenManager: NewTokenManager(config),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Request makes an authenticated HTTP request
func (c *APIClient) Request(method, endpoint string, body interface{}) (*http.Response, error) {
	url := c.config.GetAPIURL(endpoint)
	
	// Prepare request body
	var requestBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		requestBody = bytes.NewBuffer(jsonData)
	}

	// Create request
	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "promptbucket-cli")

	// Add auth headers if authenticated
	if c.tokenManager.IsAuthenticated() {
		authHeaders, err := c.tokenManager.GetAuthHeaders()
		if err != nil {
			return nil, fmt.Errorf("failed to get auth headers: %w", err)
		}
		for key, value := range authHeaders {
			req.Header.Set(key, value)
		}
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Handle 401 Unauthorized - token may be expired
	if resp.StatusCode == 401 && c.tokenManager.IsAuthenticated() {
		// Clear expired token
		c.tokenManager.ClearToken()
		return nil, fmt.Errorf("authentication failed - please run 'promptbucket login' to re-authenticate")
	}

	return resp, nil
}

// Get makes a GET request
func (c *APIClient) Get(endpoint string) (*http.Response, error) {
	return c.Request("GET", endpoint, nil)
}

// Post makes a POST request
func (c *APIClient) Post(endpoint string, body interface{}) (*http.Response, error) {
	return c.Request("POST", endpoint, body)
}

// Put makes a PUT request
func (c *APIClient) Put(endpoint string, body interface{}) (*http.Response, error) {
	return c.Request("PUT", endpoint, body)
}

// Delete makes a DELETE request
func (c *APIClient) Delete(endpoint string) (*http.Response, error) {
	return c.Request("DELETE", endpoint, nil)
}

// IsAuthenticated checks if the client is authenticated
func (c *APIClient) IsAuthenticated() bool {
	return c.tokenManager.IsAuthenticated()
}

// GetCurrentUser returns the current authenticated user info
func (c *APIClient) GetCurrentUser() (*Token, error) {
	return c.tokenManager.GetToken()
}