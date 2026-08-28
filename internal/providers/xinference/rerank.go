// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package xinference

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureRerank(spec *openai.Spec, _ provider.Config) {
	spec.RerankPath = "/rerank"
	spec.RerankDocumentsKey = "documents"
	spec.RerankTopKey = "top_n"
}
