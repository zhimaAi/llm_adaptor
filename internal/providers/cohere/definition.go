// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package cohere

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

const serviceBaseURL = "https://api.cohere.com"

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDCohere, DefaultBaseURL: "https://api.cohere.ai/compatibility/v1", DefaultServiceBaseURL: serviceBaseURL, Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding, provider.CapabilityRerank}}
	return openai.Definition(info.DefaultBaseURL, serviceBaseURL, false, info, configureChat, configureEmbedding, configureRerank)
}
