package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "persona.yaml"), []byte(
		"spec_version: \"0.1\"\nname: test-persona\nversion: \"0.1.0\"\nidentity:\n  role: Test\n  tone: direct\n",
	), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "system.md"), []byte("Test system prompt."), 0644); err != nil {
		t.Fatal(err)
	}
	return &Server{personaDir: dir}
}

func TestMakeMemorySlug_Normal(t *testing.T) {
	slug := makeMemorySlug("User prefers tabs over spaces in Go files")
	if slug != "user-prefers-tabs-over-spaces" {
		t.Errorf("unexpected slug: %q", slug)
	}
}

func TestMakeMemorySlug_Short(t *testing.T) {
	slug := makeMemorySlug("MongoDB")
	if slug != "mongodb" {
		t.Errorf("unexpected slug: %q", slug)
	}
}

func TestMakeMemorySlug_Punctuation(t *testing.T) {
	// "Use", "`go", "test", "./...`", "always" → after stripping non-alnum: "use","go","test","","always"
	// empty parts filtered → "use-go-test-always"
	slug := makeMemorySlug("Use `go test ./...` always")
	if slug != "use-go-test-always" {
		t.Errorf("unexpected slug: %q", slug)
	}
}

func TestHandleSaveMemory_WritesFile(t *testing.T) {
	srv := makeTestServer(t)

	args := json.RawMessage(`{"content":"User prefers tabs over spaces.","layer":"semantic","importance":0.8}`)
	result, err := srv.handleSaveMemory(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	res := result.(map[string]interface{})
	if res["saved"] != true {
		t.Errorf("expected saved=true, got %v", res["saved"])
	}

	memDir := filepath.Join(srv.personaDir, "memory", "semantic")
	entries, err := os.ReadDir(memDir)
	if err != nil {
		t.Fatalf("memory dir not created: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}

	content, err := os.ReadFile(filepath.Join(memDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "User prefers tabs over spaces.") {
		t.Errorf("memory file missing expected content, got:\n%s", content)
	}
	body := string(content)
	for _, want := range []string{"---\n", "importance: 0.8", "layer: semantic", "created:", "last_accessed:"} {
		if !strings.Contains(body, want) {
			t.Errorf("memory file missing expected field %q\nfull content:\n%s", want, body)
		}
	}
}

func TestHandleSaveMemory_InvalidLayer(t *testing.T) {
	srv := makeTestServer(t)
	args := json.RawMessage(`{"content":"test","layer":"invalid","importance":0.5}`)
	_, err := srv.handleSaveMemory(args)
	if err == nil {
		t.Error("expected error for invalid layer")
	}
}

func TestHandleSaveMemory_MissingContent(t *testing.T) {
	srv := makeTestServer(t)
	args := json.RawMessage(`{"layer":"episodic","importance":0.5}`)
	_, err := srv.handleSaveMemory(args)
	if err == nil {
		t.Error("expected error for missing content")
	}
}

func TestHandleSaveMemory_AllLayers(t *testing.T) {
	for _, layer := range []string{"episodic", "semantic", "procedural"} {
		t.Run(layer, func(t *testing.T) {
			srv := makeTestServer(t)
			args := json.RawMessage(`{"content":"test memory","layer":"` + layer + `","importance":0.6}`)
			_, err := srv.handleSaveMemory(args)
			if err != nil {
				t.Fatalf("layer %q: unexpected error: %v", layer, err)
			}
			memDir := filepath.Join(srv.personaDir, "memory", layer)
			entries, err := os.ReadDir(memDir)
			if err != nil {
				t.Fatalf("layer %q: memory dir not created: %v", layer, err)
			}
			if len(entries) != 1 {
				t.Errorf("layer %q: expected 1 file, got %d", layer, len(entries))
			}
		})
	}
}
