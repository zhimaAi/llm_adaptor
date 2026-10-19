// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package siliconflow

import (
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("frequency_penalty", "max_tokens", "n", "presence_penalty", "response_format", "seed", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options")
	spec.SupportsInputAudio = true
	spec.SupportsReasoningContent = true
	spec.ApplyReasoning = func(model string, effort chat.ReasoningEffort, body map[string]any) {
		shared.ApplyReasoningBoolean("enable_thinking")(model, effort, body)
		if effort == "" || effort == chat.ReasoningEffortNone {
			return
		}
		model = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(model)), "pro/")
		if openai.HasModelPrefix(model, "deepseek-ai/deepseek-v4", "zai-org/glm-5.2") {
			// low/medium/xhigh are accepted aliases, not adapter-side mappings.
			if effort == chat.ReasoningEffortMinimal {
				effort = chat.ReasoningEffortHigh
			}
			body["reasoning_effort"] = string(effort)
		}
	}
}
