package packager

type Variable struct {
    Name        string   `yaml:"name,omitempty"`
    Description string   `yaml:"description,omitempty"`
    Example     string   `yaml:"example,omitempty"`
    Enum        []string `yaml:"enum,omitempty"`
}

type Manifest struct {
    Name        string     `yaml:"name"`
    Version     string     `yaml:"version"`
    Licence     string     `yaml:"licence"`
    Description string     `yaml:"description,omitempty"`
    Prompt      string     `yaml:"prompt"`
    Digest      string     `yaml:"digest,omitempty"`
    Authors     []string   `yaml:"authors,omitempty"`
    Tags        []string   `yaml:"tags,omitempty"`
    Variables   []Variable `yaml:"variables,omitempty"`
}
