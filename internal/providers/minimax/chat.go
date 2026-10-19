// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package minimax

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

// https://platform.minimax.cn/docs/api-reference/text-chat-openai
var legacyModels = []string{
	"MiniMax-M3", "MiniMax-M2.7", "MiniMax-M2.7-highspeed", "MiniMax-M2.5", "MiniMax-M2.5-highspeed",
	"MiniMax-M2.1", "MiniMax-M2.1-highspeed", "MiniMax-M2",
}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("max_tokens", "max_completion_tokens", "temperature", "tools", "top_p", "stream_options")
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	enabled := shared.ReasoningEnabled(effort)
	if shared.IsModel(model, "MiniMax-M3.1-Flash", "MiniMax-M3.1-Flash-Preview") {
		body["thinking"] = map[string]any{"type": "adaptive"}
		shared.ApplyReasoningEffort(model, effort, body)
		body["reasoning_split"] = true
		openai.MoveMaxTokensToCompletionTokens(body)
		return
	}
	if shared.IsModel(model, "MiniMax-M3") {
		shared.ApplyAdaptiveThinkingType(model, effort, body)
	}
	body["reasoning_split"] = enabled
	if shared.IsModel(model, legacyModels...) {
		delete(body, "reasoning_effort")
	} else {
		shared.ApplyReasoningEffort(model, effort, body)
	}
	openai.MoveMaxTokensToCompletionTokens(body)
}
