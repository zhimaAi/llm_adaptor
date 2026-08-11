// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import (
	"strings"

	"github.com/zhimaAi/llm_adaptor/api/openai"
	"github.com/zhimaAi/llm_adaptor/basics"
)

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
	var result, reasoningContent = "", ""
	if responseOpenAI.Usage.PromptTokens > 0 {
		promptTokens = responseOpenAI.Usage.PromptTokens
	}
	if responseOpenAI.Usage.CompletionTokens > 0 {
		completionTokens = responseOpenAI.Usage.CompletionTokens
	}
	var toolCalls basics.ToolCalls
	if len(responseOpenAI.Choices) > 0 {
		result = responseOpenAI.Choices[0].Delta.Content
		reasoningContent = responseOpenAI.Choices[0].Delta.GetReasoningContent()
		// Compatible with moonlight
		if responseOpenAI.Choices[0].Usage.PromptTokens > 0 {
			promptTokens = responseOpenAI.Choices[0].Usage.PromptTokens
		}
		if responseOpenAI.Choices[0].Usage.CompletionTokens > 0 {
			completionTokens = responseOpenAI.Choices[0].Usage.CompletionTokens
		}
		toolCalls = responseOpenAI.Choices[0].Delta.ToolCalls
	}

	return ZhimaChatCompletionResponse{
		Result:            result,
		ReasoningContent:  reasoningContent,
		ToolCalls:         toolCalls,
		FunctionToolCalls: toolCalls.FunctionToolCalls(),
		PromptToken:       promptTokens,
		CompletionToken:   completionTokens,
	}, nil
}

type miniMaxStreamResult struct {
	*OpenAIStreamResult
	contentBuffer   string
	reasoningBuffer string
}

func (r *miniMaxStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	response, err := r.OpenAIStreamResult.Read()
	if err != nil {
		return response, err
	}
	response.Result = cumulativeDelta(response.Result, &r.contentBuffer)
	response.ReasoningContent = cumulativeDelta(response.ReasoningContent, &r.reasoningBuffer)
	return response, nil
}

func cumulativeDelta(current string, previous *string) string {
	if current == "" {
		return ""
	}
	if strings.HasPrefix(current, *previous) {
		delta := current[len(*previous):]
		*previous = current
		return delta
	}

	// Be tolerant if an endpoint switches back to standard token deltas.
	*previous += current
	return current
}

type OpenAIImageGenerationStreamResult struct {
	*openai.ImageGenerationStream
	Ext string
}

func (r *OpenAIImageGenerationStreamResult) Read() (ZhimaImageGenerationResp, error) {
	res, err := r.Recv()
	if err != nil {
		return ZhimaImageGenerationResp{}, err
	}
	inputToken := res.Usage.TotalTokens - res.Usage.OutputTokens
	outputToken := res.Usage.OutputTokens
	datas := make([]*ImageGenerationData, 0)
	if res.Type == `image_generation.completed` {
		//
	} else if res.Type == `image_generation.partial_failed` {
		datas = append(datas, &ImageGenerationData{
			Error: DataError{
				Code:    res.Error.Code,
				Message: res.Error.Message,
			},
		})
	} else if res.Type == `image_generation.partial_succeeded` {
		datas = append(datas, &ImageGenerationData{
			Url:     res.Url,
			B64Json: res.B64Json,
			Size:    res.Size,
			Error:   DataError{},
			Ext:     r.Ext,
		})
	}
	return ZhimaImageGenerationResp{
		InputToken:  inputToken,
		OutputToken: outputToken,
		Datas:       datas,
	}, nil
}

// OpenAIChatCompletionImageStreamResult is a wrapper for chat completion stream that returns images
type OpenAIChatCompletionImageStreamResult struct {
	*openai.ChatCompletionStream
	Ext string
}

func (r *OpenAIChatCompletionImageStreamResult) Read() (ZhimaImageGenerationResp, error) {
	responseOpenAI, err := r.Recv()
	if err != nil {
		return ZhimaImageGenerationResp{}, err
	}

	var promptTokens int
	var completionTokens int
	if responseOpenAI.Usage.PromptTokens > 0 {
		promptTokens = responseOpenAI.Usage.PromptTokens
	}
	if responseOpenAI.Usage.CompletionTokens > 0 {
		completionTokens = responseOpenAI.Usage.CompletionTokens
	}

	datas := make([]*ImageGenerationData, 0)
	if len(responseOpenAI.Choices) > 0 {
		for _, choice := range responseOpenAI.Choices {
			if len(choice.Delta.Images) > 0 {
				for _, image := range choice.Delta.Images {
					ext, b64Content := parseDataURL(image.ImageUrl.Url)
					datas = append(datas, &ImageGenerationData{
						Url:     ``,
						B64Json: b64Content,
						Ext:     ext,
					})
				}
			}
		}
	}

	return ZhimaImageGenerationResp{
		InputToken:  promptTokens,
		OutputToken: completionTokens,
		Datas:       datas,
	}, nil
}
