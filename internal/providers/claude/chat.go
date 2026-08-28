// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

const (
	streamInitialBuffer     = 64 * 1024
	streamMaximumBuffer     = 16 * 1024 * 1024
	claudeMessagesPath      = "/messages"
	claudeAPIKeyHeader      = "x-api-key"
	claudeVersionHeader     = "anthropic-version"
	claudeDefaultAPIVersion = "2023-06-01"
	claudeDefaultMaxTokens  = 4096
	claudeRoleUser          = "user"
	claudeRoleAssistant     = "assistant"
	claudeContentText       = "text"
	claudeContentImage      = "image"
	claudeContentToolUse    = "tool_use"
	claudeContentToolResult = "tool_result"
	claudeSourceBase64      = "base64"
	claudeSourceURL         = "url"
)

type Provider struct{ config provider.Config }

func (p *Provider) Info() provider.Info {
	return provider.Info{ID: provider.IDClaude, DefaultBaseURL: "https://api.anthropic.com/v1", Capabilities: []provider.Capability{provider.CapabilityChat}}
}

type claudeContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	Thinking string          `json:"thinking,omitempty"`
	ID       string          `json:"id,omitempty"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
}

type claudeRequestMessage struct {
	Role    string                 `json:"role"`
	Content []claudeRequestContent `json:"content"`
}

type claudeRequestContent struct {
	Type      string             `json:"type"`
	Text      string             `json:"text,omitempty"`
	ID        string             `json:"id,omitempty"`
	Name      string             `json:"name,omitempty"`
	Input     any                `json:"input,omitempty"`
	ToolUseID string             `json:"tool_use_id,omitempty"`
	Content   string             `json:"content,omitempty"`
	Source    *claudeImageSource `json:"source,omitempty"`
}

type claudeImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
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

func (p *Provider) CreateChat(ctx context.Context, selected provider.Credential, request *chat.CreateRequest) (*chat.CreateResponse, error) {
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
	if err := transport.DecodeJSONResponse(provider.IDClaude, selected.Hint, raw, &source); err != nil {
		return nil, err
	}
	result := &chat.CreateResponse{ID: source.ID, Object: "chat.completion", Model: source.Model}
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

func (p *Provider) StreamChat(ctx context.Context, selected provider.Credential, request *chat.StreamRequest) (chat.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: chat request is nil", provider.ErrInvalidRequest)
	}
	body, err := buildClaudeRequest(&request.CreateRequest, true)
	if err != nil {
		return nil, err
	}
	streamContext, cancel := context.WithCancel(shared.NormalizeContext(ctx))
	response, err := p.do(streamContext, selected, body)
	if err != nil {
		cancel()
		return nil, err
	}
	return newClaudeStream(streamContext, response.Body, cancel, selected.Hint), nil
}

func buildClaudeRequest(request *chat.CreateRequest, stream bool) (map[string]any, error) {
	if request == nil || request.Model == "" || len(request.Messages) == 0 {
		return nil, fmt.Errorf("%w: model and messages are required", provider.ErrInvalidRequest)
	}
	messages := make([]claudeRequestMessage, 0, len(request.Messages))
	var system strings.Builder
	for _, message := range request.Messages {
		if message.Role == chat.RoleSystem || message.Role == chat.RoleDeveloper {
			if message.Content.Text != nil && message.Content.Parts != nil {
				return nil, fmt.Errorf("%w: Claude system content text and parts cannot both be set", provider.ErrInvalidRequest)
			}
			if message.Content.Text != nil {
				system.WriteString(*message.Content.Text)
			} else {
				for _, part := range message.Content.Parts {
					if part.Type == chat.ContentPartText {
						system.WriteString(part.Text)
					}
				}
			}
			continue
		}
		if message.Role != chat.RoleUser && message.Role != chat.RoleAssistant && message.Role != chat.RoleTool {
			continue
		}
		converted, err := convertClaudeMessage(message)
		if err != nil {
			return nil, err
		}
		messages = append(messages, converted)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("%w: Claude request contains no supported messages", provider.ErrInvalidRequest)
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
	if len(request.Stop) > 0 {
		body["stop_sequences"] = append([]string(nil), request.Stop...)
	}
	if request.User != "" {
		body["metadata"] = map[string]any{"user_id": request.User}
	}
	if len(request.Tools) > 0 {
		tools := make([]map[string]any, 0, len(request.Tools))
		for _, tool := range request.Tools {
			if tool.Type != "" && tool.Type != "function" {
				continue
			}
			tools = append(tools, map[string]any{
				"name": tool.Function.Name, "description": tool.Function.Description,
				"input_schema": append(json.RawMessage(nil), tool.Function.Parameters...),
			})
		}
		if len(tools) > 0 {
			body["tools"] = tools
		}
	}
	if _, hasTools := body["tools"]; hasTools {
		toolChoice, err := buildClaudeToolChoice(request.ToolChoice, request.ParallelToolCalls)
		if err != nil {
			return nil, err
		}
		if toolChoice != nil {
			body["tool_choice"] = toolChoice
		}
	}
	if err := applyReasoning(request.Model, request.ReasoningEffort, body); err != nil {
		return nil, err
	}
	return shared.MergeExtraBody(body, request.ExtraBody)
}

func buildClaudeToolChoice(value any, parallel *bool) (map[string]any, error) {
	result := map[string]any{}
	if value != nil {
		switch choice := value.(type) {
		case string:
			switch choice {
			case "auto":
				result["type"] = "auto"
			case "required":
				result["type"] = "any"
			case "none":
				result["type"] = "none"
			default:
				// Ignore tool choices Claude cannot represent.
			}
		default:
			raw, err := json.Marshal(value)
			if err != nil {
				break
			}
			var openAIChoice struct {
				Type     string `json:"type"`
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			}
			if err := json.Unmarshal(raw, &openAIChoice); err != nil || openAIChoice.Type != "function" || openAIChoice.Function.Name == "" {
				break
			}
			result["type"] = "tool"
			result["name"] = openAIChoice.Function.Name
		}
	}
	if parallel != nil && result["type"] != "none" {
		if len(result) == 0 {
			result["type"] = "auto"
		}
		result["disable_parallel_tool_use"] = !*parallel
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

func convertClaudeMessage(message chat.Message) (claudeRequestMessage, error) {
	converted := claudeRequestMessage{Role: string(message.Role)}
	if message.Role == chat.RoleTool {
		if strings.TrimSpace(message.ToolCallID) == "" {
			return claudeRequestMessage{}, fmt.Errorf("%w: Claude tool result requires tool_call_id", provider.ErrInvalidRequest)
		}
		converted.Role = claudeRoleUser
		if message.Content.Text != nil && message.Content.Parts != nil {
			return claudeRequestMessage{}, fmt.Errorf("%w: Claude tool result text and parts cannot both be set", provider.ErrInvalidRequest)
		}
		var content strings.Builder
		if message.Content.Text != nil {
			content.WriteString(*message.Content.Text)
		} else {
			for _, part := range message.Content.Parts {
				if part.Type == chat.ContentPartText {
					content.WriteString(part.Text)
				}
			}
		}
		converted.Content = []claudeRequestContent{{Type: claudeContentToolResult, ToolUseID: message.ToolCallID, Content: content.String()}}
		return converted, nil
	}
	if converted.Role != claudeRoleUser && converted.Role != claudeRoleAssistant {
		return claudeRequestMessage{}, fmt.Errorf("%w: Claude does not support message role %q", provider.ErrInvalidRequest, message.Role)
	}
	if message.Content.Text != nil && (*message.Content.Text != "" || len(message.ToolCalls) == 0) {
		converted.Content = append(converted.Content, claudeRequestContent{Type: claudeContentText, Text: *message.Content.Text})
	} else {
		for _, part := range message.Content.Parts {
			switch part.Type {
			case chat.ContentPartText:
				converted.Content = append(converted.Content, claudeRequestContent{Type: claudeContentText, Text: part.Text})
			case chat.ContentPartImageURL:
				if part.ImageURL == nil || strings.TrimSpace(part.ImageURL.URL) == "" {
					return claudeRequestMessage{}, fmt.Errorf("%w: Claude image_url content is empty", provider.ErrInvalidRequest)
				}
				source, err := claudeImageSourceFromURL(part.ImageURL.URL)
				if err != nil {
					return claudeRequestMessage{}, err
				}
				converted.Content = append(converted.Content, claudeRequestContent{Type: claudeContentImage, Source: source})
			default:
				continue
			}
		}
	}
	for _, call := range message.ToolCalls {
		if call.Type != "" && call.Type != "function" {
			continue
		}
		if converted.Role != claudeRoleAssistant {
			return claudeRequestMessage{}, fmt.Errorf("%w: Claude tool_use must be in an assistant message", provider.ErrInvalidRequest)
		}
		if strings.TrimSpace(call.ID) == "" || strings.TrimSpace(call.Function.Name) == "" {
			return claudeRequestMessage{}, fmt.Errorf("%w: Claude tool_use requires id and function name", provider.ErrInvalidRequest)
		}
		input := make(map[string]any)
		if strings.TrimSpace(call.Function.Arguments) != "" {
			if err := json.Unmarshal([]byte(call.Function.Arguments), &input); err != nil {
				return claudeRequestMessage{}, fmt.Errorf("%w: invalid Claude tool arguments: %v", provider.ErrInvalidRequest, err)
			}
			if input == nil {
				input = make(map[string]any)
			}
		}
		converted.Content = append(converted.Content, claudeRequestContent{
			Type: claudeContentToolUse, ID: call.ID, Name: call.Function.Name, Input: input,
		})
	}
	if len(converted.Content) == 0 {
		return claudeRequestMessage{}, fmt.Errorf("%w: Claude message contains no supported content", provider.ErrInvalidRequest)
	}
	return converted, nil
}

func claudeImageSourceFromURL(value string) (*claudeImageSource, error) {
	if strings.HasPrefix(value, "data:") {
		mediaType, data, err := shared.ParseImageDataURL(value)
		if err != nil {
			return nil, err
		}
		return &claudeImageSource{Type: claudeSourceBase64, MediaType: mediaType, Data: data}, nil
	}
	return &claudeImageSource{Type: claudeSourceURL, URL: value}, nil
}

func (p *Provider) do(ctx context.Context, selected provider.Credential, body any) (*http.Response, error) {
	request, err := transport.NewJSONRequest(ctx, p.config, "", p.config.BaseURL+claudeMessagesPath, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set(claudeAPIKeyHeader, selected.APIKey)
	version := p.config.APIVersion
	if version == "" {
		version = claudeDefaultAPIVersion
	}
	request.Header.Set(claudeVersionHeader, version)
	response, err := p.config.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	if err := transport.CheckHTTPResponse(provider.IDClaude, selected.Hint, response); err != nil {
		response.Body.Close()
		return nil, err
	}
	return response, nil
}

type claudeStream struct {
	ctx         context.Context
	scanner     *bufio.Scanner
	terminal    *transport.StreamTerminal
	model       string
	id          string
	toolIndexes map[int]int
	hint        string
}

func newClaudeStream(ctx context.Context, body io.ReadCloser, cancel context.CancelFunc, hint string) *claudeStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &claudeStream{ctx: ctx, scanner: scanner, terminal: transport.NewStreamTerminal(cancel, body.Close), toolIndexes: make(map[int]int), hint: hint}
}

func (s *claudeStream) Recv() (*chat.StreamChunk, error) {
	if s.terminal.IsDone() {
		return nil, io.EOF
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 || bytes.HasPrefix(line, []byte("event:")) {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if err := transport.DecodeStreamAPIError(provider.IDClaude, s.hint, line); err != nil {
			return nil, s.terminal.Fail(s.ctx, err)
		}
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
		if err := transport.DecodeJSONResponse(provider.IDClaude, s.hint, line, &event); err != nil {
			return nil, s.terminal.Fail(s.ctx, err)
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
				idx, exists := s.toolIndexes[event.Index]
				if !exists {
					idx = len(s.toolIndexes)
					s.toolIndexes[event.Index] = idx
				}
				choice.Delta.ToolCalls = []chat.ToolCall{{Index: &idx, ID: event.ContentBlock.ID, Type: "function", Function: chat.FunctionCall{Name: event.ContentBlock.Name}}}
			}
		case "content_block_delta":
			choice.Delta.Content = chat.TextContent(event.Delta.Text)
			choice.Delta.ReasoningContent = event.Delta.Thinking
			if event.Delta.PartialJSON != "" {
				idx, exists := s.toolIndexes[event.Index]
				if !exists {
					err := fmt.Errorf("%w: Claude tool delta references unknown content index %d", provider.ErrInvalidRequest, event.Index)
					return nil, s.terminal.Fail(s.ctx, err)
				}
				choice.Delta.ToolCalls = []chat.ToolCall{{Index: &idx, Function: chat.FunctionCall{Arguments: event.Delta.PartialJSON}}}
			}
		case "message_delta":
			choice.FinishReason = event.Delta.StopReason
			usage := chat.Usage{CompletionTokens: event.Usage.OutputTokens}
			chunk.Usage = &usage
		case "message_stop":
			s.terminal.Finish()
			return nil, io.EOF
		default:
			continue
		}
		chunk.Choices = []chat.ChunkChoice{choice}
		return chunk, nil
	}
	if err := s.scanner.Err(); err != nil {
		return nil, s.terminal.Fail(s.ctx, err)
	}
	if s.terminal.IsDone() {
		return nil, io.EOF
	}
	if s.ctx != nil && s.ctx.Err() != nil {
		return nil, s.terminal.Fail(s.ctx, s.ctx.Err())
	}
	s.terminal.Finish()
	return nil, io.EOF
}

func (s *claudeStream) Close() error {
	return s.terminal.Close()
}

var _ provider.Chat = (*Provider)(nil)
var _ chat.Stream = (*claudeStream)(nil)
