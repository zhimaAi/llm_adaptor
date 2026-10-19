// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package doubao

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.SupportsInputAudio = true
	spec.SupportsVideoURL = true
	spec.SupportsReasoningContent = true
	spec.ApplyReasoning = func(model string, effort chat.ReasoningEffort, body map[string]any) {
		shared.ApplyThinkingType(model, effort, body)
		if effort == "" {
			return
		}
		if openai.HasModelPrefix(model, "glm-5-3") {
			body["thinking"] = map[string]any{"type": "enabled"}
		}
		// ep-* inference endpoints are opaque: let Ark resolve their capabilities.
		if openai.HasModelPrefix(model, "doubao-seed-2-", "doubao-seed-evolving", "doubao-seed-character-",
			"doubao-seed-1-8", "doubao-seed-1-6-251015", "glm-5-", "deepseek-v4", "ep-") {
			body["reasoning_effort"] = string(effort)
		}
	}
}
