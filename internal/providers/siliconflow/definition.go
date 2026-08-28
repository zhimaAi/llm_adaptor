// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package siliconflow

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDSiliconFlow, DefaultBaseURL: "https://api.siliconflow.cn/v1", Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding, provider.CapabilityRerank}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureChat, configureEmbedding, configureRerank)
}
