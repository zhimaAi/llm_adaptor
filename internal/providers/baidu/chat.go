// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package baidu

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

var enableThinkingModels = []string{
	"qwen3-235b-a22b", "qwen3-32b", "qwen3-30b-a3b", "qwen3-14b", "qwen3-8b",
	"ernie-4.5-turbo-vl", "ernie-4.5-vl-28b-a3b", "ernie-5.0-thinking-preview",
}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("frequency_penalty", "max_tokens", "max_completion_tokens", "presence_penalty", "response_format", "stop", "temperature", "tool_choice", "tools", "top_p", "user", "stream_options")
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	if shared.IsModel(model, enableThinkingModels...) {
		shared.ApplyReasoningBoolean("enable_thinking")(model, effort, body)
		return
	}
	if shared.IsModel(model, "deepseek-v4-pro", "deepseek-v4-flash") {
		shared.ApplyReasoningEffort(model, effort, body)
		return
	}
	// Retain the existing auxiliary thinking setting for other endpoints.
	shared.ApplyThinkingType(model, effort, body)
	shared.ApplyReasoningEffort(model, effort, body)
}
