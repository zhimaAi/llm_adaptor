// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package adaptor

import (
	"errors"
	"github.com/zhimaAi/llm_adaptor/api/spark"
	"io"
)

type SparkStreamResult struct {
	*spark.ChatCompletionStream
}

func (r *SparkStreamResult) Read() (resp ZhimaChatCompletionResponse, err error) {
	responseSpark, err := r.Recv()
	var functionToolCalls []FunctionToolCall
	if err != nil {
		if errors.Is(err, io.EOF) {
			if len(responseSpark.Payload.Choices.Text[0].FunctionCall.Name) > 0 {
				functionToolCalls = append(functionToolCalls, FunctionToolCall{
					Name:      responseSpark.Payload.Choices.Text[0].FunctionCall.Name,
					Arguments: responseSpark.Payload.Choices.Text[0].FunctionCall.Arguments,
				})
			}
			resp = ZhimaChatCompletionResponse{
				Result:            responseSpark.Payload.Choices.Text[0].Content,
				FunctionToolCalls: functionToolCalls,
				PromptToken:       responseSpark.Payload.Usage.Text.PromptTokens,
				CompletionToken:   responseSpark.Payload.Usage.Text.CompletionTokens,
			}
		}
	} else {
		if len(responseSpark.Payload.Choices.Text[0].FunctionCall.Name) > 0 {
			functionToolCalls = append(functionToolCalls, FunctionToolCall{
				Name:      responseSpark.Payload.Choices.Text[0].FunctionCall.Name,
				Arguments: responseSpark.Payload.Choices.Text[0].FunctionCall.Arguments,
			})
		}
		resp = ZhimaChatCompletionResponse{
			Result:            responseSpark.Payload.Choices.Text[0].Content,
			FunctionToolCalls: functionToolCalls,
			PromptToken:       responseSpark.Payload.Usage.Text.PromptTokens,
			CompletionToken:   responseSpark.Payload.Usage.Text.CompletionTokens,
		}
	}

	return
}
