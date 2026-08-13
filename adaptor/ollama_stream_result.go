// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import (
	"github.com/zhimaAi/llm_adaptor/api/ollama"
)

type OllamaStreamResult struct {
	*ollama.ChatCompletionStream
}

func (r *OllamaStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	responseOllama, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}
	toolCalls := responseOllama.Message.ToolCalls
	reasoningContent := responseOllama.Message.ReasoningContent
	if reasoningContent == "" {
		reasoningContent = responseOllama.Message.Thinking
	}
	return ZhimaChatCompletionResponse{
		Result:            responseOllama.Message.Content,
		ReasoningContent:  reasoningContent,
		ToolCalls:         toolCalls,
		FunctionToolCalls: toolCalls.FunctionToolCalls(),
		PromptToken:       responseOllama.Metrics.PromptEvalCount,
		CompletionToken:   responseOllama.Metrics.EvalCount,
	}, nil
}
