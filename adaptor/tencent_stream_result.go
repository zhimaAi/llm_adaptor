// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import "github.com/zhimaAi/llm_adaptor/api/hunyuan"

type TencentStreamResult struct {
	*hunyuan.ChatCompletionStream
}

func (r *TencentStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	responseTencent, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}
	return ZhimaChatCompletionResponse{
		Result:           *responseTencent.Choices[0].Delta.Content,
		ReasoningContent: stringValue(responseTencent.Choices[0].Delta.ReasoningContent),
		PromptToken:      int(*responseTencent.Usage.PromptTokens),
		CompletionToken:  int(*responseTencent.Usage.CompletionTokens),
	}, nil
}
