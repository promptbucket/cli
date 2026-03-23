package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// TestSuite is the top-level configuration structure.
type TestSuite struct {
	Tests []TestCase `yaml:"tests"`
}

// TestCase defines a single test with prompt, models, and assertions.
type TestCase struct {
	Name   string      `yaml:"name"`
	Prompt string      `yaml:"prompt"`
	System string      `yaml:"system,omitempty"`
	Models []string    `yaml:"models"`
	Assert []Assertion `yaml:"assert"`
}

// Assertion describes a single check to run against a model response.
type Assertion struct {
	Type     string  `yaml:"type"`
	Value    string  `yaml:"value,omitempty"`
	Pattern  string  `yaml:"pattern,omitempty"`
	Criteria string  `yaml:"criteria,omitempty"`
	Max      float64 `yaml:"max,omitempty"`
}

// Load reads and validates a YAML test suite from the given path.
func Load(path string) (*TestSuite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config %s: %w", path, err)
	}

	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s: %w", path, err)
	}

	if err := validate(&suite); err != nil {
		return nil, err
	}

	return &suite, nil
}

func validate(suite *TestSuite) error {
	if len(suite.Tests) == 0 {
		return fmt.Errorf("config must contain at least one test")
	}

	for i, tc := range suite.Tests {
		if tc.Name == "" {
			return fmt.Errorf("test #%d: name is required", i+1)
		}
		if tc.Prompt == "" {
			return fmt.Errorf("test %q: prompt is required", tc.Name)
		}
		if len(tc.Models) == 0 {
			return fmt.Errorf("test %q: at least one model is required", tc.Name)
		}
	}

	return nil
}

// ResolveSystem returns the system prompt content. If tc.System looks like
// a file path (contains "/" or ends in .txt/.md), the file is read and its
// contents returned. Otherwise the raw string value is returned.
func (tc *TestCase) ResolveSystem() (string, error) {
	s := tc.System
	if s == "" {
		return "", nil
	}

	isPath := strings.Contains(s, "/") ||
		strings.HasSuffix(s, ".txt") ||
		strings.HasSuffix(s, ".md")

	if !isPath {
		return s, nil
	}

	// Resolve relative paths against the working directory.
	abs, err := filepath.Abs(s)
	if err != nil {
		return "", fmt.Errorf("cannot resolve system path %q: %w", s, err)
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("cannot read system prompt file %q: %w", abs, err)
	}

	return string(data), nil
}
