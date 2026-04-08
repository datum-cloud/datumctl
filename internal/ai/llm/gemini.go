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
	geminiDefaultModel   = "gemini-2.0-flash"
	geminiAPIBase        = "https://generativelanguage.googleapis.com/v1beta/models"
	geminiMaxRetries     = 3
	geminiRetryInitialMs = 500
	geminiRetryMaxMs     = 30000
)

type geminiClient struct {
	apiKey string
	model  string
}

func newGeminiClient(cfg Config) (LLMClient, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		key = cfg.GeminiAPIKey
	}
	if key == "" {
		return nil, fmt.Errorf("no Gemini API key; set GEMINI_API_KEY or run: datumctl ai config set gemini_api_key <key>")
	}
	model := cfg.Model
	if model == "" {
		model = geminiDefaultModel
	}
	return &geminiClient{apiKey: key, model: model}, nil
}

func (c *geminiClient) Provider() string { return "gemini" }
func (c *geminiClient) Model() string    { return c.model }

// --- wire types ---

type geminiRequest struct {
	Contents          []geminiContent      `json:"contents"`
	Tools             []geminiToolList     `json:"tools,omitempty"`
	SystemInstruction *geminiContent       `json:"systemInstruction,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string             `json:"text,omitempty"`
	FunctionCall *geminiFunctionCall `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiToolList struct {
	FunctionDeclarations []geminiFunctionDecl `json:"functionDeclarations"`
}

type geminiFunctionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Chat implements LLMClient for Gemini.
func (c *geminiClient) Chat(ctx context.Context, systemPrompt string, messages []Message, tools []ToolDef) (Message, error) {
	req := geminiRequest{
		Contents: toGeminiContents(messages),
	}
	if systemPrompt != "" {
		req.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		}
	}
	if len(tools) > 0 {
		var decls []geminiFunctionDecl
		for _, t := range tools {
			params := t.InputSchema
			if params == nil {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			decls = append(decls, geminiFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			})
		}
		req.Tools = []geminiToolList{{FunctionDeclarations: decls}}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return Message{}, fmt.Errorf("gemini: marshal request: %w", err)
	}

	var resp geminiResponse
	if err := c.doWithRetry(ctx, body, &resp); err != nil {
		return Message{}, err
	}
	if resp.Error != nil {
		return Message{}, fmt.Errorf("gemini: %s (%d): %s", resp.Error.Status, resp.Error.Code, resp.Error.Message)
	}
	if len(resp.Candidates) == 0 {
		return Message{}, fmt.Errorf("gemini: empty candidates in response")
	}

	return fromGeminiContent(resp.Candidates[0].Content), nil
}

