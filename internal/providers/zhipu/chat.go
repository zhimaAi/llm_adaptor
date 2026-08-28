// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package zhipu

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.FieldsWithout(
		openai.AllChatFields,
		"frequency_penalty",
		"max_completion_tokens",
		"n",
		"parallel_tool_calls",
		"presence_penalty",
		"seed",
	)
	spec.ChatFieldAliases = map[string]string{"user": "user_id"}
	spec.ApplyReasoning = shared.ApplyThinkingType
}
