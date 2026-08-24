// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

type openAIChatWireRequest struct {
	Model               string                    `json:"model"`
	Messages            []openAIMessageWire       `json:"messages"`
	FrequencyPenalty    *float64                  `json:"frequency_penalty,omitempty"`
	MaxTokens           *int                      `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int                      `json:"max_completion_tokens,omitempty"`
	N                   *int                      `json:"n,omitempty"`
	ParallelToolCalls   *bool                     `json:"parallel_tool_calls,omitempty"`
	PresencePenalty     *float64                  `json:"presence_penalty,omitempty"`
	ReasoningEffort     string                    `json:"reasoning_effort,omitempty"`
	ResponseFormat      *openAIResponseFormatWire `json:"response_format,omitempty"`
	Seed                *int64                    `json:"seed,omitempty"`
	Stop                []string                  `json:"stop,omitempty"`
	Temperature         *float64                  `json:"temperature,omitempty"`
	ToolChoice          any                       `json:"tool_choice,omitempty"`
	Tools               []openAIToolWire          `json:"tools,omitempty"`
	TopP                *float64                  `json:"top_p,omitempty"`
	User                string                    `json:"user,omitempty"`
	Stream              bool                      `json:"stream"`
	StreamOptions       *chat.StreamOptions       `json:"stream_options,omitempty"`
}

type openAIMessageWire struct {
	Role       string               `json:"role"`
	Content    any                  `json:"content"`
	Name       string               `json:"name,omitempty"`
	ToolCalls  []openAIToolCallWire `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
}

type openAIContentPartWire struct {
	Type       string                `json:"type"`
	Text       string                `json:"text,omitempty"`
	ImageURL   *openAIImageURLWire   `json:"image_url,omitempty"`
	InputAudio *openAIInputAudioWire `json:"input_audio,omitempty"`
	VideoURL   *openAIVideoURLWire   `json:"video_url,omitempty"`
}

type openAIImageURLWire struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type openAIInputAudioWire struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type openAIVideoURLWire struct {
	URL string `json:"url"`
}

type openAIFunctionCallWire struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type openAIToolCallWire struct {
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Function openAIFunctionCallWire `json:"function"`
}

type openAIFunctionDefinitionWire struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type openAIToolWire struct {
	Type     string                       `json:"type"`
	Function openAIFunctionDefinitionWire `json:"function"`
}

type openAIResponseFormatWire struct {
	Type       string          `json:"type"`
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

func BuildChatRequest(spec Spec, request *chat.CreateRequest, stream bool, streamOptions *chat.StreamOptions) (map[string]any, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: chat request is nil", provider.ErrInvalidRequest)
	}
	messages, err := buildOpenAIMessages(spec, request.Messages)
	if err != nil {
		return nil, err
	}
	tools := make([]openAIToolWire, 0, len(request.Tools))
	for _, tool := range request.Tools {
		if tool.Type != "" && tool.Type != "function" {
			continue
		}
		toolType := tool.Type
		if toolType == "" {
			toolType = "function"
		}
		tools = append(tools, openAIToolWire{
			Type: toolType,
			Function: openAIFunctionDefinitionWire{
				Name: tool.Function.Name, Description: tool.Function.Description,
				Parameters: append(json.RawMessage(nil), tool.Function.Parameters...), Strict: tool.Function.Strict,
			},
		})
	}
	var responseFormat *openAIResponseFormatWire
	if request.ResponseFormat != nil {
		responseFormat = &openAIResponseFormatWire{
			Type:       request.ResponseFormat.Type,
			JSONSchema: append(json.RawMessage(nil), request.ResponseFormat.JSONSchema...),
		}
	}
	parallelToolCalls := request.ParallelToolCalls
	toolChoice := request.ToolChoice
	if len(tools) == 0 {
		parallelToolCalls = nil
		toolChoice = nil
	}
	wire := openAIChatWireRequest{
		Model: request.Model, Messages: messages, FrequencyPenalty: request.FrequencyPenalty,
		MaxTokens: request.MaxTokens, MaxCompletionTokens: request.MaxCompletionTokens,
		N: request.N, ParallelToolCalls: parallelToolCalls,
		PresencePenalty: request.PresencePenalty, ReasoningEffort: string(request.ReasoningEffort),
		ResponseFormat: responseFormat, Seed: request.Seed,
		Stop: request.Stop, Temperature: request.Temperature, ToolChoice: toolChoice,
		Tools: tools, TopP: request.TopP, User: request.User, Stream: stream,
	}
	if stream {
		wire.StreamOptions = DefaultChatStreamOptions(streamOptions)
	}
	raw, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	body := make(map[string]any)
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	filterFields(body, spec.ChatFields, AllChatFields)
	if spec.ApplyReasoning != nil {
		spec.ApplyReasoning(request.Model, request.ReasoningEffort, body)
	}
	if err := mergeChatExtraBody(body, request.ExtraBody); err != nil {
		return nil, err
	}
	return body, nil
}

