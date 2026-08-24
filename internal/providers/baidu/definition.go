// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package baidu

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDBaidu, DefaultBaseURL: "https://qianfan.baidubce.com/v2", Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureChat, configureEmbedding)
}
