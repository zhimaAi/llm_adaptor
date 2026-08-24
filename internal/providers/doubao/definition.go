// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package doubao

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDDoubao, DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3", Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding, provider.CapabilityImage}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureChat, configureEmbedding, configureImage)
}
