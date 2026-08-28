// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package transport

import (
	"encoding/json"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func DecodeStreamAPIError(providerID provider.ID, credentialHint string, raw []byte) error {
	var envelope struct {
		Type    string          `json:"type"`
		Code    json.RawMessage `json:"code"`
		Message string          `json:"message"`
		Error   json.RawMessage `json:"error"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &envelope) != nil {
		return nil
	}
	if (len(envelope.Error) == 0 || string(envelope.Error) == "null") && !strings.EqualFold(envelope.Type, "error") {
		return nil
	}
	apiError := &provider.APIError{Provider: providerID, CredentialHint: credentialHint, Raw: append([]byte(nil), raw...), Type: envelope.Type, Message: envelope.Message}
	decodeCode := func(value json.RawMessage) string {
		var code string
		if json.Unmarshal(value, &code) == nil {
			return code
		}
		return strings.Trim(string(value), `"`)
	}
	if len(envelope.Code) > 0 {
		apiError.Code = decodeCode(envelope.Code)
	}
	if len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		var message string
		if json.Unmarshal(envelope.Error, &message) == nil {
			apiError.Message = message
		} else {
			var detail struct {
				Code    json.RawMessage `json:"code"`
				Type    string          `json:"type"`
				Param   string          `json:"param"`
				Message string          `json:"message"`
			}
			if json.Unmarshal(envelope.Error, &detail) == nil {
				apiError.Type, apiError.Param, apiError.Message = detail.Type, detail.Param, detail.Message
				if len(detail.Code) > 0 {
					apiError.Code = decodeCode(detail.Code)
				}
			}
		}
	}
	if apiError.Message == "" {
		apiError.Message = "stream returned an API error"
	}
	return apiError
}
