package packager

import (
    "archive/tar"
    "bytes"
    "compress/gzip"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "net/http"
    "os"
    "regexp"
    "strings"

    "gopkg.in/yaml.v3"
)

type Artifact struct {
    Path   string
    Size   int64
    Digest string
}

// Build reads promptbucket.yaml and produces a .promptbucket package in the current directory.
func Build() (Artifact, error) {
    var art Artifact
    data, err := os.ReadFile(ManifestFile)
    if err != nil {
        return art, err
    }

    var m Manifest
    if err := yaml.Unmarshal(data, &m); err != nil {
        return art, err
    }
    if m.Name == "" || m.Version == "" || m.Licence == "" || m.Prompt == "" {
        return art, fmt.Errorf("manifest missing required fields")
    }

    // tar
    var tarBuf bytes.Buffer
    tw := tar.NewWriter(&tarBuf)
    hdr := &tar.Header{Name: ManifestFile, Mode: 0644, Size: int64(len(data))}
    if err := tw.WriteHeader(hdr); err != nil {
        return art, err
    }
    if _, err := tw.Write(data); err != nil {
        return art, err
    }
    if err := tw.Close(); err != nil {
        return art, err
    }

    // gzip
    var gzBuf bytes.Buffer
    gw := gzip.NewWriter(&gzBuf)
    if _, err := io.Copy(gw, &tarBuf); err != nil {
        return art, err
    }
    if err := gw.Close(); err != nil {
        return art, err
    }

    payload := append([]byte(MagicHeader), gzBuf.Bytes()...)
    sum := sha256.Sum256(payload)
    digest := "sha256:" + hex.EncodeToString(sum[:])

    out := fmt.Sprintf("%s-%s.promptbucket", m.Name, m.Version)
    if err := os.WriteFile(out, payload, 0644); err != nil {
        return art, err
    }
    info, err := os.Stat(out)
    if err != nil {
        return art, err
    }

    art.Path = out
    art.Size = info.Size()
    art.Digest = digest
    return art, nil
}

// ValidateVariables checks if all required variables are provided
func ValidateVariables(m *Manifest, vars map[string]string) error {
    for _, v := range m.Variables {
        if _, exists := vars[v.Name]; !exists {
            return fmt.Errorf("missing required variable: %s", v.Name)
        }
    }
    return nil
}

// SubstituteVariables replaces {{variable}} patterns in the prompt
func SubstituteVariables(prompt string, vars map[string]string) string {
    re := regexp.MustCompile(`\{\{(\w+)\}\}`)
    return re.ReplaceAllStringFunc(prompt, func(match string) string {
        varName := strings.Trim(match, "{}")
        if value, exists := vars[varName]; exists {
            return value
        }
        return match
    })
}

// ParseVarFlags converts --var key=value flags into a map
func ParseVarFlags(varFlags []string) (map[string]string, error) {
    vars := make(map[string]string)
    for _, flag := range varFlags {
        parts := strings.SplitN(flag, "=", 2)
        if len(parts) != 2 {
            return nil, fmt.Errorf("invalid variable format: %s (expected key=value)", flag)
        }
        vars[parts[0]] = parts[1]
    }
    return vars, nil
}

// LoadManifestFromPath loads a manifest from a local file or URL
func LoadManifestFromPath(path string) (*Manifest, error) {
    var data []byte
    var err error
    
    if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
        resp, err := http.Get(path)
        if err != nil {
            return nil, fmt.Errorf("failed to fetch %s: %w", path, err)
        }
        defer resp.Body.Close()
        
        data, err = io.ReadAll(resp.Body)
        if err != nil {
            return nil, fmt.Errorf("failed to read response from %s: %w", path, err)
        }
    } else {
        data, err = os.ReadFile(path)
        if err != nil {
            return nil, fmt.Errorf("failed to read file %s: %w", path, err)
        }
    }
    
    var m Manifest
    if err := yaml.Unmarshal(data, &m); err != nil {
        return nil, fmt.Errorf("failed to parse YAML from %s: %w", path, err)
    }
    
    return &m, nil
}

// FlattenManifest resolves inheritance chain up to 2 levels
func FlattenManifest(m *Manifest) (*Manifest, error) {
    result := *m
    depth := 0
    current := &result
    
    for current.From != "" && depth < 2 {
        parent, err := LoadManifestFromPath(current.From)
        if err != nil {
            return nil, fmt.Errorf("failed to load parent manifest from %s: %w", current.From, err)
        }
        
        // Merge parent into current (child overrides parent)
        merged := mergeManifests(parent, current)
        current = merged
        depth++
    }
    
    if current.From != "" && depth >= 2 {
        return nil, fmt.Errorf("inheritance chain too deep (max 2 levels)")
    }
    
    return current, nil
}

