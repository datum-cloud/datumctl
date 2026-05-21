package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	openaiDefaultModel   = "gpt-4o"
	openaiAPIURL         = "https://api.openai.com/v1/chat/completions"
	openaiMaxRetries     = 3
	openaiRetryInitialMs = 500
	openaiRetryMaxMs     = 30000
)

type openaiClient struct {
	apiKey string
	model  string
}

func newOpenAIClient(cfg Config) (LLMClient, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		key = cfg.OpenAIAPIKey
	}
	if key == "" {
		return nil, fmt.Errorf("no OpenAI API key; set OPENAI_API_KEY or run: datumctl ai config set openai_api_key <key>")
	}
	model := cfg.Model
	if model == "" {
		model = openaiDefaultModel
	}
	return &openaiClient{apiKey: key, model: model}, nil
}

func (c *openaiClient) Provider() string { return "openai" }
func (c *openaiClient) Model() string    { return c.model }

// --- wire types ---

type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
	Tools    []openaiTool    `json:"tools,omitempty"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content"` // string or null
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

type openaiToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type openaiTool struct {
	Type     string       `json:"type"`
	Function openaiToolDef `json:"function"`
}

type openaiToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openaiResponse struct {
	Choices []openaiChoice `json:"choices"`
	Error   *openaiError   `json:"error,omitempty"`
}

