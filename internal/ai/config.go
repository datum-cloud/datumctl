package ai

import (
	"fmt"
	"os"
	"path/filepath"

	syaml "sigs.k8s.io/yaml"
)

// Config holds user-level AI preferences loaded from the config file.
// All fields are optional; zero values fall back to provider/env defaults.
// Flag values always take precedence over config file values.
type Config struct {
	// LLM provider: "anthropic", "openai", or "gemini". Auto-detected from API keys if empty.
	Provider string `json:"provider,omitempty" yaml:"provider"`

	// Model overrides the provider default (e.g. "claude-sonnet-4-6").
	Model string `json:"model,omitempty" yaml:"model"`

	// MaxIterations caps the agentic loop. Defaults to 20.
	MaxIterations int `json:"max_iterations,omitempty" yaml:"max_iterations"`

	// Stream is reserved for v2. Always false in v1.
	Stream bool `json:"stream,omitempty" yaml:"stream"`

	// Default context — overridden by --organization/--project/--namespace flags.
	Organization string `json:"organization,omitempty" yaml:"organization"`
	Project      string `json:"project,omitempty" yaml:"project"`
	Namespace    string `json:"namespace,omitempty" yaml:"namespace"`

	// API keys — overridden by ANTHROPIC_API_KEY / OPENAI_API_KEY / GEMINI_API_KEY env vars.
	// Stored here so users don't have to export env vars in every shell session.
	AnthropicAPIKey string `json:"anthropic_api_key,omitempty" yaml:"anthropic_api_key"`
	OpenAIAPIKey    string `json:"openai_api_key,omitempty" yaml:"openai_api_key"`
	GeminiAPIKey    string `json:"gemini_api_key,omitempty" yaml:"gemini_api_key"`
}

// ConfigFilePath returns the platform-appropriate path to the AI config file.
// On Linux/macOS: ~/.config/datumctl/ai.yaml
// On Windows:     %AppData%\datumctl\ai.yaml
func ConfigFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, "datumctl", "ai.yaml"), nil
}

// LoadConfig reads the config file and returns a Config. A missing file is not
// an error — the returned Config will have all zero values.
func LoadConfig() (Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read AI config %s: %w", path, err)
	}
	var cfg Config
	if err := syaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse AI config %s: %w", path, err)
	}
	return cfg, nil
}

// SaveConfig writes cfg to the config file, creating the directory if needed.
func SaveConfig(cfg Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := syaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write AI config %s: %w", path, err)
	}
	return nil
}

// ApplyEnvOverrides replaces empty API key fields with values from environment
// variables. Environment variables always win over config file values.
func (c *Config) ApplyEnvOverrides() {
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		c.AnthropicAPIKey = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		c.OpenAIAPIKey = v
	}
	if v := os.Getenv("GEMINI_API_KEY"); v != "" {
		c.GeminiAPIKey = v
	}
}
