// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package baidu

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

var enableThinkingPrefixes = []string{"qwen3-", "ernie-4.5-turbo-vl", "ernie-4.5-vl-28b-a3b", "ernie-5.0-thinking-preview"}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("frequency_penalty", "max_tokens", "max_completion_tokens", "presence_penalty", "response_format", "stop", "temperature", "tool_choice", "tools", "top_p", "user", "stream_options")
	spec.ApplyReasoning = func(model string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		if openai.HasModelPrefix(model, enableThinkingPrefixes...) {
			delete(body, "reasoning_effort")
			body["enable_thinking"] = shared.ReasoningEnabled(effort)
			return
		}
		shared.ApplyThinkingType(model, effort, body)
	}
}
