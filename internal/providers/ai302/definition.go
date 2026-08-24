// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ai302

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.ID302AI, DefaultBaseURL: "https://api.302ai.cn", Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityImage}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureChat, configureImage)
}
