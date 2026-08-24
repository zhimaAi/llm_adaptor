// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package siliconflow

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("frequency_penalty", "max_tokens", "n", "presence_penalty", "response_format", "seed", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options")
	spec.SupportsInputAudio = true
	spec.ApplyReasoning = shared.ApplyReasoningBoolean("enable_thinking")
}
