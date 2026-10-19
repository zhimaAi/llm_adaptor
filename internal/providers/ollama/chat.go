// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ollama

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

// Explicit boolean-thinking model names and tags; custom/future tags use the
// OpenAI-compatible reasoning_effort parameter without changing its value.
var switchOnlyModels = []string{
	"qwen3", "qwen3:latest", "qwen3:0.6b", "qwen3:1.7b", "qwen3:4b", "qwen3:8b",
	"qwen3:14b", "qwen3:30b", "qwen3:32b", "qwen3:235b",
	"deepseek-r1", "deepseek-r1:latest", "deepseek-r1:1.5b", "deepseek-r1:7b", "deepseek-r1:8b",
	"deepseek-r1:14b", "deepseek-r1:32b", "deepseek-r1:70b", "deepseek-r1:671b",
}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.FieldsWithout(openai.AllChatFields, "max_completion_tokens", "parallel_tool_calls")
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	if shared.IsModel(model, switchOnlyModels...) {
		shared.ApplyReasoningBoolean("think")(model, effort, body)
		return
	}
	shared.ApplyReasoningEffort(model, effort, body)
}
