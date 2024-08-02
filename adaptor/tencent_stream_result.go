// Copyright © 2016- 2024 Sesame Network Technology all right reserved

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
		Result:          *responseTencent.Choices[0].Delta.Content,
		PromptToken:     int(*responseTencent.Usage.PromptTokens),
		CompletionToken: int(*responseTencent.Usage.CompletionTokens),
	}, nil
}
