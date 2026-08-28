// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package zhipu

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDZhipu, DefaultBaseURL: "https://open.bigmodel.cn/api/paas/v4", Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureChat, configureEmbedding)
}
