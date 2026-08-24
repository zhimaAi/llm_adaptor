// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

const (
	streamInitialBuffer = 64 * 1024
	streamMaximumBuffer = 16 * 1024 * 1024
	ChatPath            = "/chat/completions"
	EmbeddingPath       = "/embeddings"
	ImagePath           = "/images/generations"
	ImageEditPath       = "/images/edits"
)

type Provider struct {
	config provider.Config
	spec   Spec
}

func New(config provider.Config, spec Spec) *Provider {
	return &Provider{config: config, spec: spec}
}

func (p *Provider) Info() provider.Info { return p.spec.Info }

func (p *Provider) Config() provider.Config { return p.config }

func (p *Provider) DoJSON(ctx context.Context, selected provider.Credential, path string, body any) ([]byte, error) {
	response, err := p.DoStream(ctx, selected, path, body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}

func (p *Provider) DoStream(ctx context.Context, selected provider.Credential, path string, body any) (*http.Response, error) {
	endpoint := path
	if !strings.HasPrefix(path, "https://") && !strings.HasPrefix(path, "http://") {
		endpoint = transport.JoinURLPath(p.config.BaseURL, path)
	}
	request, err := transport.NewJSONRequest(ctx, p.config, "", endpoint, body)
	if err != nil {
		return nil, err
	}
	if p.spec.AuthorizationPrefix != nil && selected.APIKey != "" {
		request.Header.Set(p.spec.AuthorizationHeader, *p.spec.AuthorizationPrefix+selected.APIKey)
	}
	response, err := p.config.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	if err := transport.CheckHTTPResponse(p.spec.Info.ID, selected.Hint, response); err != nil {
		response.Body.Close()
		return nil, err
	}
	return response, nil
}
