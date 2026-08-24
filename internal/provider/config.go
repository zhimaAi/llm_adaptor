// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package provider

import "net/http"

type CredentialConfig struct {
	APIKeys string `json:"api_keys"`
}

type Config struct {
	Provider       ID               `json:"provider"`
	BaseURL        string           `json:"base_url,omitempty"`
	ServiceBaseURL string           `json:"service_base_url,omitempty"`
	Credentials    CredentialConfig `json:"credentials"`
	APIVersion     string           `json:"api_version,omitempty"`
	HTTPClient     *http.Client     `json:"-"`
	DefaultHeaders http.Header      `json:"-"`
}

type Info struct {
	ID                    ID           `json:"id"`
	DefaultBaseURL        string       `json:"default_base_url,omitempty"`
	DefaultServiceBaseURL string       `json:"default_service_base_url,omitempty"`
	Capabilities          []Capability `json:"capabilities"`
}
