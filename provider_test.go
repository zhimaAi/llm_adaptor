// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/registry"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
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

func TestPublicProviderIdentifiersMatchInternalIDs(t *testing.T) {
	pairs := []struct {
		public   Provider
		internal internalprovider.ID
	}{
		{Provider302AI, internalprovider.ID302AI},
		{ProviderAli, internalprovider.IDAli},
		{ProviderAzure, internalprovider.IDAzure},
		{ProviderBAAI, internalprovider.IDBAAI},
		{ProviderBaichuan, internalprovider.IDBaichuan},
		{ProviderBaidu, internalprovider.IDBaidu},
		{ProviderClaude, internalprovider.IDClaude},
		{ProviderCohere, internalprovider.IDCohere},
		{ProviderDeepSeek, internalprovider.IDDeepSeek},
		{ProviderDoubao, internalprovider.IDDoubao},
		{ProviderGemini, internalprovider.IDGemini},
		{ProviderHunyuan, internalprovider.IDHunyuan},
		{ProviderJina, internalprovider.IDJina},
		{ProviderLingYiWanWu, internalprovider.IDLingYiWanWu},
		{ProviderMiniMax, internalprovider.IDMiniMax},
		{ProviderMoonshot, internalprovider.IDMoonshot},
		{ProviderOllama, internalprovider.IDOllama},
		{ProviderOpenAI, internalprovider.IDOpenAI},
		{ProviderOpenAIAgent, internalprovider.IDOpenAIAgent},
		{ProviderOpenRouter, internalprovider.IDOpenRouter},
		{ProviderSiliconFlow, internalprovider.IDSiliconFlow},
		{ProviderSpark, internalprovider.IDSpark},
		{ProviderVoyage, internalprovider.IDVoyage},
		{ProviderXinference, internalprovider.IDXinference},
		{ProviderZhipu, internalprovider.IDZhipu},
	}
	if len(pairs) != 25 {
		t.Fatalf("provider identifier count = %d, want 25", len(pairs))
	}
	for _, pair := range pairs {
		if string(pair.public) != string(pair.internal) {
			t.Errorf("public provider %q does not match internal id %q", pair.public, pair.internal)
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

	t.Run("OpenAI image edit multipart", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/images/edits" {
				t.Errorf("path = %q", request.URL.Path)
			}
			if err := request.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			prompt := request.FormValue("prompt")
			if (prompt != "edit" && prompt != "override") || request.FormValue("stream") != "false" {
				t.Errorf("form = %#v", request.MultipartForm.Value)
			}
			if prompt == "override" {
				files := request.MultipartForm.File["image"]
				if len(files) != 1 || files[0].Filename != "override.png" {
					t.Errorf("override files = %#v", files)
				}
			} else {
				files := request.MultipartForm.File["image[]"]
				if len(files) != 2 || files[0].Filename != "one.png" || files[1].Filename != "two.png" {
					t.Errorf("files = %#v", files)
				}
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"data":[{"b64_json":"aGVsbG8="}],"output_format":"png"}`)
		}))
		defer server.Close()

		client, err := NewClient(ClientConfig{Provider: ProviderOpenAI, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client()})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Images.Edit(nil, &image.EditRequest{
			Model: "gpt-image", Prompt: "edit", ResponseFormat: "b64_json",
			Images: []image.File{
				{Filename: "one.png", ContentType: "image/png", Reader: bytes.NewBufferString("one")},
				{Filename: "two.png", ContentType: "image/png", Reader: bytes.NewBufferString("two")},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Images.Edit(nil, &image.EditRequest{
			Prompt: "override", ResponseFormat: "b64_json",
			ExtraBody: map[string]any{"image": image.File{Filename: "override.png", ContentType: "image/png", Reader: bytes.NewBufferString("override")}},
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Images.Edit(nil, &image.EditRequest{Prompt: "invalid", Images: []image.File{{}}})
		if !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("nil reader error = %v, want ErrInvalidRequest", err)
		}
	})

	t.Run("Doubao native image edit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/images/generations" {
				t.Errorf("path = %q", request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Error(err)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			images, _ := body["image"].([]any)
			if len(images) != 1 || !strings.HasPrefix(images[0].(string), "data:image/png;base64,") {
				t.Errorf("image = %#v", body["image"])
			}
			if body["sequential_image_generation"] != "auto" {
				t.Errorf("sequential mode = %#v", body["sequential_image_generation"])
			}
			options, _ := body["sequential_image_generation_options"].(map[string]any)
			if options["max_images"] != float64(2) {
				t.Errorf("sequential options = %#v", options)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"data":[{"b64_json":"aGVsbG8="}],"usage":{"output_tokens":2,"total_tokens":2}}`)
		}))
		defer server.Close()

		count := 2
		client, err := NewClient(ClientConfig{Provider: ProviderDoubao, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client()})
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Images.Edit(nil, &image.EditRequest{
			Model: "doubao-seedream", Prompt: "edit", N: &count, ResponseFormat: "b64_json", OutputFormat: "png",
			Images: []image.File{{Filename: "input.png", ContentType: "image/png", Reader: bytes.NewBufferString("input")}},
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Doubao native image stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Error(err)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			if body["stream"] != true {
				t.Errorf("stream = %#v", body["stream"])
			}
			writer.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(writer, "data: {\"created\":1,\"data\":[{\"b64_json\":\"b25l\"},{\"b64_json\":\"dHdv\"}],\"usage\":{\"output_tokens\":2,\"total_tokens\":2}}\n\ndata: [DONE]\n\n")
		}))
		defer server.Close()

		client, err := NewClient(ClientConfig{Provider: ProviderDoubao, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client()})
		if err != nil {
			t.Fatal(err)
		}
		stream, err := client.Images.Stream(nil, &image.StreamRequest{GenerateRequest: image.GenerateRequest{Model: "doubao-seedream", Prompt: "draw", ResponseFormat: "b64_json", OutputFormat: "png"}})
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		first, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		second, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if first.B64JSON != "b25l" || second.B64JSON != "dHdv" || first.Usage.TotalTokens != 2 || second.Usage.TotalTokens != 0 {
			t.Fatalf("unexpected chunks: first=%#v second=%#v", first, second)
		}
		if _, err = stream.Recv(); err != io.EOF {
			t.Fatalf("final error = %v, want EOF", err)
		}
	})

	t.Run("BAAI optional bearer authentication", func(t *testing.T) {
		for _, test := range []struct {
			name       string
			apiKey     string
			wantHeader string
		}{{name: "anonymous"}, {name: "api key", apiKey: "secret", wantHeader: "Bearer secret"}} {
			t.Run(test.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					if got := request.Header.Get("Authorization"); got != test.wantHeader {
						t.Errorf("authorization = %q, want %q", got, test.wantHeader)
					}
					writer.Header().Set("Content-Type", "application/json")
					switch request.URL.Path {
					case "/v1/embeddings":
						_, _ = io.WriteString(writer, `{"data":[{"index":0,"embedding":[1]}]}`)
					case "/v1/rerank":
						_, _ = io.WriteString(writer, `{"results":[{"index":0,"relevance_score":1}]}`)
					default:
						t.Errorf("path = %q", request.URL.Path)
					}
				}))
				defer server.Close()
				client, err := NewClient(ClientConfig{Provider: ProviderBAAI, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: test.apiKey}, HTTPClient: server.Client()})
				if err != nil {
					t.Fatal(err)
				}
				input := "hello"
				if _, err = client.Embeddings.Create(nil, &embedding.CreateRequest{Model: "bge-m3", Input: embedding.Input{Text: &input}}); err != nil {
					t.Fatal(err)
				}
				if _, err = client.Rerank.Create(nil, &rerank.CreateRequest{Model: "bge-m3", Query: "q", Documents: []string{"d"}}); err != nil {
					t.Fatal(err)
				}
			})
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
