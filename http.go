// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	headerAuthorization = "Authorization"
	headerContentType   = "Content-Type"
	mediaTypeJSON       = "application/json"
	bearerPrefix        = "Bearer "
)

func newJSONRequest(ctx context.Context, config ClientConfig, authorization, url string, body any) (*http.Request, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for key, values := range config.DefaultHeaders {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set(headerContentType, mediaTypeJSON)
	if authorization != "" {
		req.Header.Set(headerAuthorization, authorization)
	}
	return req, nil
}

func checkHTTPResponse(provider Provider, credentialHint string, response *http.Response) error {
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusBadRequest {
		return nil
	}
	raw, readErr := io.ReadAll(response.Body)
	apiError := &APIError{
		Provider:       provider,
		StatusCode:     response.StatusCode,
		RequestID:      response.Header.Get("x-request-id"),
		RetryAfter:     response.Header.Get("retry-after"),
		CredentialHint: credentialHint,
		Raw:            raw,
		Err:            readErr,
	}
	var decoded struct {
		Error struct {
			Code    string `json:"code"`
			Type    string `json:"type"`
			Param   string `json:"param"`
			Message string `json:"message"`
		} `json:"error"`
		BaseResponse struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	if len(raw) > 0 && json.Unmarshal(raw, &decoded) == nil {
		apiError.Code = decoded.Error.Code
		apiError.Type = decoded.Error.Type
		apiError.Param = decoded.Error.Param
		apiError.Message = decoded.Error.Message
		if apiError.Message == "" {
			apiError.Message = decoded.BaseResponse.StatusMsg
			apiError.Code = fmt.Sprint(decoded.BaseResponse.StatusCode)
		}
	}
	if apiError.Message == "" {
		apiError.Message = response.Status
	}
	return apiError
}

func mergeExtraBody(value any, extra map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any)
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	for key, item := range extra {
		result[key] = item
	}
	return result, nil
}
