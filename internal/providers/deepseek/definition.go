// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package deepseek

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDDeepSeek, DefaultBaseURL: "https://api.deepseek.com", Capabilities: []provider.Capability{provider.CapabilityChat}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureChat)
}
