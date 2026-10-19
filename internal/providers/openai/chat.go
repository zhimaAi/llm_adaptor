// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	openprotocol "github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openprotocol.Spec, _ provider.Config) {
	spec.SupportsInputAudio = true
	spec.ApplyReasoning = applyReasoning
}

func applyReasoning(model string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	shared.ApplyReasoningEffort(model, effort, body)
	openprotocol.MoveMaxTokensToCompletionTokens(body)
	delete(body, "temperature")
}
