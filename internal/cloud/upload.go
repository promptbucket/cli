package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/promptbucket/cli/internal/runner"
)

const (
	defaultAPIURL = "https://api.promptbucket.co"
	uploadTimeout = 30 * time.Second
)

// UploadResults POSTs the SuiteResult to the PromptBucket cloud API.
// It prints a success message on success or a warning on failure, but never
// returns a hard error so that the CLI does not abort.
func UploadResults(apiKey string, results *runner.SuiteResult) {
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Warning: --cloud flag set but no API key provided. Set --api-key or PROMPTBUCKET_API_KEY.")
		return
	}

	apiURL := os.Getenv("PROMPTBUCKET_API_URL")
	if apiURL == "" {
		apiURL = defaultAPIURL
	}

	body, err := json.Marshal(results)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to marshal results for upload: %v\n", err)
		return
	}

	endpoint := apiURL + "/v1/results"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to create upload request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{Timeout: uploadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to upload results to cloud: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "Warning: cloud upload returned HTTP %d\n", resp.StatusCode)
		return
	}

	fmt.Println("Results uploaded to promptbucket.co")
}
