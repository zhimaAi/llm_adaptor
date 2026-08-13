// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import (
	"github.com/zhimaAi/llm_adaptor/api/spark"
	"github.com/zhimaAi/llm_adaptor/basics"
)

type SparkStreamResult struct {
	*spark.ChatCompletionStream
}

func (r *SparkStreamResult) Read() (resp ZhimaChatCompletionResponse, err error) {
	responseSpark, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}

	resp.PromptToken = responseSpark.Payload.Usage.Text.PromptTokens
	resp.CompletionToken = responseSpark.Payload.Usage.Text.CompletionTokens
	var toolCalls basics.ToolCalls
	if len(responseSpark.Payload.Choices.Text) == 0 {
		return resp, nil
	}
	if len(responseSpark.Payload.Choices.Text[0].FunctionCall.Name) > 0 {
		toolCalls = append(toolCalls, basics.NewFunctionToolCall("", responseSpark.Payload.Choices.Text[0].FunctionCall.Name, responseSpark.Payload.Choices.Text[0].FunctionCall.Arguments))
	}
	resp.Result = responseSpark.Payload.Choices.Text[0].Content
	resp.ReasoningContent = responseSpark.Payload.Choices.Text[0].ReasoningContent
	resp.ToolCalls = toolCalls
	resp.FunctionToolCalls = toolCalls.FunctionToolCalls()
	return resp, nil
}
