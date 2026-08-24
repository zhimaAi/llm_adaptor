// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/registry"
	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

func TestProviderRegistryContract(t *testing.T) {
	ids := registry.IDs()
	if len(ids) != 25 {
		t.Fatalf("provider count = %d, want 25", len(ids))
	}
	for _, id := range ids {
		definition, ok := registry.Lookup(id)
		if !ok {
			t.Fatalf("provider %q is not registered", id)
		}
		baseURL := definition.DefaultBaseURL
		if baseURL == "" {
			baseURL = "http://localhost/v1"
		}
		config := internalprovider.Config{Provider: id, BaseURL: baseURL, APIVersion: "v1", HTTPClient: http.DefaultClient}
		if definition.Normalize != nil {
			if err := definition.Normalize(&config); err != nil {
				t.Fatalf("normalize %q: %v", id, err)
			}
		}
		implementation := definition.New(config)
		info := implementation.Info()
		if info.ID != id {
			t.Fatalf("provider info id %q does not match %q", info.ID, id)
		}
		for _, capability := range info.Capabilities {
			switch capability {
			case internalprovider.CapabilityChat:
				assertCapability[internalprovider.Chat](t, implementation, Capability(capability))
			case internalprovider.CapabilityEmbedding:
				assertCapability[internalprovider.Embedding](t, implementation, Capability(capability))
			case internalprovider.CapabilityImage:
				assertCapability[internalprovider.Image](t, implementation, Capability(capability))
			case internalprovider.CapabilityRerank:
				assertCapability[internalprovider.Rerank](t, implementation, Capability(capability))
			case internalprovider.CapabilitySpeech:
				assertCapability[internalprovider.Speech](t, implementation, Capability(capability))
			default:
				t.Fatalf("unknown capability %q", capability)
			}
		}
	}
}

func TestProviderDirectoryContract(t *testing.T) {
	want := []string{"ai302", "ali", "azure", "baai", "baichuan", "baidu", "claude", "cohere", "deepseek", "doubao", "gemini", "hunyuan", "jina", "lingyiwanwu", "minimax", "moonshot", "ollama", "openai", "openaiagent", "openrouter", "siliconflow", "spark", "voyage", "xinference", "zhipu"}
	entries, err := os.ReadDir(filepath.Join("internal", "providers"))
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		got = append(got, entry.Name())
		if _, err := os.Stat(filepath.Join("internal", "providers", entry.Name(), "definition.go")); err != nil {
			t.Errorf("provider %s has no definition.go", entry.Name())
		}
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("provider directories = %v", got)
	}

	for _, root := range []string{filepath.Join("internal", "providers"), filepath.Join("internal", "protocol"), filepath.Join("internal", "shared"), filepath.Join("internal", "transport")} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || entry.IsDir() || filepath.Ext(path) != ".go" {
				return walkErr
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(content)
			if strings.HasPrefix(filepath.ToSlash(path), "internal/providers/") && strings.Contains(text, `"github.com/zhimaAi/llm_adaptor/v2"`) {
				t.Errorf("provider imports root package: %s", path)
			}
			if !strings.HasPrefix(filepath.ToSlash(path), "internal/providers/") && strings.Contains(text, "/internal/providers/") {
				t.Errorf("shared package imports provider implementation: %s", path)
			}
			if strings.Contains(text, "func init(") {
				t.Errorf("implicit provider registration in %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCriticalProviderFlows(t *testing.T) {
	t.Run("OpenAI-compatible stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/v1/chat/completions" {
				t.Errorf("path = %q", request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			streamOptions, _ := body["stream_options"].(map[string]any)
			if streamOptions["include_usage"] != true {
				t.Errorf("stream_options = %#v", streamOptions)
			}
			writer.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(writer, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"<think>plan</think>answer\"}}]}\n\ndata: [DONE]\n\n")
		}))
		defer server.Close()

		client, err := NewClient(ClientConfig{Provider: ProviderOpenAIAgent, BaseURL: server.URL, APIVersion: "v1", HTTPClient: server.Client()})
		if err != nil {
			t.Fatal(err)
		}
		stream, err := client.Chat.Stream(nil, &chat.StreamRequest{CreateRequest: chat.CreateRequest{Model: "model", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}}}})
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		chunk, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		content := chunk.Choices[0].Delta.Content.Text
		if chunk.Choices[0].Delta.ReasoningContent != "plan" || content == nil || *content != "answer" {
			t.Fatalf("unexpected chunk: %#v", chunk)
		}
	})

	t.Run("MiniMax speech", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != MiniMaxSpeechPath {
				t.Errorf("path = %q", request.URL.Path)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"data":{"audio":"00ff","status":2},"trace_id":"trace","base_resp":{"status_code":0}}`)
		}))
		defer server.Close()

		client, err := NewClient(ClientConfig{Provider: ProviderMiniMax, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client()})
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Speech.Create(nil, &speech.CreateRequest{Model: "speech-2.6-hd", Text: "hello"})
		if err != nil {
			t.Fatal(err)
		}
		if response.Data == nil || response.Data.Audio != "00ff" {
			t.Fatalf("unexpected response: %#v", response)
		}
	})
}

func TestOpenAIAgentPublicContract(t *testing.T) {
	client, err := NewClient(ClientConfig{Provider: ProviderOpenAIAgent, BaseURL: "https://gateway.example/custom", APIVersion: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	if client.config.BaseURL != "https://gateway.example/custom/v1" {
		t.Fatalf("base URL = %q", client.config.BaseURL)
	}
	for _, capability := range []Capability{CapabilityChat, CapabilityEmbedding, CapabilityImage} {
		if !client.supports(capability) {
			t.Errorf("missing capability %s", capability)
		}
	}
}

func assertCapability[T any](t *testing.T, implementation internalprovider.Implementation, capability Capability) {
	t.Helper()
	if _, ok := any(implementation).(T); !ok {
		t.Fatalf("declared capability %q is not implemented", capability)
	}
}
