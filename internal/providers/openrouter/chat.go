// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openrouter

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ApplyReasoning = func(_ string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		delete(body, "reasoning_effort")
		body["reasoning"] = map[string]any{"effort": string(effort)}
	}
}
