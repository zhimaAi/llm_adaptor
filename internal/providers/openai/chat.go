// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	openprotocol "github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openprotocol.Spec, _ provider.Config) {
	spec.SupportsInputAudio = true
	spec.ApplyReasoning = func(model string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		switch {
		case openprotocol.HasModelPrefix(model, "gpt-6-astra", "gpt-6.1-sol"):
			effort = shared.SupportedReasoningEffort(effort, chat.ReasoningEffortLow,
				chat.ReasoningEffortMedium, chat.ReasoningEffortHigh, chat.ReasoningEffortXHigh, chat.ReasoningEffortMax)
		case openprotocol.HasModelPrefix(model, "gpt-6-sol", "gpt-6-luna"):
			effort = shared.SupportedReasoningEffort(effort, chat.ReasoningEffortNone, chat.ReasoningEffortLow,
				chat.ReasoningEffortMedium, chat.ReasoningEffortHigh, chat.ReasoningEffortXHigh, chat.ReasoningEffortMax)
		case openprotocol.HasModelPrefix(model, "o1", "o3", "o4-mini"):
			effort = shared.SupportedReasoningEffort(effort,
				chat.ReasoningEffortLow, chat.ReasoningEffortMedium, chat.ReasoningEffortHigh)
		}
		body["reasoning_effort"] = string(effort)
		openprotocol.MoveMaxTokensToCompletionTokens(body)
		delete(body, "temperature")
	}
}
