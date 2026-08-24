// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package gemini

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.SupportsInputAudio = true
	spec.SupportsVideoURL = true
	spec.ApplyReasoning = func(model string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		body["reasoning_effort"] = reasoningEffort(model, effort)
	}
}

func reasoningEffort(model string, effort chat.ReasoningEffort) string {
	if !openai.IsKnownReasoningEffort(effort) {
		return string(effort)
	}
	if (effort == chat.ReasoningEffortXHigh || effort == chat.ReasoningEffortMax) && openai.HasModelPrefix(model, "gemini-2.5", "gemini-3") {
		return string(chat.ReasoningEffortHigh)
	}
	if openai.HasModelPrefix(model, "gemini-3.1-pro") && (effort == chat.ReasoningEffortNone || effort == chat.ReasoningEffortMinimal) {
		return string(chat.ReasoningEffortLow)
	}
	if effort == chat.ReasoningEffortNone && openai.HasModelPrefix(model, "gemini-3", "gemini-2.5-pro") {
		return string(chat.ReasoningEffortMinimal)
	}
	return string(effort)
}
