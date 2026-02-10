package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMemoryExists(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, ".stormtrooper", "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("remember this"), 0644)

	content, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "remember this" {
		t.Fatalf("expected 'remember this', got %q", content)
	}
}

func TestLoadMemoryMissing(t *testing.T) {
	dir := t.TempDir()

	content, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty string for missing file, got %q", content)
	}
}

func TestLoadMemoryEmptyFile(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, ".stormtrooper", "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte(""), 0644)

	content, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty string, got %q", content)
	}
}

func TestDir(t *testing.T) {
	result := Dir("/some/project")
	expected := filepath.Join("/some/project", ".stormtrooper", "memory")
	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}
}
