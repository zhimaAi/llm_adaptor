// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import "github.com/zhimaAi/llm_adaptor/api/gemini"

type GeminiStreamResult struct {
	*gemini.ChatCompletionStream
}

func (c *GeminiStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	responseGemini, err := c.ChatCompletionStream.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}
	result, reasoningContent := geminiTextAndThinking(responseGemini.Candidates[0].Content.Parts)
	return ZhimaChatCompletionResponse{
		Result:           result,
		ReasoningContent: reasoningContent,
		PromptToken:      responseGemini.UsageMetadata.PromptTokenCount,
		CompletionToken:  geminiCompletionTokens(responseGemini.UsageMetadata),
	}, nil
}
