// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidAPIKeyConfig = errors.New("invalid api key configuration")
	ErrCredentialSelection = errors.New("credential selection failed")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrNilContext          = errors.New("context is nil")
	ErrUnsupportedProvider = errors.New("unsupported provider")
)

type UnsupportedCapabilityError struct {
	Provider   Provider
	Capability Capability
}

type UnsupportedParameterError struct {
	Provider   Provider
	Capability Capability
	Parameter  string
}

func (e *UnsupportedParameterError) Error() string {
	return fmt.Sprintf("provider %q does not support parameter %q for capability %q", e.Provider, e.Parameter, e.Capability)
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

func (e *APIError) Unwrap() error {
	return e.Err
}
