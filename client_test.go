// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	llm "github.com/zhimaAi/llm_adaptor/v2"
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

func TestPublicClientCapabilities(t *testing.T) {
	t.Run("chat create and stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/v1/chat/completions" {
				t.Errorf("path = %q", request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			if stream, _ := body["stream"].(bool); stream {
				writer.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(writer, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"<think>plan</think>answer\"}}]}\n\ndata: [DONE]\n\n")
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"id":"id","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"answer"}}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
		}))
		defer server.Close()

		client := newClient(t, llm.ClientConfig{Provider: llm.ProviderOpenAIAgent, BaseURL: server.URL, APIVersion: "v1", HTTPClient: server.Client()})
		request := chat.CreateRequest{Model: "model", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}}}
		response, err := client.Chat.Create(nil, &request)
		if err != nil {
			t.Fatal(err)
		}
		if len(response.Choices) != 1 || response.Choices[0].Message.Content.Text == nil || *response.Choices[0].Message.Content.Text != "answer" {
			t.Fatalf("unexpected response: %#v", response)
		}

		stream, err := client.Chat.Stream(nil, &chat.StreamRequest{CreateRequest: request})
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		chunk, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if len(chunk.Choices) != 1 || chunk.Choices[0].Delta.Content.Text == nil || *chunk.Choices[0].Delta.Content.Text != "answer" || chunk.Choices[0].Delta.ReasoningContent != "plan" {
			t.Fatalf("unexpected chunk: %#v", chunk)
		}
		if _, err = stream.Recv(); err != io.EOF {
			t.Fatalf("stream end error = %v, want io.EOF", err)
		}
	})

	t.Run("embedding", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/v1/embeddings" {
				t.Errorf("path = %q", request.URL.Path)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"object":"list","data":[{"object":"embedding","index":0,"embedding":[1,2]}]}`)
		}))
		defer server.Close()

		client := newClient(t, llm.ClientConfig{Provider: llm.ProviderBAAI, BaseURL: server.URL, HTTPClient: server.Client()})
		input := "hello"
		response, err := client.Embeddings.Create(nil, &embedding.CreateRequest{Model: "bge-m3", Input: embedding.Input{Text: &input}})
		if err != nil {
			t.Fatal(err)
		}
		values, err := response.Data[0].Embedding.Float64s()
		if err != nil || len(values) != 2 || values[0] != 1 || values[1] != 2 {
			t.Fatalf("unexpected embedding: %#v, %v", values, err)
		}
	})

	t.Run("image generate and edit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			switch request.URL.Path {
			case "/images/generations":
				var body map[string]any
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["prompt"] != "draw" {
					t.Errorf("body = %#v", body)
				}
			case "/images/edits":
				if err := request.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
					return
				}
				files := request.MultipartForm.File["image"]
				if request.FormValue("prompt") != "edit" || len(files) != 1 || files[0].Filename != "input.png" {
					t.Errorf("form = %#v, files = %#v", request.MultipartForm.Value, files)
				}
			default:
				t.Errorf("path = %q", request.URL.Path)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"data":[{"b64_json":"aGVsbG8="}],"output_format":"png"}`)
		}))
		defer server.Close()

		client := newClient(t, llm.ClientConfig{Provider: llm.ProviderOpenAI, BaseURL: server.URL, Credentials: llm.CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client()})
		generated, err := client.Images.Generate(nil, &image.GenerateRequest{Model: "gpt-image", Prompt: "draw", ResponseFormat: "b64_json", OutputFormat: "png"})
		if err != nil || len(generated.Data) != 1 || generated.Data[0].B64JSON == "" {
			t.Fatalf("unexpected generated image: %#v, %v", generated, err)
		}
		edited, err := client.Images.Edit(nil, &image.EditRequest{
			Model: "gpt-image", Prompt: "edit", ResponseFormat: "b64_json", OutputFormat: "png",
			Images: []image.File{{Filename: "input.png", ContentType: "image/png", Reader: bytes.NewBufferString("image")}},
		})
		if err != nil || len(edited.Data) != 1 || edited.Data[0].B64JSON == "" {
			t.Fatalf("unexpected edited image: %#v, %v", edited, err)
		}
	})

	t.Run("rerank", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/v1/rerank" {
				t.Errorf("path = %q", request.URL.Path)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"id":"id","results":[{"index":1,"relevance_score":0.9}]}`)
		}))
		defer server.Close()

		client := newClient(t, llm.ClientConfig{Provider: llm.ProviderBAAI, BaseURL: server.URL, HTTPClient: server.Client()})
		response, err := client.Rerank.Create(nil, &rerank.CreateRequest{Model: "bge-reranker", Query: "q", Documents: []string{"a", "b"}})
		if err != nil || len(response.Results) != 1 || response.Results[0].Index != 1 {
			t.Fatalf("unexpected rerank response: %#v, %v", response, err)
		}
	})

	t.Run("speech create and stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/t2a_v2" {
				t.Errorf("path = %q", request.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			if stream, _ := body["stream"].(bool); stream {
				writer.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(writer, "data: {\"data\":{\"audio\":\"00\",\"status\":1},\"base_resp\":{\"status_code\":0}}\n\ndata: {\"data\":{\"audio\":\"\",\"status\":2},\"base_resp\":{\"status_code\":0}}\n\n")
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(writer, `{"data":{"audio":"00ff","status":2},"trace_id":"trace","base_resp":{"status_code":0}}`)
		}))
		defer server.Close()

		client := newClient(t, llm.ClientConfig{Provider: llm.ProviderMiniMax, BaseURL: server.URL, Credentials: llm.CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client()})
		request := speech.CreateRequest{Model: "speech-2.8-hd", Text: "hello"}
		response, err := client.Speech.Create(nil, &request)
		if err != nil || response.Data == nil || response.Data.Status != speech.AudioStatusComplete {
			t.Fatalf("unexpected speech response: %#v, %v", response, err)
		}

		stream, err := client.Speech.Stream(nil, &speech.StreamRequest{CreateRequest: request})
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		first, err := stream.Recv()
		if err != nil || first.Data == nil || first.Data.Status != speech.AudioStatusStreaming {
			t.Fatalf("unexpected streaming chunk: %#v, %v", first, err)
		}
		last, err := stream.Recv()
		if err != nil || last.Data == nil || last.Data.Status != speech.AudioStatusComplete {
			t.Fatalf("unexpected complete chunk: %#v, %v", last, err)
		}
		if _, err = stream.Recv(); err != io.EOF {
			t.Fatalf("stream end error = %v, want io.EOF", err)
		}
	})

	t.Run("api key selection", func(t *testing.T) {
		response, err := llm.SelectAPIKey(llm.SelectAPIKeyRequest{Credentials: llm.CredentialConfig{APIKeys: "only-key@10"}})
		if err != nil || response.APIKey != "only-key" || response.CredentialHint == "" {
			t.Fatalf("unexpected selection: %#v, %v", response, err)
		}
	})
}

func newClient(t *testing.T, config llm.ClientConfig) *llm.Client {
	t.Helper()
	client, err := llm.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	return client
}
