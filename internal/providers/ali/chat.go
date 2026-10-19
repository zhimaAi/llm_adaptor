// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ali

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

// Documented legacy IDs and explicitly opted-in families. Unmatched models
// default to effort passthrough.
// https://help.aliyun.com/zh/model-studio/deep-thinking
var switchOnlyModels = []string{
	"qwen3.7-max", "qwen3.7-max-2026-05-20", "qwen3.7-max-2026-06-08",
	"qwen3.7-max-preview", "qwen3.7-max-2026-05-17",
	"qwen3.7-plus", "qwen3.7-plus-2026-05-26", "qwen3.7-flash", "qwen3.7-flash-2026-07-15",
	"qwen3.6-plus", "qwen3.6-plus-2026-04-02", "qwen3.6-flash", "qwen3.6-flash-2026-04-16", "qwen3.6-35b-a3b",
	"qwen3.5-plus", "qwen3.5-plus-2026-02-15", "qwen3.5-flash", "qwen3.5-flash-2026-02-23",
	"qwen3.5-397b-a17b", "qwen3.5-122b-a10b", "qwen3.5-27b", "qwen3.5-35b-a3b",
	"qwen-plus", "qwen-plus-*",
	"qwen-flash", "qwen-flash-2025-07-28", "qwen-turbo", "qwen-turbo-latest", "qwen-turbo-2025-04-28",
	"qwen3-235b-a22b", "qwen3-32b", "qwen3-30b-a3b", "qwen3-14b", "qwen3-8b", "qwen3-4b", "qwen3-1.7b", "qwen3-0.6b",
	"qwen3-omni-flash", "qwen3-omni-flash-2025-09-15", "qwen3-vl-plus", "qwen3-vl-flash",
	"deepseek-v3.2", "deepseek-v3.2-exp", "deepseek-v3.1", "deepseek-v3.1-terminus",
	"siliconflow/deepseek-v3.2", "siliconflow/deepseek-v3.1-terminus",
	"vanchin/deepseek-v3.2-think", "vanchin/deepseek-v3.1-terminus",
	"kimi-k2.5", "kimi-k2.6", "kimi-k2.7-code",
	"kimi/kimi-k2.5", "kimi/kimi-k2.6", "kimi/kimi-k2.7-code", "kimi/kimi-k2.7-code-highspeed",
	"glm-4.5", "glm-4.5-air", "glm-4.6", "glm-4.7",
}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.FieldsWithout(
		openai.AllChatFields,
		"frequency_penalty", "max_completion_tokens", "parallel_tool_calls", "response_format", "tool_choice", "user",
	)
	spec.SupportsInputAudio = true
	spec.SupportsVideoURL = true
	spec.SupportsReasoningContent = true
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	if shared.IsModel(model, "MiniMax/MiniMax-M3", "MiniMax-M3") {
		shared.ApplyAdaptiveThinkingType(model, effort, body)
		return
	}
	shared.ApplyReasoningBoolean("enable_thinking")(model, effort, body)
	if shared.IsModel(model, switchOnlyModels...) {
		return
	}
	// Preserve documented auxiliary settings without changing the effort.
	if shared.IsModel(model, "kimi/kimi-k3", "glm-5.3", "glm-5.3-flash", "glm-5.3-flashx",
		"ZHIPU/GLM-5.3", "ZHIPU/GLM-5.3-Flash", "ZHIPU/GLM-5.3-FlashX") {
		body["enable_thinking"] = true
	}
	shared.ApplyReasoningEffort(model, effort, body)
}
