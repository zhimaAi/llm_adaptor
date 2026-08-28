// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"

	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

type EmbeddingService struct{ client *Client }

func (s EmbeddingService) Create(ctx context.Context, request *embedding.CreateRequest) (*embedding.CreateResponse, error) {
	ctx = shared.NormalizeContext(ctx)
	if !s.client.supports(CapabilityEmbedding) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityEmbedding}
	}
	implementation, ok := s.client.provider.(internalprovider.Embedding)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityEmbedding}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	response, err := implementation.CreateEmbedding(ctx, internalCredential(selected), request)
	return response, normalizeProviderError(err)
}
