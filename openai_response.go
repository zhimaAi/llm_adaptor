// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"encoding/json"
	"fmt"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
)

type openAIFunctionCallResponseWire struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type openAIToolCallResponseWire struct {
	Index    *int                           `json:"index,omitempty"`
	ID       string                         `json:"id,omitempty"`
	Type     string                         `json:"type,omitempty"`
	Function openAIFunctionCallResponseWire `json:"function"`
}

type openAIAudioResponseWire struct {
	ID         string `json:"id"`
	Data       string `json:"data,omitempty"`
	ExpiresAt  int64  `json:"expires_at,omitempty"`
	Transcript string `json:"transcript,omitempty"`
}

type openAIURLCitationResponseWire struct {
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	URL        string `json:"url"`
	Title      string `json:"title"`
}

type openAIAnnotationResponseWire struct {
	Type        string                         `json:"type"`
	URLCitation *openAIURLCitationResponseWire `json:"url_citation,omitempty"`
}

type openAIMessageResponseWire struct {
	Role             string                          `json:"role"`
	Content          json.RawMessage                 `json:"content"`
	Name             string                          `json:"name,omitempty"`
	ToolCalls        []openAIToolCallResponseWire    `json:"tool_calls,omitempty"`
	ToolCallID       string                          `json:"tool_call_id,omitempty"`
	FunctionCall     *openAIFunctionCallResponseWire `json:"function_call,omitempty"`
	ReasoningContent string                          `json:"reasoning_content,omitempty"`
	Refusal          string                          `json:"refusal,omitempty"`
	Audio            *openAIAudioResponseWire        `json:"audio,omitempty"`
	Annotations      []openAIAnnotationResponseWire  `json:"annotations,omitempty"`
}

