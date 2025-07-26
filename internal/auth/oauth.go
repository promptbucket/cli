package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// OAuthResult represents the result of OAuth authentication
type OAuthResult struct {
	Success bool
	Token   *Token
	Error   string
}

// OAuthServer handles the local callback server for OAuth
type OAuthServer struct {
	config   *Config
	server   *http.Server
	resultCh chan OAuthResult
}

// NewOAuthServer creates a new OAuth callback server
func NewOAuthServer(config *Config) *OAuthServer {
	return &OAuthServer{
		config:   config,
		resultCh: make(chan OAuthResult, 1),
	}
}

// Start starts the OAuth callback server and waits for authentication
func (os *OAuthServer) Start(ctx context.Context) (*OAuthResult, error) {
	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", os.handleCallback)

	os.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", os.config.CallbackPort),
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := os.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			os.resultCh <- OAuthResult{Success: false, Error: fmt.Sprintf("Server error: %v", err)}
		}
	}()

	// Wait for result or timeout
	select {
	case result := <-os.resultCh:
		return &result, nil
	case <-ctx.Done():
		os.Stop()
		return &OAuthResult{Success: false, Error: "OAuth timeout"}, nil
	}
}

// Stop stops the OAuth callback server
func (os *OAuthServer) Stop() {
	if os.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		os.server.Shutdown(ctx)
	}
}

// handleCallback handles the OAuth callback request
func (os *OAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	// Check for error
	if errorMsg := query.Get("error"); errorMsg != "" {
		os.sendErrorResponse(w, errorMsg)
		
		// Delay shutdown to allow page to load
		go func() {
			time.Sleep(5 * time.Second)
			os.Stop()
		}()
		
		os.resultCh <- OAuthResult{Success: false, Error: errorMsg}
		return
	}

	// Extract auth data
	token := query.Get("token")
	userID := query.Get("user_id")
	email := query.Get("email")
	name := query.Get("name")
	avatarURL := query.Get("avatar_url")
	provider := query.Get("provider")

	// Validate required fields
	if token == "" || userID == "" || email == "" {
		os.sendErrorResponse(w, "Missing required authentication parameters")
		
		// Delay shutdown to allow page to load
		go func() {
			time.Sleep(5 * time.Second)
			os.Stop()
		}()
		
		os.resultCh <- OAuthResult{Success: false, Error: "Missing required OAuth parameters"}
		return
	}

	// Create token object
	authToken := &Token{
		AccessToken: token,
		UserID:      userID,
		Email:       email,
		Name:        name,
		AvatarURL:   avatarURL,
		Provider:    provider,
	}

	// Send success response
	os.sendSuccessResponse(w)
	
	// Delay shutdown to allow page to load and auto-close
	go func() {
		time.Sleep(3 * time.Second)
		os.Stop()
	}()
	
	os.resultCh <- OAuthResult{Success: true, Token: authToken}
}

// sendSuccessResponse sends a success HTML response
func (srv *OAuthServer) sendSuccessResponse(w http.ResponseWriter) {
	// Try to load HTML from assets/auth-success.html
	htmlPath := filepath.Join("assets", "auth-success.html")
	htmlContent, err := os.ReadFile(htmlPath)

	var html string
	if err != nil {
		// Fallback to embedded HTML if file read fails
		html = `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
               padding: 40px; text-align: center; background: #f8fafc; }
        .success { color: #059669; }
        .container { max-width: 400px; margin: 0 auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="success">✅ Authentication Successful!</h1>
        <p>You have been successfully authenticated with PromptBucket.</p>
        <p>You can now close this window and return to your terminal.</p>
    </div>
    <script>
        setTimeout(function() { window.close(); }, 2000);
    </script>
</body>
</html>`
	} else {
		html = string(htmlContent)
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// sendErrorResponse sends an error HTML response
func (srv *OAuthServer) sendErrorResponse(w http.ResponseWriter, errorMsg string) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Authentication Failed</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
               padding: 40px; text-align: center; background: #f8fafc; }
        .error { color: #dc2626; }
        .container { max-width: 400px; margin: 0 auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="error">❌ Authentication Failed</h1>
        <p>%s</p>
        <p>You can close this window and try again.</p>
    </div>
</body>
</html>`, errorMsg)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(html))
}

// IsPortAvailable checks if the callback port is available
func (os *OAuthServer) IsPortAvailable() bool {
	addr := fmt.Sprintf(":%d", os.config.CallbackPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
