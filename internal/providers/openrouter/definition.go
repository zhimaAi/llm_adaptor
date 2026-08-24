// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openrouter

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

const defaultBaseURL = "https://openrouter.ai/api/v1"

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDOpenRouter, DefaultBaseURL: defaultBaseURL, Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityImage}}
	return provider.Definition{
		DefaultBaseURL: defaultBaseURL,
		New: func(config provider.Config) provider.Implementation {
			spec := openai.DefaultSpec(info)
			configureChat(&spec, config)
			configureImage(&spec, config)
			return &Provider{Provider: openai.New(config, spec)}
		},
	}
}

func imageChatSpec() openai.Spec {
	spec := openai.DefaultSpec(provider.Info{ID: provider.IDOpenRouter, DefaultBaseURL: defaultBaseURL, Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityImage}})
	configureChat(&spec, provider.Config{})
	configureImage(&spec, provider.Config{})
	return spec
}