type openAIPromptTokensDetailsWire struct {
	AudioTokens  int `json:"audio_tokens,omitempty"`
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type openAICompletionTokensDetailsWire struct {
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`
	AudioTokens              int `json:"audio_tokens,omitempty"`
	ReasoningTokens          int `json:"reasoning_tokens,omitempty"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

type openAIUsageWire struct {
	PromptTokens            int                               `json:"prompt_tokens"`
	CompletionTokens        int                               `json:"completion_tokens"`
	TotalTokens             int                               `json:"total_tokens"`
	PromptTokensDetails     openAIPromptTokensDetailsWire     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails openAICompletionTokensDetailsWire `json:"completion_tokens_details,omitempty"`
}

type openAITopLogProbWire struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

type openAITokenLogProbWire struct {
	Token       string                 `json:"token"`
	LogProb     float64                `json:"logprob"`
	Bytes       []int                  `json:"bytes,omitempty"`
	TopLogProbs []openAITopLogProbWire `json:"top_logprobs,omitempty"`
}

type openAILogProbsWire struct {
	Content []openAITokenLogProbWire `json:"content,omitempty"`
	Refusal []openAITokenLogProbWire `json:"refusal,omitempty"`
}

type openAIChoiceResponseWire struct {
	Index        int                       `json:"index"`
	Message      openAIMessageResponseWire `json:"message"`
	FinishReason string                    `json:"finish_reason,omitempty"`
	LogProbs     *openAILogProbsWire       `json:"logprobs,omitempty"`
}

type openAIChatResponseWire struct {
	ID                string                     `json:"id"`
	Object            string                     `json:"object"`
	Created           int64                      `json:"created"`
	Model             string                     `json:"model"`
	SystemFingerprint string                     `json:"system_fingerprint,omitempty"`
	ServiceTier       string                     `json:"service_tier,omitempty"`
	Choices           []openAIChoiceResponseWire `json:"choices"`
	Usage             openAIUsageWire            `json:"usage"`
}

type openAIChunkChoiceResponseWire struct {
	Index        int                       `json:"index"`
	Delta        openAIMessageResponseWire `json:"delta"`
	FinishReason string                    `json:"finish_reason,omitempty"`
	LogProbs     *openAILogProbsWire       `json:"logprobs,omitempty"`
}

type openAIChatStreamResponseWire struct {
	ID                string                          `json:"id"`
	Object            string                          `json:"object"`
	Created           int64                           `json:"created"`
	Model             string                          `json:"model"`
	SystemFingerprint string                          `json:"system_fingerprint,omitempty"`
	ServiceTier       string                          `json:"service_tier,omitempty"`
	Choices           []openAIChunkChoiceResponseWire `json:"choices"`
	Usage             *openAIUsageWire                `json:"usage,omitempty"`
}

func decodeOpenAIChatResponse(raw []byte) (*chat.CreateResponse, error) {
	var source openAIChatResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &chat.CreateResponse{
		ID: source.ID, Object: source.Object, Created: source.Created, Model: source.Model,
		SystemFingerprint: source.SystemFingerprint, ServiceTier: source.ServiceTier,
		Usage: mapOpenAIUsage(source.Usage), Choices: make([]chat.Choice, len(source.Choices)),
	}
	for index, choice := range source.Choices {
		message, err := mapOpenAIMessage(choice.Message)
		if err != nil {
			return nil, err
		}
		result.Choices[index] = chat.Choice{
			Index: choice.Index, Message: message, FinishReason: choice.FinishReason, LogProbs: mapOpenAILogProbs(choice.LogProbs),
		}
	}
	return result, nil
}

func decodeOpenAIChatStreamResponse(raw []byte) (*chat.StreamChunk, error) {
	var source openAIChatStreamResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &chat.StreamChunk{
		ID: source.ID, Object: source.Object, Created: source.Created, Model: source.Model,
		SystemFingerprint: source.SystemFingerprint, ServiceTier: source.ServiceTier,
		Choices: make([]chat.ChunkChoice, len(source.Choices)),
	}
	if source.Usage != nil {
		usage := mapOpenAIUsage(*source.Usage)
		result.Usage = &usage
	}
	for index, choice := range source.Choices {
		delta, err := mapOpenAIMessage(choice.Delta)
		if err != nil {
			return nil, err
		}
		result.Choices[index] = chat.ChunkChoice{
			Index: choice.Index, Delta: delta, FinishReason: choice.FinishReason, LogProbs: mapOpenAILogProbs(choice.LogProbs),
		}
	}
	return result, nil
}

func mapOpenAILogProbs(source *openAILogProbsWire) *chat.LogProbs {
	if source == nil {
		return nil
	}
	return &chat.LogProbs{
		Content: mapOpenAITokenLogProbs(source.Content),
		Refusal: mapOpenAITokenLogProbs(source.Refusal),
	}
}

func mapOpenAITokenLogProbs(source []openAITokenLogProbWire) []chat.TokenLogProb {
	result := make([]chat.TokenLogProb, len(source))
	for index, item := range source {
		result[index] = chat.TokenLogProb{
			Token: item.Token, LogProb: item.LogProb, Bytes: append([]int(nil), item.Bytes...),
			TopLogProbs: make([]chat.TopLogProb, len(item.TopLogProbs)),
		}
		for topIndex, top := range item.TopLogProbs {
			result[index].TopLogProbs[topIndex] = chat.TopLogProb{
				Token: top.Token, LogProb: top.LogProb, Bytes: append([]int(nil), top.Bytes...),
			}
		}
	}
	return result
}

func mapOpenAIMessage(source openAIMessageResponseWire) (chat.Message, error) {
	content, err := decodeOpenAIMessageContent(source.Content)
	if err != nil {
		return chat.Message{}, err
	}
	result := chat.Message{
		Role: chat.Role(source.Role), Content: content, Name: source.Name, ToolCallID: source.ToolCallID,
		ReasoningContent: source.ReasoningContent, Refusal: source.Refusal,
		ToolCalls: make([]chat.ToolCall, len(source.ToolCalls)), Annotations: make([]chat.Annotation, len(source.Annotations)),
	}
	if source.FunctionCall != nil {
		result.FunctionCall = &chat.FunctionCall{Name: source.FunctionCall.Name, Arguments: source.FunctionCall.Arguments}
	}
	if source.Audio != nil {
		result.Audio = &chat.Audio{
			ID: source.Audio.ID, Data: source.Audio.Data, ExpiresAt: source.Audio.ExpiresAt, Transcript: source.Audio.Transcript,
		}
	}
	for index, toolCall := range source.ToolCalls {
		result.ToolCalls[index] = chat.ToolCall{
			Index: toolCall.Index, ID: toolCall.ID, Type: toolCall.Type,
			Function: chat.FunctionCall{Name: toolCall.Function.Name, Arguments: toolCall.Function.Arguments},
		}
	}
	for index, annotation := range source.Annotations {
		result.Annotations[index] = chat.Annotation{Type: annotation.Type}
		if annotation.URLCitation != nil {
			result.Annotations[index].URLCitation = &chat.URLCitation{
				StartIndex: annotation.URLCitation.StartIndex, EndIndex: annotation.URLCitation.EndIndex,
				URL: annotation.URLCitation.URL, Title: annotation.URLCitation.Title,
			}
		}
	}
	return result, nil
}

func decodeOpenAIMessageContent(raw json.RawMessage) (chat.MessageContent, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return chat.MessageContent{}, nil
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return chat.TextContent(text), nil
	}
	var parts []struct {
		Type       string                `json:"type"`
		Text       string                `json:"text,omitempty"`
		ImageURL   *openAIImageURLWire   `json:"image_url,omitempty"`
		InputAudio *openAIInputAudioWire `json:"input_audio,omitempty"`
		VideoURL   *openAIVideoURLWire   `json:"video_url,omitempty"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return chat.MessageContent{}, fmt.Errorf("decode chat message content: %w", err)
	}
	result := make([]chat.ContentPart, len(parts))
	for index, part := range parts {
		result[index] = chat.ContentPart{Type: chat.ContentPartType(part.Type), Text: part.Text}
		if part.ImageURL != nil {
			result[index].ImageURL = &chat.ImageURL{URL: part.ImageURL.URL, Detail: part.ImageURL.Detail}
		}
		if part.InputAudio != nil {
			result[index].InputAudio = &chat.InputAudio{Data: part.InputAudio.Data, Format: part.InputAudio.Format}
		}
		if part.VideoURL != nil {
			result[index].VideoURL = &chat.VideoURL{URL: part.VideoURL.URL}
		}
	}
	return chat.PartsContent(result...), nil
}

func mapOpenAIUsage(source openAIUsageWire) chat.Usage {
	return chat.Usage{
		PromptTokens: source.PromptTokens, CompletionTokens: source.CompletionTokens, TotalTokens: source.TotalTokens,
		PromptTokensDetails: chat.PromptTokensDetails{
			AudioTokens: source.PromptTokensDetails.AudioTokens, CachedTokens: source.PromptTokensDetails.CachedTokens,
		},
		CompletionTokensDetails: chat.CompletionTokensDetails{
			AcceptedPredictionTokens: source.CompletionTokensDetails.AcceptedPredictionTokens,
			AudioTokens:              source.CompletionTokensDetails.AudioTokens,
			ReasoningTokens:          source.CompletionTokensDetails.ReasoningTokens,
			RejectedPredictionTokens: source.CompletionTokensDetails.RejectedPredictionTokens,
		},
	}
}

type openAIEmbeddingDataWire struct {
	Object    string          `json:"object"`
	Embedding json.RawMessage `json:"embedding"`
	Index     int             `json:"index"`
}

type openAIEmbeddingResponseWire struct {
	Object string                    `json:"object"`
	Data   []openAIEmbeddingDataWire `json:"data"`
	Model  string                    `json:"model"`
	Usage  struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func decodeOpenAIEmbeddingResponse(raw []byte) (*embedding.CreateResponse, error) {
	var source openAIEmbeddingResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &embedding.CreateResponse{
		Object: source.Object, Model: source.Model,
		Usage: embedding.Usage{PromptTokens: source.Usage.PromptTokens, TotalTokens: source.Usage.TotalTokens},
		Data:  make([]embedding.Data, len(source.Data)),
	}
	for index, item := range source.Data {
		var value embedding.EmbeddingValue
		if err := json.Unmarshal(item.Embedding, &value); err != nil {
			return nil, err
		}
		result.Data[index] = embedding.Data{Object: item.Object, Embedding: value, Index: item.Index}
	}
	return result, nil
}

type openAIImageDataWire struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type openAIImageTokenDetailsWire struct {
	ImageTokens int `json:"image_tokens,omitempty"`
	TextTokens  int `json:"text_tokens,omitempty"`
}

type openAIImageResponseWire struct {
	Created      int64                 `json:"created,omitempty"`
	Background   string                `json:"background,omitempty"`
	Data         []openAIImageDataWire `json:"data"`
	OutputFormat string                `json:"output_format,omitempty"`
	Quality      string                `json:"quality,omitempty"`
	Size         string                `json:"size,omitempty"`
	Usage        struct {
		InputTokens         int                         `json:"input_tokens,omitempty"`
		InputTokensDetails  openAIImageTokenDetailsWire `json:"input_tokens_details,omitempty"`
		OutputTokens        int                         `json:"output_tokens,omitempty"`
		OutputTokensDetails openAIImageTokenDetailsWire `json:"output_tokens_details,omitempty"`
		TotalTokens         int                         `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

func decodeOpenAIImageResponse(raw []byte) (*image.GenerateResponse, error) {
	var source openAIImageResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &image.GenerateResponse{
		Created: source.Created, Background: source.Background, OutputFormat: source.OutputFormat,
		Quality: source.Quality, Size: source.Size, Data: make([]image.Data, len(source.Data)),
		Usage: image.Usage{
			InputTokens:         source.Usage.InputTokens,
			InputTokensDetails:  image.TokenDetails{ImageTokens: source.Usage.InputTokensDetails.ImageTokens, TextTokens: source.Usage.InputTokensDetails.TextTokens},
			OutputTokens:        source.Usage.OutputTokens,
			OutputTokensDetails: image.TokenDetails{ImageTokens: source.Usage.OutputTokensDetails.ImageTokens, TextTokens: source.Usage.OutputTokensDetails.TextTokens},
			TotalTokens:         source.Usage.TotalTokens,
		},
	}
	for index, item := range source.Data {
		result.Data[index] = image.Data{URL: item.URL, B64JSON: item.B64JSON, RevisedPrompt: item.RevisedPrompt}
	}
	return result, nil
}
