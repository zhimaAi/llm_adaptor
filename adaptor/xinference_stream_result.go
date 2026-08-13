// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import (
	"github.com/zhimaAi/llm_adaptor/api/xinference"
)

type XinferenceStreamResult struct {
	*xinference.ChatCompletionStream
}

func (r *XinferenceStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	responseXinference, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}
	return ZhimaChatCompletionResponse{
		Result:           responseXinference.Choices[0].Delta.Content,
		ReasoningContent: responseXinference.Choices[0].Delta.ReasoningContent,
		PromptToken:      responseXinference.Usage.PromptTokens,
		CompletionToken:  responseXinference.Usage.CompletionTokens,
	}, nil
}
