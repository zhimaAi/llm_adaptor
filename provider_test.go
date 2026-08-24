// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import "testing"

func TestOpenAIAgentProviderContract(t *testing.T) {
	if ProviderOpenAIAgent != Provider("openaiAgent") {
		t.Fatalf("provider value = %q", ProviderOpenAIAgent)
	}
	client, err := NewClient(ClientConfig{
		Provider:   ProviderOpenAIAgent,
		BaseURL:    "https://gateway.example/custom",
		APIVersion: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range []Capability{CapabilityChat, CapabilityEmbedding, CapabilityImage} {
		if !client.supports(capability) {
			t.Errorf("missing capability %s", capability)
		}
	}
	if client.config.BaseURL != "https://gateway.example/custom/v1" {
		t.Fatalf("base URL = %q", client.config.BaseURL)
	}
}

func TestProviderCapabilityImplementations(t *testing.T) {
	for id, definition := range providerDefinitions {
		t.Run(string(id), func(t *testing.T) {
			baseURL := definition.defaultBaseURL
			if baseURL == "" {
				baseURL = "http://localhost/v1"
			}
			implementation := definition.newProvider(ClientConfig{Provider: id, BaseURL: baseURL})
			info := implementation.info()
			if info.ID != id {
				t.Fatalf("provider info id %q does not match definition %q", info.ID, id)
			}
			for _, capability := range info.Capabilities {
				switch capability {
				case CapabilityChat:
					if _, ok := implementation.(chatProvider); !ok {
						t.Fatalf("declared chat capability is not implemented")
					}
				case CapabilityEmbedding:
					if _, ok := implementation.(embeddingProvider); !ok {
						t.Fatalf("declared embedding capability is not implemented")
					}
				case CapabilityImage:
					if _, ok := implementation.(imageProvider); !ok {
						t.Fatalf("declared image capability is not implemented")
					}
				case CapabilityRerank:
					if _, ok := implementation.(rerankProvider); !ok {
						t.Fatalf("declared rerank capability is not implemented")
					}
				case CapabilitySpeech:
					if _, ok := implementation.(speechProvider); !ok {
						t.Fatalf("declared speech capability is not implemented")
					}
				default:
					t.Fatalf("unknown capability %q", capability)
				}
			}
		})
	}
}
