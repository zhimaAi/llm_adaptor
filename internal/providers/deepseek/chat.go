// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package deepseek

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ChatFields = openai.Fields("frequency_penalty", "max_tokens", "presence_penalty", "response_format", "seed", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options")
	spec.ApplyReasoning = shared.ApplyThinkingType
}
