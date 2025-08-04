package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/promptbucket/cli/internal/auth"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull [package]",
	Short: "Pull a package from the registry",
	Long: `Pull a package from the PromptBucket registry and save it as a YAML file.

Package can be specified as:
  - org/name:version (e.g., rawte.mayur/Ui-Artist:0.1.0)
  - org/name (pulls latest version)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageSpec := args[0]
		outputDir, _ := cmd.Flags().GetString("output")

		// Parse package specification
		var org, packageName, version string
		
		// Split by / to get org and package
		parts := strings.Split(packageSpec, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid package format: %s (expected org/name[:version])", packageSpec)
		}
		org = parts[0]
		
		// Check for version in the package name part
		if idx := strings.LastIndex(parts[1], ":"); idx > 0 {
			packageName = parts[1][:idx]
			version = parts[1][idx+1:]
		} else {
			packageName = parts[1]
			// For now, we'll need to specify version explicitly
			// TODO: Add API endpoint to get latest version
			return fmt.Errorf("please specify version (e.g., %s:0.1.0)", packageSpec)
		}

		// Create auth config and API client
		config := auth.NewConfig()
		apiClient := auth.NewAPIClient(config)

		// Build endpoint for manifest
		endpoint := fmt.Sprintf("/manifests/%s/%s/%s", org, packageName, version)

		// Download manifest
		fmt.Printf("📥 Pulling %s/%s:%s...\n", org, packageName, version)
		
		resp, err := apiClient.Get(endpoint)
		if err != nil {
			return fmt.Errorf("failed to pull package: %w", err)
		}
		defer resp.Body.Close()

		// Check status
		if resp.StatusCode == 404 {
			return fmt.Errorf("package not found: %s/%s:%s", org, packageName, version)
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("pull failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Read manifest data
		manifestData, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to download manifest: %w", err)
		}

		// Generate filename
		filename := fmt.Sprintf("%s-%s.yaml", packageName, version)
		
		// Determine output path
		outputPath := filename
		if outputDir != "" {
			// Create output directory if it doesn't exist
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}
			outputPath = filepath.Join(outputDir, filename)
		}

		// Write manifest file
		if err := os.WriteFile(outputPath, manifestData, 0644); err != nil {
			return fmt.Errorf("failed to save manifest: %w", err)
		}

		fileInfo, _ := os.Stat(outputPath)
		fmt.Printf("✅ Downloaded %s (%.2f KB)\n", outputPath, float64(fileInfo.Size())/1024)

		// Show next steps
		fmt.Println("\nNext steps:")
		fmt.Printf("  • View the manifest: cat %s\n", outputPath)
		fmt.Printf("  • Build with variables: promptbucket run --var key=value\n")
		fmt.Printf("  • Validate: promptbucket validate %s\n", outputPath)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
	pullCmd.Flags().StringP("output", "o", "", "Output directory for the manifest file")
}