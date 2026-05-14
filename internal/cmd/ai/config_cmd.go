package ai

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	datumai "go.datum.net/datumctl/internal/ai"
)

// validKeys is the set of keys accepted by `datumctl ai config set`.
var validKeys = map[string]string{
	"organization":      "Default organization ID (overridden by --organization flag)",
	"project":           "Default project ID (overridden by --project flag)",
	"namespace":         "Default namespace (overridden by --namespace flag)",
	"provider":          "LLM provider: anthropic, openai, or gemini",
	"model":             "LLM model override (e.g. claude-sonnet-4-6, gpt-4o)",
	"max_iterations":    "Agentic loop iteration cap (default 20)",
	"anthropic_api_key": "Anthropic API key (overridden by ANTHROPIC_API_KEY env var)",
	"openai_api_key":    "OpenAI API key (overridden by OPENAI_API_KEY env var)",
	"gemini_api_key":    "Gemini API key (overridden by GEMINI_API_KEY env var)",
}

func configCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage datumctl ai configuration",
		Long: `Read and write the datumctl ai configuration file.

Settings in the config file provide defaults for every 'datumctl ai' invocation.
Flag values always override config file values; environment variables always
override config file API keys.`,
	}
	cmd.AddCommand(configSetCommand(), configShowCommand(), configUnsetCommand())
	return cmd
}

func configSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a key in the datumctl ai configuration file.

Valid keys:
  organization      Default organization ID
  project           Default project ID
  namespace         Default namespace (default: "default")
  provider          LLM provider: anthropic, openai, or gemini
  model             LLM model (e.g. claude-sonnet-4-6, gpt-4o, gemini-2.0-flash)
  max_iterations    Agentic loop iteration cap (default 20)
  anthropic_api_key Anthropic API key
  openai_api_key    OpenAI API key
  gemini_api_key    Gemini API key`,
		Example: `  datumctl ai config set organization datum-demos-iy50km
  datumctl ai config set anthropic_api_key sk-ant-...
  datumctl ai config set model claude-sonnet-4-6`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			if _, ok := validKeys[key]; !ok {
				return fmt.Errorf("unknown config key %q; valid keys: %s",
					key, strings.Join(sortedKeys(validKeys), ", "))
			}

			cfg, err := datumai.LoadConfig()
			if err != nil {
				return err
			}

			if err := setConfigKey(&cfg, key, value); err != nil {
				return err
			}

			if err := datumai.SaveConfig(cfg); err != nil {
				return err
			}

			path, _ := datumai.ConfigFilePath()
			fmt.Fprintf(cmd.OutOrStdout(), "Set %s in %s\n", key, path)
			return nil
		},
	}
}

func configUnsetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove a configuration value",
		Example: `  datumctl ai config unset organization
  datumctl ai config unset anthropic_api_key`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if _, ok := validKeys[key]; !ok {
				return fmt.Errorf("unknown config key %q", key)
			}

			cfg, err := datumai.LoadConfig()
			if err != nil {
				return err
			}

			if err := setConfigKey(&cfg, key, ""); err != nil {
				return err
			}

			if err := datumai.SaveConfig(cfg); err != nil {
				return err
			}

			path, _ := datumai.ConfigFilePath()
			fmt.Fprintf(cmd.OutOrStdout(), "Unset %s in %s\n", key, path)
			return nil
		},
	}
}

func configShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "show",
		Short:   "Show the current configuration",
		Example: `  datumctl ai config show`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := datumai.ConfigFilePath()
			if err != nil {
				return err
			}

			cfg, err := datumai.LoadConfig()
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			row := func(key, value string) {
				if value == "" {
					fmt.Fprintf(w, "  %-22s (not set)\n", key)
				} else {
					fmt.Fprintf(w, "  %-22s %s\n", key, value)
				}
			}

			fmt.Fprintf(w, "Patch configuration\n")
			fmt.Fprintf(w, "%s\n\n", path)

			fmt.Fprintf(w, "CONTEXT\n")
			row("organization", cfg.Organization)
			row("project", cfg.Project)
			ns := cfg.Namespace
			if ns == "" {
				ns = "default"
			}
			row("namespace", ns)

			fmt.Fprintf(w, "\nMODEL\n")
			row("provider", cfg.Provider)
			row("model", cfg.Model)
			iters := ""
			if cfg.MaxIterations > 0 {
				iters = fmt.Sprintf("%d", cfg.MaxIterations)
			} else {
				iters = "20 (default)"
			}
			row("max_iterations", iters)

			fmt.Fprintf(w, "\nAPI KEYS\n")
			row("anthropic_api_key", redact(cfg.AnthropicAPIKey))
			row("openai_api_key", redact(cfg.OpenAIAPIKey))
			row("gemini_api_key", redact(cfg.GeminiAPIKey))

			fmt.Fprintln(w)
			return nil
		},
	}
}

func setConfigKey(cfg *datumai.Config, key, value string) error {
	switch key {
	case "organization":
		cfg.Organization = value
	case "project":
		cfg.Project = value
	case "namespace":
		cfg.Namespace = value
	case "provider":
		cfg.Provider = value
	case "model":
		cfg.Model = value
	case "max_iterations":
		if value == "" {
			cfg.MaxIterations = 0
			return nil
		}
		var n int
		if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
			return fmt.Errorf("max_iterations must be an integer, got %q", value)
		}
		cfg.MaxIterations = n
	case "anthropic_api_key":
		cfg.AnthropicAPIKey = value
	case "openai_api_key":
		cfg.OpenAIAPIKey = value
	case "gemini_api_key":
		cfg.GeminiAPIKey = value
	}
	return nil
}

func redact(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
