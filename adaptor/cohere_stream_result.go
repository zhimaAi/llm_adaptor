// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package adaptor

import "github.com/zhimaAi/llm_adaptor/api/cohere"

type CohereStreamResult struct {
	*cohere.ChatCompletionStream
}

func (r *CohereStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	responseCohere, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}
	return ZhimaChatCompletionResponse{
		Result:          responseCohere.Text,
		PromptToken:     responseCohere.Response.Meta.Tokens.InputTokens,
		CompletionToken: responseCohere.Response.Meta.Tokens.OutputTokens,
	}, nil
}
