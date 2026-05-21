package persona

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestMemory(t *testing.T, path string, importance float64, created time.Time, content string) {
	t.Helper()
	text := fmt.Sprintf(
		"---\nimportance: %.1f\nlayer: episodic\ncreated: %s\nlast_accessed: %s\n---\n\n%s\n",
		importance,
		created.UTC().Format(time.RFC3339),
		created.UTC().Format(time.RFC3339),
		content,
	)
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestParseMemoryFile_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mem.md")
	writeTestMemory(t, path, 0.8, time.Now(), "User prefers tabs over spaces.")

	content, fm, err := parseMemoryFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if content != "User prefers tabs over spaces." {
		t.Errorf("unexpected content: %q", content)
	}
	if fm.Importance == nil || *fm.Importance != 0.8 {
		t.Errorf("expected importance 0.8, got %v", fm.Importance)
	}
	if fm.Created.IsZero() {
		t.Error("expected non-zero created time")
	}
}

func TestParseMemoryFile_NoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.md")
	if err := os.WriteFile(path, []byte("Plain memory content"), 0644); err != nil {
		t.Fatal(err)
	}

	content, fm, err := parseMemoryFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if content != "Plain memory content" {
		t.Errorf("unexpected content: %q", content)
	}
	if fm.Importance == nil || *fm.Importance != 0.5 {
		t.Errorf("expected default importance 0.5, got %v", fm.Importance)
	}
}

func TestLoadTopKMemories_Empty(t *testing.T) {
	dir := t.TempDir()
	memories := loadTopKMemories(dir, 8)
	if len(memories) != 0 {
		t.Errorf("expected 0 memories, got %d", len(memories))
	}
}

func TestLoadTopKMemories_HighImportanceFirst(t *testing.T) {
	dir := t.TempDir()
	layerDir := filepath.Join(dir, "memory", "episodic")
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeTestMemory(t, filepath.Join(layerDir, "low.md"), 0.1, now, "Low importance memory")
	writeTestMemory(t, filepath.Join(layerDir, "high.md"), 0.9, now, "High importance memory")

	memories := loadTopKMemories(dir, 8)
	if len(memories) != 2 {
		t.Fatalf("expected 2 memories, got %d", len(memories))
	}
	if memories[0] != "High importance memory" {
		t.Errorf("expected high importance first, got: %q", memories[0])
	}
}

func TestLoadTopKMemories_CapsAtK(t *testing.T) {
	dir := t.TempDir()
	layerDir := filepath.Join(dir, "memory", "episodic")
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	for i := 0; i < 10; i++ {
		writeTestMemory(t, filepath.Join(layerDir, fmt.Sprintf("mem%02d.md", i)), 0.5, now, fmt.Sprintf("Memory %d", i))
	}

	memories := loadTopKMemories(dir, 8)
	if len(memories) != 8 {
		t.Errorf("expected 8 (capped at k), got %d", len(memories))
	}
}

func TestLoadTopKMemories_SeedAndMemoryMerged(t *testing.T) {
	dir := t.TempDir()
	seedDir := filepath.Join(dir, "seed-memory", "semantic")
	memDir := filepath.Join(dir, "memory", "procedural")
	if err := os.MkdirAll(seedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(memDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	writeTestMemory(t, filepath.Join(seedDir, "seed.md"), 0.6, now, "Seed memory")
	writeTestMemory(t, filepath.Join(memDir, "live.md"), 0.7, now, "Live memory")

	memories := loadTopKMemories(dir, 8)
	if len(memories) != 2 {
		t.Fatalf("expected 2 memories (one from each dir), got %d", len(memories))
	}
	if memories[0] != "Live memory" {
		t.Errorf("expected live memory first, got: %q", memories[0])
	}
}

func TestLoadTopKMemories_RecencyDecay(t *testing.T) {
	dir := t.TempDir()
	layerDir := filepath.Join(dir, "memory", "episodic")
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	old := now.AddDate(0, -3, 0)

	writeTestMemory(t, filepath.Join(layerDir, "old.md"), 0.9, old, "Old but important memory")
	writeTestMemory(t, filepath.Join(layerDir, "fresh.md"), 0.5, now, "Fresh lower importance memory")

	memories := loadTopKMemories(dir, 8)
	if len(memories) != 2 {
		t.Fatalf("expected 2 memories, got %d", len(memories))
	}
	// fresh (0.5 * 1.0 = 0.5) > old (0.9 * e^-3 ≈ 0.045)
	if memories[0] != "Fresh lower importance memory" {
		t.Errorf("expected fresh memory first due to recency decay, got: %q", memories[0])
	}
}
