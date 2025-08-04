package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/promptbucket/cli/internal/auth"
	"github.com/promptbucket/cli/internal/packager"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push a .promptbucket package to the registry",
	Long: `Push a .promptbucket package to the PromptBucket registry.

This command builds the package from the current directory's promptbucket.yaml
and uploads it to the registry. You must be authenticated to push packages.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create auth config and API client
		config := auth.NewConfig()
		apiClient := auth.NewAPIClient(config)

		// Check if authenticated
		if !apiClient.IsAuthenticated() {
			return fmt.Errorf("not authenticated. Run 'promptbucket login' first")
		}

		// Get current user info
		user, err := apiClient.GetCurrentUser()
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		fmt.Printf("🚀 Pushing package as %s\n", user.Email)

		// Read the manifest
		manifestData, err := os.ReadFile(packager.ManifestFile)
		if err != nil {
			return fmt.Errorf("failed to read manifest: %w", err)
		}

		var manifest packager.Manifest
		if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
			return fmt.Errorf("failed to parse manifest: %w", err)
		}

		// Validate required fields
		if manifest.Name == "" || manifest.Version == "" || manifest.Licence == "" || manifest.Prompt == "" {
			return fmt.Errorf("manifest missing required fields (name, version, licence, prompt)")
		}

		// Extract username from email (everything before @)
		username := user.Email
		if idx := strings.Index(user.Email, "@"); idx != -1 {
			username = user.Email[:idx]
		}

		// Calculate digest
		hash := sha256.Sum256(manifestData)
		digest := hex.EncodeToString(hash[:])

		fmt.Printf("📦 Pushing %s/%s:%s\n", username, manifest.Name, manifest.Version)
		fmt.Printf("   Digest: sha256:%s\n", digest)

		// Upload the manifest using the manifest endpoint
		endpoint := fmt.Sprintf("/manifests/%s/%s/%s", username, manifest.Name, manifest.Version)
		fmt.Println("📤 Uploading to registry...")

		// Create request with YAML body
		req, err := http.NewRequest("PUT", config.GetAPIURL(endpoint), strings.NewReader(string(manifestData)))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/x-yaml")
		req.Header.Set("User-Agent", "promptbucket-cli")

		// Get auth headers directly
		tokenManager := auth.NewTokenManager(config)
		if tokenManager.IsAuthenticated() {
			headers, err := tokenManager.GetAuthHeaders()
			if err != nil {
				return fmt.Errorf("failed to get auth headers: %w", err)
			}
			for key, value := range headers {
				req.Header.Set(key, value)
			}
		}

		// Make request using http client
		httpClient := &http.Client{}
		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to upload package: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Check status
		if resp.StatusCode == 409 {
			return fmt.Errorf("package version %s/%s:%s already exists", username, manifest.Name, manifest.Version)
		}
		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
		}

		// Success!
		fmt.Println("✅ Package pushed successfully!")
		fmt.Printf("   Package: %s/%s:%s\n", username, manifest.Name, manifest.Version)
		fmt.Printf("   Digest: sha256:%s\n", digest)
		
		// Show how to pull the package
		fmt.Println("\nTo pull this package:")
		fmt.Printf("   promptbucket pull %s/%s:%s\n", username, manifest.Name, manifest.Version)
		
		// Show how to fetch and run
		fmt.Println("\nTo fetch and run:")
		fmt.Printf("   promptbucket fetch https://api.promptbucket.co/v1/manifests/%s/%s/%s\n", username, manifest.Name, manifest.Version)

		// Optionally build the .promptbucket archive file
		if buildFlag, _ := cmd.Flags().GetBool("build"); buildFlag {
			fmt.Println("\n📦 Building local package archive...")
			artifact, err := packager.Build()
			if err != nil {
				fmt.Printf("⚠️  Warning: Could not build package archive: %v\n", err)
			} else {
				fmt.Printf("✅ Built %s (%.2f KB)\n", artifact.Path, float64(artifact.Size)/1024)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().Bool("build", false, "Also build a local .promptbucket archive file")
}