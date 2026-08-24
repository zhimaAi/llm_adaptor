// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package azure

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	rawPrefix := ""
	spec.AuthorizationHeader = "api-key"
	spec.AuthorizationPrefix = &rawPrefix
	spec.SupportsInputAudio = true
	spec.ApplyReasoning = func(_ string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		openai.MoveMaxTokensToCompletionTokens(body)
		delete(body, "temperature")
	}
}
