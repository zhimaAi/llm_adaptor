// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func CheckHTTPResponse(providerID provider.ID, credentialHint string, response *http.Response) error {
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusBadRequest {
		return nil
	}
	raw, readErr := io.ReadAll(response.Body)
	apiError := &provider.APIError{
		Provider:       providerID,
		StatusCode:     response.StatusCode,
		RequestID:      response.Header.Get("x-request-id"),
		RetryAfter:     response.Header.Get("retry-after"),
		CredentialHint: credentialHint,
		Raw:            raw,
		Err:            readErr,
	}
	var decoded struct {
		Error struct {
			Code    json.RawMessage `json:"code"`
			Type    string          `json:"type"`
			Param   string          `json:"param"`
			Message string          `json:"message"`
		} `json:"error"`
		BaseResponse struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	if len(raw) > 0 && json.Unmarshal(raw, &decoded) == nil {
		apiError.Code = decodeAPIErrorCode(decoded.Error.Code)
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

func decodeAPIErrorCode(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	return strings.TrimSpace(string(raw))
}
