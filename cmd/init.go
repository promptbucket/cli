package cmd

import (
    "bufio"
    "errors"
    "fmt"
    "os"
    "strings"

    "github.com/promptbucket/cli/internal/auth"
    "github.com/promptbucket/cli/internal/packager"
    "github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Scaffold promptbucket.yaml with interactive prompts",
    RunE: func(cmd *cobra.Command, args []string) error {
        if _, err := os.Stat("promptbucket.yaml"); err == nil {
            return errors.New("promptbucket.yaml already exists")
        }
        
        return createInteractiveManifest()
    },
}

func createInteractiveManifest() error {
    reader := bufio.NewReader(os.Stdin)
    
    // Get logged-in user info
    config := auth.NewConfig()
    tokenManager := auth.NewTokenManager(config)
    var loggedInUser *auth.Token
    if tokenManager.IsAuthenticated() {
        var err error
        loggedInUser, err = tokenManager.GetToken()
        if err != nil {
            fmt.Printf("⚠️  Warning: Could not read auth token: %v\n", err)
        }
    }
    
    fmt.Println("🚀 Creating a new PromptBucket manifest")
    fmt.Println()
    
    // Ask for persona name
    fmt.Print("📝 Enter persona name (e.g., 'code-reviewer', 'data-analyst'): ")
    personaName, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read persona name: %w", err)
    }
    personaName = strings.TrimSpace(personaName)
    if personaName == "" {
        personaName = "helpful-assistant"
    }
    
    // Ask for description
    fmt.Print("📄 Enter description (optional): ")
    description, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read description: %w", err)
    }
    description = strings.TrimSpace(description)
    
    // Ask for persona details
    fmt.Println()
    fmt.Println("🎭 Persona Configuration (all optional - press Enter to skip)")
    
    var persona *packager.Persona
    
    fmt.Print("👤 Persona role (e.g., 'Senior Software Engineer', 'Data Scientist'): ")
    roleInput, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read role: %w", err)
    }
    role := strings.TrimSpace(roleInput)
    
    fmt.Print("🧠 Expertise areas (comma-separated, e.g., 'Python, Machine Learning, APIs'): ")
    expertiseInput, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read expertise: %w", err)
    }
    expertiseStr := strings.TrimSpace(expertiseInput)
    
    fmt.Print("💬 Communication tone (e.g., 'professional', 'friendly', 'encouraging'): ")
    toneInput, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read tone: %w", err)
    }
    tone := strings.TrimSpace(toneInput)
    
    fmt.Print("🎯 Focus areas (comma-separated, e.g., 'security, performance, best practices'): ")
    focusInput, err := reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read focus: %w", err)
    }
    focusStr := strings.TrimSpace(focusInput)
    
    // Create persona if any fields were provided
    if role != "" || expertiseStr != "" || tone != "" || focusStr != "" {
        persona = &packager.Persona{}
        
        if role != "" {
            persona.Role = role
        }
        
        if expertiseStr != "" {
            expertise := strings.Split(expertiseStr, ",")
            for i, exp := range expertise {
                expertise[i] = strings.TrimSpace(exp)
            }
            persona.Expertise = expertise
        }
        
        if tone != "" {
            persona.Tone = tone
        }
        
        if focusStr != "" {
            focus := strings.Split(focusStr, ",")
            for i, f := range focus {
                focus[i] = strings.TrimSpace(f)
            }
            persona.Focus = focus
        }
    }
    
    // Ask for prompt
    fmt.Println("💬 Enter the system prompt (press Enter twice when done):")
    var promptLines []string
    emptyLineCount := 0
    for {
        line, err := reader.ReadString('\n')
        if err != nil {
            return fmt.Errorf("failed to read prompt: %w", err)
        }
        
        line = strings.TrimRight(line, "\n")
        if line == "" {
            emptyLineCount++
            if emptyLineCount >= 2 {
                break
            }
        } else {
            emptyLineCount = 0
        }
        promptLines = append(promptLines, line)
    }
    
    // Remove trailing empty lines
    for len(promptLines) > 0 && promptLines[len(promptLines)-1] == "" {
        promptLines = promptLines[:len(promptLines)-1]
    }
    
    prompt := strings.Join(promptLines, "\n")
    if prompt == "" {
        prompt = "You are a helpful assistant."
    }
    
    // Build manifest content
    var manifestBuilder strings.Builder
    manifestBuilder.WriteString(fmt.Sprintf("name: %s\n", personaName))
    manifestBuilder.WriteString("version: 0.1.0\n")
    manifestBuilder.WriteString("licence: Apache-2.0\n")
    
    if description != "" {
        manifestBuilder.WriteString(fmt.Sprintf("description: %s\n", description))
    }
    
    // Add author if logged in
    if loggedInUser != nil && loggedInUser.Name != "" {
        manifestBuilder.WriteString("authors:\n")
        manifestBuilder.WriteString(fmt.Sprintf("  - %s\n", loggedInUser.Name))
        fmt.Printf("✅ Added %s as author (from logged-in user)\n", loggedInUser.Name)
    }
    
    // Add persona if configured
    if persona != nil {
        manifestBuilder.WriteString("persona:\n")
        if persona.Role != "" {
            manifestBuilder.WriteString(fmt.Sprintf("  role: %s\n", persona.Role))
        }
        if len(persona.Expertise) > 0 {
            manifestBuilder.WriteString("  expertise:\n")
            for _, exp := range persona.Expertise {
                manifestBuilder.WriteString(fmt.Sprintf("    - %s\n", exp))
            }
        }
        if persona.Tone != "" {
            manifestBuilder.WriteString(fmt.Sprintf("  tone: %s\n", persona.Tone))
        }
        if len(persona.Focus) > 0 {
            manifestBuilder.WriteString("  focus:\n")
            for _, f := range persona.Focus {
                manifestBuilder.WriteString(fmt.Sprintf("    - %s\n", f))
            }
        }
        fmt.Printf("✅ Added persona configuration\n")
    }
    
    manifestBuilder.WriteString("prompt: |-\n")
    for _, line := range strings.Split(prompt, "\n") {
        manifestBuilder.WriteString(fmt.Sprintf("  %s\n", line))
    }
    
    // Write to file
    manifestContent := manifestBuilder.String()
    if err := os.WriteFile("promptbucket.yaml", []byte(manifestContent), 0644); err != nil {
        return fmt.Errorf("failed to write promptbucket.yaml: %w", err)
    }
    
    fmt.Println()
    fmt.Printf("✅ Created promptbucket.yaml for persona '%s'\n", personaName)
    if loggedInUser == nil {
        fmt.Printf("💡 Tip: Run 'promptbucket login' to automatically populate author info\n")
    }
    
    return nil
}

func init() { rootCmd.AddCommand(initCmd) }
