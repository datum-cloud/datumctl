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
	anthropicDefaultModel   = "claude-sonnet-4-6"
	anthropicAPIURL         = "https://api.anthropic.com/v1/messages"
	anthropicVersion        = "2023-06-01"
	anthropicMaxTokens      = 4096
	anthropicMaxRetries     = 3
	anthropicRetryInitialMs = 500
	anthropicRetryMaxMs     = 30000
)

type anthropicClient struct {
	apiKey string
	model  string
}

func newAnthropicClient(cfg Config) (LLMClient, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		key = cfg.AnthropicAPIKey
	}
	if key == "" {
		return nil, fmt.Errorf("no Anthropic API key; set ANTHROPIC_API_KEY or run: datumctl ai config set anthropic_api_key <key>")
	}
	model := cfg.Model
	if model == "" {
		model = anthropicDefaultModel
	}
	return &anthropicClient{apiKey: key, model: model}, nil
}

func (c *anthropicClient) Provider() string { return "anthropic" }
func (c *anthropicClient) Model() string    { return c.model }

// --- wire types ---

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []anthropicContent
}

type anthropicContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     *map[string]any `json:"input,omitempty"` // pointer so nil=omit, &{}="{}"
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContent `json:"content"`
	StopReason string             `json:"stop_reason"`
	Error      *anthropicError    `json:"error,omitempty"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Chat implements LLMClient. It converts the internal history to Anthropic's
// wire format, sends the request with retry logic, and converts the response
// back to the internal Message type.
func (c *anthropicClient) Chat(ctx context.Context, systemPrompt string, messages []Message, tools []ToolDef) (Message, error) {
	wireMessages := toAnthropicMessages(messages)

	req := anthropicRequest{
		Model:     c.model,
		MaxTokens: anthropicMaxTokens,
		System:    systemPrompt,
		Messages:  wireMessages,
	}
	for _, t := range tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		req.Tools = append(req.Tools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return Message{}, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	var resp anthropicResponse
	if err := c.doWithRetry(ctx, body, &resp); err != nil {
		return Message{}, err
	}
	if resp.Error != nil {
		return Message{}, fmt.Errorf("anthropic: %s: %s", resp.Error.Type, resp.Error.Message)
	}

	return fromAnthropicResponse(resp), nil
}

// toAnthropicMessages converts the internal history to Anthropic wire format.
// Key rules:
//   - RoleUser → role "user", string content
//   - RoleAssistant → role "assistant", content array (text + tool_use blocks)
//   - RoleToolResult → batched into a single role "user" message with
//     tool_result content blocks (consecutive RoleToolResult entries share one message)
func toAnthropicMessages(messages []Message) []anthropicMessage {
	var result []anthropicMessage

	for i := 0; i < len(messages); {
		msg := messages[i]
		switch msg.Role {
		case RoleUser:
			result = append(result, anthropicMessage{Role: "user", Content: msg.Content})
			i++

		case RoleAssistant:
			var blocks []anthropicContent
			if msg.Content != "" {
				blocks = append(blocks, anthropicContent{Type: "text", Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				input := tc.Arguments
				if input == nil {
					input = map[string]any{}
				}
				blocks = append(blocks, anthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.ToolName,
					Input: &input,
				})
			}
			result = append(result, anthropicMessage{Role: "assistant", Content: blocks})
			i++

		case RoleToolResult:
			// Batch all consecutive tool_result messages into one user message.
			var blocks []anthropicContent
			for i < len(messages) && messages[i].Role == RoleToolResult {
				tr := messages[i].ToolResult
				block := anthropicContent{
					Type:      "tool_result",
					ToolUseID: tr.CallID,
					Content:   tr.Content,
				}
				if tr.IsError {
					block.IsError = true
				}
				blocks = append(blocks, block)
				i++
			}
			result = append(result, anthropicMessage{Role: "user", Content: blocks})

		default:
			i++
		}
	}

	return result
}

// fromAnthropicResponse converts an Anthropic response to the internal Message type.
func fromAnthropicResponse(resp anthropicResponse) Message {
	msg := Message{Role: RoleAssistant}
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			msg.Content += block.Text
		case "tool_use":
			var args map[string]any
			if block.Input != nil {
				args = *block.Input
			} else {
				args = map[string]any{}
			}
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:        block.ID,
				ToolName:  block.Name,
				Arguments: args,
			})
		}
	}
	return msg
}

// --- streaming wire types ---

type anthropicStreamData struct {
	Type         string                `json:"type"`
	Index        int                   `json:"index"`
	Error        *anthropicError       `json:"error,omitempty"`
	ContentBlock *anthropicStreamBlock `json:"content_block,omitempty"`
	Delta        *anthropicStreamDelta `json:"delta,omitempty"`
}

type anthropicStreamBlock struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type anthropicStreamDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

// StreamChat implements LLMClient. It uses Anthropic's streaming Messages API
// to write text delta chunks to textOut as they arrive, and reconstructs tool
// calls from accumulated input_json_delta events.
func (c *anthropicClient) StreamChat(ctx context.Context, systemPrompt string, messages []Message, tools []ToolDef, textOut io.Writer) (Message, error) {
	wireMessages := toAnthropicMessages(messages)
	req := anthropicRequest{
		Model:     c.model,
		MaxTokens: anthropicMaxTokens,
		System:    systemPrompt,
		Messages:  wireMessages,
		Stream:    true,
	}
	for _, t := range tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		req.Tools = append(req.Tools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return Message{}, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return Message{}, fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return Message{}, fmt.Errorf("anthropic: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// blockType tracks whether each content block index is "text" or "tool_use".
	type toolCallAccum struct {
		id   string
		name string
		args strings.Builder
	}
	blockType := map[int]string{}
	toolCalls := map[int]*toolCallAccum{}
	toolOrder := []int{} // preserves insertion order

	var textBuf strings.Builder
	for ev := range scanSSE(resp.Body) {
		if ev.data == "" || ev.data == "[DONE]" {
			continue
		}
		var d anthropicStreamData
		if err := json.Unmarshal([]byte(ev.data), &d); err != nil {
			continue
		}
		switch d.Type {
		case "error":
			if d.Error != nil {
				return Message{}, fmt.Errorf("anthropic: %s: %s", d.Error.Type, d.Error.Message)
			}
		case "content_block_start":
			if d.ContentBlock == nil {
				continue
			}
			blockType[d.Index] = d.ContentBlock.Type
			if d.ContentBlock.Type == "tool_use" {
				toolCalls[d.Index] = &toolCallAccum{
					id:   d.ContentBlock.ID,
					name: d.ContentBlock.Name,
				}
				toolOrder = append(toolOrder, d.Index)
			}
		case "content_block_delta":
			if d.Delta == nil {
				continue
			}
			switch d.Delta.Type {
			case "text_delta":
				textBuf.WriteString(d.Delta.Text)
				if textOut != nil {
					fmt.Fprint(textOut, d.Delta.Text)
				}
			case "input_json_delta":
				if tc, ok := toolCalls[d.Index]; ok {
					tc.args.WriteString(d.Delta.PartialJSON)
				}
			}
		}
	}

	msg := Message{Role: RoleAssistant, Content: textBuf.String()}
	for _, idx := range toolOrder {
		tc := toolCalls[idx]
		var args map[string]any
		if tc.args.Len() > 0 {
			_ = json.Unmarshal([]byte(tc.args.String()), &args)
		}
		if args == nil {
			args = map[string]any{}
		}
		msg.ToolCalls = append(msg.ToolCalls, ToolCall{
			ID:       tc.id,
			ToolName: tc.name,
			Arguments: args,
		})
	}
	return msg, nil
}

func (c *anthropicClient) doWithRetry(ctx context.Context, body []byte, out *anthropicResponse) error {
	delay := time.Duration(anthropicRetryInitialMs) * time.Millisecond
	for attempt := 0; attempt < anthropicMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			if delay > time.Duration(anthropicRetryMaxMs)*time.Millisecond {
				delay = time.Duration(anthropicRetryMaxMs) * time.Millisecond
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("anthropic: build request: %w", err)
		}
		httpReq.Header.Set("x-api-key", c.apiKey)
		httpReq.Header.Set("anthropic-version", anthropicVersion)
		httpReq.Header.Set("content-type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("anthropic: http: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("anthropic: read response: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusInternalServerError ||
			resp.StatusCode == http.StatusServiceUnavailable {
			if attempt < anthropicMaxRetries-1 {
				continue
			}
			return fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("anthropic: unmarshal response: %w", err)
		}
		return nil
	}
	return fmt.Errorf("anthropic: max retries exceeded")
}
