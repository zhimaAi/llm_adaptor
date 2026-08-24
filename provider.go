// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import "net/http"

type Provider string

const (
	Provider302AI       Provider = "302ai"
	ProviderAli         Provider = "ali"
	ProviderAzure       Provider = "azure"
	ProviderBAAI        Provider = "baai"
	ProviderBaichuan    Provider = "baichuan"
	ProviderBaidu       Provider = "baidu"
	ProviderClaude      Provider = "claude"
	ProviderCohere      Provider = "cohere"
	ProviderDeepSeek    Provider = "deepseek"
	ProviderDoubao      Provider = "doubao"
	ProviderGemini      Provider = "gemini"
	ProviderHunyuan     Provider = "hunyuan"
	ProviderJina        Provider = "jina"
	ProviderLingYiWanWu Provider = "lingyiwanwu"
	ProviderMiniMax     Provider = "minimax"
	ProviderMoonshot    Provider = "moonshot"
	ProviderOllama      Provider = "ollama"
	ProviderOpenAI      Provider = "openai"
	ProviderOpenAIAgent Provider = "openaiAgent"
	ProviderOpenRouter  Provider = "openrouter"
	ProviderSiliconFlow Provider = "siliconflow"
	ProviderSpark       Provider = "spark"
	ProviderVoyage      Provider = "voyage"
	ProviderXinference  Provider = "xinference"
	ProviderZhipu       Provider = "zhipu"
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
