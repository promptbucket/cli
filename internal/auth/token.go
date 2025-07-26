package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Token represents the authentication token and user info
type Token struct {
	AccessToken string    `json:"access_token"`
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	Name        string    `json:"name,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// TokenManager handles token storage and retrieval
type TokenManager struct {
	config *Config
}

// NewTokenManager creates a new token manager
func NewTokenManager(config *Config) *TokenManager {
	return &TokenManager{config: config}
}

// SaveToken saves the auth token to disk
func (tm *TokenManager) SaveToken(token *Token) error {
	// Ensure config directory exists
	if err := os.MkdirAll(tm.config.ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal token to JSON
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write to file
	tokenPath := tm.config.GetTokenPath()
	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// GetToken retrieves the stored auth token
func (tm *TokenManager) GetToken() (*Token, error) {
	tokenPath := tm.config.GetTokenPath()
	
	// Check if file exists
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return nil, nil // No token file
	}

	// Read file
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	// Unmarshal token
	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	// Check if token is expired
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		tm.ClearToken() // Clear expired token
		return nil, nil
	}

	return &token, nil
}

// ClearToken removes the stored auth token
func (tm *TokenManager) ClearToken() error {
	tokenPath := tm.config.GetTokenPath()
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to clear
	}
	
	if err := os.Remove(tokenPath); err != nil {
		return fmt.Errorf("failed to remove token file: %w", err)
	}
	
	return nil
}

// IsAuthenticated checks if user is authenticated with a valid token
func (tm *TokenManager) IsAuthenticated() bool {
	token, err := tm.GetToken()
	return err == nil && token != nil && token.AccessToken != ""
}

// GetAuthHeaders returns HTTP headers for authenticated requests
func (tm *TokenManager) GetAuthHeaders() (map[string]string, error) {
	token, err := tm.GetToken()
	if err != nil {
		return nil, err
	}
	if token == nil {
		return map[string]string{}, nil
	}
	
	return map[string]string{
		"Authorization": "Bearer " + token.AccessToken,
	}, nil
}