// mergeManifests merges child manifest into parent (child takes precedence)
func mergeManifests(parent, child *Manifest) *Manifest {
    result := *parent
    
    // Child overrides parent for all fields except variables which are merged
    if child.Name != "" {
        result.Name = child.Name
    }
    if child.Version != "" {
        result.Version = child.Version
    }
    if child.Licence != "" {
        result.Licence = child.Licence
    }
    if child.Description != "" {
        result.Description = child.Description
    }
    if len(child.Authors) > 0 {
        result.Authors = child.Authors
    }
    if len(child.Tags) > 0 {
        result.Tags = child.Tags
    }
    if child.Language != "" {
        result.Language = child.Language
    }
    if child.ModelHint != "" {
        result.ModelHint = child.ModelHint
    }
    if child.Prompt != "" {
        result.Prompt = child.Prompt
    }
    
    // Merge persona (child completely overrides parent persona)
    if child.Persona != nil {
        result.Persona = child.Persona
    }
    
    // Merge variables (child variables with same name override parent)
    varMap := make(map[string]Variable)
    for _, v := range parent.Variables {
        varMap[v.Name] = v
    }
    for _, v := range child.Variables {
        varMap[v.Name] = v
    }
    
    result.Variables = make([]Variable, 0, len(varMap))
    for _, v := range varMap {
        result.Variables = append(result.Variables, v)
    }
    
    // Clear 'from' field in result
    result.From = ""
    
    return &result
}

// BuildWithVariables builds a prompt with variable substitution and generates final_prompt.md
func BuildWithVariables(varFlags []string) error {
    // Parse variables
    vars, err := ParseVarFlags(varFlags)
    if err != nil {
        return err
    }
    
    // Load and parse manifest
    data, err := os.ReadFile(ManifestFile)
    if err != nil {
        return err
    }
    
    var m Manifest
    if err := yaml.Unmarshal(data, &m); err != nil {
        return err
    }
    
    // Validate required fields
    if m.Name == "" || m.Version == "" || m.Licence == "" || m.Prompt == "" {
        return fmt.Errorf("manifest missing required fields")
    }
    
    // Flatten inheritance
    flattened, err := FlattenManifest(&m)
    if err != nil {
        return err
    }
    
    // Validate variables
    if err := ValidateVariables(flattened, vars); err != nil {
        return err
    }
    
    // Generate persona-aware prompt and then substitute variables
    personaPrompt := GeneratePersonaPrompt(flattened)
    finalPrompt := SubstituteVariables(personaPrompt, vars)
    
    // Generate filename based on name and version
    filename := fmt.Sprintf("%s-%s-prompt.md", flattened.Name, flattened.Version)
    
    // Write prompt file
    if err := os.WriteFile(filename, []byte(finalPrompt), 0644); err != nil {
        return fmt.Errorf("failed to write %s: %w", filename, err)
    }
    
    fmt.Printf("Generated %s with resolved variables\n", filename)
    return nil
}

// FetchAndBuild downloads YAML from registry and builds with variables
func FetchAndBuild(url string, varFlags []string) error {
    // Parse variables
    vars, err := ParseVarFlags(varFlags)
    if err != nil {
        return err
    }
    
    // Fetch manifest from URL
    m, err := LoadManifestFromPath(url)
    if err != nil {
        return err
    }
    
    // Validate required fields
    if m.Name == "" || m.Version == "" || m.Licence == "" || m.Prompt == "" {
        return fmt.Errorf("manifest missing required fields")
    }
    
    // Flatten inheritance
    flattened, err := FlattenManifest(m)
    if err != nil {
        return err
    }
    
    // Validate variables
    if err := ValidateVariables(flattened, vars); err != nil {
        return err
    }
    
    // Generate persona-aware prompt and then substitute variables
    personaPrompt := GeneratePersonaPrompt(flattened)
    finalPrompt := SubstituteVariables(personaPrompt, vars)
    
    // Generate filename based on name and version
    filename := fmt.Sprintf("%s-%s-prompt.md", flattened.Name, flattened.Version)
    
    // Write prompt file
    if err := os.WriteFile(filename, []byte(finalPrompt), 0644); err != nil {
        return fmt.Errorf("failed to write %s: %w", filename, err)
    }
    
    fmt.Printf("Fetched %s and generated %s with resolved variables\n", url, filename)
    return nil
}

