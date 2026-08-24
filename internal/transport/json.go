// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package transport

import (
	"bytes"
	"context"
	"encoding/json"
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
	if ctx == nil {
		ctx = context.Background()
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for key, values := range config.DefaultHeaders {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	request.Header.Set(HeaderContentType, MediaTypeJSON)
	if authorization != "" {
		request.Header.Set(HeaderAuthorization, authorization)
	}
	return request, nil
}
