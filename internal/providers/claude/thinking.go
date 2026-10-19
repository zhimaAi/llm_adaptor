// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package claude

import (
	"encoding/json"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
)

const (
	thinkingEnabled      = "enabled"
	thinkingDisabled     = "disabled"
	thinkingAdaptive     = "adaptive"
	thinkingDisplay      = "summarized"
	legacyThinkingBudget = 1024
	legacyMinimumTokens  = 2048
)

type reasoningCapability struct {
	prefix           string
	adaptiveThinking bool
	effort           bool
	cannotDisable    bool
	disableType      string
	defaultSampling  bool
	supportsXHigh    bool
	supportsMax      bool
}

var reasoningCapabilities = []reasoningCapability{
	{prefix: "claude-opus-5-5", adaptiveThinking: true, effort: true, cannotDisable: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-opus-5.5", adaptiveThinking: true, effort: true, cannotDisable: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-sonnet-5-5", adaptiveThinking: true, effort: true, disableType: "between_tools", defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-sonnet-5.5", adaptiveThinking: true, effort: true, disableType: "between_tools", defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-haiku-5-5", adaptiveThinking: true, effort: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
	{prefix: "claude-haiku-5.5", adaptiveThinking: true, effort: true, defaultSampling: true, supportsXHigh: true, supportsMax: true},
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

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) error {
	if effort == "" {
		return nil
	}
	capability := capabilityForModel(model)
	if effort == chat.ReasoningEffortNone {
		if !capability.cannotDisable {
			mode := thinkingDisabled
			if capability.disableType != "" {
				mode = capability.disableType
				body["output_config"] = map[string]any{"effort": string(chat.ReasoningEffortLow)}
			}
			body["thinking"] = map[string]any{"type": mode}
		} else {
			body["thinking"] = map[string]any{"type": thinkingAdaptive, "display": thinkingDisplay}
			body["output_config"] = map[string]any{"effort": string(chat.ReasoningEffortLow)}
		}
		if capability.cannotDisable || capability.defaultSampling {
			delete(body, "temperature")
		}
		return nil
	}
	if capability.adaptiveThinking {
		body["thinking"] = map[string]any{"type": thinkingAdaptive, "display": thinkingDisplay}
	} else {
		body["thinking"] = map[string]any{"type": thinkingEnabled, "budget_tokens": legacyThinkingBudget}
		if maxTokens, ok := numberAsInt(body["max_tokens"]); ok && maxTokens <= legacyThinkingBudget {
			body["max_tokens"] = legacyMinimumTokens
		}
	}
	if capability.effort || !openai.IsKnownReasoningEffort(effort) {
		body["output_config"] = map[string]any{"effort": reasoningEffort(model, effort)}
	}
	delete(body, "temperature")
	return nil
}

func reasoningEffort(model string, effort chat.ReasoningEffort) string {
	if !openai.IsKnownReasoningEffort(effort) {
		return string(effort)
	}
	capability := capabilityForModel(model)
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

func capabilityForModel(model string) reasoningCapability {
	model = strings.ToLower(strings.TrimSpace(model))
	for _, capability := range reasoningCapabilities {
		if strings.HasPrefix(model, capability.prefix) {
			return capability
		}
	}
	return reasoningCapability{}
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
