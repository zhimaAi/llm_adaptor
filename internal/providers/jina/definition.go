// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package jina

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDJina, DefaultBaseURL: "https://api.jina.ai/v1", Capabilities: []provider.Capability{provider.CapabilityEmbedding, provider.CapabilityRerank}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureEmbedding, configureRerank)
}
