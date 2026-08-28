// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"

	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
)

type RerankService struct{ client *Client }

func (s RerankService) Create(ctx context.Context, request *rerank.CreateRequest) (*rerank.CreateResponse, error) {
	ctx = shared.NormalizeContext(ctx)
	if !s.client.supports(CapabilityRerank) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityRerank}
	}
	implementation, ok := s.client.provider.(internalprovider.Rerank)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityRerank}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	response, err := implementation.CreateRerank(ctx, internalCredential(selected), request)
	return response, normalizeProviderError(err)
}
