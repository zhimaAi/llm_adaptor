// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"errors"
	"fmt"
	"strings"

	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

var (
	ErrInvalidAPIKeyConfig = errors.New("invalid api key configuration")
	ErrCredentialSelection = errors.New("credential selection failed")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrUnsupportedProvider = errors.New("unsupported provider")
)

type UnsupportedCapabilityError struct {
	Provider   Provider
	Capability Capability
}

func (e *UnsupportedCapabilityError) Error() string {
	return fmt.Sprintf("provider %q does not support capability %q", e.Provider, e.Capability)
}

type APIError struct {
	Provider       Provider `json:"provider"`
	StatusCode     int      `json:"status_code"`
	Code           string   `json:"code,omitempty"`
	Type           string   `json:"type,omitempty"`
	Param          string   `json:"param,omitempty"`
	Message        string   `json:"message"`
	RequestID      string   `json:"request_id,omitempty"`
	RetryAfter     string   `json:"retry_after,omitempty"`
	CredentialHint string   `json:"credential_hint,omitempty"`
	Raw            []byte   `json:"-"`
	Err            error    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "llm api error"
}

func (e *APIError) Unwrap() error { return e.Err }

func normalizeProviderError(err error) error {
	if err == nil {
		return nil
	}
	var apiError *internalprovider.APIError
	if errors.As(err, &apiError) {
		return &APIError{
			Provider: Provider(apiError.Provider), StatusCode: apiError.StatusCode,
			Code: apiError.Code, Type: apiError.Type, Param: apiError.Param, Message: apiError.Message,
			RequestID: apiError.RequestID, RetryAfter: apiError.RetryAfter,
			CredentialHint: apiError.CredentialHint, Raw: append([]byte(nil), apiError.Raw...), Err: apiError.Err,
		}
	}
	var unsupported *internalprovider.UnsupportedCapabilityError
	if errors.As(err, &unsupported) {
		return &UnsupportedCapabilityError{Provider: Provider(unsupported.Provider), Capability: Capability(unsupported.Capability)}
	}
	if errors.Is(err, internalprovider.ErrInvalidRequest) {
		detail := strings.TrimSpace(strings.TrimPrefix(err.Error(), internalprovider.ErrInvalidRequest.Error()))
		detail = strings.TrimSpace(strings.TrimPrefix(detail, ":"))
		if detail == "" {
			return ErrInvalidRequest
		}
		return fmt.Errorf("%w: %s", ErrInvalidRequest, detail)
	}
	return err
}
