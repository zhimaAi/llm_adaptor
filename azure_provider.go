// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
)

const (
	azureAPIKeyHeader = "api-key"
	azureAPIVersion   = "api-version"
)

type azureProvider struct{ config ClientConfig }

func (p *azureProvider) info() ProviderInfo {
	return ProviderInfo{ID: ProviderAzure, Capabilities: []Capability{CapabilityChat, CapabilityEmbedding}}
}

func (p *azureProvider) createChat(ctx context.Context, selected credential, request *chat.CreateRequest) (*chat.CreateResponse, error) {
	if request == nil || request.Model == "" || len(request.Messages) == 0 {
		return nil, fmt.Errorf("%w: model and messages are required", ErrInvalidRequest)
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	delete(body, "model")
	body["stream"] = false
	raw, err := p.do(ctx, selected, request.Model, "chat/completions", body, false)
	if err != nil {
		return nil, err
	}
	result := &chat.CreateResponse{}
	if err := json.Unmarshal(raw, result); err != nil {
		return nil, err
	}
	result.RawResponse = append(result.RawResponse[:0], raw...)
	normalizeThinkTaggedResponse(result)
	return result, nil
}

func (p *azureProvider) streamChat(ctx context.Context, selected credential, request *chat.StreamRequest) (chat.Stream, error) {
	if request == nil || request.Model == "" || len(request.Messages) == 0 {
		return nil, fmt.Errorf("%w: model and messages are required", ErrInvalidRequest)
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	delete(body, "model")
	body["stream"] = true
	streamContext, cancel := context.WithCancel(ctx)
	response, err := p.doResponse(streamContext, selected, request.Model, "chat/completions", body)
	if err != nil {
		cancel()
		return nil, err
	}
	return newThinkTagStream(newOpenAIChatStream(response.Body, cancel, ProviderAzure, selected.hint)), nil
}

func (p *azureProvider) createEmbedding(ctx context.Context, selected credential, request *embedding.CreateRequest) (*embedding.CreateResponse, error) {
	if request == nil || request.Model == "" {
		return nil, fmt.Errorf("%w: embedding model is required", ErrInvalidRequest)
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	delete(body, "model")
	raw, err := p.do(ctx, selected, request.Model, "embeddings", body, false)
	if err != nil {
		return nil, err
	}
	result := &embedding.CreateResponse{}
	if err := json.Unmarshal(raw, result); err != nil {
		return nil, err
	}
	result.RawResponse = append(result.RawResponse[:0], raw...)
	return result, nil
}

func (p *azureProvider) do(ctx context.Context, selected credential, deployment, operation string, body any, _ bool) ([]byte, error) {
	response, err := p.doResponse(ctx, selected, deployment, operation, body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}

func (p *azureProvider) doResponse(ctx context.Context, selected credential, deployment, operation string, body any) (*http.Response, error) {
	endpoint := strings.TrimRight(p.config.BaseURL, "/") + "/openai/deployments/" + url.PathEscape(deployment) + "/" + operation
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	query.Set(azureAPIVersion, p.config.APIVersion)
	parsed.RawQuery = query.Encode()
	request, err := newJSONRequest(ctx, p.config, "", parsed.String(), body)
	if err != nil {
		return nil, err
	}
	request.Header.Set(azureAPIKeyHeader, selected.apiKey)
	response, err := p.config.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPResponse(ProviderAzure, selected.hint, response); err != nil {
		response.Body.Close()
		return nil, err
	}
	return response, nil
}

var _ chatProvider = (*azureProvider)(nil)
var _ embeddingProvider = (*azureProvider)(nil)
