// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package moonshot

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

// https://platform.kimi.com/docs/guide/use-thinking-models
var switchOnlyModels = []string{
	"kimi-k2.5", "kimi-k2.6", "kimi-k2.7-code", "kimi-k2.7-code-highspeed",
	"kimi-k2-thinking", "kimi-k2-thinking-turbo",
}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("frequency_penalty", "max_tokens", "n", "presence_penalty", "response_format", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options")
	spec.SupportsReasoningContent = true
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	if shared.IsModel(model, switchOnlyModels...) {
		shared.ApplyThinkingType(model, effort, body)
		return
	}
	shared.ApplyReasoningEffort(model, effort, body)
}
