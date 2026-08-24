// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package provider

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRequest = errors.New("invalid request")
)

type UnsupportedCapabilityError struct {
	Provider   ID
	Capability Capability
}

func (e *UnsupportedCapabilityError) Error() string {
	return fmt.Sprintf("provider %q does not support capability %q", e.Provider, e.Capability)
}

type APIError struct {
	Provider       ID     `json:"provider"`
	StatusCode     int    `json:"status_code"`
	Code           string `json:"code,omitempty"`
	Type           string `json:"type,omitempty"`
	Param          string `json:"param,omitempty"`
	Message        string `json:"message"`
	RequestID      string `json:"request_id,omitempty"`
	RetryAfter     string `json:"retry_after,omitempty"`
	CredentialHint string `json:"credential_hint,omitempty"`
	Raw            []byte `json:"-"`
	Err            error  `json:"-"`
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
