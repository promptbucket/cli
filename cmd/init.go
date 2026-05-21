package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var (
	initName      string
	initRole      string
	initExpertise string
	initTone      string
	initProvider  string
	initModel     string
	initForce     bool
)

const personaYAMLTemplate = `spec_version: "0.1"
name: "{{.Name}}"
version: "0.1.0"
license: "MIT"
description: "{{.Role}} persona."
authors:
  - "{{.Author}}"

identity:
  role: "{{.Role}}"
  expertise:
{{range .ExpertiseList}}  - "{{.}}"
{{end}}  tone: "{{.Tone}}"
  focus:
    - "correctness"
    - "shipping"

base_model:
  provider: "{{.Provider}}"
  model: "{{.Model}}"
  fallback: "claude-haiku-4-5-20251001"

reflection:
  enabled: true
  model: "claude-haiku-4-5-20251001"
  triggers:
    - "session_end"
  max_notes_per_session: 5

memory:
  layers:
    - "episodic"
    - "semantic"
    - "procedural"
  retrieval:
    top_k: 8
    weights:
      importance: 0.4
      recency: 0.3
      relevance: 0.3
  forgetting:
    enabled: true
    half_life_days: 60
    floor_importance: 0.8

mcp:
  expose_as: "server"
  resources:
    - "memory://episodic"
    - "memory://semantic"
`

const systemMDTemplate = `# {{.Role}}

You are a {{.Role}}. Your expertise covers: {{.Expertise}}.

Your communication style is {{.Tone}}.

## How you work

- Be direct and specific. No unnecessary explanation.
- When you are uncertain, say so rather than guessing.
- Ask one clarifying question at a time if you need more context.

## What you know about this project

(Add project-specific context here. This file is committed to the repo and shared with anyone who uses this persona.)
`

const gitignoreContent = `memory/
transcripts/
`

const readmeMDTemplate = `# {{.Name}}

{{.Role}} persona powered by [PromptBucket](https://promptbucket.io).

## Usage

` + "```" + `bash
# Start as MCP server (works with Claude Code, Cursor, Cline)
promptbucket serve

# Chat in the terminal
promptbucket chat
` + "```" + `

## Memory

Private memory is stored in ` + "`" + `.promptbucket/memory/` + "`" + ` and is gitignored by default.
Seed memory in ` + "`" + `.promptbucket/seed-memory/` + "`" + ` is committed and shared with teammates.
`

type initTemplateData struct {
	Name          string
	Role          string
	Expertise     string
	ExpertiseList []string
	Tone          string
	Provider      string
	Model         string
	Author        string
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a persona in the current project",
	Long:  "Creates a .promptbucket/ directory with persona.yaml, system.md, and memory layer folders.",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := ".promptbucket"

		if !initForce {
			if _, err := os.Stat(dir); err == nil {
				return fmt.Errorf("%s already exists (use --force to overwrite)", dir)
			}
		}

		// Parse name: if no slash, use "local/name"
		name := initName
		if !strings.Contains(name, "/") {
			name = "local/" + name
		}

		// Get author from git config or fallback
		author := gitConfigUser()

		expertiseList := parseCSV(initExpertise)

		data := initTemplateData{
			Name:          name,
			Role:          initRole,
			Expertise:     initExpertise,
			ExpertiseList: expertiseList,
			Tone:          initTone,
			Provider:      initProvider,
			Model:         initModel,
			Author:        author,
		}

		// Directories to create
		dirs := []string{
			dir,
			filepath.Join(dir, "memory", "episodic"),
			filepath.Join(dir, "memory", "semantic"),
			filepath.Join(dir, "memory", "procedural"),
			filepath.Join(dir, "transcripts"),
			filepath.Join(dir, "seed-memory", "semantic"),
			filepath.Join(dir, "seed-memory", "procedural"),
		}

		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", d, err)
			}
		}

		// .gitkeep files to make empty dirs trackable
		gitkeepDirs := []string{
			filepath.Join(dir, "memory", "episodic"),
			filepath.Join(dir, "memory", "semantic"),
			filepath.Join(dir, "memory", "procedural"),
			filepath.Join(dir, "transcripts"),
			filepath.Join(dir, "seed-memory", "semantic"),
			filepath.Join(dir, "seed-memory", "procedural"),
		}

		for _, d := range gitkeepDirs {
			p := filepath.Join(d, ".gitkeep")
			if err := os.WriteFile(p, []byte(""), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", p, err)
			}
		}

		// Write .gitignore
		if err := writeFile(filepath.Join(dir, ".gitignore"), gitignoreContent); err != nil {
			return err
		}

		// Write persona.yaml from template
		if err := writeTemplate(filepath.Join(dir, "persona.yaml"), personaYAMLTemplate, data); err != nil {
			return err
		}

		// Write system.md from template
		if err := writeTemplate(filepath.Join(dir, "system.md"), systemMDTemplate, data); err != nil {
			return err
		}

		// Write README.md from template
		if err := writeTemplate(filepath.Join(dir, "README.md"), readmeMDTemplate, data); err != nil {
			return err
		}

		fmt.Printf("Created %s/\n", dir)
		fmt.Println()
		fmt.Printf("  persona: %s\n", name)
		fmt.Printf("  role:    %s\n", initRole)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Printf("  1. Edit %s/system.md — add project context\n", dir)
		fmt.Printf("  2. Edit %s/persona.yaml — adjust expertise and model\n", dir)
		fmt.Printf("  3. Run: promptbucket serve\n")
		fmt.Printf("  4. Connect Claude Code or Cursor to the MCP server\n")
		return nil
	},
}

func writeFile(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}
	return nil
}

func writeTemplate(path, tmpl string, data interface{}) error {
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return fmt.Errorf("template parse error for %s: %w", path, err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer f.Close()
	if err := t.Execute(f, data); err != nil {
		return fmt.Errorf("template execute error for %s: %w", path, err)
	}
	return nil
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func gitConfigUser() string {
	// Try to get name from git config
	nameBytes, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".gitconfig"))
	if err != nil {
		return "Unknown"
	}
	for _, line := range strings.Split(string(nameBytes), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name = ") {
			return strings.TrimPrefix(line, "name = ")
		}
	}
	return "Unknown"
}

func init() {
	initCmd.Flags().StringVar(&initName, "name", "my-persona", "Persona name (format: author/name or just name)")
	initCmd.Flags().StringVar(&initRole, "role", "Software Engineer", "Role description")
	initCmd.Flags().StringVar(&initExpertise, "expertise", "Go, API design", "Comma-separated expertise areas")
	initCmd.Flags().StringVar(&initTone, "tone", "direct, pragmatic", "Communication tone")
	initCmd.Flags().StringVar(&initProvider, "provider", "anthropic", "Model provider (anthropic, openai, google)")
	initCmd.Flags().StringVar(&initModel, "model", "claude-sonnet-4-6", "Preferred model ID")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing .promptbucket/ directory")
	rootCmd.AddCommand(initCmd)
}
