package auth

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// Config holds auth configuration
type Config struct {
	BaseURL    string
	APIVersion string
	CallbackPort    int
	ConfigDir       string
	TokenFile       string
}

// NewConfig creates a new auth config with defaults
func NewConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".promptbucket")
	
	return &Config{
		BaseURL:    getEnvOrDefault("PROMPTBUCKET_BASE_URL", "https://harbor.promptbucket.co"),
		APIVersion: getEnvOrDefault("PROMPTBUCKET_API_VERSION", "v1"),
		CallbackPort:    3456,
		ConfigDir:       configDir,
		TokenFile:       "token.json",
	}
}

// GetAPIURL returns the full API URL
func (c *Config) GetAPIURL(endpoint string) string {
	baseURL := c.BaseURL
	if baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}
	if endpoint != "" && endpoint[0] != '/' {
		endpoint = "/" + endpoint
	}
	return baseURL + "/" + c.APIVersion + endpoint
}

// GetOAuthURL returns the OAuth URL for a provider with CLI callback
func (c *Config) GetOAuthURL(provider string) string {
	callbackURL := fmt.Sprintf("http://localhost:%d/callback", c.CallbackPort)
	oauthURL := c.GetAPIURL("/auth/" + provider)
	return oauthURL + "?redirect_uri=" + url.QueryEscape(callbackURL)
}

// GetTokenPath returns the full path to the token file
func (c *Config) GetTokenPath() string {
	return filepath.Join(c.ConfigDir, c.TokenFile)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}