// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ollama

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDOllama, DefaultBaseURL: "http://localhost:11434/v1", Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding}}
	definition := openai.Definition(info.DefaultBaseURL, "", true, info, configureChat, configureEmbedding)
	definition.Normalize = func(config *provider.Config) error {
		config.BaseURL = transport.AppendURLSegment(config.BaseURL, "v1")
		return nil
	}
	return definition
}
