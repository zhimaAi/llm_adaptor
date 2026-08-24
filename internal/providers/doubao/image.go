// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package doubao

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

func configureImage(spec *openai.Spec, _ provider.Config) {
	spec.ImageFields = openai.Fields("n", "response_format", "size", "output_format")
}
