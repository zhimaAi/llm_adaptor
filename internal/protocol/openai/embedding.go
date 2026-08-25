// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"context"
	"fmt"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

type embeddingWireRequest struct {
	Model          string `json:"model"`
	Input          any    `json:"input"`
	EncodingFormat string `json:"encoding_format,omitempty"`
	Dimensions     *int   `json:"dimensions,omitempty"`
	User           string `json:"user,omitempty"`
}

func (p *Provider) CreateEmbedding(ctx context.Context, selected provider.Credential, request *embedding.CreateRequest) (*embedding.CreateResponse, error) {
	if request == nil || strings.TrimSpace(request.Model) == "" {
		return nil, fmt.Errorf("%w: embedding model is required", provider.ErrInvalidRequest)
	}
	input, err := embeddingInput(request.Input)
	if err != nil {
		return nil, err
	}
	if p.spec.TransformEmbeddingInput != nil {
		input, err = p.spec.TransformEmbeddingInput(input)
		if err != nil {
			return nil, err
		}
	}
	wire := embeddingWireRequest{Model: request.Model, Input: input, EncodingFormat: request.EncodingFormat, Dimensions: request.Dimensions, User: request.User}
	body, err := shared.MergeExtraBody(wire, nil)
	if err != nil {
		return nil, err
	}
	filterFields(body, p.spec.EmbeddingFields, AllEmbeddingFields)
	applyFieldAliases(body, p.spec.EmbeddingAliases)
	body, err = shared.MergeExtraBody(body, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	raw, err := p.DoJSON(ctx, selected, p.spec.EmbeddingPath, body)
	if err != nil {
		return nil, err
	}
	result, err := DecodeEmbeddingResponse(raw)
	if err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("%w: response contains no embeddings", provider.ErrInvalidRequest)
	}
	return result, nil
}

func embeddingInput(input embedding.Input) (any, error) {
	count := 0
	if input.Text != nil {
		count++
	}
	if input.Texts != nil {
		count++
	}
	if input.Tokens != nil {
		count++
	}
	if input.TokenBatches != nil {
		count++
	}
	if count != 1 {
		return nil, fmt.Errorf("%w: embedding input must contain exactly one input representation", provider.ErrInvalidRequest)
	}
	switch {
	case input.Text != nil:
		return *input.Text, nil
	case input.Texts != nil:
		return append([]string(nil), input.Texts...), nil
	case input.Tokens != nil:
		return append([]int(nil), input.Tokens...), nil
	default:
		batches := make([][]int, len(input.TokenBatches))
		for index := range input.TokenBatches {
			batches[index] = append([]int(nil), input.TokenBatches[index]...)
		}
		return batches, nil
	}
}

var _ provider.Embedding = (*Provider)(nil)
