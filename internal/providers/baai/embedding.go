// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package baai

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureEmbedding(spec *openai.Spec, _ provider.Config) {
	spec.EmbeddingPath = "/v1/embeddings"
	spec.EmbeddingFields = openai.Fields()
}
