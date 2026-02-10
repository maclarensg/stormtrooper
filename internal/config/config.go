// Package config handles configuration loading with layering:
// defaults -> global config -> project config -> env vars -> CLI flags.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all runtime configuration.
type Config struct {
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
	BaseURL string `yaml:"base_url"`
}

// defaults returns a Config populated with hardcoded default values.
func defaults() Config {
	return Config{
		Model:   "moonshotai/kimi-k2",
		BaseURL: "https://openrouter.ai/api/v1",
	}
}

// Load reads config from all layers and returns the merged result.
// cliModel is the --model flag value (empty string if not set).
func Load(cliModel string) (*Config, error) {
	cfg := defaults()

	// Layer 2: Global config
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".stormtrooper", "config.yaml")
		if err := mergeFromFile(&cfg, globalPath); err != nil {
			return nil, fmt.Errorf("global config %s: %w", globalPath, err)
		}
	}

	// Layer 3: Project config
	projectPath := filepath.Join(".stormtrooper", "config.yaml")
	if err := mergeFromFile(&cfg, projectPath); err != nil {
		return nil, fmt.Errorf("project config %s: %w", projectPath, err)
	}

	// Layer 4: Environment variables
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		cfg.APIKey = key
	}

	// Layer 5: CLI flags
	if cliModel != "" {
		cfg.Model = cliModel
	}

	// Validate
	if cfg.APIKey == "" {
		return nil, errors.New("OPENROUTER_API_KEY not set. Set it as an environment variable or in ~/.stormtrooper/config.yaml")
	}

	return &cfg, nil
}

// mergeFromFile reads a YAML config file and merges non-zero values into cfg.
// If the file does not exist, it is silently skipped.
func mergeFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var fileCfg Config
	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	if fileCfg.APIKey != "" {
		cfg.APIKey = fileCfg.APIKey
	}
	if fileCfg.Model != "" {
		cfg.Model = fileCfg.Model
	}
	if fileCfg.BaseURL != "" {
		cfg.BaseURL = fileCfg.BaseURL
	}

	return nil
}
