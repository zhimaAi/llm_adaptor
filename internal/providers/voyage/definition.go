// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package voyage

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDVoyage, DefaultBaseURL: "https://api.voyageai.com/v1", Capabilities: []provider.Capability{provider.CapabilityEmbedding}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureEmbedding)
}
