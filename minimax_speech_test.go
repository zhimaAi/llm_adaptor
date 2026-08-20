package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

func TestMiniMaxSpeechCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != MiniMaxSpeechPath {
			t.Errorf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get(headerAuthorization) != bearerPrefix+"test-key" {
			t.Errorf("unexpected authorization header")
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body["stream"] != false || body["custom"] != "value" {
			t.Errorf("unexpected body: %#v", body)
		}
		writer.Header().Set(headerContentType, mediaTypeJSON)
		_, _ = io.WriteString(writer, `{"data":{"audio":"abcd","status":2},"extra_info":{"usage_characters":4},"trace_id":"trace-1","provider_field":"kept","base_resp":{"status_code":0,"status_msg":"success"}}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Provider:    ProviderMiniMax,
		BaseURL:     server.URL,
		Credentials: CredentialConfig{APIKeys: "test-key"},
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Speech.Create(context.Background(), &speech.CreateRequest{
		Model:     "speech-2.8-hd",
		Text:      "test",
		ExtraBody: map[string]any{"custom": "value"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Data == nil || response.Data.Audio != "abcd" || response.TraceID != "trace-1" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if _, exists := response.ExtraFields["provider_field"]; !exists {
		t.Fatalf("provider extension was not preserved: %#v", response.ExtraFields)
	}
}

func TestMiniMaxSpeechStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set(headerContentType, "text/event-stream")
		_, _ = io.WriteString(writer, "data: {\"data\":{\"audio\":\"aa\",\"status\":1},\"trace_id\":\"trace-1\",\"base_resp\":{\"status_code\":0,\"status_msg\":\"success\"}}\n\n")
		_, _ = io.WriteString(writer, "{\"data\":{\"audio\":\"bb\",\"status\":2},\"trace_id\":\"trace-1\",\"base_resp\":{\"status_code\":0,\"status_msg\":\"success\"}}\n")
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{
		Provider:    ProviderMiniMax,
		BaseURL:     server.URL,
		Credentials: CredentialConfig{APIKeys: "test-key"},
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := client.Speech.Stream(context.Background(), &speech.StreamRequest{
		CreateRequest: speech.CreateRequest{Model: "speech-2.8-turbo", Text: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	first, err := stream.Recv()
	if err != nil || first.Data.Audio != "aa" {
		t.Fatalf("unexpected first chunk: %#v, %v", first, err)
	}
	second, err := stream.Recv()
	if err != nil || second.Data.Audio != "bb" {
		t.Fatalf("unexpected second chunk: %#v, %v", second, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestMiniMaxSpeechContextCancellation(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-release
	}))

	client, err := NewClient(ClientConfig{
		Provider:    ProviderMiniMax,
		BaseURL:     server.URL,
		Credentials: CredentialConfig{APIKeys: "test-key"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err = client.Speech.Create(ctx, &speech.CreateRequest{Model: "speech-2.8-hd", Text: "test"})
	close(release)
	server.Close()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}
