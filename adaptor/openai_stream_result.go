// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package adaptor

import "github.com/zhimaAi/llm_adaptor/api/openai"

type OpenAIStreamResult struct {
	*openai.ChatCompletionStream
}

func (r *OpenAIStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	responseOpenAI, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}

	var promptTokens int
	var completionTokens int
	var result string
	if responseOpenAI.Usage.PromptTokens > 0 {
		promptTokens = responseOpenAI.Usage.PromptTokens
	}
	if responseOpenAI.Usage.CompletionTokens > 0 {
		completionTokens = responseOpenAI.Usage.CompletionTokens
	}
	var functionToolCalls []FunctionToolCall
	if len(responseOpenAI.Choices) > 0 {
		result = responseOpenAI.Choices[0].Delta.Content
		// Compatible with moonlight
		if responseOpenAI.Choices[0].Usage.PromptTokens > 0 {
			promptTokens = responseOpenAI.Choices[0].Usage.PromptTokens
		}
		if responseOpenAI.Choices[0].Usage.CompletionTokens > 0 {
			completionTokens = responseOpenAI.Choices[0].Usage.CompletionTokens
		}
		for _, toolCall := range responseOpenAI.Choices[0].Delta.ToolCalls {
			functionToolCalls = append(functionToolCalls, FunctionToolCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			})
		}
	}

	return ZhimaChatCompletionResponse{
		Result:            result,
		FunctionToolCalls: functionToolCalls,
		PromptToken:       promptTokens,
		CompletionToken:   completionTokens,
	}, nil
}
