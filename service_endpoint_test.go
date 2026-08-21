// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
)

func TestServiceBaseURLPreservesCustomSubpath(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		wantPath string
		response string
		call     func(context.Context, *Client) error
	}{
		{
			name: "ali rerank", provider: ProviderAli,
			wantPath: "/native/custom/api/v1/services/rerank/text-rerank/text-rerank",
			response: `{"request_id":"id","output":{"results":[{"index":0,"relevance_score":0.9}]},"usage":{"total_tokens":2}}`,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.Rerank.Create(ctx, &rerank.CreateRequest{Model: "model", Query: "q", Documents: []string{"d"}})
				return err
			},
		},
		{
			name: "cohere rerank", provider: ProviderCohere,
			wantPath: "/native/custom/v2/rerank",
			response: `{"results":[{"index":0,"relevance_score":0.9}]}`,
			call: func(ctx context.Context, client *Client) error {
				_, err := client.Rerank.Create(ctx, &rerank.CreateRequest{Model: "model", Query: "q", Documents: []string{"d"}})
				return err
			},
		},
		{
			name: "gemini embedding", provider: ProviderGemini,
			wantPath: "/native/custom/models/model:embedContent",
			response: `{"embedding":{"values":[1,2]}}`,
			call: func(ctx context.Context, client *Client) error {
				text := "hello"
				_, err := client.Embeddings.Create(ctx, &embedding.CreateRequest{Model: "model", Input: embedding.Input{Text: &text}})
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != test.wantPath {
					t.Fatalf("request path = %q, want %q", request.URL.Path, test.wantPath)
				}
				_, _ = io.WriteString(writer, test.response)
			}))
			defer server.Close()
			client, err := NewClient(ClientConfig{Provider: test.provider, BaseURL: server.URL + "/openai/v1", ServiceBaseURL: server.URL + "/native/custom", Credentials: CredentialConfig{APIKeys: "key"}})
			if err != nil {
				t.Fatal(err)
			}
			if err := test.call(context.Background(), client); err != nil {
				t.Fatal(err)
			}
		})
	}
}
