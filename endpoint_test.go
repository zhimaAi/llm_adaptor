// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import "testing"

func TestProviderEndpointNormalization(t *testing.T) {
	tests := []struct {
		name        string
		config      ClientConfig
		wantBase    string
		wantService string
	}{
		{name: "openai agent appends version", config: ClientConfig{Provider: ProviderOpenAIAgent, BaseURL: "https://gateway.example/custom", APIVersion: "v3", Credentials: CredentialConfig{APIKeys: "key"}}, wantBase: "https://gateway.example/custom/v3"},
		{name: "openai agent keeps existing version", config: ClientConfig{Provider: ProviderOpenAIAgent, BaseURL: "https://gateway.example/custom/v3/", APIVersion: "v3", Credentials: CredentialConfig{APIKeys: "key"}}, wantBase: "https://gateway.example/custom/v3"},
		{name: "xinference keeps existing version", config: ClientConfig{Provider: ProviderXinference, BaseURL: "https://gateway.example/custom/v1/", APIVersion: "v1", Credentials: CredentialConfig{APIKeys: "key"}}, wantBase: "https://gateway.example/custom/v1"},
		{name: "azure appends openai v1", config: ClientConfig{Provider: ProviderAzure, BaseURL: "https://resource.openai.azure.com/custom", APIVersion: "ignored", Credentials: CredentialConfig{APIKeys: "key"}}, wantBase: "https://resource.openai.azure.com/custom/openai/v1"},
		{name: "azure keeps complete openai v1", config: ClientConfig{Provider: ProviderAzure, BaseURL: "https://resource.openai.azure.com/custom/openai/v1/", Credentials: CredentialConfig{APIKeys: "key"}}, wantBase: "https://resource.openai.azure.com/custom/openai/v1"},
		{name: "ollama appends openai version", config: ClientConfig{Provider: ProviderOllama, BaseURL: "https://ollama.example/custom", Credentials: CredentialConfig{APIKeys: "key"}}, wantBase: "https://ollama.example/custom/v1"},
		{name: "ali preserves explicit service path", config: ClientConfig{Provider: ProviderAli, BaseURL: "https://gateway.example/openai/v1", ServiceBaseURL: "https://gateway.example/native/custom", Credentials: CredentialConfig{APIKeys: "key"}}, wantBase: "https://gateway.example/openai/v1", wantService: "https://gateway.example/native/custom"},
		{name: "minimax uses China default", config: ClientConfig{Provider: ProviderMiniMax, Credentials: CredentialConfig{APIKeys: "key"}}, wantBase: "https://api.minimaxi.com/v1"},
		{name: "minimax preserves overseas override", config: ClientConfig{Provider: ProviderMiniMax, BaseURL: "https://api.minimax.io/v1", Credentials: CredentialConfig{APIKeys: "key"}}, wantBase: "https://api.minimax.io/v1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewClient(test.config)
			if err != nil {
				t.Fatal(err)
			}
			if client.config.BaseURL != test.wantBase || client.config.ServiceBaseURL != test.wantService {
				t.Fatalf("unexpected endpoints: base=%q service=%q", client.config.BaseURL, client.config.ServiceBaseURL)
			}
		})
	}
}

func TestVersionedProvidersRequireAPIVersion(t *testing.T) {
	for _, provider := range []Provider{ProviderOpenAIAgent, ProviderXinference} {
		if _, err := NewClient(ClientConfig{Provider: provider, BaseURL: "https://example.com", Credentials: CredentialConfig{APIKeys: "key"}}); err == nil {
			t.Fatalf("provider %s accepted an empty API version", provider)
		}
	}
}