// toGeminiContents converts the internal history to Gemini wire format.
// Gemini uses "user" and "model" roles; tool calls and responses are embedded
// as function call/response parts within those turns.
func toGeminiContents(messages []Message) []geminiContent {
	var result []geminiContent

	for i := 0; i < len(messages); {
		msg := messages[i]
		switch msg.Role {
		case RoleUser:
			result = append(result, geminiContent{
				Role:  "user",
				Parts: []geminiPart{{Text: msg.Content}},
			})
			i++

		case RoleAssistant:
			var parts []geminiPart
			if msg.Content != "" {
				parts = append(parts, geminiPart{Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				parts = append(parts, geminiPart{
					FunctionCall: &geminiFunctionCall{Name: tc.ToolName, Args: tc.Arguments},
				})
			}
			result = append(result, geminiContent{Role: "model", Parts: parts})
			i++

		case RoleToolResult:
			// Batch consecutive tool results into a single "user" turn.
			var parts []geminiPart
			for i < len(messages) && messages[i].Role == RoleToolResult {
				tr := messages[i].ToolResult
				responseMap := map[string]any{"output": tr.Content}
				if tr.IsError {
					responseMap = map[string]any{"error": tr.Content}
				}
				parts = append(parts, geminiPart{
					FunctionResponse: &geminiFunctionResponse{
						Name:     tr.CallID,
						Response: responseMap,
					},
				})
				i++
			}
			result = append(result, geminiContent{Role: "user", Parts: parts})

		default:
			i++
		}
	}

	return result
}

// fromGeminiContent converts a Gemini response content to the internal Message type.
func fromGeminiContent(c geminiContent) Message {
	msg := Message{Role: RoleAssistant}
	for _, part := range c.Parts {
		if part.Text != "" {
			msg.Content += part.Text
		}
		if part.FunctionCall != nil {
			// Gemini does not provide stable IDs; use tool name as a stand-in.
			// This works because we batch all results before the next Chat call.
			msg.ToolCalls = append(msg.ToolCalls, ToolCall{
				ID:        part.FunctionCall.Name,
				ToolName:  part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			})
		}
	}
	return msg
}

// StreamChat implements LLMClient using Gemini's streamGenerateContent endpoint.
// Gemini streams a JSON array where each element is a full geminiResponse chunk.
// Text parts are written to textOut as they arrive; function call parts are
// accumulated and returned in the Message.
func (c *geminiClient) StreamChat(ctx context.Context, systemPrompt string, messages []Message, tools []ToolDef, textOut io.Writer) (Message, error) {
	req := geminiRequest{Contents: toGeminiContents(messages)}
	if systemPrompt != "" {
		req.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
		}
	}
	if len(tools) > 0 {
		var decls []geminiFunctionDecl
		for _, t := range tools {
			params := t.InputSchema
			if params == nil {
				params = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			decls = append(decls, geminiFunctionDecl{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			})
		}
		req.Tools = []geminiToolList{{FunctionDeclarations: decls}}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return Message{}, fmt.Errorf("gemini: marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s:streamGenerateContent?alt=sse&key=%s", geminiAPIBase, c.model, c.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Message{}, fmt.Errorf("gemini: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return Message{}, fmt.Errorf("gemini: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return Message{}, fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var textBuf strings.Builder
	msg := Message{Role: RoleAssistant}

	for ev := range scanSSE(resp.Body) {
		if ev.data == "" || ev.data == "[DONE]" {
			continue
		}
		var chunk geminiResponse
		if err := json.Unmarshal([]byte(ev.data), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil {
			return Message{}, fmt.Errorf("gemini: %s (%d): %s", chunk.Error.Status, chunk.Error.Code, chunk.Error.Message)
		}
		if len(chunk.Candidates) == 0 {
			continue
		}
		for _, part := range chunk.Candidates[0].Content.Parts {
			if part.Text != "" {
				textBuf.WriteString(part.Text)
				if textOut != nil {
					fmt.Fprint(textOut, part.Text)
				}
			}
			if part.FunctionCall != nil {
				msg.ToolCalls = append(msg.ToolCalls, ToolCall{
					ID:        part.FunctionCall.Name,
					ToolName:  part.FunctionCall.Name,
					Arguments: part.FunctionCall.Args,
				})
			}
		}
	}

	msg.Content = textBuf.String()
	return msg, nil
}

func (c *geminiClient) doWithRetry(ctx context.Context, body []byte, out *geminiResponse) error {
	url := fmt.Sprintf("%s/%s:generateContent?key=%s", geminiAPIBase, c.model, c.apiKey)
	delay := time.Duration(geminiRetryInitialMs) * time.Millisecond

	for attempt := 0; attempt < geminiMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
			if delay > time.Duration(geminiRetryMaxMs)*time.Millisecond {
				delay = time.Duration(geminiRetryMaxMs) * time.Millisecond
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("gemini: build request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("gemini: http: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("gemini: read response: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusInternalServerError ||
			resp.StatusCode == http.StatusServiceUnavailable {
			if attempt < geminiMaxRetries-1 {
				continue
			}
			return fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("gemini: HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("gemini: unmarshal response: %w", err)
		}
		return nil
	}
	return fmt.Errorf("gemini: max retries exceeded")
}
