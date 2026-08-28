// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package baai

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDBAAI, Capabilities: []provider.Capability{provider.CapabilityEmbedding, provider.CapabilityRerank}}
	return openai.Definition("", "", true, info, configureEmbedding, configureRerank)
}
