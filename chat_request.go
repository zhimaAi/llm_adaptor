// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
)

const (
	chatThinkingEnabled        = "enabled"
	chatThinkingDisabled       = "disabled"
	chatThinkingAdaptive       = "adaptive"
	chatThinkingDisplay        = "summarized"
	claudeLegacyThinkingBudget = 1024
	claudeLegacyMinimumTokens  = 2048
)

var baiduEnableThinkingModelPrefixes = []string{
	"qwen3-", "ernie-4.5-turbo-vl", "ernie-4.5-vl-28b-a3b", "ernie-5.0-thinking-preview",
}

type claudeReasoningCapability struct {
	prefix           string
	adaptiveThinking bool
	effort           bool
	cannotDisable    bool
	defaultSampling  bool
	supportsXHigh    bool
	supportsMax      bool
}

var claudeReasoningCapabilities = []claudeReasoningCapability{
	{prefix: "claude-fable-5", adaptiveThinking: true, effort: true, cannotDisable: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-mythos-5", adaptiveThinking: true, effort: true, cannotDisable: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-mythos-preview", adaptiveThinking: true, effort: true, cannotDisable: true, defaultSampling: true, supportsMax: true},
	{prefix: "claude-opus-5", adaptiveThinking: true, effort: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-sonnet-5", adaptiveThinking: true, effort: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-opus-4-8", adaptiveThinking: true, effort: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-opus-4-7", adaptiveThinking: true, effort: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-opus-4-6", adaptiveThinking: true, effort: true, supportsMax: true},
	{prefix: "claude-sonnet-4-6", adaptiveThinking: true, effort: true, supportsMax: true},
	{prefix: "claude-opus-4-5", effort: true},
}

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

func buildOpenAIChatRequest(provider Provider, request *chat.CreateRequest, stream bool, streamOptions *chat.StreamOptions) (map[string]any, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: chat request is nil", ErrInvalidRequest)
	}
	messages, err := buildOpenAIMessages(provider, request.Messages)
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
		wire.StreamOptions = defaultChatStreamOptions(streamOptions)
	}
	raw, err := json.Marshal(wire)
	if err != nil {
		return nil, err
	}
	body := make(map[string]any)
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	filterRequestFields(body, chatProviderParameterFields[provider], allOpenAIChatFields)
	applyProviderReasoning(provider, request.Model, request.ReasoningEffort, body)
	if err := mergeChatExtraBody(body, request.ExtraBody); err != nil {
		return nil, err
	}
	return body, nil
}

func buildOpenAIMessages(provider Provider, messages []chat.Message) ([]openAIMessageWire, error) {
	result := make([]openAIMessageWire, len(messages))
	for index, message := range messages {
		content, err := buildOpenAIMessageContent(provider, message.Content)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid message at index %d: %v", ErrInvalidRequest, index, err)
		}
		toolCalls := make([]openAIToolCallWire, len(message.ToolCalls))
		for toolIndex, toolCall := range message.ToolCalls {
			toolCalls[toolIndex] = openAIToolCallWire{
				ID: toolCall.ID, Type: toolCall.Type,
				Function: openAIFunctionCallWire{Name: toolCall.Function.Name, Arguments: toolCall.Function.Arguments},
			}
		}
		if content == nil && len(toolCalls) == 0 && message.ToolCallID == "" {
			return nil, fmt.Errorf("%w: message at index %d contains no supported content", ErrInvalidRequest, index)
		}
		result[index] = openAIMessageWire{
			Role: string(message.Role), Content: content, Name: message.Name,
			ToolCalls: toolCalls, ToolCallID: message.ToolCallID,
		}
	}
	return result, nil
}

