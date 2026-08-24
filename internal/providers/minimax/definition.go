// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package minimax

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

const DefaultBaseURL = "https://api.minimaxi.com/v1"

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDMiniMax, DefaultBaseURL: DefaultBaseURL, Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilitySpeech}}
	return provider.Definition{
		DefaultBaseURL: DefaultBaseURL,
		New: func(config provider.Config) provider.Implementation {
			spec := openai.DefaultSpec(info)
			configureChat(&spec, config)
			return &Provider{Provider: openai.New(config, spec)}
		},
	}
}
