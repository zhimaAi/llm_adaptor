// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package azure

import (
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDAzure, Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding}}
	definition := openai.Definition("", "", false, info, configureChat, configureEmbedding)
	definition.Normalize = func(config *provider.Config) error {
		config.BaseURL = appendOpenAIV1Path(config.BaseURL)
		return nil
	}
	return definition
}

func appendOpenAIV1Path(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	lower := strings.ToLower(baseURL)
	if strings.HasSuffix(lower, "/openai/v1") {
		return baseURL
	}
	if strings.HasSuffix(lower, "/openai") {
		return transport.JoinURLPath(baseURL, "v1")
	}
	return transport.JoinURLPath(baseURL, "openai/v1")
}
