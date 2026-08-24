// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
)

func TestProviderParameterProfilesCoverRegisteredCapabilities(t *testing.T) {
	if len(providerDefinitions) != 25 {
		t.Fatalf("provider count = %d, want 25", len(providerDefinitions))
	}
	for provider, definition := range providerDefinitions {
		implementation := definition.newProvider(ClientConfig{})
		for _, capability := range implementation.info().Capabilities {
			switch capability {
			case CapabilityChat:
				if _, ok := chatProviderParameterFields[provider]; !ok {
					t.Errorf("chat parameter profile is missing for %s", provider)
				}
			case CapabilityEmbedding:
				if _, ok := embeddingProviderParameterFields[provider]; !ok {
					t.Errorf("embedding parameter profile is missing for %s", provider)
				}
			case CapabilityImage:
				if _, ok := imageProviderParameterFields[provider]; !ok {
					t.Errorf("image parameter profile is missing for %s", provider)
				}
			}
		}
	}
}

func TestRestrictedChatParameterProfiles(t *testing.T) {
	integer := 2
	parallel := true
	seed := int64(3)
	request := &chat.CreateRequest{
		Model: "model", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}},
		N: &integer, MaxCompletionTokens: &integer, ParallelToolCalls: &parallel, Seed: &seed,
		User: "user", ToolChoice: "required", Tools: []chat.Tool{{Function: chat.FunctionDefinition{Name: "tool"}}},
	}
	tests := []struct {
		provider Provider
		absent   []string
		present  []string
	}{
		{provider: ProviderCohere, absent: []string{"n", "parallel_tool_calls", "tool_choice", "user", "max_completion_tokens", "stream_options"}, present: []string{"tools", "seed"}},
		{provider: ProviderMiniMax, absent: []string{"n", "parallel_tool_calls", "tool_choice", "user", "seed"}, present: []string{"tools", "max_completion_tokens", "stream_options"}},
		{provider: ProviderSpark, absent: []string{"n", "parallel_tool_calls", "tool_choice", "max_completion_tokens", "seed"}, present: []string{"tools", "user"}},
		{provider: ProviderOllama, absent: []string{"max_completion_tokens", "n", "parallel_tool_calls", "tool_choice", "user"}, present: []string{"tools", "seed", "stream_options"}},
	}
	for _, test := range tests {
		t.Run(string(test.provider), func(t *testing.T) {
			body, err := buildOpenAIChatRequest(test.provider, request, true, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range test.absent {
				if _, ok := body[field]; ok {
					t.Errorf("unsupported field %q was sent: %#v", field, body)
				}
			}
			for _, field := range test.present {
				if _, ok := body[field]; !ok {
					t.Errorf("supported field %q was filtered: %#v", field, body)
				}
			}
		})
	}
}

func TestRestrictedParameterExtraBodyHasFinalPriority(t *testing.T) {
	request := &chat.CreateRequest{
		Model: "model", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}},
		User: "ignored", ExtraBody: map[string]any{"user": "forced"},
	}
	body, err := buildOpenAIChatRequest(ProviderMiniMax, request, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if body["user"] != "forced" {
		t.Fatalf("ExtraBody did not restore filtered field: %#v", body)
	}
}

func TestAzureUsesOpenAIV1Protocol(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.RequestURI())
		if request.Header.Get(azureAPIKeyHeader) != "azure-key" {
			t.Errorf("api-key header = %q", request.Header.Get(azureAPIKeyHeader))
		}
		if request.Header.Get(headerAuthorization) != "" {
			t.Errorf("unexpected Authorization header = %q", request.Header.Get(headerAuthorization))
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "deployment-name" {
			t.Errorf("model = %#v", body["model"])
		}
		switch {
		case strings.HasSuffix(request.URL.Path, "/chat/completions") && body["stream"] == true:
			writer.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(writer, "data: {\"id\":\"chat\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")
		case strings.HasSuffix(request.URL.Path, "/chat/completions"):
			_, _ = io.WriteString(writer, `{"id":"chat","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`)
		case strings.HasSuffix(request.URL.Path, "/embeddings"):
			_, _ = io.WriteString(writer, `{"object":"list","data":[{"object":"embedding","embedding":[1,2],"index":0}],"model":"deployment-name","usage":{"prompt_tokens":1,"total_tokens":1}}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Provider: ProviderAzure, BaseURL: server.URL + "/gateway", APIVersion: "2024-10-21",
		Credentials: CredentialConfig{APIKeys: "azure-key"}, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	chatRequest := chat.CreateRequest{
		Model: "deployment-name", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}},
	}
	if _, err = client.Chat.Create(context.Background(), &chatRequest); err != nil {
		t.Fatal(err)
	}
	stream, err := client.Chat.Stream(context.Background(), &chat.StreamRequest{CreateRequest: chatRequest})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if err = stream.Close(); err != nil {
		t.Fatal(err)
	}
	text := "hello"
	if _, err = client.Embeddings.Create(context.Background(), &embedding.CreateRequest{
		Model: "deployment-name", Input: embedding.Input{Text: &text},
	}); err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("request paths = %#v", paths)
	}
	for _, path := range paths {
		if !strings.HasPrefix(path, "/gateway/openai/v1/") || strings.Contains(path, "api-version") || strings.Contains(path, "/deployments/") {
			t.Errorf("legacy Azure path was used: %s", path)
		}
	}
}
