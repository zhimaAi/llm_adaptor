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
	client, err := NewClient(ClientConfig{Provider: ProviderOpenCompatible, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}})
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

func TestImageRequestUsesTypedInputAndNormalizesDownloadedBase64(t *testing.T) {
	png := []byte("fake-png")
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/images/generations":
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			images, ok := body["image"].([]any)
			if !ok || len(images) != 1 || images[0] != "https://input.example/a.png" {
				t.Fatalf("typed image input missing: %#v", body)
			}
			_, _ = io.WriteString(writer, `{"data":[{"url":"`+server.URL+`/generated","size":"1024x1024","error":{"code":"","message":""}}]}`)
		case "/generated":
			writer.Header().Set("Content-Type", "image/png")
			_, _ = writer.Write(png)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderOpenCompatible, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Images.Generate(context.Background(), &image.GenerateRequest{Model: "model", Prompt: "draw", Image: []string{"https://input.example/a.png"}, ResponseFormat: "b64_json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 1 || response.Data[0].B64JSON != base64.StdEncoding.EncodeToString(png) || response.Data[0].Format != "png" || response.Data[0].URL != "" || response.Data[0].Size != "1024x1024" {
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
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(writer, `data: {"id":"id","choices":[{"index":0,"delta":{"images":[{"type":"image_url","image_url":{"url":"data:image/png;base64,`+encoded+`"}}]}}]}`+"\n\n")
		_, _ = io.WriteString(writer, "data: [DONE]\n\n")
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderOpenRouter, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := client.Images.Stream(context.Background(), &image.StreamRequest{GenerateRequest: image.GenerateRequest{Model: "model", Prompt: "draw", ResponseFormat: "b64_json"}})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	chunk, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if chunk.B64JSON != encoded || chunk.Format != "png" || chunk.URL != "" {
		t.Fatalf("unexpected image chunk: %#v", chunk)
	}
}

func TestImageResponsePreservesPerItemError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = io.WriteString(writer, `{"data":[{"error":{"code":"content_policy","message":"blocked"}}]}`)
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderOpenCompatible, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Images.Generate(context.Background(), &image.GenerateRequest{Model: "model", Prompt: "draw", ResponseFormat: "b64_json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 1 || response.Data[0].Error.Code != "content_policy" || response.Data[0].Error.Message != "blocked" {
		t.Fatalf("unexpected image error item: %#v", response.Data)
	}
}

func TestImageStreamPreservesPartialFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(writer, `data: {"type":"image_generation.partial_failed","error":{"code":"render_failed","message":"try again"}}`+"\n\n")
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderOpenCompatible, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := client.Images.Stream(context.Background(), &image.StreamRequest{GenerateRequest: image.GenerateRequest{Model: "model", Prompt: "draw"}})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	chunk, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if chunk.Error.Code != "render_failed" || chunk.Error.Message != "try again" {
		t.Fatalf("unexpected partial failure: %#v", chunk)
	}
}

func TestAliImageStreamReturnsEveryImageAndUsageOnce(t *testing.T) {
	response := &image.GenerateResponse{
		Data:  []image.Data{{URL: "one"}, {URL: "two"}, {URL: "three"}},
		Usage: image.Usage{InputTokens: 2, OutputTokens: 3, TotalTokens: 5}, RawResponse: []byte(`{"data":[]}`),
	}
	stream := &singleImageStream{response: response, terminal: newStreamTerminal(nil, nil)}
	for index, wantURL := range []string{"one", "two", "three"} {
		chunk, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if chunk.URL != wantURL {
			t.Fatalf("chunk %d URL = %q, want %q", index, chunk.URL, wantURL)
		}
		if index == 0 {
			if chunk.Usage.TotalTokens != 5 || len(chunk.RawResponse) == 0 {
				t.Fatalf("first chunk lost accounting metadata: %#v", chunk)
			}
		} else if chunk.Usage.TotalTokens != 0 || len(chunk.RawResponse) != 0 {
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
	request := &image.GenerateRequest{ResponseFormat: imageResponseFormatBase64}
	if err := normalizeImageData(context.Background(), ClientConfig{HTTPClient: server.Client()}, ProviderOpenAI, "hint", request, &data); err != nil {
		t.Fatal(err)
	}
	if data.Format != imageFormatJPEG || data.MIMEType != "image/jpeg" {
		t.Fatalf("unexpected normalized image: %#v", data)
	}
}

func TestUnknownImageFormatDefaultsToJPG(t *testing.T) {
	for _, data := range []image.Data{
		{B64JSON: base64.StdEncoding.EncodeToString([]byte("image"))},
		{URL: "https://example.com/generated.bin"},
	} {
		if err := normalizeImageData(context.Background(), ClientConfig{}, ProviderOpenAI, "hint", &image.GenerateRequest{}, &data); err != nil {
			t.Fatal(err)
		}
		if data.Format != imageFormatJPG || data.MIMEType != "image/jpeg" {
			t.Fatalf("unexpected fallback image metadata: %#v", data)
		}
	}
}
