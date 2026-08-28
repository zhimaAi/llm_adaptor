// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package doubao

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

type Provider struct{ *openai.Provider }

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDDoubao, DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3", Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding, provider.CapabilityImage}}
	return provider.Definition{
		DefaultBaseURL: info.DefaultBaseURL,
		New: func(config provider.Config) provider.Implementation {
			spec := openai.DefaultSpec(info)
			configureChat(&spec, config)
			configureEmbedding(&spec, config)
			return &Provider{Provider: openai.New(config, spec)}
		},
	}
}
