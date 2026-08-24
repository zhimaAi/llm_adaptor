// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
)

func TestOpenAIMessageBuilderIgnoresUnsupportedParts(t *testing.T) {
	content, err := buildOpenAIMessageContent(ProviderClaude, chat.PartsContent(
		chat.ContentPart{Type: chat.ContentPartInputAudio, InputAudio: &chat.InputAudio{Data: "audio", Format: "wav"}},
		chat.ContentPart{Type: chat.ContentPartText, Text: "kept"},
		chat.ContentPart{Type: chat.ContentPartType("future_part")},
	))
	if err != nil {
		t.Fatal(err)
	}
	parts, ok := content.([]openAIContentPartWire)
	if !ok || len(parts) != 1 || parts[0].Type != string(chat.ContentPartText) || parts[0].Text != "kept" {
		t.Fatalf("content = %#v", content)
	}
}

func TestOpenAIMessageBuilderRejectsMessageWithoutSupportedContent(t *testing.T) {
	_, err := buildOpenAIChatRequest(ProviderClaude, &chat.CreateRequest{
		Model: "model",
		Messages: []chat.Message{{
			Role: chat.RoleUser,
			Content: chat.PartsContent(chat.ContentPart{
				Type: chat.ContentPartVideoURL, VideoURL: &chat.VideoURL{URL: "https://example.com/video.mp4"},
			}),
		}},
	}, false, nil)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v", err)
	}
}

func TestClaudeBuilderIgnoresUnsupportedPublicFields(t *testing.T) {
	value := 1
	seed := int64(2)
	body, err := buildClaudeRequest(&chat.CreateRequest{
		Model: "claude-sonnet",
		Messages: []chat.Message{
			{Role: chat.RoleSystem, Content: chat.PartsContent(
				chat.ContentPart{Type: chat.ContentPartText, Text: "system"},
				chat.ContentPart{Type: chat.ContentPartVideoURL, VideoURL: &chat.VideoURL{URL: "https://example.com/video.mp4"}},
			)},
			{Role: chat.RoleUser, Content: chat.TextContent("hello")},
		},
		N: &value, FrequencyPenalty: float64Pointer(0.2), PresencePenalty: float64Pointer(0.3),
		Seed: &seed, ResponseFormat: &chat.ResponseFormat{Type: "json_object"},
		ToolChoice: "future_choice",
		Tools:      []chat.Tool{{Type: "future_tool"}},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if body["system"] != "system" {
		t.Fatalf("system = %#v", body["system"])
	}
	for _, key := range []string{"n", "frequency_penalty", "presence_penalty", "seed", "response_format", "tool_choice", "tools"} {
		if _, exists := body[key]; exists {
			t.Fatalf("unsupported field %q was sent: %#v", key, body[key])
		}
	}
}

func TestGeminiEmbeddingIgnoresUnsupportedFieldsButExtraBodyCanInjectThem(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		bodies = append(bodies, body)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"embedding":{"values":[1,2]}}`))
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{
		Provider: ProviderGemini, BaseURL: server.URL, ServiceBaseURL: server.URL,
		Credentials: CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	text := "hello"
	if _, err = client.Embeddings.Create(context.Background(), &embedding.CreateRequest{
		Model: "model", Input: embedding.Input{Text: &text}, EncodingFormat: "base64", User: "ignored",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Embeddings.Create(context.Background(), &embedding.CreateRequest{
		Model: "model", Input: embedding.Input{Text: &text}, EncodingFormat: "base64", User: "ignored",
		ExtraBody: map[string]any{"user": "forced", "encoding_format": "future"},
	}); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests = %d", len(bodies))
	}
	if _, exists := bodies[0]["user"]; exists {
		t.Fatalf("structured user was sent: %#v", bodies[0])
	}
	if _, exists := bodies[0]["encoding_format"]; exists {
		t.Fatalf("structured encoding_format was sent: %#v", bodies[0])
	}
	if bodies[1]["user"] != "forced" || bodies[1]["encoding_format"] != "future" {
		t.Fatalf("ExtraBody was not injected: %#v", bodies[1])
	}
}

func TestAliImageIgnoresUnsupportedFields(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		bodies = append(bodies, body)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"output":{"choices":[{"message":{"content":[{"image":"data:image/png;base64,aGVsbG8="}]}}]}}`))
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{
		Provider: ProviderAli, BaseURL: server.URL, ServiceBaseURL: server.URL,
		Credentials: CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	generate := image.GenerateRequest{Model: "qwen-image", Prompt: "draw", User: "ignored", ExtraBody: map[string]any{"user": "forced"}}
	if _, err = client.Images.Generate(context.Background(), &generate); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Images.Stream(context.Background(), &image.StreamRequest{GenerateRequest: generate}); err != nil {
		t.Fatal(err)
	}
	mask := image.Input{ImageURL: "https://example.com/mask.png"}
	edit := image.EditRequest{
		Model: "qwen-image", Prompt: "edit", User: "ignored", Mask: &mask,
		Images: []image.Input{{FileID: "unsupported"}, {FileID: "ignored", ImageURL: "https://example.com/input.png"}},
	}
	if _, err = client.Images.Edit(context.Background(), &edit); err != nil {
		t.Fatal(err)
	}
	if _, err = client.Images.EditStream(context.Background(), &image.EditStreamRequest{EditRequest: edit}); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 4 {
		t.Fatalf("requests = %d", len(bodies))
	}
	for index, body := range bodies {
		if _, exists := body["user"]; exists {
			t.Fatalf("request %d sent structured user at top level: %#v", index, body)
		}
	}
	parameters, ok := bodies[0]["parameters"].(map[string]any)
	if !ok || parameters["user"] != "forced" {
		t.Fatalf("Ali image ExtraBody was not injected into parameters: %#v", bodies[0])
	}
	content := aliImageRequestContent(t, bodies[2])
	if len(content) != 2 || content[0]["text"] != "edit" || content[1]["image"] != "https://example.com/input.png" {
		t.Fatalf("filtered edit content = %#v", content)
	}
}