func buildOpenAIMessageContent(provider Provider, content chat.MessageContent) (any, error) {
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
			if !providerSupportsInputAudio(provider) {
				continue
			}
			if part.InputAudio == nil || part.InputAudio.Data == "" || part.InputAudio.Format == "" {
				return nil, fmt.Errorf("input_audio part at index %d is invalid", index)
			}
			wire.InputAudio = &openAIInputAudioWire{Data: part.InputAudio.Data, Format: part.InputAudio.Format}
		case chat.ContentPartVideoURL:
			if !providerSupportsVideoURL(provider) {
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

func providerSupportsInputAudio(provider Provider) bool {
	switch provider {
	case ProviderOpenAI, ProviderAzure, ProviderOpenCompatible,
		ProviderAli, ProviderDoubao, ProviderGemini, ProviderSiliconFlow:
		return true
	default:
		return false
	}
}

func providerSupportsVideoURL(provider Provider) bool {
	switch provider {
	case ProviderOpenCompatible, ProviderAli, ProviderDoubao, ProviderGemini, ProviderHunyuan:
		return true
	default:
		return false
	}
}

func isKnownReasoningEffort(effort chat.ReasoningEffort) bool {
	switch effort {
	case "", chat.ReasoningEffortNone, chat.ReasoningEffortMinimal, chat.ReasoningEffortLow,
		chat.ReasoningEffortMedium, chat.ReasoningEffortHigh, chat.ReasoningEffortXHigh,
		chat.ReasoningEffortMax:
		return true
	default:
		return false
	}
}

func defaultChatStreamOptions(options *chat.StreamOptions) *chat.StreamOptions {
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

func normalizeChatStreamRequest(request *chat.StreamRequest) *chat.StreamRequest {
	if request == nil {
		return nil
	}
	result := *request
	result.StreamOptions = defaultChatStreamOptions(request.StreamOptions)
	return &result
}

func mergeChatExtraBody(body map[string]any, extra map[string]any) error {
	for key, value := range extra {
		body[key] = value
	}
	return nil
}

func applyProviderReasoning(provider Provider, model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	enabled := effort != chat.ReasoningEffortNone
	enabledType := chatThinkingDisabled
	if enabled {
		enabledType = chatThinkingEnabled
	}
	switch provider {
	case ProviderOpenAI, ProviderAzure:
		moveMaxTokensToCompletionTokens(body)
		delete(body, "temperature")
	case ProviderGemini:
		body["reasoning_effort"] = geminiReasoningEffort(model, effort)
	case ProviderAli, ProviderSiliconFlow, ProviderXinference, ProviderHunyuan:
		delete(body, "reasoning_effort")
		body["enable_thinking"] = enabled
	case ProviderOllama:
		delete(body, "reasoning_effort")
		body["think"] = enabled
	case ProviderBaidu:
		delete(body, "reasoning_effort")
		if hasModelPrefix(model, baiduEnableThinkingModelPrefixes...) {
			body["enable_thinking"] = enabled
		} else {
			body["thinking"] = map[string]any{"type": enabledType}
		}
	case ProviderDeepSeek, ProviderMoonshot, ProviderZhipu, ProviderDoubao, ProviderCohere, ProviderSpark:
		delete(body, "reasoning_effort")
		body["thinking"] = map[string]any{"type": enabledType}
	case ProviderOpenRouter:
		delete(body, "reasoning_effort")
		body["reasoning"] = map[string]any{"effort": string(effort)}
	case ProviderMiniMax:
		delete(body, "reasoning_effort")
		if hasModelPrefix(model, "minimax-m3") {
			thinkingType := chatThinkingDisabled
			if enabled {
				thinkingType = chatThinkingAdaptive
			}
			body["thinking"] = map[string]any{"type": thinkingType}
		}
		body["reasoning_split"] = enabled
		moveMaxTokensToCompletionTokens(body)
	}
}

func applyClaudeReasoning(model string, effort chat.ReasoningEffort, body map[string]any) error {
	if effort == "" {
		return nil
	}
	capability := claudeReasoningCapabilityForModel(model)
	if effort == chat.ReasoningEffortNone {
		if !capability.cannotDisable {
			body["thinking"] = map[string]any{"type": chatThinkingDisabled}
		} else {
			body["thinking"] = map[string]any{"type": chatThinkingAdaptive, "display": chatThinkingDisplay}
			body["output_config"] = map[string]any{"effort": string(chat.ReasoningEffortLow)}
		}
		if capability.cannotDisable || capability.defaultSampling {
			delete(body, "temperature")
		}
		return nil
	}
	if capability.adaptiveThinking {
		body["thinking"] = map[string]any{"type": chatThinkingAdaptive, "display": chatThinkingDisplay}
	} else {
		body["thinking"] = map[string]any{"type": chatThinkingEnabled, "budget_tokens": claudeLegacyThinkingBudget}
		if maxTokens, ok := numberAsInt(body["max_tokens"]); ok && maxTokens <= claudeLegacyThinkingBudget {
			body["max_tokens"] = claudeLegacyMinimumTokens
		}
	}
	if capability.effort || !isKnownReasoningEffort(effort) {
		body["output_config"] = map[string]any{"effort": claudeReasoningEffort(model, effort)}
	}
	delete(body, "temperature")
	return nil
}

func geminiReasoningEffort(model string, effort chat.ReasoningEffort) string {
	if !isKnownReasoningEffort(effort) {
		return string(effort)
	}
	if (effort == chat.ReasoningEffortXHigh || effort == chat.ReasoningEffortMax) &&
		hasModelPrefix(model, "gemini-2.5", "gemini-3") {
		return string(chat.ReasoningEffortHigh)
	}
	if hasModelPrefix(model, "gemini-3.1-pro") && (effort == chat.ReasoningEffortNone || effort == chat.ReasoningEffortMinimal) {
		return string(chat.ReasoningEffortLow)
	}
	if effort == chat.ReasoningEffortNone && hasModelPrefix(model,
		"gemini-3", "gemini-2.5-pro") {
		return string(chat.ReasoningEffortMinimal)
	}
	return string(effort)
}

func claudeReasoningEffort(model string, effort chat.ReasoningEffort) string {
	if !isKnownReasoningEffort(effort) {
		return string(effort)
	}
	capability := claudeReasoningCapabilityForModel(model)
	switch effort {
	case chat.ReasoningEffortMinimal:
		return string(chat.ReasoningEffortLow)
	case chat.ReasoningEffortXHigh:
		if !capability.supportsXHigh {
			return string(chat.ReasoningEffortHigh)
		}
	case chat.ReasoningEffortMax:
		if !capability.supportsMax {
			if capability.supportsXHigh {
				return string(chat.ReasoningEffortXHigh)
			}
			return string(chat.ReasoningEffortHigh)
		}
	}
	return string(effort)
}

func claudeReasoningCapabilityForModel(model string) claudeReasoningCapability {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, capability := range claudeReasoningCapabilities {
		if strings.HasPrefix(model, capability.prefix) {
			return capability
		}
	}
	return claudeReasoningCapability{}
}

func moveMaxTokensToCompletionTokens(body map[string]any) {
	if value, ok := body["max_tokens"]; ok {
		body["max_completion_tokens"] = value
		delete(body, "max_tokens")
	}
}

func numberAsInt(value any) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, true
	case float64:
		return int(number), true
	case json.Number:
		parsed, err := number.Int64()
		return int(parsed), err == nil
	default:
		return 0, false
	}
}

func hasModelPrefix(model string, prefixes ...string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, prefix := range prefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}
