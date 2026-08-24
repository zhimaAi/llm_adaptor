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

func TestOpenAIBuilderDropsToolControlsWhenAllToolsAreUnsupported(t *testing.T) {
	parallel := true
	body, err := buildOpenAIChatRequest(ProviderOpenAI, &chat.CreateRequest{
		Model: "model", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}},
		Tools: []chat.Tool{{Type: "future_tool"}}, ToolChoice: "required", ParallelToolCalls: &parallel,
	}, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"tools", "tool_choice", "parallel_tool_calls"} {
		if _, exists := body[key]; exists {
			t.Fatalf("filtered tool control %q was sent: %#v", key, body)
		}
	}
}

func TestClaudeBuilderIgnoresUnsupportedPublicFields(t *testing.T) {
	value := 1
	seed := int64(2)
	parallel := true
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
		ToolChoice: "future_choice", ParallelToolCalls: &parallel,
		Tools: []chat.Tool{{Type: "future_tool"}},
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

func TestClaudeBuilderRejectsInvalidMixedSystemContent(t *testing.T) {
	content := chat.TextContent("text")
	content.Parts = []chat.ContentPart{{Type: chat.ContentPartText, Text: "part"}}
	_, err := buildClaudeRequest(&chat.CreateRequest{
		Model: "claude-sonnet",
		Messages: []chat.Message{
			{Role: chat.RoleSystem, Content: content},
			{Role: chat.RoleUser, Content: chat.TextContent("hello")},
		},
	}, false)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v", err)
	}
}

func TestClaudeBuilderIgnoresUnsupportedRolesAndToolCalls(t *testing.T) {
	body, err := buildClaudeRequest(&chat.CreateRequest{
		Model: "claude-sonnet",
		Messages: []chat.Message{
			{Role: chat.Role("future_role"), Content: chat.TextContent("ignored")},
			{Role: chat.RoleUser, Content: chat.TextContent("hello")},
			{Role: chat.RoleAssistant, ToolCalls: []chat.ToolCall{{Type: "future_tool"}}, Content: chat.TextContent("kept")},
		},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	messages, ok := body["messages"].([]claudeRequestMessage)
	if !ok || len(messages) != 2 {
		t.Fatalf("messages = %#v", body["messages"])
	}
	if len(messages[1].Content) != 1 || messages[1].Content[0].Type != claudeContentText || messages[1].Content[0].Text != "kept" {
		t.Fatalf("assistant content = %#v", messages[1].Content)
	}

	_, err = buildClaudeRequest(&chat.CreateRequest{
		Model: "claude-sonnet", Messages: []chat.Message{{Role: chat.Role("future_role"), Content: chat.TextContent("ignored")}},
	}, false)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v", err)
	}
}

func TestClaudeToolResultKeepsSupportedTextParts(t *testing.T) {
	converted, err := convertClaudeMessage(chat.Message{
		Role: chat.RoleTool, ToolCallID: "call-1",
		Content: chat.PartsContent(
			chat.ContentPart{Type: chat.ContentPartText, Text: "kept"},
			chat.ContentPart{Type: chat.ContentPartVideoURL, VideoURL: &chat.VideoURL{URL: "https://example.com/video.mp4"}},
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(converted.Content) != 1 || converted.Content[0].Content != "kept" {
		t.Fatalf("content = %#v", converted.Content)
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
	generate := image.GenerateRequest{Model: "qwen-image", Prompt: "draw", User: "ignored", Quality: "hd", ExtraBody: map[string]any{"user": "forced"}}
	response, err := client.Images.Generate(context.Background(), &generate)
	if err != nil {
		t.Fatal(err)
	}
	if response.Quality != "" {
		t.Fatalf("unsupported quality was copied to response: %#v", response)
	}
	stream, err := client.Images.Stream(context.Background(), &image.StreamRequest{GenerateRequest: generate})
	if err != nil {
		t.Fatal(err)
	}
	chunk, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if chunk.Quality != "" {
		t.Fatalf("unsupported quality was copied to stream chunk: %#v", chunk)
	}
	if err = stream.Close(); err != nil {
		t.Fatal(err)
	}
	mask := image.Input{ImageURL: "https://example.com/mask.png"}
	edit := image.EditRequest{
		Model: "qwen-image", Prompt: "edit", User: "ignored", Quality: "hd", Mask: &mask,
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
	if _, exists := parameters["quality"]; exists {
		t.Fatalf("unsupported quality was sent: %#v", parameters)
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
	body, err := buildOpenRouterImageBody("model", "draw", nil, "1024x1024", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"n", "quality", "user"} {
		if _, exists := body[key]; exists {
			t.Fatalf("unsupported field %q was sent: %#v", key, body)
		}
	}
	body, err = buildOpenRouterImageBody("model", "draw", nil, "1024x1024", map[string]any{
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
