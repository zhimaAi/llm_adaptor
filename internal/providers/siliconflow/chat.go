// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package siliconflow

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

// https://api-docs.siliconflow.cn/docs/api/chat-completions-post
var switchOnlyModels = []string{
	"Qwen/Qwen3-235B-A22B", "Qwen/Qwen3-30B-A3B", "Qwen/Qwen3-32B", "Qwen/Qwen3-14B", "Qwen/Qwen3-8B",
	"Pro/Qwen/Qwen3-8B", "deepseek-ai/DeepSeek-V3.1", "Pro/deepseek-ai/DeepSeek-V3.1",
	"deepseek-ai/DeepSeek-V3.2", "Pro/deepseek-ai/DeepSeek-V3.2",
	"zai-org/GLM-4.5", "zai-org/GLM-4.5-Air", "zai-org/GLM-4.6", "zai-org/GLM-4.7",
	"Pro/zai-org/GLM-4.5", "Pro/zai-org/GLM-4.6", "Pro/zai-org/GLM-4.7",
}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("frequency_penalty", "max_tokens", "n", "presence_penalty", "response_format", "seed", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options")
	spec.SupportsInputAudio = true
	spec.SupportsReasoningContent = true
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	shared.ApplyReasoningBoolean("enable_thinking")(model, effort, body)
	if !shared.IsModel(model, switchOnlyModels...) {
		shared.ApplyReasoningEffort(model, effort, body)
	}
}
