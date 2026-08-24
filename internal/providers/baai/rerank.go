// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package baai

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureRerank(spec *openai.Spec, _ provider.Config) {
	spec.RerankPath = "/v1/rerank"
	spec.RerankDocumentsKey = "passages"
	spec.RerankTopKey = "top_k"
}
