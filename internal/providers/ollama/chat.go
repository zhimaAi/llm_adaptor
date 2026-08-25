// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ollama

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.FieldsWithout(openai.AllChatFields, "max_completion_tokens", "parallel_tool_calls")
	spec.ApplyReasoning = func(_ string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		delete(body, "reasoning_effort")
		switch effort {
		case chat.ReasoningEffortNone:
			body["think"] = false
		case chat.ReasoningEffortMinimal, chat.ReasoningEffortLow:
			body["think"] = string(chat.ReasoningEffortLow)
		case chat.ReasoningEffortMedium:
			body["think"] = string(chat.ReasoningEffortMedium)
		case chat.ReasoningEffortHigh, chat.ReasoningEffortXHigh, chat.ReasoningEffortMax:
			body["think"] = string(chat.ReasoningEffortHigh)
		default:
			body["think"] = true
		}
	}
}
