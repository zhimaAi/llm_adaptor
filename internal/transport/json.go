// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

const (
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	MediaTypeJSON       = "application/json"
	BearerPrefix        = "Bearer "
)

func NewJSONRequest(ctx context.Context, config provider.Config, authorization, url string, body any) (*http.Request, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return NewRequest(ctx, config, authorization, url, MediaTypeJSON, bytes.NewReader(payload))
}

func NewRequest(ctx context.Context, config provider.Config, authorization, url, contentType string, body io.Reader) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	for key, values := range config.DefaultHeaders {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	if contentType != "" {
		request.Header.Set(HeaderContentType, contentType)
	}
	if authorization != "" {
		request.Header.Set(HeaderAuthorization, authorization)
	}
	return request, nil
}
