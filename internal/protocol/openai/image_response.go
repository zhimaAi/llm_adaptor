// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"encoding/json"

	"github.com/zhimaAi/llm_adaptor/v2/image"
)

type imageDataWire struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}
type imageTokenDetailsWire struct {
	ImageTokens int `json:"image_tokens,omitempty"`
	TextTokens  int `json:"text_tokens,omitempty"`
}
type imageResponseWire struct {
	Created      int64           `json:"created,omitempty"`
	Background   string          `json:"background,omitempty"`
	Data         []imageDataWire `json:"data"`
	OutputFormat string          `json:"output_format,omitempty"`
	Quality      string          `json:"quality,omitempty"`
	Size         string          `json:"size,omitempty"`
	Usage        struct {
		InputTokens         int                   `json:"input_tokens,omitempty"`
		InputTokensDetails  imageTokenDetailsWire `json:"input_tokens_details,omitempty"`
		OutputTokens        int                   `json:"output_tokens,omitempty"`
		OutputTokensDetails imageTokenDetailsWire `json:"output_tokens_details,omitempty"`
		TotalTokens         int                   `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

func DecodeImageResponse(raw []byte) (*image.GenerateResponse, error) {
	var source imageResponseWire
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
