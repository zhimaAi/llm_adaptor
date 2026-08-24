// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/rerank"
)

func TestSixRerankProvidersBuildNativeRequests(t *testing.T) {
	tests := []struct {
		name         string
		provider     Provider
		model        string
		apiVersion   string
		wantPath     string
		documentsKey string
		topKey       string
		aliLegacy    bool
	}{
		{name: "siliconflow", provider: ProviderSiliconFlow, model: "BAAI/bge-reranker-v2-m3", wantPath: "/rerank", documentsKey: "documents", topKey: "top_n"},
		{name: "cohere", provider: ProviderCohere, model: "rerank-v3.5", wantPath: "/v2/rerank", documentsKey: "documents", topKey: "top_n"},
		{name: "ali qwen3", provider: ProviderAli, model: "qwen3-rerank", wantPath: aliCompatibleRerankPath, documentsKey: "documents", topKey: "top_n"},
		{name: "ali legacy", provider: ProviderAli, model: "gte-rerank-v2", wantPath: aliRerankPath, aliLegacy: true},
		{name: "baai", provider: ProviderBAAI, model: "bge-reranker", wantPath: "/v1/rerank", documentsKey: "passages", topKey: "top_k"},
		{name: "xinference", provider: ProviderXinference, model: "bge-reranker", apiVersion: "v1", wantPath: "/v1/rerank", documentsKey: "documents", topKey: "top_n"},
		{name: "jina", provider: ProviderJina, model: "jina-reranker-v2-base-multilingual", wantPath: "/rerank", documentsKey: "documents", topKey: "top_n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var body map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != test.wantPath {
					t.Errorf("path = %q, want %q", request.URL.Path, test.wantPath)
				}
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
					t.Errorf("decode request: %v", err)
				}
				if test.aliLegacy {
					_, _ = io.WriteString(writer, `{"request_id":"id","output":{"results":[{"index":1,"relevance_score":0.9}]},"usage":{"total_tokens":7}}`)
					return
				}
				_, _ = io.WriteString(writer, `{"id":"id","results":[{"index":1,"relevance_score":0.9}],"meta":{"api_version":{"version":"2","is_deprecated":false,"is_experimental":false},"billed_units":{"search_units":1},"tokens":{"input_tokens":7},"cached_tokens":2,"warnings":["notice"]},"usage":{"total_tokens":7}}`)
			}))
			defer server.Close()

			config := ClientConfig{
				Provider: test.provider, BaseURL: server.URL, ServiceBaseURL: server.URL,
				APIVersion: test.apiVersion, Credentials: CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client(),
			}
			client, err := NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			topN := 1
			response, err := client.Rerank.Create(context.Background(), &rerank.CreateRequest{
				Model: test.model, Query: "query", Documents: []string{"first", "second"}, TopN: &topN,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(response.Results) != 1 || response.Results[0].Index != 1 || response.Results[0].RelevanceScore != 0.9 {
				t.Fatalf("unexpected response: %#v", response)
			}
			if test.aliLegacy {
				input := body["input"].(map[string]any)
				parameters := body["parameters"].(map[string]any)
				if _, ok := input["documents"].([]any); !ok || parameters["top_n"] != float64(1) {
					t.Fatalf("unexpected Ali legacy body: %#v", body)
				}
				return
			}
			if _, ok := body[test.documentsKey].([]any); !ok || body[test.topKey] != float64(1) {
				t.Fatalf("unexpected request body: %#v", body)
			}
			if test.provider == ProviderCohere {
				if response.Meta == nil || response.Meta.APIVersion == nil || response.Meta.APIVersion.Version != "2" || response.Meta.BilledUnits == nil || response.Meta.BilledUnits.SearchUnits == nil || response.Meta.Tokens == nil || response.Meta.CachedTokens == nil || len(response.Meta.Warnings) != 1 {
					t.Fatalf("Cohere v2 meta was not preserved: %#v", response.Meta)
				}
			}
		})
	}
}

func TestRerankExtraBodyConflictAndInjection(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewDecoder(request.Body).Decode(&body)
		_, _ = io.WriteString(writer, `{"results":[{"index":0,"relevance_score":1}]}`)
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{Provider: ProviderJina, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key"}, HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	request := &rerank.CreateRequest{Model: "model", Query: "q", Documents: []string{"d"}, ExtraBody: map[string]any{"priority": 1}}
	if _, err := client.Rerank.Create(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if body["priority"] != float64(1) || request.ExtraBody["priority"] != 1 {
		t.Fatalf("ExtraBody injection or immutability failed: body=%#v request=%#v", body, request)
	}
	request.ExtraBody = map[string]any{"top_n": 2}
	if _, err := client.Rerank.Create(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if body["top_n"] != float64(2) {
		t.Fatalf("ExtraBody top_n did not override provider value: %#v", body)
	}
}
