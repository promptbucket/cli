package persona

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Persona is the parsed persona.yaml.
type Persona struct {
	SpecVersion string   `yaml:"spec_version"`
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	License     string   `yaml:"license"`
	Description string   `yaml:"description"`
	Authors     []string `yaml:"authors"`
	Identity    Identity `yaml:"identity"`
	BaseModel   Model    `yaml:"base_model"`
}

// Identity holds the persona's role and behavioural traits.
type Identity struct {
	Role      string   `yaml:"role"`
	Expertise []string `yaml:"expertise"`
	Tone      string   `yaml:"tone"`
	Focus     []string `yaml:"focus"`
}

// Model holds the preferred LLM provider and model name.
type Model struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	Fallback string `yaml:"fallback"`
}

// Load reads and parses the persona.yaml from dir.
func Load(dir string) (*Persona, error) {
	yamlPath := filepath.Join(dir, "persona.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", yamlPath, err)
	}
	var p Persona
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", yamlPath, err)
	}
	return &p, nil
}

// SystemPrompt builds the effective system prompt:
//  1. Identity block from persona.yaml
//  2. Contents of system.md
//  3. Top-8 memories scored by importance × recency
//  4. Memory tool instruction block
func SystemPrompt(dir string, p *Persona) (string, error) {
	var sb strings.Builder

	// Identity header
	sb.WriteString("# Identity\n\n")
	sb.WriteString(fmt.Sprintf("**Role:** %s\n\n", p.Identity.Role))
	sb.WriteString(fmt.Sprintf("**Expertise:** %s\n\n", strings.Join(p.Identity.Expertise, ", ")))
	sb.WriteString(fmt.Sprintf("**Tone:** %s\n\n", p.Identity.Tone))
	if len(p.Identity.Focus) > 0 {
		sb.WriteString(fmt.Sprintf("**Focus:** %s\n\n", strings.Join(p.Identity.Focus, ", ")))
	}
	sb.WriteString("---\n\n")

	// system.md
	sysPath := filepath.Join(dir, "system.md")
	sysContent, err := os.ReadFile(sysPath)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", sysPath, err)
	}
	sb.Write(sysContent)

	// Top-K memories
	memories := loadTopKMemories(dir, 8)
	if len(memories) > 0 {
		sb.WriteString("\n\n---\n\n## Relevant memories\n\n")
		for _, m := range memories {
			sb.WriteString(m)
			sb.WriteString("\n\n")
		}
	}

	// Memory tool instruction
	sb.WriteString("\n\n---\n\n## Memory\n\n")
	sb.WriteString("You have a save_memory tool. Use it during this session to save things worth keeping for future sessions:\n\n")
	sb.WriteString("- Project facts, stack decisions, user preferences → layer: semantic, importance: 0.7–0.9\n")
	sb.WriteString("- What happened this session, what worked, what failed → layer: episodic, importance: 0.5–0.8\n")
	sb.WriteString("- Reusable procedures you figured out → layer: procedural, importance: 0.8–1.0\n\n")
	sb.WriteString("Keep each memory 1–3 sentences, self-contained. Call it as you go — do not wait until the end.\n")

	return sb.String(), nil
}

// memoryFrontmatter holds the YAML frontmatter fields of a memory file.
type memoryFrontmatter struct {
	Importance *float64  `yaml:"importance"`
	Created    time.Time `yaml:"created"`
	Layer      string    `yaml:"layer"`
}

// parseMemoryFile reads a memory .md file and returns its body text and frontmatter.
// If the file has no frontmatter, importance defaults to 0.5 and created to file mtime.
func parseMemoryFile(path string) (content string, fm memoryFrontmatter, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	text := string(data)
	lines := strings.Split(text, "\n")

	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "---" {
		defaultImportance := 0.5
		fm.Importance = &defaultImportance
		fm.Created = fileModTime(path)
		content = strings.TrimSpace(text)
		return
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		defaultImportance := 0.5
		fm.Importance = &defaultImportance
		fm.Created = fileModTime(path)
		content = strings.TrimSpace(text)
		return
	}

	fmText := strings.Join(lines[1:end], "\n")
	if parseErr := yaml.Unmarshal([]byte(fmText), &fm); parseErr != nil {
		defaultImportance := 0.5
		fm.Importance = &defaultImportance
	}
	if fm.Importance == nil {
		defaultImportance := 0.5
		fm.Importance = &defaultImportance
	}
	if fm.Created.IsZero() {
		fm.Created = fileModTime(path)
	}
	content = strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	return
}

// fileModTime returns the modification time of path, or time.Now() on error.
func fileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
}

// memoryEntry holds a scored memory for sorting.
type memoryEntry struct {
	content string
	score   float64
}

// loadTopKMemories reads all .md files from seed-memory/ and memory/ subdirectories,
// scores each by importance × e^(-days/30), and returns the top k content strings.
func loadTopKMemories(dir string, k int) []string {
	var entries []memoryEntry
	now := time.Now()
	layers := []string{"procedural", "semantic", "episodic"}

	for _, base := range []string{"seed-memory", "memory"} {
		for _, layer := range layers {
			layerDir := filepath.Join(dir, base, layer)
			files, err := os.ReadDir(layerDir)
			if err != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
					continue
				}
				path := filepath.Join(layerDir, f.Name())
				content, fm, err := parseMemoryFile(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[promptbucket] skipping %s: %v\n", path, err)
					continue
				}
				if strings.TrimSpace(content) == "" {
					continue
				}
				days := now.Sub(fm.Created).Hours() / 24
				score := *fm.Importance * math.Exp(-days/30)
				entries = append(entries, memoryEntry{content: content, score: score})
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].score > entries[j].score
	})
	if len(entries) > k {
		entries = entries[:k]
	}

	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = e.content
	}
	return out
}
