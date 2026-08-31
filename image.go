// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"

	"github.com/zhimaAi/llm_adaptor/v2/image"
	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

type ImageService struct{ client *Client }

func (s ImageService) Generate(ctx context.Context, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	ctx = NormalizeContext(ctx)
	implementation, selected, err := s.generateProvider()
	if err != nil {
		return nil, err
	}
	response, err := implementation.GenerateImage(ctx, internalCredential(selected), request)
	return response, normalizeProviderError(err)
}

func (s ImageService) Stream(ctx context.Context, request *image.StreamRequest) (image.Stream, error) {
	ctx = NormalizeContext(ctx)
	if !s.client.supports(CapabilityImage) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	implementation, ok := s.client.provider.(internalprovider.ImageStream)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	stream, err := implementation.StreamImage(ctx, internalCredential(selected), request)
	if err != nil {
		return nil, normalizeProviderError(err)
	}
	return &imageErrorStream{Stream: stream}, nil
}

func (s ImageService) Edit(ctx context.Context, request *image.EditRequest) (*image.GenerateResponse, error) {
	ctx = NormalizeContext(ctx)
	if !s.client.supports(CapabilityImage) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	implementation, ok := s.client.provider.(internalprovider.ImageEdit)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	response, err := implementation.EditImage(ctx, internalCredential(selected), request)
	return response, normalizeProviderError(err)
}

func (s ImageService) EditStream(ctx context.Context, request *image.EditStreamRequest) (image.Stream, error) {
	ctx = NormalizeContext(ctx)
	if !s.client.supports(CapabilityImage) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	implementation, ok := s.client.provider.(internalprovider.ImageEditStream)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	stream, err := implementation.StreamImageEdit(ctx, internalCredential(selected), request)
	if err != nil {
		return nil, normalizeProviderError(err)
	}
	return &imageErrorStream{Stream: stream}, nil
}

type imageErrorStream struct{ image.Stream }

func (s *imageErrorStream) Recv() (*image.StreamChunk, error) {
	chunk, err := s.Stream.Recv()
	return chunk, normalizeProviderError(err)
}

func (s *imageErrorStream) Close() error { return normalizeProviderError(s.Stream.Close()) }

func (s ImageService) generateProvider() (internalprovider.Image, credential, error) {
	if !s.client.supports(CapabilityImage) {
		return nil, credential{}, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	implementation, ok := s.client.provider.(internalprovider.Image)
	if !ok {
		return nil, credential{}, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	selected, err := s.client.credentials.selectCredential()
	return implementation, selected, err
}
