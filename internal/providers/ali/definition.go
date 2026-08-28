// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ali

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

const (
	defaultBaseURL        = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	defaultServiceBaseURL = "https://dashscope.aliyuncs.com"
)

type Provider struct{ *openai.Provider }

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDAli, DefaultBaseURL: defaultBaseURL, DefaultServiceBaseURL: defaultServiceBaseURL, Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding, provider.CapabilityRerank, provider.CapabilityImage}}
	return provider.Definition{
		DefaultBaseURL: defaultBaseURL, DefaultServiceBaseURL: defaultServiceBaseURL,
		New: func(config provider.Config) provider.Implementation {
			spec := openai.DefaultSpec(info)
			configureChat(&spec, config)
			configureEmbedding(&spec, config)
			configureImage(&spec, config)
			return &Provider{Provider: openai.New(config, spec)}
		},
	}
}
