// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package adaptor

import "github.com/zhimaAi/llm_adaptor/api/azure"

type AzureStreamResult struct {
	*azure.ChatCompletionStream
}

func (r *AzureStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	responseAzure, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}
	var result string
	var functionToolCalls []FunctionToolCall
	if len(responseAzure.Choices) > 0 {
		result = responseAzure.Choices[0].Delta.Content
		for _, toolCall := range responseAzure.Choices[0].Delta.ToolCalls {
			functionToolCalls = append(functionToolCalls, FunctionToolCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			})
		}
	}
	return ZhimaChatCompletionResponse{
		Result:            result,
		FunctionToolCalls: functionToolCalls,
		PromptToken:       responseAzure.Usage.PromptTokens,
		CompletionToken:   responseAzure.Usage.CompletionTokens,
	}, nil
}
