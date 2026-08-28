// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ali

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
		"parallel_tool_calls",
		"response_format",
		"tool_choice",
		"user",
	)
	spec.SupportsInputAudio = true
	spec.SupportsVideoURL = true
	spec.ApplyReasoning = shared.ApplyReasoningBoolean("enable_thinking")
}
