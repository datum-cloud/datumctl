// Package ai defines the `datumctl ai` cobra command.
package ai

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	datumai "go.datum.net/datumctl/internal/ai"
	"go.datum.net/datumctl/internal/ai/llm"
	"go.datum.net/datumctl/internal/client"
	mcpsvc "go.datum.net/datumctl/internal/mcp"
)

// Command returns the cobra.Command for `datumctl ai`.
func Command() *cobra.Command {
	var (
		organization string
		project      string
		namespace    string
		model        string
		maxIter      int
	)

	cmd := &cobra.Command{
		Use:   "ai [query]",
		Short: "Ask a natural-language question about your Datum Cloud resources",
		Long: `Start an AI-powered assistant that translates natural language into
Datum Cloud operations.

In interactive mode (no query argument), the assistant maintains conversation
context so you can ask follow-up questions. Read operations execute immediately;
write operations always show a preview and ask for confirmation.

Configuration is read from the ai config file (see 'datumctl ai config show').
Flag values override config file values. API keys in the config file are
overridden by environment variables (ANTHROPIC_API_KEY, OPENAI_API_KEY, GEMINI_API_KEY).`,
		Example: `  # First-time setup (store API key and default org)
  datumctl ai config set anthropic_api_key sk-ant-...
  datumctl ai config set organization my-org-id

  # Then just run it — no flags needed
  datumctl ai "list all DNS zones"

  # Override the default org for one query
  datumctl ai "list projects" --organization other-org-id

  # Interactive session
  datumctl ai --project my-project-id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load config file first — flags override below.
			aiCfg, err := datumai.LoadConfig()
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "[ai] warning: could not load config: %v\n", err)
			}

			// Environment variables override config file API keys.
			aiCfg.ApplyEnvOverrides()

			// Flags override config file context.
			if organization != "" {
				aiCfg.Organization = organization
			}
			if project != "" {
				aiCfg.Project = project
			}
			if namespace != "" {
				aiCfg.Namespace = namespace
			}
			if model != "" {
				aiCfg.Model = model
			}
			if cmd.Flags().Changed("max-iterations") {
				aiCfg.MaxIterations = maxIter
			}
			if aiCfg.MaxIterations <= 0 {
				aiCfg.MaxIterations = 20
			}
			if aiCfg.Namespace == "" {
				aiCfg.Namespace = "default"
			}

			// Validate: cannot set both org and project.
			if aiCfg.Organization != "" && aiCfg.Project != "" {
				return errors.New("organization and project are mutually exclusive; unset one with 'datumctl ai config unset'")
			}

			hasContext := aiCfg.Organization != "" || aiCfg.Project != ""

			// Determine terminal/interactive mode.
			isTTY := term.IsTerminal(int(os.Stdin.Fd()))
			isInteractive := isTTY && len(args) == 0

			// Resolve the initial query.
			query, err := resolveQuery(args, isTTY)
			if err != nil {
				return err
			}

			// Construct the LLM client.
			llmClient, err := llm.NewClient(llm.Config{
				Provider:        aiCfg.Provider,
				Model:           aiCfg.Model,
				AnthropicAPIKey: aiCfg.AnthropicAPIKey,
				OpenAIAPIKey:    aiCfg.OpenAIAPIKey,
				GeminiAPIKey:    aiCfg.GeminiAPIKey,
			})
			if err != nil {
				return fmt.Errorf("initialize LLM: %w\n\nRun 'datumctl ai config set anthropic_api_key <key>' to save your key", err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "[ai] using %s/%s\n", llmClient.Provider(), llmClient.Model())

			var registry *datumai.Registry
			if hasContext {
				cfg, err := client.RestConfigForContext(cmd.Context(), aiCfg.Organization, aiCfg.Project)
				if err != nil {
					return err
				}
				k, err := client.NewK8sFromRESTConfig(cfg)
				if err != nil {
					return err
				}
				k.Namespace = aiCfg.Namespace
				if err := k.Preflight(cmd.Context()); err != nil {
					return err
				}
				svc := mcpsvc.NewService(k)
				registry = datumai.NewRegistry(svc)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "[ai] no organization or project set — running without resource tools\n")
				fmt.Fprintf(cmd.ErrOrStderr(), "[ai] tip: run 'datumctl ai config set organization <id>' to set a default\n")
				registry = datumai.NewEmptyRegistry()
			}

			systemPrompt := datumai.BuildSystemPrompt(aiCfg.Organization, aiCfg.Project, aiCfg.Namespace)

			agent := datumai.NewAgent(datumai.AgentOptions{
				LLM:           llmClient,
				Registry:      registry,
				SystemPrompt:  systemPrompt,
				MaxIterations: aiCfg.MaxIterations,
				In:            cmd.InOrStdin(),
				Out:           cmd.OutOrStdout(),
				ErrOut:        cmd.ErrOrStderr(),
				Interactive:   isInteractive,
				IsTerminal:    isTTY,
			})

			if isInteractive {
				fmt.Fprintf(cmd.OutOrStdout(), "Datum Cloud AI assistant (type 'exit' to quit)\n")
				if hasContext {
					fmt.Fprintf(cmd.OutOrStdout(), "Organization: %s  Project: %s  Namespace: %s\n\n",
						aiCfg.Organization, aiCfg.Project, aiCfg.Namespace)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "No context set — add --organization or --project to manage resources.\n\n")
				}
			}

			return agent.Run(cmd.Context(), query)
		},
	}

	cmd.Flags().StringVar(&organization, "organization", "", "Organization context (overrides config file)")
	cmd.Flags().StringVar(&project, "project", "", "Project context (overrides config file)")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Default namespace (overrides config file)")
	cmd.Flags().StringVar(&model, "model", "", "Model override, e.g. claude-sonnet-4-6, gpt-4o, gemini-2.0-flash")
	cmd.Flags().IntVar(&maxIter, "max-iterations", 20, "Agentic loop iteration cap")

	cmd.AddCommand(configCommand())

	return cmd
}

// resolveQuery determines the initial query string from CLI args or stdin.
func resolveQuery(args []string, isTTY bool) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	if !isTTY {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		q := strings.TrimSpace(string(data))
		if q == "" {
			return "", errors.New("no query provided via argument or stdin")
		}
		return q, nil
	}
	return "Hello! What would you like to do with your Datum Cloud resources?", nil
}
