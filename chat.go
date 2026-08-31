// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

type ChatService struct{ client *Client }

func (s ChatService) Create(ctx context.Context, request *chat.CreateRequest) (*chat.CreateResponse, error) {
	ctx = NormalizeContext(ctx)
	if !s.client.supports(CapabilityChat) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityChat}
	}
	implementation, ok := s.client.provider.(internalprovider.Chat)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityChat}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	response, err := implementation.CreateChat(ctx, internalCredential(selected), request)
	return response, normalizeProviderError(err)
}

func (s ChatService) Stream(ctx context.Context, request *chat.StreamRequest) (chat.Stream, error) {
	ctx = NormalizeContext(ctx)
	if !s.client.supports(CapabilityChat) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityChat}
	}
	implementation, ok := s.client.provider.(internalprovider.Chat)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityChat}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	stream, err := implementation.StreamChat(ctx, internalCredential(selected), openai.NormalizeChatStreamRequest(request))
	if err != nil {
		return nil, normalizeProviderError(err)
	}
	return &chatErrorStream{Stream: stream}, nil
}

type chatErrorStream struct{ chat.Stream }

func (s *chatErrorStream) Recv() (*chat.StreamChunk, error) {
	chunk, err := s.Stream.Recv()
	return chunk, normalizeProviderError(err)
}

func (s *chatErrorStream) Close() error { return normalizeProviderError(s.Stream.Close()) }
