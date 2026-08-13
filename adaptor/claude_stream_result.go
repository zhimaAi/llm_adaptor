// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import (
	"github.com/zhimaAi/llm_adaptor/api/claude"
	"github.com/zhimaAi/llm_adaptor/basics"
)

type ClaudeStreamResult struct {
	*claude.ChatCompletionStream
}

func (r *ClaudeStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	responseClaude, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}
	var toolCalls basics.ToolCalls
	if responseClaude.ContentBlock.Type == `tool_use` {
		toolCalls = append(toolCalls, basics.NewFunctionToolCall(responseClaude.ContentBlock.Id, responseClaude.ContentBlock.Name, ""))
	}
	if responseClaude.Delta.Type == `input_json_delta` {
		toolCalls = append(toolCalls, basics.NewFunctionToolCall("", "", responseClaude.Delta.PartialJson))
	}
	return ZhimaChatCompletionResponse{
		Result:            responseClaude.Delta.Text,
		ReasoningContent:  responseClaude.Delta.Thinking,
		ToolCalls:         toolCalls,
		FunctionToolCalls: toolCalls.FunctionToolCalls(),
		PromptToken:       responseClaude.Message.Usage.InputTokens,
		CompletionToken:   responseClaude.Message.Usage.OutputTokens,
	}, nil
}
