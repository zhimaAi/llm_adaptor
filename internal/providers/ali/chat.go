// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ali

import (
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.FieldsWithout(
		openai.AllChatFields,
		"frequency_penalty",
		"max_completion_tokens",
		"parallel_tool_calls",
		"response_format",
		"tool_choice",
		"user",
	)
	spec.SupportsInputAudio = true
	spec.SupportsVideoURL = true
	spec.SupportsReasoningContent = true
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	fullModel := strings.ToLower(strings.TrimSpace(model))
	model = fullModel[strings.LastIndex(fullModel, "/")+1:]
	if openai.HasModelPrefix(model, "minimax-m3") {
		delete(body, "reasoning_effort")
		mode := "adaptive"
		if effort == chat.ReasoningEffortNone {
			mode = "disabled"
		}
		body["thinking"] = map[string]any{"type": mode}
		return
	}
	shared.ApplyReasoningBoolean("enable_thinking")(model, effort, body)
	switch {
	case fullModel == "kimi/kimi-k3":
		body["enable_thinking"] = true
		body["reasoning_effort"] = string(shared.SupportedReasoningEffort(effort, chat.ReasoningEffortMax))
	case openai.HasModelPrefix(model, "glm-5.3", "kimi-k3"):
		if strings.HasPrefix(model, "glm-") {
			body["enable_thinking"] = true
			effort = shared.SupportedReasoningEffort(effort, chat.ReasoningEffortLow, chat.ReasoningEffortHigh, chat.ReasoningEffortMax)
		} else if effort != chat.ReasoningEffortNone {
			effort = shared.SupportedReasoningEffort(effort, chat.ReasoningEffortLow, chat.ReasoningEffortHigh, chat.ReasoningEffortMax)
		} else {
			return
		}
		body["reasoning_effort"] = string(effort)
	case openai.HasModelPrefix(model, "qwen3.8-"):
		// All seven levels, including compatibility aliases, are accepted.
		body["reasoning_effort"] = string(effort)
	case openai.HasModelPrefix(model, "deepseek-v4", "glm-5"):
		if effort == chat.ReasoningEffortNone {
			return // Use enable_thinking=false rather than an unsupported level.
		}
		if effort == chat.ReasoningEffortMinimal && !openai.HasModelPrefix(model, "deepseek-v4.1-") {
			// These older hosted models do not document minimal as an alias.
			effort = chat.ReasoningEffortHigh
			if openai.HasModelPrefix(model, "deepseek-v4-flash-0731", "deepseek-v4-pro-0813") {
				effort = chat.ReasoningEffortLow
			}
		}
		body["reasoning_effort"] = string(effort)
	}
}
