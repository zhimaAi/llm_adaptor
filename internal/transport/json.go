// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

const (
	HeaderAuthorization        = "Authorization"
	HeaderContentType          = "Content-Type"
	MediaTypeJSON              = "application/json"
	BearerPrefix               = "Bearer "
	APIErrorTypeResponseDecode = "response_decode_error"
	responseDecodeRawMaxBytes  = 16 * 1024
	responseDecodeRawSuffix    = "..."
)

func DecodeJSONResponse(providerID provider.ID, credentialHint string, raw []byte, target any) error {
	if err := json.Unmarshal(raw, target); err != nil {
		return NewResponseDecodeError(providerID, credentialHint, raw, err)
	}
	return nil
}

func NewResponseDecodeError(providerID provider.ID, credentialHint string, raw []byte, err error) error {
	if err == nil {
		return nil
	}
	return &provider.APIError{
		Provider:       providerID,
		Type:           APIErrorTypeResponseDecode,
		Message:        fmt.Sprintf("decode JSON response: %v, raw response: %q", err, responseDecodeRawSummary(raw)),
		CredentialHint: credentialHint,
		Raw:            append([]byte(nil), raw...),
		Err:            err,
	}
}

func responseDecodeRawSummary(raw []byte) string {
	value := strings.ToValidUTF8(string(raw), "\uFFFD")
	if len(value) <= responseDecodeRawMaxBytes {
		return value
	}
	end := responseDecodeRawMaxBytes - len(responseDecodeRawSuffix)
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + responseDecodeRawSuffix
}

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
