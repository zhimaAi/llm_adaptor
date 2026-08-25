// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

type FieldSet map[string]struct{}

func Fields(fields ...string) FieldSet {
	result := make(FieldSet, len(fields))
	for _, field := range fields {
		result[field] = struct{}{}
	}
	return result
}

func FieldsWithout(source FieldSet, fields ...string) FieldSet {
	result := make(FieldSet, len(source))
	for field := range source {
		result[field] = struct{}{}
	}
	for _, field := range fields {
		delete(result, field)
	}
	return result
}

var AllChatFields = Fields(
	"frequency_penalty", "max_tokens", "max_completion_tokens", "n", "parallel_tool_calls",
	"presence_penalty", "response_format", "seed", "stop", "temperature", "tool_choice",
	"tools", "top_p", "user", "stream_options",
)

var AllEmbeddingFields = Fields("encoding_format", "dimensions", "user")
var AllImageFields = Fields("n", "quality", "response_format", "size", "user", "output_format", "mask")

type ReasoningFunc func(model string, effort chat.ReasoningEffort, body map[string]any)
type ConfigureFunc func(*Spec, provider.Config)

type Spec struct {
	Info                provider.Info
	AuthorizationHeader string
	AuthorizationPrefix *string
	ChatPath            string
	EmbeddingPath       string
	ImagePath           string
	ImageEditPath       string
	RerankPath          string
	RerankBaseURL       string
	RerankDocumentsKey  string
	RerankTopKey        string
	ChatFields          FieldSet
	EmbeddingFields     FieldSet
	ImageFields         FieldSet
	ChatFieldAliases    map[string]string
	EmbeddingAliases    map[string]string
	SupportsInputAudio  bool
	SupportsVideoURL    bool
	ApplyReasoning      ReasoningFunc
}

func DefaultSpec(info provider.Info) Spec {
	prefix := "Bearer "
	return Spec{
		Info:                info,
		AuthorizationHeader: "Authorization",
		AuthorizationPrefix: &prefix,
		ChatPath:            ChatPath,
		EmbeddingPath:       EmbeddingPath,
		ImagePath:           ImagePath,
		ImageEditPath:       ImageEditPath,
		ChatFields:          AllChatFields,
		EmbeddingFields:     AllEmbeddingFields,
		ImageFields:         AllImageFields,
	}
}

func (s *Spec) SetImagePaths(generatePath, editPath string) {
	s.ImagePath = generatePath
	s.ImageEditPath = editPath
}

func Definition(baseURL, serviceBaseURL string, credentialsOptional bool, info provider.Info, configure ...ConfigureFunc) provider.Definition {
	return provider.Definition{
		DefaultBaseURL:        baseURL,
		DefaultServiceBaseURL: serviceBaseURL,
		CredentialsOptional:   credentialsOptional,
		New: func(config provider.Config) provider.Implementation {
			spec := DefaultSpec(info)
			for _, apply := range configure {
				if apply != nil {
					apply(&spec, config)
				}
			}
			return New(config, spec)
		},
	}
}

func filterFields(body map[string]any, supported, known FieldSet) {
	for field := range known {
		if _, ok := supported[field]; !ok {
			delete(body, field)
		}
	}
}

func applyFieldAliases(body map[string]any, aliases map[string]string) {
	for source, target := range aliases {
		value, exists := body[source]
		if !exists {
			continue
		}
		delete(body, source)
		body[target] = value
	}
}
