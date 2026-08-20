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
			_, _ = io.WriteString(writer, `{"data":[{"url":"`+server.URL+`/generated"}]}`)
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
	if len(response.Data) != 1 || response.Data[0].B64JSON != base64.StdEncoding.EncodeToString(png) || response.Data[0].Format != "png" || response.Data[0].URL != "" {
		t.Fatalf("unexpected image response: %#v", response)
	}
}

func TestOpenRouterImageStreamConvertsDeltaImages(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("image"))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
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
