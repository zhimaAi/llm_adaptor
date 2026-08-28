// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package hunyuan

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("frequency_penalty", "max_tokens", "presence_penalty", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options")
	spec.SupportsVideoURL = true
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	if effort == chat.ReasoningEffortNone {
		shared.ApplyThinkingType(model, effort, body)
		return
	}
	switch effort {
	case chat.ReasoningEffortMinimal, chat.ReasoningEffortLow:
		body["reasoning_effort"] = string(chat.ReasoningEffortLow)
	case chat.ReasoningEffortMedium, chat.ReasoningEffortHigh:
		body["reasoning_effort"] = string(effort)
	case chat.ReasoningEffortXHigh, chat.ReasoningEffortMax:
		body["reasoning_effort"] = string(chat.ReasoningEffortHigh)
	default:
		body["reasoning_effort"] = string(effort)
	}
}
