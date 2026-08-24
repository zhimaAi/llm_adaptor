// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"fmt"
	"net/http"
	"strings"

	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/registry"
)

const (
	DefaultBaseURLMiniMax = "https://api.minimaxi.com/v1"
	MiniMaxSpeechPath     = "/t2a_v2"
	ChatCompletionsPath   = "/chat/completions"
	EmbeddingsPath        = "/embeddings"
	ImageGenerationsPath  = "/images/generations"
	ImageEditsPath        = "/images/edits"
)

type Client struct {
	config      ClientConfig
	credentials *credentialPool
	provider    internalprovider.Implementation

	Chat       ChatService
	Embeddings EmbeddingService
	Images     ImageService
	Rerank     RerankService
	Speech     SpeechService
}

func NewClient(config ClientConfig) (*Client, error) {
	definition, ok := registry.Lookup(internalprovider.ID(config.Provider))
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, config.Provider)
	}
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.ServiceBaseURL = strings.TrimRight(strings.TrimSpace(config.ServiceBaseURL), "/")
	if config.BaseURL == "" {
		config.BaseURL = definition.DefaultBaseURL
	}
	if config.BaseURL == "" {
		return nil, fmt.Errorf("%w: base_url is required for provider %s", ErrInvalidRequest, config.Provider)
	}
	if config.ServiceBaseURL == "" {
		config.ServiceBaseURL = definition.DefaultServiceBaseURL
	}
	internalConfig := internalprovider.Config{
		Provider: internalprovider.ID(config.Provider), BaseURL: config.BaseURL,
		ServiceBaseURL: config.ServiceBaseURL,
		Credentials:    internalprovider.CredentialConfig{APIKeys: config.Credentials.APIKeys},
		APIVersion:     config.APIVersion, HTTPClient: config.HTTPClient, DefaultHeaders: config.DefaultHeaders,
	}
	if definition.Normalize != nil {
		if err := definition.Normalize(&internalConfig); err != nil {
			return nil, normalizeProviderError(err)
		}
	}
	if internalConfig.HTTPClient == nil {
		internalConfig.HTTPClient = http.DefaultClient
	}
	config.BaseURL = internalConfig.BaseURL
	config.ServiceBaseURL = internalConfig.ServiceBaseURL
	config.HTTPClient = internalConfig.HTTPClient
	var credentials *credentialPool
	var err error
	if strings.TrimSpace(config.Credentials.APIKeys) == "" && definition.CredentialsOptional {
		credentials = newAnonymousCredentialPool()
	} else {
		credentials, err = newCredentialPool(config.Credentials)
		if err != nil {
			return nil, err
		}
	}
	client := &Client{config: config, credentials: credentials, provider: definition.New(internalConfig)}
	client.Chat = ChatService{client: client}
	client.Embeddings = EmbeddingService{client: client}
	client.Images = ImageService{client: client}
	client.Rerank = RerankService{client: client}
	client.Speech = SpeechService{client: client}
	return client, nil
}

func (c *Client) ProviderInfo() ProviderInfo {
	info := c.provider.Info()
	result := ProviderInfo{
		ID: Provider(info.ID), DefaultBaseURL: info.DefaultBaseURL,
		DefaultServiceBaseURL: info.DefaultServiceBaseURL,
		Capabilities:          make([]Capability, len(info.Capabilities)),
	}
	for index, capability := range info.Capabilities {
		result.Capabilities[index] = Capability(capability)
	}
	return result
}

func (c *Client) supports(capability Capability) bool {
	for _, supported := range c.provider.Info().Capabilities {
		if Capability(supported) == capability {
			return true
		}
	}
	return false
}

func internalCredential(selected credential) internalprovider.Credential {
	return internalprovider.Credential{APIKey: selected.apiKey, Hint: selected.hint}
}
