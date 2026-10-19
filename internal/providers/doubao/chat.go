// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package doubao

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

// https://docs.volcengine.com/docs/ark/model-deprecation-migration-guide
var switchOnlyModels = []string{
	"doubao-seed-1-6-250615", "doubao-seed-1-6-flash-250615", "doubao-seed-1-6-flash-250828",
	"doubao-seed-1-6-vision-250815", "doubao-seed-code-preview-251028",
	"deepseek-v3-2-251201", "glm-4-7-251222", "kimi-k2-5-260127",
}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.SupportsInputAudio = true
	spec.SupportsVideoURL = true
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
	if shared.IsModel(model, "glm-5-3-flash-260828") {
		body["thinking"] = map[string]any{"type": "enabled"}
	}
	shared.ApplyReasoningEffort(model, effort, body)
}
