package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/embedding"
)

type geminiProvider struct{ *openAICompatibleProvider }

func newGeminiProvider(config ClientConfig) *geminiProvider {
	return &geminiProvider{newOpenAICompatibleProvider(config, ProviderInfo{
		ID: ProviderGemini, DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		Capabilities: []Capability{CapabilityChat, CapabilityEmbedding},
	})}
}

func (p *geminiProvider) createEmbedding(ctx context.Context, selected credential, request *embedding.CreateRequest) (*embedding.CreateResponse, error) {
	if request == nil || request.Model == "" {
		return nil, fmt.Errorf("%w: embedding model is required", ErrInvalidRequest)
	}
	texts := embeddingTexts(request.Input)
	if len(texts) == 0 {
		return nil, fmt.Errorf("%w: embedding input is required", ErrInvalidRequest)
	}
	result := &embedding.CreateResponse{Object: "list", Model: request.Model, Data: make([]embedding.Data, 0, len(texts))}
	for index, text := range texts {
		body := map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": text}}}}
		for key, value := range request.ExtraBody {
			body[key] = value
		}
		endpoint, err := p.geminiEmbeddingURL(request.Model, selected.apiKey)
		if err != nil {
			return nil, err
		}
		httpRequest, err := newJSONRequest(ctx, p.config, "", endpoint, body)
		if err != nil {
			return nil, err
		}
		response, err := p.config.HTTPClient.Do(httpRequest)
		if err != nil {
			return nil, err
		}
		if err := checkHTTPResponse(ProviderGemini, selected.hint, response); err != nil {
			response.Body.Close()
			return nil, err
		}
		raw, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			return nil, err
		}
		var source struct {
			Embedding struct {
				Values []float64 `json:"values"`
			} `json:"embedding"`
		}
		if err := json.Unmarshal(raw, &source); err != nil {
			return nil, err
		}
		result.Data = append(result.Data, embedding.Data{Object: "embedding", Embedding: source.Embedding.Values, Index: index})
		result.RawResponse = raw
	}
	return result, nil
}

func (p *geminiProvider) geminiEmbeddingURL(model, apiKey string) (string, error) {
	serviceBase := p.config.BaseURL
	if value, ok := p.config.Extra["service_base_url"].(string); ok && strings.TrimSpace(value) != "" {
		serviceBase = strings.TrimRight(strings.TrimSpace(value), "/")
	} else if index := strings.Index(serviceBase, "/openai"); index >= 0 {
		serviceBase = serviceBase[:index]
	}
	endpoint, err := url.Parse(strings.TrimRight(serviceBase, "/") + "/models/" + url.PathEscape(model) + ":embedContent")
	if err != nil {
		return "", err
	}
	query := endpoint.Query()
	query.Set("key", apiKey)
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}

func embeddingTexts(input embedding.Input) []string {
	if input.Text != nil {
		return []string{*input.Text}
	}
	return input.Texts
}

var _ embeddingProvider = (*geminiProvider)(nil)
