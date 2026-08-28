// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package minimax

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("max_tokens", "max_completion_tokens", "temperature", "tools", "top_p", "stream_options")
	spec.ApplyReasoning = func(model string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		enabled := shared.ReasoningEnabled(effort)
		delete(body, "reasoning_effort")
		if openai.HasModelPrefix(model, "minimax-m3") {
			thinkingType := "disabled"
			if enabled {
				thinkingType = "adaptive"
			}
			body["thinking"] = map[string]any{"type": thinkingType}
		}
		body["reasoning_split"] = enabled
		openai.MoveMaxTokensToCompletionTokens(body)
	}
}
