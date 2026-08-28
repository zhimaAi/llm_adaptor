// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package cohere

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureRerank(spec *openai.Spec, config provider.Config) {
	spec.RerankPath = "/v2/rerank"
	spec.RerankBaseURL = config.ServiceBaseURL
	spec.RerankDocumentsKey = "documents"
	spec.RerankTopKey = "top_n"
}