func TestAliAndOpenRouterRejectEditWithoutSupportedImages(t *testing.T) {
	ali := newAliProvider(ClientConfig{ServiceBaseURL: "https://example.com"})
	_, err := ali.editImage(context.Background(), credential{}, &image.EditRequest{
		Model: "model", Prompt: "edit", Images: []image.Input{{FileID: "file"}},
	})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Ali error = %v", err)
	}
	_, err = openRouterInputImages([]image.Input{{FileID: "file"}})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("OpenRouter error = %v", err)
	}
}

func TestOpenRouterImageIgnoresUnsupportedFieldsButExtraBodyCanInjectThem(t *testing.T) {
	n := 4
	body, err := buildOpenRouterImageBody("model", "draw", nil, &n, "hd", "1024x1024", "ignored", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"n", "quality", "user"} {
		if _, exists := body[key]; exists {
			t.Fatalf("unsupported field %q was sent: %#v", key, body)
		}
	}
	body, err = buildOpenRouterImageBody("model", "draw", nil, &n, "hd", "1024x1024", "ignored", map[string]any{
		"n": 3, "quality": "forced", "user": "forced",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if body["n"] != 3 || body["quality"] != "forced" || body["user"] != "forced" {
		t.Fatalf("ExtraBody was not injected: %#v", body)
	}
}

func aliImageRequestContent(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	input, ok := body["input"].(map[string]any)
	if !ok {
		t.Fatalf("input = %#v", body["input"])
	}
	messages, ok := input["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("messages = %#v", input["messages"])
	}
	message, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("message = %#v", messages[0])
	}
	rawContent, ok := message["content"].([]any)
	if !ok {
		t.Fatalf("content = %#v", message["content"])
	}
	content := make([]map[string]any, len(rawContent))
	for index, raw := range rawContent {
		content[index], ok = raw.(map[string]any)
		if !ok {
			t.Fatalf("content[%d] = %#v", index, raw)
		}
	}
	return content
}

func float64Pointer(value float64) *float64 {
	return &value
}