type openaiChoice struct {
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Chat implements LLMClient for OpenAI.
func (c *openaiClient) Chat(ctx context.Context, systemPrompt string, messages []Message, tools []ToolDef) (Message, error) {
	wireMessages := []openaiMessage{}
	if systemPrompt != "" {
		wireMessages = append(wireMessages, openaiMessage{Role: "system", Content: systemPrompt})
	}
	wireMessages = append(wireMessages, toOpenAIMessages(messages)...)

	req := openaiRequest{
		Model:    c.model,
		Messages: wireMessages,
	}
	for _, t := range tools {
		params := t.InputSchema
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		req.Tools = append(req.Tools, openaiTool{
			Type: "function",
			Function: openaiToolDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return Message{}, fmt.Errorf("openai: marshal request: %w", err)
	}

	var resp openaiResponse
	if err := c.doWithRetry(ctx, body, &resp); err != nil {
		return Message{}, err
	}
	if resp.Error != nil {
		return Message{}, fmt.Errorf("openai: %s: %s", resp.Error.Type, resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return Message{}, fmt.Errorf("openai: empty choices in response")
	}

	return fromOpenAIMessage(resp.Choices[0].Message), nil
}

// toOpenAIMessages converts internal history to OpenAI wire format.
func toOpenAIMessages(messages []Message) []openaiMessage {
	var result []openaiMessage
	for _, msg := range messages {
		switch msg.Role {
		case RoleUser:
			result = append(result, openaiMessage{Role: "user", Content: msg.Content})

		case RoleAssistant:
			m := openaiMessage{Role: "assistant", Content: msg.Content}
			for _, tc := range msg.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				m.ToolCalls = append(m.ToolCalls, openaiToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openaiToolFunction{
						Name:      tc.ToolName,
						Arguments: string(argsJSON),
					},
				})
			}
			if len(m.ToolCalls) > 0 {
				m.Content = nil // OpenAI expects null content when tool_calls present
			}
			result = append(result, m)

		case RoleToolResult:
			tr := msg.ToolResult
			content := tr.Content
			if tr.IsError {
				content = "error: " + content
			}
			result = append(result, openaiMessage{
				Role:       "tool",
				Content:    content,
				ToolCallID: tr.CallID,
			})
		}
	}
	return result
}

// fromOpenAIMessage converts an OpenAI response message to the internal type.
func fromOpenAIMessage(m openaiMessage) Message {
	msg := Message{Role: RoleAssistant}
	if s, ok := m.Content.(string); ok {
		msg.Content = s
	}
	for _, tc := range m.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		msg.ToolCalls = append(msg.ToolCalls, ToolCall{
			ID:        tc.ID,
			ToolName:  tc.Function.Name,
			Arguments: args,
		})
	}
	return msg
}

// StreamChat implements LLMClient using OpenAI's streaming chat completions API.
func (c *openaiClient) StreamChat(ctx context.Context, systemPrompt string, messages []Message, tools []ToolDef, textOut io.Writer) (Message, error) {
	wireMessages := []openaiMessage{}
	if systemPrompt != "" {
		wireMessages = append(wireMessages, openaiMessage{Role: "system", Content: systemPrompt})
	}
	wireMessages = append(wireMessages, toOpenAIMessages(messages)...)

	type openaiStreamRequest struct {
		Model    string          `json:"model"`
		Messages []openaiMessage `json:"messages"`
		Tools    []openaiTool    `json:"tools,omitempty"`
		Stream   bool            `json:"stream"`
	}
	req := openaiStreamRequest{
		Model:    c.model,
		Messages: wireMessages,
		Stream:   true,
	}
	for _, t := range tools {
		params := t.InputSchema
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		req.Tools = append(req.Tools, openaiTool{
			Type: "function",
			Function: openaiToolDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return Message{}, fmt.Errorf("openai: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openaiAPIURL, bytes.NewReader(body))
	if err != nil {
		return Message{}, fmt.Errorf("openai: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return Message{}, fmt.Errorf("openai: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	type openaiIndexedTC struct {
		Index    int                `json:"index"`
		ID       string             `json:"id"`
		Function openaiToolFunction `json:"function"`
	}
	type openaiStreamChunkRich struct {
		Choices []struct {
			Delta struct {
				Content   string            `json:"content"`
				ToolCalls []openaiIndexedTC `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
		Error *openaiError `json:"error,omitempty"`
	}

	type toolAccum struct {
		id   string
		name string
		args strings.Builder
	}
	toolByIndex := map[int]*toolAccum{}
	toolOrder := []int{}

	var textBuf strings.Builder
	for ev := range scanSSE(resp.Body) {
		if ev.data == "" || ev.data == "[DONE]" {
			continue
		}
		var chunk openaiStreamChunkRich
		if err := json.Unmarshal([]byte(ev.data), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil {
			return Message{}, fmt.Errorf("openai: %s: %s", chunk.Error.Type, chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			textBuf.WriteString(delta.Content)
			if textOut != nil {
				fmt.Fprint(textOut, delta.Content)
			}
		}
		for _, tc := range delta.ToolCalls {
			if _, exists := toolByIndex[tc.Index]; !exists {
				toolByIndex[tc.Index] = &toolAccum{}
				toolOrder = append(toolOrder, tc.Index)
			}
			ta := toolByIndex[tc.Index]
			if tc.ID != "" && ta.id == "" {
				ta.id = tc.ID
			}
			if tc.Function.Name != "" && ta.name == "" {
				ta.name = tc.Function.Name
			}
			ta.args.WriteString(tc.Function.Arguments)
		}
	}

	msg := Message{Role: RoleAssistant, Content: textBuf.String()}
	for _, idx := range toolOrder {
		ta := toolByIndex[idx]
		var args map[string]any
		if ta.args.Len() > 0 {
			_ = json.Unmarshal([]byte(ta.args.String()), &args)
		}
		if args == nil {
			args = map[string]any{}
		}
		msg.ToolCalls = append(msg.ToolCalls, ToolCall{
			ID:        ta.id,
			ToolName:  ta.name,
			Arguments: args,
		})
	}
	return msg, nil
}

func (c *openaiClient) doWithRetry(ctx context.Context, body []byte, out *openaiResponse) error {
	delay := time.Duration(openaiRetryInitialMs) * time.Millisecond
	for attempt := 0; attempt < openaiMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			if delay > time.Duration(openaiRetryMaxMs)*time.Millisecond {
				delay = time.Duration(openaiRetryMaxMs) * time.Millisecond
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, openaiAPIURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("openai: build request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("openai: http: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("openai: read response: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusInternalServerError ||
			resp.StatusCode == http.StatusServiceUnavailable {
			if attempt < openaiMaxRetries-1 {
				continue
			}
			return fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("openai: unmarshal response: %w", err)
		}
		return nil
	}
	return fmt.Errorf("openai: max retries exceeded")
}
