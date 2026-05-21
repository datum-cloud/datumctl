// Package llm provides a provider-agnostic LLM client interface and shared
// types used by the datumctl ai agentic loop.
package llm

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Role identifies the speaker of a conversation turn.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	// RoleToolResult is an internal role for tool result messages stored in the
	// shared history slice. It is NEVER sent verbatim to any provider.
	// Each provider's Chat() implementation translates it to the appropriate
	// wire format (Anthropic: role "user" with tool_result content blocks;
	// OpenAI: role "tool").
	RoleToolResult Role = "tool_result"
)

// Message is one turn in the conversation history.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall  // populated when the LLM requests tool invocations
	ToolResult *ToolResult // populated for RoleToolResult messages
}

// ToolCall represents the LLM's request to invoke a named tool.
type ToolCall struct {
	ID        string         // provider-assigned ID used to correlate results
	ToolName  string
	Arguments map[string]any
}

// ToolResult is the response fed back to the LLM after a tool executes.
type ToolResult struct {
	CallID  string
	Content string
	IsError bool
}

// ToolDef is the schema the LLM sees when deciding which tools to call.
type ToolDef struct {
	Name        string
	Description string
	InputSchema map[string]any // JSON Schema object
}

// LLMClient is the provider abstraction used by the agentic loop.
type LLMClient interface {
	// Chat sends the conversation history and available tools to the provider
	// and returns the next assistant message. The system prompt is not included
	// in messages — providers receive it separately via their constructor config.
	Chat(ctx context.Context, systemPrompt string, messages []Message, tools []ToolDef) (Message, error)

	// StreamChat is like Chat but streams text delta chunks to textOut as they
	// arrive from the provider. Tool-call arguments are accumulated internally
	// and returned in the Message. textOut may be nil to suppress streaming.
	StreamChat(ctx context.Context, systemPrompt string, messages []Message, tools []ToolDef, textOut io.Writer) (Message, error)

	Provider() string
	Model() string
}

// Config holds provider, model, and API key preferences for NewClient.
// API keys here are fallbacks; environment variables always take precedence.
type Config struct {
	Provider        string
	Model           string
	AnthropicAPIKey string // fallback if ANTHROPIC_API_KEY env var is unset
	OpenAIAPIKey    string // fallback if OPENAI_API_KEY env var is unset
	GeminiAPIKey    string // fallback if GEMINI_API_KEY env var is unset
}

// NewClient constructs an LLMClient using the following priority:
//  1. Model name prefix: claude-→Anthropic, gpt-/o1/o3→OpenAI, gemini-→Gemini
//  2. cfg.Provider explicit override
//  3. Which API key is available (env var > config file key)
func NewClient(cfg Config) (LLMClient, error) {
	if cfg.Model != "" {
		switch {
		case strings.HasPrefix(cfg.Model, "claude-"):
			return newAnthropicClient(cfg)
		case strings.HasPrefix(cfg.Model, "gpt-"),
			strings.HasPrefix(cfg.Model, "o1"),
			strings.HasPrefix(cfg.Model, "o3"):
			return newOpenAIClient(cfg)
		case strings.HasPrefix(cfg.Model, "gemini-"):
			return newGeminiClient(cfg)
		}
	}

	switch cfg.Provider {
	case "anthropic":
		return newAnthropicClient(cfg)
	case "openai":
		return newOpenAIClient(cfg)
	case "gemini":
		return newGeminiClient(cfg)
	}

	// Auto-detect from available API keys.
	switch {
	case cfg.AnthropicAPIKey != "":
		return newAnthropicClient(cfg)
	case cfg.OpenAIAPIKey != "":
		return newOpenAIClient(cfg)
	case cfg.GeminiAPIKey != "":
		return newGeminiClient(cfg)
	default:
		return nil, fmt.Errorf(
			"no LLM API key found; set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GEMINI_API_KEY,\n" +
				"or run: datumctl ai config set anthropic_api_key <key>")
	}
}
