// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package hunyuan

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

const DefaultBaseURL = "https://tokenhub.tencentmaas.com/v1"

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDHunyuan, DefaultBaseURL: DefaultBaseURL, Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureChat, configureEmbedding)
}
