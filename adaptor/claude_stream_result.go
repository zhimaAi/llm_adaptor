// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package adaptor

import "github.com/zhimaAi/llm_adaptor/api/claude"

type ClaudeStreamResult struct {
	*claude.ChatCompletionStream
}

func (r *ClaudeStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	responseClaude, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}
	var functionToolCalls []FunctionToolCall
	if responseClaude.ContentBlock.Type == `tool_use` {
		functionToolCalls = append(functionToolCalls, FunctionToolCall{
			Name: responseClaude.ContentBlock.Name,
		})
	}
	if responseClaude.Delta.Type == `input_json_delta` {
		functionToolCalls = append(functionToolCalls, FunctionToolCall{
			Arguments: responseClaude.Delta.PartialJson,
		})
	}
	return ZhimaChatCompletionResponse{
		Result:            responseClaude.Delta.Text,
		FunctionToolCalls: functionToolCalls,
		PromptToken:       responseClaude.Message.Usage.InputTokens,
		CompletionToken:   responseClaude.Message.Usage.OutputTokens,
	}, nil
}
