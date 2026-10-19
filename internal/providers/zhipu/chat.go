// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package zhipu

import (
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
		"n",
		"parallel_tool_calls",
		"presence_penalty",
		"seed",
	)
	spec.ChatFieldAliases = map[string]string{"user": "user_id"}
	spec.SupportsReasoningContent = true
	spec.ApplyReasoning = func(model string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		if openai.HasModelPrefix(model, "glm-5.3") {
			effort = shared.SupportedReasoningEffort(effort,
				chat.ReasoningEffortLow, chat.ReasoningEffortHigh, chat.ReasoningEffortMax)
			body["thinking"] = map[string]any{"type": "enabled"}
			body["reasoning_effort"] = string(effort)
			return
		}
		shared.ApplyThinkingType(model, effort, body)
		if openai.HasModelPrefix(model, "glm-5.2") {
			body["reasoning_effort"] = string(effort)
		}
	}
}
