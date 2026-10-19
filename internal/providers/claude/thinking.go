// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package claude

import (
	"encoding/json"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
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
	manualThinking   bool
	cannotDisable    bool
	disableType      string
	defaultSampling  bool
}

// Only these documented legacy IDs use budget-based thinking without effort.
// https://platform.claude.com/docs/en/build-with-claude/extended-thinking
var switchOnlyModels = []string{
	"claude-3-7-sonnet-latest", "claude-3-7-sonnet-20250219",
	"claude-opus-4", "claude-opus-4-0", "claude-opus-4-20250514",
	"claude-opus-4-1", "claude-opus-4-1-20250805",
	"claude-sonnet-4", "claude-sonnet-4-0", "claude-sonnet-4-20250514",
	"claude-sonnet-4-5", "claude-sonnet-4-5-20250929",
	"claude-haiku-4-5", "claude-haiku-4-5-20251001",
}

// These capabilities select auxiliary protocol settings, never effort values.
var reasoningCapabilities = []reasoningCapability{
	{prefix: "claude-opus-5-5", adaptiveThinking: true, cannotDisable: true, defaultSampling: true},
	{prefix: "claude-opus-5.5", adaptiveThinking: true, cannotDisable: true, defaultSampling: true},
	{prefix: "claude-sonnet-5-5", adaptiveThinking: true, disableType: "between_tools", defaultSampling: true},
	{prefix: "claude-sonnet-5.5", adaptiveThinking: true, disableType: "between_tools", defaultSampling: true},
	{prefix: "claude-haiku-5-5", adaptiveThinking: true, defaultSampling: true},
	{prefix: "claude-haiku-5.5", adaptiveThinking: true, defaultSampling: true},
	{prefix: "claude-fable-5", adaptiveThinking: true, cannotDisable: true, defaultSampling: true},
	{prefix: "claude-mythos-5", adaptiveThinking: true, cannotDisable: true, defaultSampling: true},
	{prefix: "claude-mythos-preview", adaptiveThinking: true, cannotDisable: true, defaultSampling: true},
	{prefix: "claude-opus-5", adaptiveThinking: true, defaultSampling: true},
	{prefix: "claude-sonnet-5", adaptiveThinking: true, defaultSampling: true},
	{prefix: "claude-opus-4-8", adaptiveThinking: true, defaultSampling: true},
	{prefix: "claude-opus-4-7", adaptiveThinking: true, defaultSampling: true},
	{prefix: "claude-opus-4-6", adaptiveThinking: true},
	{prefix: "claude-sonnet-4-6", adaptiveThinking: true},
	{prefix: "claude-opus-4-5", manualThinking: true},
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) error {
	if effort == "" {
		return nil
	}
	legacy := shared.IsModel(model, switchOnlyModels...)
	capability := capabilityForModel(model)
	if !legacy {
		body["output_config"] = map[string]any{"effort": string(effort)}
	}
	if effort == chat.ReasoningEffortNone {
		if capability.cannotDisable {
			body["thinking"] = map[string]any{"type": thinkingAdaptive, "display": thinkingDisplay}
		} else {
			mode := thinkingDisabled
			if capability.disableType != "" {
				mode = capability.disableType
			}
			body["thinking"] = map[string]any{"type": mode}
		}
		if capability.cannotDisable || capability.defaultSampling {
			delete(body, "temperature")
		}
		return nil
	}
	if capability.adaptiveThinking {
		body["thinking"] = map[string]any{"type": thinkingAdaptive, "display": thinkingDisplay}
	} else if legacy || capability.manualThinking {
		body["thinking"] = map[string]any{"type": thinkingEnabled, "budget_tokens": legacyThinkingBudget}
		if maxTokens, ok := numberAsInt(body["max_tokens"]); ok && maxTokens <= legacyThinkingBudget {
			body["max_tokens"] = legacyMinimumTokens
		}
	}
	delete(body, "temperature")
	return nil
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
