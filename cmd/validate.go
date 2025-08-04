package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/promptbucket/cli/internal/packager"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var validateCmd = &cobra.Command{
	Use:   "validate [prompt_name]",
	Short: "Validate a prompt manifest file",
	Long: `Validate a prompt manifest file against the schema.

If no prompt_name is specified, validates the current directory's promptbucket.yaml.
If prompt_name is specified, looks for promptbucket.yaml in that directory or treats it as a file path.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var manifestPath string
		
		if len(args) == 0 {
			// No argument provided, use current directory
			manifestPath = packager.ManifestFile
		} else {
			promptName := args[0]
			
			// Check if it's a file path
			if filepath.Ext(promptName) == ".yaml" || filepath.Ext(promptName) == ".yml" {
				manifestPath = promptName
			} else {
				// Assume it's a directory name
				manifestPath = filepath.Join(promptName, packager.ManifestFile)
			}
		}
		
		return validateManifest(manifestPath)
	},
}

func validateManifest(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("manifest file not found: %s", path)
	}
	
	// Read the manifest file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read manifest file %s: %w", path, err)
	}
	
	// Parse YAML
	var manifest packager.Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("invalid YAML syntax in %s: %w", path, err)
	}
	
	// Validate required fields
	var errors []string
	
	if manifest.Name == "" {
		errors = append(errors, "missing required field: name")
	}
	
	if manifest.Version == "" {
		errors = append(errors, "missing required field: version")
	}
	
	if manifest.Licence == "" {
		errors = append(errors, "missing required field: licence")
	}
	
	if manifest.Prompt == "" {
		errors = append(errors, "missing required field: prompt")
	}
	
	// Validate name pattern
	if manifest.Name != "" {
		if !validateNamePattern(manifest.Name) {
			errors = append(errors, "name must match pattern: ^[a-z0-9]([a-z0-9-_]{0,38}[a-z0-9])?$")
		}
	}
	
	// Validate version pattern (semantic versioning)
	if manifest.Version != "" {
		if !validateVersionPattern(manifest.Version) {
			errors = append(errors, "version must follow semantic versioning: major.minor.patch")
		}
	}
	
	// Validate variables
	if len(manifest.Variables) > 0 {
		if err := validateVariables(manifest.Variables); err != nil {
			errors = append(errors, err.Error())
		}
	}
	
	// Validate persona
	if manifest.Persona != nil {
		if err := validatePersona(manifest.Persona); err != nil {
			errors = append(errors, err.Error())
		}
	}
	
	// Report validation results
	if len(errors) > 0 {
		fmt.Printf("❌ Validation failed for %s:\n", path)
		for _, err := range errors {
			fmt.Printf("  - %s\n", err)
		}
		return fmt.Errorf("validation failed")
	}
	
	fmt.Printf("✅ %s is valid\n", path)
	
	// Try to flatten manifest if it has inheritance
	if manifest.From != "" {
		fmt.Printf("ℹ️  Checking inheritance chain...\n")
		_, err := packager.FlattenManifest(&manifest)
		if err != nil {
			fmt.Printf("⚠️  Warning: inheritance validation failed: %s\n", err)
		} else {
			fmt.Printf("✅ Inheritance chain is valid\n")
		}
	}
	
	return nil
}

func validateNamePattern(name string) bool {
	// Pattern: ^[a-z0-9]([a-z0-9-_]{0,38}[a-z0-9])?$
	if len(name) == 0 || len(name) > 40 {
		return false
	}
	
	// First character must be alphanumeric
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= '0' && first <= '9')) {
		return false
	}
	
	// Single character names are valid
	if len(name) == 1 {
		return true
	}
	
	// Last character must be alphanumeric
	last := name[len(name)-1]
	if !((last >= 'a' && last <= 'z') || (last >= '0' && last <= '9')) {
		return false
	}
	
	// Middle characters can be alphanumeric, dash, or underscore
	for i := 1; i < len(name)-1; i++ {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	
	return true
}

func validateVersionPattern(version string) bool {
	// Basic semantic versioning check: major.minor.patch with optional pre-release
	parts := make([]rune, 0)
	dotCount := 0
	
	for _, r := range version {
		if r == '.' {
			dotCount++
			if dotCount > 2 {
				return false
			}
		} else if r >= '0' && r <= '9' {
			// digits are ok
		} else if r == '-' && dotCount == 2 {
			// pre-release suffix after patch version is ok
			break
		} else {
			return false
		}
		parts = append(parts, r)
	}
	
	return dotCount == 2
}

func validateVariables(variables []packager.Variable) error {
	names := make(map[string]bool)
	
	for _, v := range variables {
		// Check for duplicate names
		if names[v.Name] {
			return fmt.Errorf("duplicate variable name: %s", v.Name)
		}
		names[v.Name] = true
		
		// Validate variable name pattern
		if !validateVariableName(v.Name) {
			return fmt.Errorf("variable name '%s' must match pattern: ^[a-zA-Z][a-zA-Z0-9_]*$", v.Name)
		}
		
		// Validate description length
		if len(v.Description) > 120 {
			return fmt.Errorf("variable '%s' description exceeds 120 characters", v.Name)
		}
		
		// Validate example length
		if len(v.Example) > 120 {
			return fmt.Errorf("variable '%s' example exceeds 120 characters", v.Name)
		}
	}
	
	return nil
}

func validateVariableName(name string) bool {
	if len(name) == 0 {
		return false
	}
	
	// First character must be a letter
	first := name[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z')) {
		return false
	}
	
	// Remaining characters must be letters, digits, or underscores
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	
	return true
}

func validatePersona(persona *packager.Persona) error {
	// Validate field lengths according to schema
	if len(persona.Name) > 100 {
		return fmt.Errorf("persona name exceeds 100 characters")
	}
	
	if len(persona.Role) > 100 {
		return fmt.Errorf("persona role exceeds 100 characters")
	}
	
	if len(persona.Experience) > 100 {
		return fmt.Errorf("persona experience exceeds 100 characters")
	}
	
	if len(persona.Background) > 500 {
		return fmt.Errorf("persona background exceeds 500 characters")
	}
	
	if len(persona.Tone) > 50 {
		return fmt.Errorf("persona tone exceeds 50 characters")
	}
	
	if len(persona.Style) > 50 {
		return fmt.Errorf("persona style exceeds 50 characters")
	}
	
	if len(persona.LanguageLevel) > 50 {
		return fmt.Errorf("persona language_level exceeds 50 characters")
	}
	
	if len(persona.Approach) > 50 {
		return fmt.Errorf("persona approach exceeds 50 characters")
	}
	
	if len(persona.InteractionStyle) > 200 {
		return fmt.Errorf("persona interaction_style exceeds 200 characters")
	}
	
	if len(persona.OutputFormat) > 50 {
		return fmt.Errorf("persona output_format exceeds 50 characters")
	}
	
	// Validate array limits
	if len(persona.Personality) > 10 {
		return fmt.Errorf("persona personality array exceeds 10 items")
	}
	
	if len(persona.Expertise) > 20 {
		return fmt.Errorf("persona expertise array exceeds 20 items")
	}
	
	if len(persona.Focus) > 10 {
		return fmt.Errorf("persona focus array exceeds 10 items")
	}
	
	if len(persona.Constraints) > 10 {
		return fmt.Errorf("persona constraints array exceeds 10 items")
	}
	
	if len(persona.Preferences) > 10 {
		return fmt.Errorf("persona preferences array exceeds 10 items")
	}
	
	// Validate individual array item lengths
	for _, p := range persona.Personality {
		if len(p) > 50 {
			return fmt.Errorf("persona personality item '%s' exceeds 50 characters", p)
		}
	}
	
	for _, e := range persona.Expertise {
		if len(e) > 50 {
			return fmt.Errorf("persona expertise item '%s' exceeds 50 characters", e)
		}
	}
	
	for _, f := range persona.Focus {
		if len(f) > 50 {
			return fmt.Errorf("persona focus item '%s' exceeds 50 characters", f)
		}
	}
	
	for _, c := range persona.Constraints {
		if len(c) > 200 {
			return fmt.Errorf("persona constraint '%s' exceeds 200 characters", c)
		}
	}
	
	for _, p := range persona.Preferences {
		if len(p) > 200 {
			return fmt.Errorf("persona preference '%s' exceeds 200 characters", p)
		}
	}
	
	return nil
}

func init() {
	rootCmd.AddCommand(validateCmd)
}