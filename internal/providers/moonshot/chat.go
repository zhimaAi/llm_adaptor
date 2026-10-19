// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package moonshot

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("frequency_penalty", "max_tokens", "n", "presence_penalty", "response_format", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options")
	spec.SupportsReasoningContent = true
	spec.ApplyReasoning = func(model string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		if openai.HasModelPrefix(model, "kimi-k3") {
			body["reasoning_effort"] = string(shared.SupportedReasoningEffort(effort,
				chat.ReasoningEffortLow, chat.ReasoningEffortHigh, chat.ReasoningEffortMax))
			return
		}
		shared.ApplyThinkingType(model, effort, body)
	}
}