func buildOpenAIMessages(spec Spec, messages []chat.Message) ([]openAIMessageWire, error) {
	result := make([]openAIMessageWire, len(messages))
	for index, message := range messages {
		content, err := buildOpenAIMessageContent(spec, message.Content)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid message at index %d: %v", provider.ErrInvalidRequest, index, err)
		}
		toolCalls := make([]openAIToolCallWire, len(message.ToolCalls))
		for toolIndex, toolCall := range message.ToolCalls {
			toolCalls[toolIndex] = openAIToolCallWire{
				ID: toolCall.ID, Type: toolCall.Type,
				Function: openAIFunctionCallWire{Name: toolCall.Function.Name, Arguments: toolCall.Function.Arguments},
			}
		}
		if content == nil && len(toolCalls) == 0 && message.ToolCallID == "" {
			return nil, fmt.Errorf("%w: message at index %d contains no supported content", provider.ErrInvalidRequest, index)
		}
		result[index] = openAIMessageWire{
			Role: string(message.Role), Content: content, Name: message.Name,
			ToolCalls: toolCalls, ToolCallID: message.ToolCallID,
		}
	}
	return result, nil
}

func buildOpenAIMessageContent(spec Spec, content chat.MessageContent) (any, error) {
	if content.Text != nil && content.Parts != nil {
		return nil, fmt.Errorf("content text and parts cannot both be set")
	}
	if content.Text != nil {
		return *content.Text, nil
	}
	if content.Parts == nil {
		return nil, nil
	}
	parts := make([]openAIContentPartWire, 0, len(content.Parts))
	for index, part := range content.Parts {
		wire := openAIContentPartWire{Type: string(part.Type), Text: part.Text}
		switch part.Type {
		case chat.ContentPartText:
			if part.Text == "" {
				return nil, fmt.Errorf("text part at index %d is empty", index)
			}
		case chat.ContentPartImageURL:
			if part.ImageURL == nil || strings.TrimSpace(part.ImageURL.URL) == "" {
				return nil, fmt.Errorf("image_url part at index %d is invalid", index)
			}
			wire.ImageURL = &openAIImageURLWire{URL: part.ImageURL.URL, Detail: part.ImageURL.Detail}
		case chat.ContentPartInputAudio:
			if !spec.SupportsInputAudio {
				continue
			}
			if part.InputAudio == nil || part.InputAudio.Data == "" || part.InputAudio.Format == "" {
				return nil, fmt.Errorf("input_audio part at index %d is invalid", index)
			}
			wire.InputAudio = &openAIInputAudioWire{Data: part.InputAudio.Data, Format: part.InputAudio.Format}
		case chat.ContentPartVideoURL:
			if !spec.SupportsVideoURL {
				continue
			}
			if part.VideoURL == nil || strings.TrimSpace(part.VideoURL.URL) == "" {
				return nil, fmt.Errorf("video_url part at index %d is invalid", index)
			}
			wire.VideoURL = &openAIVideoURLWire{URL: part.VideoURL.URL}
		default:
			continue
		}
		parts = append(parts, wire)
	}
	if len(parts) == 0 {
		return nil, nil
	}
	return parts, nil
}

func IsKnownReasoningEffort(effort chat.ReasoningEffort) bool {
	switch effort {
	case "", chat.ReasoningEffortNone, chat.ReasoningEffortMinimal, chat.ReasoningEffortLow,
		chat.ReasoningEffortMedium, chat.ReasoningEffortHigh, chat.ReasoningEffortXHigh,
		chat.ReasoningEffortMax:
		return true
	default:
		return false
	}
}

func DefaultChatStreamOptions(options *chat.StreamOptions) *chat.StreamOptions {
	result := chat.StreamOptions{}
	if options != nil {
		result = *options
	}
	if result.IncludeUsage == nil {
		includeUsage := true
		result.IncludeUsage = &includeUsage
	}
	return &result
}

func NormalizeChatStreamRequest(request *chat.StreamRequest) *chat.StreamRequest {
	if request == nil {
		return nil
	}
	result := *request
	result.StreamOptions = DefaultChatStreamOptions(request.StreamOptions)
	return &result
}

func mergeChatExtraBody(body map[string]any, extra map[string]any) error {
	for key, value := range extra {
		body[key] = value
	}
	return nil
}

func MoveMaxTokensToCompletionTokens(body map[string]any) {
	if value, ok := body["max_tokens"]; ok {
		body["max_completion_tokens"] = value
		delete(body, "max_tokens")
	}
}

func HasModelPrefix(model string, prefixes ...string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, prefix := range prefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}
