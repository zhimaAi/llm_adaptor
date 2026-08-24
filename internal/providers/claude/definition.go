// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package claude

import "github.com/zhimaAi/llm_adaptor/v2/internal/provider"

const defaultBaseURL = "https://api.anthropic.com/v1"

func Definition() provider.Definition {
	return provider.Definition{
		DefaultBaseURL: defaultBaseURL,
		New: func(config provider.Config) provider.Implementation {
			return &Provider{config: config}
		},
	}
}
