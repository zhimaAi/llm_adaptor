// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package provider

type Capability string

const (
	CapabilityChat      Capability = "chat"
	CapabilityEmbedding Capability = "embedding"
	CapabilityImage     Capability = "image"
	CapabilityRerank    Capability = "rerank"
	CapabilitySpeech    Capability = "speech"
)
