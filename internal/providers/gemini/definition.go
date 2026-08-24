// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package gemini

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

const (
	defaultBaseURL        = "https://generativelanguage.googleapis.com/v1beta/openai"
	defaultServiceBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

type Provider struct{ *openai.Provider }

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDGemini, DefaultBaseURL: defaultBaseURL, DefaultServiceBaseURL: defaultServiceBaseURL, Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding}}
	return provider.Definition{
		DefaultBaseURL: defaultBaseURL, DefaultServiceBaseURL: defaultServiceBaseURL,
		New: func(config provider.Config) provider.Implementation {
			spec := openai.DefaultSpec(info)
			configureChat(&spec, config)
			configureEmbedding(&spec, config)
			return &Provider{Provider: openai.New(config, spec)}
		},
	}
}
