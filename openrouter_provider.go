// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/image"
)

type openRouterProvider struct{ *openAICompatibleProvider }

func openRouterImageContent(prompt string, images []string) chat.MessageContent {
	if len(images) == 0 {
		return chat.TextContent(prompt)
	}
	parts := []chat.ContentPart{{Type: chat.ContentPartText, Text: prompt}}
	for _, inputImage := range images {
		parts = append(parts, chat.ContentPart{Type: chat.ContentPartImageURL, ImageURL: &chat.ImageURL{URL: inputImage}})
	}
	return chat.PartsContent(parts...)
}

func newOpenRouterProvider(config ClientConfig) *openRouterProvider {
	return &openRouterProvider{newOpenAICompatibleProvider(config, ProviderInfo{
		ID: ProviderOpenRouter, DefaultBaseURL: "https://openrouter.ai/api/v1", Capabilities: []Capability{CapabilityChat, CapabilityImage},
	})}
}

func (p *openRouterProvider) generateImage(ctx context.Context, selected credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil || request.Model == "" || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image model and prompt are required", ErrInvalidRequest)
	}
	extra := make(map[string]any, len(request.ExtraBody)+2)
	for key, value := range request.ExtraBody {
		extra[key] = value
	}
	extra["modalities"] = []string{"image", "text"}
	imageConfig := map[string]any{}
	if request.Size != "" {
		imageConfig["image_size"] = request.Size
	}
	if len(imageConfig) > 0 {
		extra["image_config"] = imageConfig
	}
	chatResponse, err := p.createChat(ctx, selected, &chat.CreateRequest{
		Model:     request.Model,
		Messages:  []chat.Message{{Role: chat.RoleUser, Content: openRouterImageContent(request.Prompt, request.Image)}},
		ExtraBody: extra,
	})
	if err != nil {
		return nil, err
	}
	result := &image.GenerateResponse{RawResponse: chatResponse.RawResponse}
	for _, choice := range chatResponse.Choices {
		for _, generated := range choice.Message.Images {
			if generated.ImageURL == nil {
				continue
			}
			result.Data = append(result.Data, image.Data{URL: generated.ImageURL.URL})
		}
	}
	result.Usage.InputTokens = chatResponse.Usage.PromptTokens
	result.Usage.OutputTokens = chatResponse.Usage.CompletionTokens
	result.Usage.TotalTokens = chatResponse.Usage.TotalTokens
	if err := normalizeImageResponse(ctx, p.config, ProviderOpenRouter, selected.hint, request, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *openRouterProvider) streamImage(ctx context.Context, selected credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil || request.Model == "" || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image model and prompt are required", ErrInvalidRequest)
	}
	extra := make(map[string]any, len(request.ExtraBody)+2)
	for key, value := range request.ExtraBody {
		extra[key] = value
	}
	extra["modalities"] = []string{"image", "text"}
	if request.Size != "" {
		extra["image_config"] = map[string]any{"image_size": request.Size}
	}
	stream, err := p.streamChat(ctx, selected, &chat.StreamRequest{CreateRequest: chat.CreateRequest{
		Model: request.Model, Messages: []chat.Message{{Role: chat.RoleUser, Content: openRouterImageContent(request.Prompt, request.Image)}}, ExtraBody: extra,
	}})
	if err != nil {
		return nil, err
	}
	return &openRouterImageStream{ctx: ctx, stream: stream, config: p.config, selected: selected, request: request.GenerateRequest}, nil
}

type openRouterImageStream struct {
	ctx      context.Context
	stream   chat.Stream
	config   ClientConfig
	selected credential
	request  image.GenerateRequest
	pending  []*image.StreamChunk
	finished bool
}

func (s *openRouterImageStream) Recv() (*image.StreamChunk, error) {
	if len(s.pending) > 0 {
		chunk := s.pending[0]
		s.pending = s.pending[1:]
		return chunk, nil
	}
	if s.finished {
		return nil, io.EOF
	}
	for {
		chunk, err := s.stream.Recv()
		if err != nil {
			if err == io.EOF {
				s.finished = true
			}
			return nil, err
		}
		usage := image.Usage{}
		if chunk.Usage != nil {
			usage.InputTokens = chunk.Usage.PromptTokens
			usage.OutputTokens = chunk.Usage.CompletionTokens
			usage.TotalTokens = chunk.Usage.TotalTokens
		}
		for _, choice := range chunk.Choices {
			for _, generated := range choice.Delta.Images {
				if generated.ImageURL == nil {
					continue
				}
				response := &image.GenerateResponse{Data: []image.Data{{URL: generated.ImageURL.URL}}}
				if err := normalizeImageResponse(s.ctx, s.config, ProviderOpenRouter, s.selected.hint, &s.request, response); err != nil {
					return nil, err
				}
				item := response.Data[0]
				s.pending = append(s.pending, &image.StreamChunk{URL: item.URL, B64JSON: item.B64JSON, Format: item.Format, MIMEType: item.MIMEType, Usage: usage, RawResponse: chunk.RawResponse})
			}
		}
		if len(s.pending) > 0 {
			return s.Recv()
		}
		if chunk.Usage != nil {
			return &image.StreamChunk{Usage: usage, RawResponse: chunk.RawResponse}, nil
		}
	}
}

func (s *openRouterImageStream) Close() error {
	s.finished = true
	return s.stream.Close()
}

var _ imageProvider = (*openRouterProvider)(nil)
var _ imageStreamProvider = (*openRouterProvider)(nil)
var _ image.Stream = (*openRouterImageStream)(nil)
