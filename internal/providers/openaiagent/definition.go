// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openaiagent

import (
	"fmt"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

func Definition() provider.Definition {
	info := provider.Info{ID: provider.IDOpenAIAgent, Capabilities: []provider.Capability{provider.CapabilityChat, provider.CapabilityEmbedding, provider.CapabilityImage}}
	definition := openai.Definition("", "", true, info, configureChat, configureEmbedding, configureImage)
	definition.Normalize = requireVersion
	return definition
}

func requireVersion(config *provider.Config) error {
	if strings.TrimSpace(config.APIVersion) == "" {
		return fmt.Errorf("%w: api_version is required for provider %s", provider.ErrInvalidRequest, config.Provider)
	}
	config.BaseURL = transport.AppendURLSegment(config.BaseURL, config.APIVersion)
	return nil
}