// GeneratePersonaPrompt creates a persona-aware prompt by combining persona info with the main prompt
func GeneratePersonaPrompt(m *Manifest) string {
    if m.Persona == nil {
        return m.Prompt
    }
    
    var promptBuilder strings.Builder
    
    // Start with identity
    if m.Persona.Name != "" || m.Persona.Role != "" {
        promptBuilder.WriteString("# Identity\n")
        if m.Persona.Name != "" && m.Persona.Role != "" {
            promptBuilder.WriteString(fmt.Sprintf("You are %s, a %s.\n", m.Persona.Name, m.Persona.Role))
        } else if m.Persona.Name != "" {
            promptBuilder.WriteString(fmt.Sprintf("You are %s.\n", m.Persona.Name))
        } else if m.Persona.Role != "" {
            promptBuilder.WriteString(fmt.Sprintf("You are a %s.\n", m.Persona.Role))
        }
        promptBuilder.WriteString("\n")
    }
    
    // Add background and expertise
    if m.Persona.Background != "" || len(m.Persona.Expertise) > 0 || m.Persona.Experience != "" {
        promptBuilder.WriteString("# Background & Expertise\n")
        
        if m.Persona.Background != "" {
            promptBuilder.WriteString(fmt.Sprintf("Background: %s\n", m.Persona.Background))
        }
        
        if m.Persona.Experience != "" {
            promptBuilder.WriteString(fmt.Sprintf("Experience: %s\n", m.Persona.Experience))
        }
        
        if len(m.Persona.Expertise) > 0 {
            promptBuilder.WriteString(fmt.Sprintf("Areas of expertise: %s\n", strings.Join(m.Persona.Expertise, ", ")))
        }
        promptBuilder.WriteString("\n")
    }
    
    // Add personality and communication style
    if len(m.Persona.Personality) > 0 || m.Persona.Tone != "" || m.Persona.Style != "" {
        promptBuilder.WriteString("# Communication Style\n")
        
        if len(m.Persona.Personality) > 0 {
            promptBuilder.WriteString(fmt.Sprintf("Personality traits: %s\n", strings.Join(m.Persona.Personality, ", ")))
        }
        
        if m.Persona.Tone != "" {
            promptBuilder.WriteString(fmt.Sprintf("Tone: %s\n", m.Persona.Tone))
        }
        
        if m.Persona.Style != "" {
            promptBuilder.WriteString(fmt.Sprintf("Communication style: %s\n", m.Persona.Style))
        }
        
        if m.Persona.LanguageLevel != "" {
            promptBuilder.WriteString(fmt.Sprintf("Technical level: %s\n", m.Persona.LanguageLevel))
        }
        
        if m.Persona.InteractionStyle != "" {
            promptBuilder.WriteString(fmt.Sprintf("Interaction approach: %s\n", m.Persona.InteractionStyle))
        }
        promptBuilder.WriteString("\n")
    }
    
    // Add approach and focus areas
    if m.Persona.Approach != "" || len(m.Persona.Focus) > 0 {
        promptBuilder.WriteString("# Approach & Focus\n")
        
        if m.Persona.Approach != "" {
            promptBuilder.WriteString(fmt.Sprintf("Problem-solving approach: %s\n", m.Persona.Approach))
        }
        
        if len(m.Persona.Focus) > 0 {
            promptBuilder.WriteString(fmt.Sprintf("Key focus areas: %s\n", strings.Join(m.Persona.Focus, ", ")))
        }
        promptBuilder.WriteString("\n")
    }
    
    // Add constraints and preferences
    if len(m.Persona.Constraints) > 0 || len(m.Persona.Preferences) > 0 {
        promptBuilder.WriteString("# Guidelines\n")
        
        if len(m.Persona.Constraints) > 0 {
            promptBuilder.WriteString("Constraints:\n")
            for _, constraint := range m.Persona.Constraints {
                promptBuilder.WriteString(fmt.Sprintf("- %s\n", constraint))
            }
        }
        
        if len(m.Persona.Preferences) > 0 {
            promptBuilder.WriteString("Preferences:\n")
            for _, preference := range m.Persona.Preferences {
                promptBuilder.WriteString(fmt.Sprintf("- %s\n", preference))
            }
        }
        promptBuilder.WriteString("\n")
    }
    
    // Add output format
    if m.Persona.OutputFormat != "" {
        promptBuilder.WriteString("# Output Format\n")
        promptBuilder.WriteString(fmt.Sprintf("Format responses in %s style.\n\n", m.Persona.OutputFormat))
    }
    
    // Add separator and main prompt
    if promptBuilder.Len() > 0 {
        promptBuilder.WriteString("---\n\n")
    }
    
    // Add the main prompt
    promptBuilder.WriteString(m.Prompt)
    
    return promptBuilder.String()
}

// GetPromptFilename returns the expected prompt filename for a manifest
func GetPromptFilename(manifestFile string) (string, error) {
    data, err := os.ReadFile(manifestFile)
    if err != nil {
        return "", err
    }
    
    var m Manifest
    if err := yaml.Unmarshal(data, &m); err != nil {
        return "", err
    }
    
    // Flatten to get final name/version
    flattened, err := FlattenManifest(&m)
    if err != nil {
        return "", err
    }
    
    return fmt.Sprintf("%s-%s-prompt.md", flattened.Name, flattened.Version), nil
}
