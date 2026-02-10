package context

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadWithStormtrooperMD(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "STORMTROOPER.md"), []byte("project instructions"), 0644)

	pc, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Instructions != "project instructions" {
		t.Fatalf("expected instructions, got %q", pc.Instructions)
	}
}

func TestLoadWithClaudeMDFallback(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("claude instructions"), 0644)

	pc, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Instructions != "claude instructions" {
		t.Fatalf("expected claude instructions, got %q", pc.Instructions)
	}
}

func TestLoadStormtrooperMDTakesPriority(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "STORMTROOPER.md"), []byte("stormtrooper wins"), 0644)
	os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("claude loses"), 0644)

	pc, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Instructions != "stormtrooper wins" {
		t.Fatalf("STORMTROOPER.md should take priority, got %q", pc.Instructions)
	}
}

func TestLoadNoInstructions(t *testing.T) {
	dir := t.TempDir()

	pc, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Instructions != "" {
		t.Fatalf("expected empty instructions, got %q", pc.Instructions)
	}
}

func TestLoadWithMemory(t *testing.T) {
	dir := t.TempDir()
	memDir := filepath.Join(dir, ".stormtrooper", "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("memory content"), 0644)

	pc, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Memory != "memory content" {
		t.Fatalf("expected memory content, got %q", pc.Memory)
	}
}

func TestLoadEnvironmentFields(t *testing.T) {
	dir := t.TempDir()

	pc, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pc.Platform != runtime.GOOS {
		t.Fatalf("expected platform %s, got %s", runtime.GOOS, pc.Platform)
	}
	expectedDate := time.Now().Format("2006-01-02")
	if pc.Date != expectedDate {
		t.Fatalf("expected date %s, got %s", expectedDate, pc.Date)
	}
	if pc.WorkingDir == "" {
		t.Fatal("WorkingDir should not be empty")
	}
}

func TestBuildSystemPromptFull(t *testing.T) {
	pc := &ProjectContext{
		WorkingDir:   "/my/project",
		Instructions: "Do this project stuff",
		Memory:       "Remember this thing",
		Platform:     "linux",
		Date:         "2026-02-10",
	}

	prompt := pc.BuildSystemPrompt()

	if !strings.Contains(prompt, "Stormtrooper") {
		t.Fatal("prompt should contain Stormtrooper identity")
	}
	if !strings.Contains(prompt, "# Project Instructions") {
		t.Fatal("prompt should contain project instructions section")
	}
	if !strings.Contains(prompt, "Do this project stuff") {
		t.Fatal("prompt should contain actual instructions")
	}
	if !strings.Contains(prompt, "# Memory") {
		t.Fatal("prompt should contain memory section")
	}
	if !strings.Contains(prompt, "Remember this thing") {
		t.Fatal("prompt should contain actual memory")
	}
	if !strings.Contains(prompt, "# Environment") {
		t.Fatal("prompt should contain environment section")
	}
	if !strings.Contains(prompt, "/my/project") {
		t.Fatal("prompt should contain working directory")
	}
	if !strings.Contains(prompt, "linux") {
		t.Fatal("prompt should contain platform")
	}
	if !strings.Contains(prompt, "2026-02-10") {
		t.Fatal("prompt should contain date")
	}
}

func TestBuildSystemPromptMinimal(t *testing.T) {
	pc := &ProjectContext{
		WorkingDir: "/my/project",
		Platform:   "linux",
		Date:       "2026-02-10",
	}

	prompt := pc.BuildSystemPrompt()

	if !strings.Contains(prompt, "Stormtrooper") {
		t.Fatal("prompt should contain Stormtrooper identity")
	}
	if strings.Contains(prompt, "# Project Instructions") {
		t.Fatal("prompt should NOT contain instructions section when empty")
	}
	if strings.Contains(prompt, "# Memory") {
		t.Fatal("prompt should NOT contain memory section when empty")
	}
	if !strings.Contains(prompt, "# Environment") {
		t.Fatal("prompt should always contain environment section")
	}
}
