// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"net/http"

	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

type Provider string

const (
	Provider302AI       Provider = Provider(internalprovider.ID302AI)
	ProviderAli         Provider = Provider(internalprovider.IDAli)
	ProviderAzure       Provider = Provider(internalprovider.IDAzure)
	ProviderBAAI        Provider = Provider(internalprovider.IDBAAI)
	ProviderBaichuan    Provider = Provider(internalprovider.IDBaichuan)
	ProviderBaidu       Provider = Provider(internalprovider.IDBaidu)
	ProviderClaude      Provider = Provider(internalprovider.IDClaude)
	ProviderCohere      Provider = Provider(internalprovider.IDCohere)
	ProviderDeepSeek    Provider = Provider(internalprovider.IDDeepSeek)
	ProviderDoubao      Provider = Provider(internalprovider.IDDoubao)
	ProviderGemini      Provider = Provider(internalprovider.IDGemini)
	ProviderHunyuan     Provider = Provider(internalprovider.IDHunyuan)
	ProviderJina        Provider = Provider(internalprovider.IDJina)
	ProviderLingYiWanWu Provider = Provider(internalprovider.IDLingYiWanWu)
	ProviderMiniMax     Provider = Provider(internalprovider.IDMiniMax)
	ProviderMoonshot    Provider = Provider(internalprovider.IDMoonshot)
	ProviderOllama      Provider = Provider(internalprovider.IDOllama)
	ProviderOpenAI      Provider = Provider(internalprovider.IDOpenAI)
	ProviderOpenAIAgent Provider = Provider(internalprovider.IDOpenAIAgent)
	ProviderOpenRouter  Provider = Provider(internalprovider.IDOpenRouter)
	ProviderSiliconFlow Provider = Provider(internalprovider.IDSiliconFlow)
	ProviderSpark       Provider = Provider(internalprovider.IDSpark)
	ProviderVoyage      Provider = Provider(internalprovider.IDVoyage)
	ProviderXinference  Provider = Provider(internalprovider.IDXinference)
	ProviderZhipu       Provider = Provider(internalprovider.IDZhipu)
)

type Capability string

const (
	CapabilityChat      Capability = "chat"
	CapabilityEmbedding Capability = "embedding"
	CapabilityImage     Capability = "image"
	CapabilityRerank    Capability = "rerank"
	CapabilitySpeech    Capability = "speech"
)

type CredentialConfig struct {
	APIKeys string `json:"api_keys"`
}

type ClientConfig struct {
	Provider       Provider         `json:"provider"`
	BaseURL        string           `json:"base_url,omitempty"`
	ServiceBaseURL string           `json:"service_base_url,omitempty"`
	Credentials    CredentialConfig `json:"credentials"`
	APIVersion     string           `json:"api_version,omitempty"`
	HTTPClient     *http.Client     `json:"-"`
	DefaultHeaders http.Header      `json:"-"`
}

type ProviderInfo struct {
	ID                    Provider     `json:"id"`
	DefaultBaseURL        string       `json:"default_base_url,omitempty"`
	DefaultServiceBaseURL string       `json:"default_service_base_url,omitempty"`
	Capabilities          []Capability `json:"capabilities"`
}
