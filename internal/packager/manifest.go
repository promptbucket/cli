package packager

type Variable struct {
    Name        string   `yaml:"name,omitempty"`
    Description string   `yaml:"description,omitempty"`
    Example     string   `yaml:"example,omitempty"`
    Enum        []string `yaml:"enum,omitempty"`
}

type Persona struct {
    // Identity
    Name         string   `yaml:"name,omitempty"`         // e.g., "Alex the Code Reviewer"
    Role         string   `yaml:"role,omitempty"`         // e.g., "Senior Software Engineer"
    Personality  []string `yaml:"personality,omitempty"`  // e.g., ["friendly", "thorough", "encouraging"]
    
    // Expertise & Background
    Expertise    []string `yaml:"expertise,omitempty"`    // e.g., ["Python", "Machine Learning", "API Design"]
    Experience   string   `yaml:"experience,omitempty"`   // e.g., "10+ years", "Senior level"
    Background   string   `yaml:"background,omitempty"`   // e.g., "Computer Science PhD with industry experience"
    
    // Communication Style
    Tone         string   `yaml:"tone,omitempty"`         // e.g., "professional", "casual", "encouraging"
    Style        string   `yaml:"style,omitempty"`        // e.g., "concise", "detailed", "step-by-step"
    LanguageLevel string  `yaml:"language_level,omitempty"` // e.g., "beginner-friendly", "expert-level"
    
    // Behavioral Traits
    Approach     string   `yaml:"approach,omitempty"`     // e.g., "analytical", "creative", "systematic"
    Focus        []string `yaml:"focus,omitempty"`        // e.g., ["security", "performance", "usability"]
    InteractionStyle string `yaml:"interaction_style,omitempty"` // e.g., "asks clarifying questions", "provides examples"
    
    // Constraints & Preferences
    Constraints  []string `yaml:"constraints,omitempty"`  // e.g., ["no profanity", "always cite sources"]
    Preferences  []string `yaml:"preferences,omitempty"`  // e.g., ["provide code examples", "explain reasoning"]
    
    // Output Format
    OutputFormat string   `yaml:"output_format,omitempty"` // e.g., "markdown", "structured", "conversational"
}

type Manifest struct {
    Name        string     `yaml:"name"`
    Version     string     `yaml:"version"`
    Licence     string     `yaml:"licence"`
    Description string     `yaml:"description,omitempty"`
    Authors     []string   `yaml:"authors,omitempty"`
    Tags        []string   `yaml:"tags,omitempty"`
    Language    string     `yaml:"language,omitempty"`
    ModelHint   string     `yaml:"model_hint,omitempty"`
    From        string     `yaml:"from,omitempty"`
    Persona     *Persona   `yaml:"persona,omitempty"`
    Variables   []Variable `yaml:"variables,omitempty"`
    Prompt      string     `yaml:"prompt"`
    Digest      string     `yaml:"digest,omitempty"`
}
