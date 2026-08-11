// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

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
	content := responseCohere.Delta.Message.Content
	result, reasoningContent := cohereTextAndThinking([]cohere.ChatContent{content})
	promptTokens := responseCohere.Delta.Usage.Tokens.InputTokens
	completionTokens := responseCohere.Delta.Usage.Tokens.OutputTokens
	if responseCohere.Type == "" && result == "" && reasoningContent == "" {
		result = responseCohere.Text
		promptTokens = responseCohere.Response.Meta.Tokens.InputTokens
		completionTokens = responseCohere.Response.Meta.Tokens.OutputTokens
	}
	return ZhimaChatCompletionResponse{
		Result:           result,
		ReasoningContent: reasoningContent,
		PromptToken:      promptTokens,
		CompletionToken:  completionTokens,
	}, nil
}
