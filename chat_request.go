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

var chatReservedRequestKeys = map[string]struct{}{
	"model": {}, "messages": {}, "frequency_penalty": {}, "logit_bias": {}, "logprobs": {},
	"top_logprobs": {}, "max_tokens": {}, "max_completion_tokens": {}, "modalities": {}, "n": {},
	"parallel_tool_calls": {}, "presence_penalty": {}, "reasoning_effort": {}, "response_format": {},
	"seed": {}, "service_tier": {}, "stop": {}, "store": {}, "temperature": {}, "tool_choice": {},
	"tools": {}, "top_p": {}, "user": {}, "metadata": {}, "stream": {}, "stream_options": {},
	"enable_thinking": {}, "think": {}, "thinking": {}, "reasoning": {}, "reasoning_split": {},
}

var baiduEnableThinkingModelPrefixes = []string{
	"qwen3-", "ernie-4.5-turbo-vl", "ernie-4.5-vl-28b-a3b", "ernie-5.0-thinking-preview",
}

var claudeAdaptiveThinkingModelPrefixes = []string{
	"claude-opus-4-6", "claude-sonnet-4-6", "claude-opus-4-7", "claude-opus-4-8",
	"claude-opus-5", "claude-sonnet-5", "claude-fable-5", "claude-mythos-5", "claude-mythos-preview",
}

var claudeCannotDisableThinkingModelPrefixes = []string{
	"claude-fable-5", "claude-mythos-5", "claude-mythos-preview",
}

var claudeDefaultSamplingModelPrefixes = []string{
	"claude-opus-4-7", "claude-opus-4-8", "claude-opus-5", "claude-sonnet-5",
	"claude-fable-5", "claude-mythos-5", "claude-mythos-preview",
}

type openAIChatWireRequest struct {
	Model               string               `json:"model"`
	Messages            []chat.Message       `json:"messages"`
	FrequencyPenalty    *float64             `json:"frequency_penalty,omitempty"`
	LogitBias           map[string]int       `json:"logit_bias,omitempty"`
	LogProbs            *bool                `json:"logprobs,omitempty"`
	TopLogProbs         *int                 `json:"top_logprobs,omitempty"`
	MaxTokens           *int                 `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int                 `json:"max_completion_tokens,omitempty"`
	Modalities          []string             `json:"modalities,omitempty"`
	N                   *int                 `json:"n,omitempty"`
	ParallelToolCalls   *bool                `json:"parallel_tool_calls,omitempty"`
	PresencePenalty     *float64             `json:"presence_penalty,omitempty"`
	ReasoningEffort     chat.ReasoningEffort `json:"reasoning_effort,omitempty"`
	ResponseFormat      *chat.ResponseFormat `json:"response_format,omitempty"`
	Seed                *int64               `json:"seed,omitempty"`
	ServiceTier         string               `json:"service_tier,omitempty"`
	Stop                []string             `json:"stop,omitempty"`
	Store               *bool                `json:"store,omitempty"`
	Temperature         *float64             `json:"temperature,omitempty"`
	ToolChoice          any                  `json:"tool_choice,omitempty"`
	Tools               []chat.Tool          `json:"tools,omitempty"`
	TopP                *float64             `json:"top_p,omitempty"`
	User                string               `json:"user,omitempty"`
	Metadata            map[string]any       `json:"metadata,omitempty"`
	Stream              bool                 `json:"stream"`
	StreamOptions       *chat.StreamOptions  `json:"stream_options,omitempty"`
}

func buildOpenAIChatRequest(provider Provider, request *chat.CreateRequest, stream bool, streamOptions *chat.StreamOptions) (map[string]any, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: chat request is nil", ErrInvalidRequest)
	}
	if err := validateReasoningEffort(request.ReasoningEffort); err != nil {
		return nil, err
	}
	wire := openAIChatWireRequest{
		Model: request.Model, Messages: request.Messages, FrequencyPenalty: request.FrequencyPenalty,
		LogitBias: request.LogitBias, LogProbs: request.LogProbs, TopLogProbs: request.TopLogProbs,
		MaxTokens: request.MaxTokens, MaxCompletionTokens: request.MaxCompletionTokens,
		Modalities: request.Modalities, N: request.N, ParallelToolCalls: request.ParallelToolCalls,
		PresencePenalty: request.PresencePenalty, ReasoningEffort: request.ReasoningEffort,
		ResponseFormat: request.ResponseFormat, Seed: request.Seed, ServiceTier: request.ServiceTier,
		Stop: request.Stop, Store: request.Store, Temperature: request.Temperature, ToolChoice: request.ToolChoice,
		Tools: request.Tools, TopP: request.TopP, User: request.User, Metadata: request.Metadata, Stream: stream,
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
	mergeChatExtraBody(body, request.ExtraBody)
	applyProviderReasoning(provider, request.Model, request.ReasoningEffort, body)
	return body, nil
}

func validateReasoningEffort(effort chat.ReasoningEffort) error {
	switch effort {
	case "", chat.ReasoningEffortNone, chat.ReasoningEffortMinimal, chat.ReasoningEffortLow,
		chat.ReasoningEffortMedium, chat.ReasoningEffortHigh, chat.ReasoningEffortXHigh:
		return nil
	default:
		return fmt.Errorf("%w: unsupported reasoning_effort %q", ErrInvalidRequest, effort)
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

func mergeChatExtraBody(body map[string]any, extra map[string]any) {
	for key, value := range extra {
		if _, reserved := chatReservedRequestKeys[key]; reserved {
			continue
		}
		body[key] = value
	}
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
	case ProviderOpenAI, ProviderOpenAIAgent, ProviderAzure:
		moveMaxTokensToCompletionTokens(body)
		delete(body, "temperature")
	case ProviderGemini:
		if !enabled && hasModelPrefix(model, "gemini-3") {
			body["reasoning_effort"] = string(chat.ReasoningEffortMinimal)
		}
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
		body["reasoning"] = map[string]any{"enabled": enabled}
	case ProviderMiniMax:
		delete(body, "reasoning_effort")
		if hasModelPrefix(model, "minimax-m3") {
			thinkingType := chatThinkingDisabled
			if enabled {
				thinkingType = chatThinkingAdaptive
			}
			body["thinking"] = map[string]any{"type": thinkingType}
			body["reasoning_split"] = enabled
		}
		moveMaxTokensToCompletionTokens(body)
	}
}

func applyClaudeReasoning(model string, effort chat.ReasoningEffort, body map[string]any) error {
	if err := validateReasoningEffort(effort); err != nil {
		return err
	}
	if effort == "" {
		return nil
	}
	cannotDisable := hasModelPrefix(model, claudeCannotDisableThinkingModelPrefixes...)
	if effort == chat.ReasoningEffortNone {
		if !cannotDisable {
			body["thinking"] = map[string]any{"type": chatThinkingDisabled}
		}
		if cannotDisable || hasModelPrefix(model, claudeDefaultSamplingModelPrefixes...) {
			delete(body, "temperature")
		}
		return nil
	}
	if hasModelPrefix(model, claudeAdaptiveThinkingModelPrefixes...) {
		body["thinking"] = map[string]any{"type": chatThinkingAdaptive, "display": chatThinkingDisplay}
	} else {
		body["thinking"] = map[string]any{"type": chatThinkingEnabled, "budget_tokens": claudeLegacyThinkingBudget}
		if maxTokens, ok := numberAsInt(body["max_tokens"]); ok && maxTokens <= claudeLegacyThinkingBudget {
			body["max_tokens"] = claudeLegacyMinimumTokens
		}
	}
	delete(body, "temperature")
	return nil
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
