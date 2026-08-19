package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
)

const (
	claudeMessagesPath      = "/messages"
	claudeAPIKeyHeader      = "x-api-key"
	claudeVersionHeader     = "anthropic-version"
	claudeDefaultAPIVersion = "2023-06-01"
	claudeDefaultMaxTokens  = 4096
)

type claudeProvider struct{ config ClientConfig }

func (p *claudeProvider) info() ProviderInfo {
	return ProviderInfo{ID: ProviderClaude, DefaultBaseURL: "https://api.anthropic.com/v1", Capabilities: []Capability{CapabilityChat}}
}

type claudeContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Thinking string          `json:"thinking,omitempty"`
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
}

type claudeResponse struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Role       string          `json:"role"`
	Model      string          `json:"model"`
	StopReason string          `json:"stop_reason"`
	Content    []claudeContent `json:"content"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *claudeProvider) createChat(ctx context.Context, selected credential, request *chat.CreateRequest) (*chat.CreateResponse, error) {
	body, err := buildClaudeRequest(request, false)
	if err != nil {
		return nil, err
	}
	response, err := p.do(ctx, selected, body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var source claudeResponse
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &chat.CreateResponse{ID: source.ID, Object: "chat.completion", Model: source.Model, RawResponse: raw}
	choice := chat.Choice{Index: 0, FinishReason: source.StopReason, Message: chat.Message{Role: chat.RoleAssistant}}
	var text, reasoning strings.Builder
	for _, content := range source.Content {
		switch content.Type {
		case "text":
			text.WriteString(content.Text)
		case "thinking", "redacted_thinking":
			reasoning.WriteString(content.Thinking)
		case "tool_use":
			choice.Message.ToolCalls = append(choice.Message.ToolCalls, chat.ToolCall{ID: content.ID, Type: "function", Function: chat.FunctionCall{Name: content.Name, Arguments: string(content.Input)}})
		}
	}
	choice.Message.Content = chat.TextContent(text.String())
	choice.Message.ReasoningContent = reasoning.String()
	result.Choices = []chat.Choice{choice}
	result.Usage.PromptTokens = source.Usage.InputTokens
	result.Usage.CompletionTokens = source.Usage.OutputTokens
	result.Usage.TotalTokens = source.Usage.InputTokens + source.Usage.OutputTokens
	return result, nil
}

func (p *claudeProvider) streamChat(ctx context.Context, selected credential, request *chat.StreamRequest) (chat.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: chat request is nil", ErrInvalidRequest)
	}
	body, err := buildClaudeRequest(&request.CreateRequest, true)
	if err != nil {
		return nil, err
	}
	streamContext, cancel := context.WithCancel(ctx)
	response, err := p.do(streamContext, selected, body)
	if err != nil {
		cancel()
		return nil, err
	}
	return newClaudeStream(response.Body, cancel), nil
}

func buildClaudeRequest(request *chat.CreateRequest, stream bool) (map[string]any, error) {
	if request == nil || request.Model == "" || len(request.Messages) == 0 {
		return nil, fmt.Errorf("%w: model and messages are required", ErrInvalidRequest)
	}
	messages := make([]chat.Message, 0, len(request.Messages))
	var system strings.Builder
	for _, message := range request.Messages {
		if message.Role == chat.RoleSystem || message.Role == chat.RoleDeveloper {
			if message.Content.Text != nil {
				system.WriteString(*message.Content.Text)
			}
			continue
		}
		messages = append(messages, message)
	}
	maxTokens := claudeDefaultMaxTokens
	if request.MaxCompletionTokens != nil {
		maxTokens = *request.MaxCompletionTokens
	} else if request.MaxTokens != nil {
		maxTokens = *request.MaxTokens
	}
	body := map[string]any{"model": request.Model, "messages": messages, "max_tokens": maxTokens, "stream": stream}
	if system.Len() > 0 {
		body["system"] = system.String()
	}
	if request.Temperature != nil {
		body["temperature"] = *request.Temperature
	}
	if request.TopP != nil {
		body["top_p"] = *request.TopP
	}
	if len(request.Tools) > 0 {
		tools := make([]map[string]any, 0, len(request.Tools))
		for _, tool := range request.Tools {
			tools = append(tools, map[string]any{"name": tool.Function.Name, "description": tool.Function.Description, "input_schema": tool.Function.Parameters})
		}
		body["tools"] = tools
	}
	for key, value := range request.ExtraBody {
		body[key] = value
	}
	return body, nil
}

func (p *claudeProvider) do(ctx context.Context, selected credential, body any) (*http.Response, error) {
	request, err := newJSONRequest(ctx, p.config, "", p.config.BaseURL+claudeMessagesPath, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set(claudeAPIKeyHeader, selected.apiKey)
	version := p.config.APIVersion
	if version == "" {
		version = claudeDefaultAPIVersion
	}
	request.Header.Set(claudeVersionHeader, version)
	response, err := p.config.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPResponse(ProviderClaude, selected.hint, response); err != nil {
		response.Body.Close()
		return nil, err
	}
	return response, nil
}

type claudeStream struct {
	body        io.ReadCloser
	scanner     *bufio.Scanner
	closeOnce   sync.Once
	finished    bool
	model       string
	id          string
	toolIndexes map[int]int
	cancel      context.CancelFunc
}

func newClaudeStream(body io.ReadCloser, cancel context.CancelFunc) *claudeStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &claudeStream{body: body, scanner: scanner, toolIndexes: make(map[int]int), cancel: cancel}
}

func (s *claudeStream) Recv() (*chat.StreamChunk, error) {
	if s.finished {
		return nil, io.EOF
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 || bytes.HasPrefix(line, []byte("event:")) {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		var event struct {
			Type         string         `json:"type"`
			Index        int            `json:"index"`
			Message      claudeResponse `json:"message"`
			ContentBlock claudeContent  `json:"content_block"`
			Delta        struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				Thinking    string `json:"thinking"`
				PartialJSON string `json:"partial_json"`
				StopReason  string `json:"stop_reason"`
			} `json:"delta"`
			Usage struct {
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, err
		}
		chunk := &chat.StreamChunk{ID: s.id, Object: "chat.completion.chunk", Model: s.model}
		choice := chat.ChunkChoice{Index: 0}
		switch event.Type {
		case "message_start":
			s.id, s.model = event.Message.ID, event.Message.Model
			chunk.ID, chunk.Model = s.id, s.model
			usage := chat.Usage{PromptTokens: event.Message.Usage.InputTokens, TotalTokens: event.Message.Usage.InputTokens}
			chunk.Usage = &usage
			choice.Delta.Role = chat.RoleAssistant
		case "content_block_start":
			if event.ContentBlock.Type == "tool_use" {
				idx := event.Index
				choice.Delta.ToolCalls = []chat.ToolCall{{Index: &idx, ID: event.ContentBlock.ID, Type: "function", Function: chat.FunctionCall{Name: event.ContentBlock.Name}}}
			}
		case "content_block_delta":
			choice.Delta.Content = chat.TextContent(event.Delta.Text)
			choice.Delta.ReasoningContent = event.Delta.Thinking
			if event.Delta.PartialJSON != "" {
				idx := event.Index
				choice.Delta.ToolCalls = []chat.ToolCall{{Index: &idx, Function: chat.FunctionCall{Arguments: event.Delta.PartialJSON}}}
			}
		case "message_delta":
			choice.FinishReason = event.Delta.StopReason
			usage := chat.Usage{CompletionTokens: event.Usage.OutputTokens}
			chunk.Usage = &usage
		case "message_stop":
			s.finished = true
			return nil, io.EOF
		default:
			continue
		}
		chunk.Choices = []chat.ChunkChoice{choice}
		return chunk, nil
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	s.finished = true
	return nil, io.EOF
}

func (s *claudeStream) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.finished = true
		if s.cancel != nil {
			s.cancel()
		}
		err = s.body.Close()
	})
	return err
}

var _ chatProvider = (*claudeProvider)(nil)
var _ chat.Stream = (*claudeStream)(nil)
