package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaults(t *testing.T) {
	d := defaults()
	if d.Model != "moonshotai/kimi-k2" {
		t.Errorf("expected default model 'moonshotai/kimi-k2', got %q", d.Model)
	}
	if d.BaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("expected default base URL, got %q", d.BaseURL)
	}
	if d.APIKey != "" {
		t.Errorf("expected empty default API key")
	}
}

func TestMergeFromFile_NotExist(t *testing.T) {
	cfg := defaults()
	err := mergeFromFile(&cfg, "/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	// Config should remain at defaults.
	if cfg.Model != "moonshotai/kimi-k2" {
		t.Errorf("model should stay at default")
	}
}

func TestMergeFromFile_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("api_key: test-key-123\nmodel: custom-model\n"), 0644)

	cfg := defaults()
	err := mergeFromFile(&cfg, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.APIKey != "test-key-123" {
		t.Errorf("expected api_key 'test-key-123', got %q", cfg.APIKey)
	}
	if cfg.Model != "custom-model" {
		t.Errorf("expected model 'custom-model', got %q", cfg.Model)
	}
	if cfg.BaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("base_url should stay at default, got %q", cfg.BaseURL)
	}
}

func TestMergeFromFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("api_key: [\ninvalid:\n  - {\n"), 0644)

	cfg := defaults()
	err := mergeFromFile(&cfg, path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	// Create a project config with an api_key.
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".stormtrooper"), 0755)
	os.WriteFile(filepath.Join(dir, ".stormtrooper", "config.yaml"),
		[]byte("api_key: file-key\nmodel: file-model\n"), 0644)

	// Change to the temp dir so project config is found.
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Set env var to override.
	t.Setenv("OPENROUTER_API_KEY", "env-key")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.APIKey != "env-key" {
		t.Errorf("expected env var to override file; got api_key=%q", cfg.APIKey)
	}
	if cfg.Model != "file-model" {
		t.Errorf("expected model from file, got %q", cfg.Model)
	}
}

func TestLoad_CLIOverridesEverything(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".stormtrooper"), 0755)
	os.WriteFile(filepath.Join(dir, ".stormtrooper", "config.yaml"),
		[]byte("api_key: file-key\nmodel: file-model\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	t.Setenv("OPENROUTER_API_KEY", "env-key")

	cfg, err := Load("cli-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Model != "cli-model" {
		t.Errorf("expected CLI model to override; got %q", cfg.Model)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("expected env api key; got %q", cfg.APIKey)
	}
}

func TestLoad_MissingAPIKey(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Ensure no env var is set.
	t.Setenv("OPENROUTER_API_KEY", "")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
	if !strings.Contains(err.Error(), "OPENROUTER_API_KEY") {
		t.Errorf("expected helpful error message about OPENROUTER_API_KEY, got: %v", err)
	}
}

func TestLoad_DefaultsUsedWhenNoFiles(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	t.Setenv("OPENROUTER_API_KEY", "test-key")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Model != "moonshotai/kimi-k2" {
		t.Errorf("expected default model, got %q", cfg.Model)
	}
	if cfg.BaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("expected default base URL, got %q", cfg.BaseURL)
	}
}
