// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"encoding/json"

	"github.com/zhimaAi/llm_adaptor/v2/embedding"
)

type embeddingDataWire struct {
	Object    string          `json:"object"`
	Embedding json.RawMessage `json:"embedding"`
	Index     int             `json:"index"`
}

type embeddingResponseWire struct {
	Object string              `json:"object"`
	Data   []embeddingDataWire `json:"data"`
	Model  string              `json:"model"`
	Usage  struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func DecodeEmbeddingResponse(raw []byte) (*embedding.CreateResponse, error) {
	var source embeddingResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &embedding.CreateResponse{Object: source.Object, Model: source.Model, Usage: embedding.Usage{PromptTokens: source.Usage.PromptTokens, TotalTokens: source.Usage.TotalTokens}, Data: make([]embedding.Data, len(source.Data))}
	for index, item := range source.Data {
		var value embedding.EmbeddingValue
		if err := json.Unmarshal(item.Embedding, &value); err != nil {
			return nil, err
		}
		result.Data[index] = embedding.Data{Object: item.Object, Embedding: value, Index: item.Index}
	}
	return result, nil
}
