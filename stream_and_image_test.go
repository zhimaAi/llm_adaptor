// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/image"
)

func TestOpenAIStreamReturnsAPIErrorOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(writer, `data: {"error":{"code":"rate_limit","type":"rate_limit_error","message":"slow down"}}`+"\n\n")
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderOpenAIAgent, BaseURL: server.URL, APIVersion: "v1", Credentials: CredentialConfig{APIKeys: "key"}})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := client.Chat.Stream(context.Background(), &chat.StreamRequest{CreateRequest: chat.CreateRequest{Model: "model", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}}}})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	_, err = stream.Recv()
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.Code != "rate_limit" || apiError.Message != "slow down" {
		t.Fatalf("unexpected stream error: %#v", err)
	}
	if _, err = stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("second Recv error = %v, want EOF", err)
	}
}

func TestImageEditUsesTypedInputAndNormalizesDownloadedBase64(t *testing.T) {
	png := []byte("fake-png")
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v1/images/edits":
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			images, ok := body["images"].([]any)
			if !ok || len(images) != 1 {
				t.Fatalf("typed image input missing: %#v", body)
			}
			_, _ = io.WriteString(writer, `{"data":[{"url":"`+server.URL+`/generated"}],"size":"1024x1024"}`)
		case "/generated":
			writer.Header().Set("Content-Type", "image/png")
			_, _ = writer.Write(png)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderOpenAIAgent, BaseURL: server.URL, APIVersion: "v1", Credentials: CredentialConfig{APIKeys: "key"}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Images.Edit(context.Background(), &image.EditRequest{Model: "model", Prompt: "draw", Images: []image.Input{{ImageURL: "https://input.example/a.png"}}, ResponseFormat: "b64_json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 1 || response.Data[0].B64JSON != base64.StdEncoding.EncodeToString(png) || response.Data[0].URL != "" || response.OutputFormat != "png" || response.Size != "1024x1024" {
		t.Fatalf("unexpected image response: %#v", response)
	}
}

func TestOpenRouterImageStreamConvertsDeltaImages(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("image"))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		modalities, _ := body["modalities"].([]any)
		streamOptions, _ := body["stream_options"].(map[string]any)
		if len(modalities) != 2 || streamOptions["include_usage"] != true {
			t.Errorf("missing typed modalities or default usage: %#v", body)
			return
		}
		for _, key := range []string{"n", "quality", "user"} {
			if _, exists := body[key]; exists {
				t.Errorf("unsupported field %q was sent: %#v", key, body)
				return
			}
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(writer, `data: {"id":"id","choices":[{"index":0,"delta":{"images":[{"type":"image_url","image_url":{"url":"data:image/png;base64,`+encoded+`"}}]}}]}`+"\n\n")
		_, _ = io.WriteString(writer, "data: [DONE]\n\n")
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderOpenRouter, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}})
	if err != nil {
		t.Fatal(err)
	}
	n := 4
	stream, err := client.Images.Stream(context.Background(), &image.StreamRequest{GenerateRequest: image.GenerateRequest{
		Model: "model", Prompt: "draw", N: &n, Quality: "hd", User: "ignored", ResponseFormat: "b64_json",
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	chunk, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if chunk.B64JSON != encoded || chunk.OutputFormat != "png" || chunk.Quality != "" {
		t.Fatalf("unexpected image chunk: %#v", chunk)
	}
}

func TestImageResponseRejectsMissingImageData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(writer, `{"data":[{"error":{"code":"content_policy","message":"blocked"}}]}`)
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderOpenAIAgent, BaseURL: server.URL, APIVersion: "v1", Credentials: CredentialConfig{APIKeys: "key"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Images.Generate(context.Background(), &image.GenerateRequest{Model: "model", Prompt: "draw", ResponseFormat: "b64_json"})
	if err == nil {
		t.Fatal("expected missing base64 image error")
	}
}

func TestImageStreamReturnsPartialFailureOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(writer, `data: {"type":"image_generation.partial_failed","error":{"code":"render_failed","message":"try again"}}`+"\n\n")
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderOpenAIAgent, BaseURL: server.URL, APIVersion: "v1", Credentials: CredentialConfig{APIKeys: "key"}})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := client.Images.Stream(context.Background(), &image.StreamRequest{GenerateRequest: image.GenerateRequest{Model: "model", Prompt: "draw"}})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	_, err = stream.Recv()
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.Code != "render_failed" {
		t.Fatalf("unexpected partial failure: %#v", err)
	}
	if _, err = stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("second Recv error = %v, want EOF", err)
	}
}

func TestAliImageStreamReturnsEveryImageAndUsageOnce(t *testing.T) {
	response := &image.GenerateResponse{
		Data:  []image.Data{{B64JSON: "one"}, {B64JSON: "two"}, {B64JSON: "three"}},
		Usage: image.Usage{InputTokens: 2, OutputTokens: 3, TotalTokens: 5},
	}
	stream := &singleImageStream{response: response, terminal: newStreamTerminal(nil, nil)}
	for index, wantBase64 := range []string{"one", "two", "three"} {
		chunk, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if chunk.B64JSON != wantBase64 {
			t.Fatalf("chunk %d B64JSON = %q, want %q", index, chunk.B64JSON, wantBase64)
		}
		if index == 0 {
			if chunk.Usage.TotalTokens != 5 {
				t.Fatalf("first chunk lost accounting metadata: %#v", chunk)
			}
		} else if chunk.Usage.TotalTokens != 0 {
			t.Fatalf("chunk %d duplicated accounting metadata: %#v", index, chunk)
		}
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestDownloadedImageFormatUsesOriginalURLSuffix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/octet-stream")
		_, _ = writer.Write([]byte("image"))
	}))
	defer server.Close()
	data := image.Data{URL: server.URL + "/generated.JPG?download=1"}
	request := imageRequestOptions{ResponseFormat: imageResponseFormatBase64}
	format, err := normalizeImageData(context.Background(), ClientConfig{HTTPClient: server.Client()}, ProviderOpenAI, "hint", request, &data)
	if err != nil {
		t.Fatal(err)
	}
	if format != imageFormatJPEG {
		t.Fatalf("unexpected normalized image: %#v", data)
	}
}

func TestUnknownImageFormatDefaultsToJPG(t *testing.T) {
	for _, data := range []image.Data{
		{B64JSON: base64.StdEncoding.EncodeToString([]byte("image"))},
		{URL: "https://example.com/generated.bin"},
	} {
		format, err := normalizeImageData(context.Background(), ClientConfig{}, ProviderOpenAI, "hint", imageRequestOptions{}, &data)
		if err != nil {
			t.Fatal(err)
		}
		if format != imageFormatJPEG {
			t.Fatalf("unexpected fallback image metadata: %#v", data)
		}
	}
}
