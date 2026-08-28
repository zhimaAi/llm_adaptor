// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package provider

type Definition struct {
	DefaultBaseURL        string
	DefaultServiceBaseURL string
	CredentialsOptional   bool
	Normalize             func(*Config) error
	New                   func(Config) Implementation
}
