// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package baai

import (
	"fmt"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureEmbedding(spec *openai.Spec, _ provider.Config) {
	spec.EmbeddingPath = "/v1/embeddings"
	spec.EmbeddingFields = openai.Fields()
	spec.TransformEmbeddingInput = transformEmbeddingInput
}

func transformEmbeddingInput(input any) (any, error) {
	switch value := input.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%w: BAAI embedding text input is empty", provider.ErrInvalidRequest)
		}
		return []string{value}, nil
	case []string:
		if len(value) == 0 {
			return nil, fmt.Errorf("%w: BAAI embedding text input is empty", provider.ErrInvalidRequest)
		}
		return append([]string(nil), value...), nil
	default:
		return nil, fmt.Errorf("%w: BAAI embedding only supports text input", provider.ErrInvalidRequest)
	}
}
