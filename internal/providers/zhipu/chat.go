// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package zhipu

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

// https://docs.bigmodel.cn/api-reference/模型-api/对话补全
var switchOnlyModels = []string{
	"glm-4.5", "glm-4.5-air", "glm-4.5-airx", "glm-4.5-flash",
	"glm-4.6", "glm-4.6v", "glm-4.6v-flash", "glm-4.7", "glm-4.7-flash", "glm-4.7-flashx",
	"glm-5", "glm-5-turbo", "glm-5.1",
}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.FieldsWithout(openai.AllChatFields,
		"frequency_penalty", "max_completion_tokens", "n", "parallel_tool_calls", "presence_penalty", "seed")
	spec.ChatFieldAliases = map[string]string{"user": "user_id"}
	spec.SupportsReasoningContent = true
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	shared.ApplyThinkingType(model, effort, body)
	if shared.IsModel(model, switchOnlyModels...) {
		return
	}
	if shared.IsModel(model, "glm-5.3", "glm-5.3-flash", "glm-5.3-flashx") {
		body["thinking"] = map[string]any{"type": "enabled"}
	}
	shared.ApplyReasoningEffort(model, effort, body)
}
