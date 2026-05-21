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
)

// Command returns the cobra.Command for `datumctl ai`.
func Command() *cobra.Command {
	var (
		organization string
		project      string
		namespace    string
		model        string
		maxIter      int
		platformWide bool
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
			if cmd.Flags().Changed("platform-wide") {
				aiCfg.PlatformWide = platformWide
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

			// Validate mutual exclusions.
			if aiCfg.PlatformWide && (aiCfg.Organization != "" || aiCfg.Project != "") {
				return errors.New("--platform-wide cannot be used with --organization or --project")
			}
			if aiCfg.Organization != "" && aiCfg.Project != "" {
				return errors.New("organization and project are mutually exclusive; unset one with 'datumctl ai config unset'")
			}

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

			// Build the factory and resolve the effective scope using priority order:
			// CLI flags → env vars → active datumctl context → ai config defaults.
			// AI config values are only applied as a last resort so that switching
			// context via `datumctl ctx use` is respected without also having to
			// clear `datumctl ai config set project/organization`.
			factory, err := client.NewDatumFactory(cmd.Context())
			if err != nil {
				return fmt.Errorf("build client factory: %w", err)
			}
			// Apply only explicit CLI flags first (organization/project/platformWide
			// vars are empty/false when the flags were not passed).
			if organization != "" {
				*factory.ConfigFlags.Organization = organization
			}
			if project != "" {
				*factory.ConfigFlags.Project = project
			}
			if cmd.Flags().Changed("platform-wide") {
				*factory.ConfigFlags.PlatformWide = platformWide
			}

			// Resolve: CLI flags → env vars → active datumctl context.
			resolvedProject, resolvedOrg, resolvedPlatformWide, err := factory.ConfigFlags.ResolvedScope()
			if err != nil {
				return fmt.Errorf("resolve scope: %w", err)
			}

			// Fall back to ai config defaults only when nothing else provided a scope.
			if !resolvedPlatformWide && resolvedOrg == "" && resolvedProject == "" {
				switch {
				case aiCfg.PlatformWide:
					*factory.ConfigFlags.PlatformWide = true
					resolvedPlatformWide = true
				case aiCfg.Organization != "":
					*factory.ConfigFlags.Organization = aiCfg.Organization
					resolvedOrg = aiCfg.Organization
				case aiCfg.Project != "":
					*factory.ConfigFlags.Project = aiCfg.Project
					resolvedProject = aiCfg.Project
				}
			}

			hasContext := resolvedPlatformWide || resolvedOrg != "" || resolvedProject != ""

			// Propagate the session namespace to the factory so tools can use it
			// as a default when the caller doesn't pass an explicit namespace.
			*factory.ConfigFlags.Namespace = aiCfg.Namespace

			// Build the tool registry.
			var registry *datumai.Registry
			if hasContext {
				// Preflight: surface auth errors early.
				if _, err := factory.ConfigFlags.ToRESTConfig(); err != nil {
					return fmt.Errorf("connect to Datum Cloud: %w\n\nRun 'datumctl login' to authenticate", err)
				}
				registry = datumai.NewRegistry(factory)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "[ai] no organization or project set — running without resource tools\n")
				fmt.Fprintf(cmd.ErrOrStderr(), "[ai] tip: run 'datumctl ai config set organization <id>' to set a default\n")
				registry = datumai.NewEmptyRegistry()
			}

			systemPrompt := datumai.BuildSystemPrompt(resolvedOrg, resolvedProject, aiCfg.Namespace, resolvedPlatformWide)

			var gate datumai.ConfirmGate
			if isTTY {
				gate = datumai.StdinGate{In: cmd.InOrStdin(), Out: cmd.ErrOrStderr()}
			} else {
				gate = datumai.AutoDeclineGate{ErrOut: cmd.ErrOrStderr()}
			}

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
				Gate:          gate,
			})

			if isInteractive {
				fmt.Fprintf(cmd.OutOrStdout(), "Patch — Datum Cloud AI assistant (type 'exit' to quit)\n")
				switch {
				case resolvedPlatformWide:
					fmt.Fprintf(cmd.OutOrStdout(), "Mode: platform-wide (staff portal)\n\n")
				case hasContext:
					fmt.Fprintf(cmd.OutOrStdout(), "Organization: %s  Project: %s  Namespace: %s\n\n",
						resolvedOrg, resolvedProject, aiCfg.Namespace)
				default:
					fmt.Fprintf(cmd.OutOrStdout(), "No context set — add --organization, --project, or --platform-wide to manage resources.\n\n")
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
	cmd.Flags().BoolVar(&platformWide, "platform-wide", false, "Access platform root (staff portal) instead of an org or project control plane")
	cmd.MarkFlagsMutuallyExclusive("organization", "project", "platform-wide")

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
