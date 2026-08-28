// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	openprotocol "github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDOpenAI, DefaultBaseURL: "https://api.openai.com/v1", Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding, provider.CapabilityImage}}
	return openprotocol.Definition(info.DefaultBaseURL, "", false, info, configureChat, configureEmbedding, configureImage)
}
