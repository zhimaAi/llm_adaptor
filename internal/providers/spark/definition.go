// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package spark

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDSpark, DefaultBaseURL: "https://spark-api-open.xf-yun.com/v1", Capabilities: []provider.Capability{provider.CapabilityChat}}
	return openai.Definition(info.DefaultBaseURL, "", false, info, configureChat)
}
