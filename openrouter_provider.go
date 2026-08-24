// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/image"
)

type openRouterProvider struct{ *openAICompatibleProvider }

type openRouterGeneratedImage struct {
	Type     string `json:"type,omitempty"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type openRouterImageResponse struct {
	ID      string `json:"id,omitempty"`
	Created int64  `json:"created,omitempty"`
	Choices []struct {
		Message struct {
			Images []openRouterGeneratedImage `json:"images,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage chat.Usage `json:"usage,omitempty"`
}

type openRouterImageStreamResponse struct {
	Created int64 `json:"created,omitempty"`
	Choices []struct {
		Delta struct {
			Images []openRouterGeneratedImage `json:"images,omitempty"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *chat.Usage `json:"usage,omitempty"`
}

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
	if request == nil {
		return nil, fmt.Errorf("%w: image request is nil", ErrInvalidRequest)
	}
	return p.createOpenRouterImage(ctx, selected, request.Model, request.Prompt, nil, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func (p *openRouterProvider) editImage(ctx context.Context, selected credential, request *image.EditRequest) (*image.GenerateResponse, error) {
	if request == nil || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image edit request and images are required", ErrInvalidRequest)
	}
	images, err := openRouterInputImages(request.Images)
	if err != nil {
		return nil, err
	}
	return p.createOpenRouterImage(ctx, selected, request.Model, request.Prompt, images, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func (p *openRouterProvider) createOpenRouterImage(ctx context.Context, selected credential, model, prompt string, images []string, size, responseFormat, outputFormat string, extraBody map[string]any) (*image.GenerateResponse, error) {
	body, err := buildOpenRouterImageBody(model, prompt, images, size, extraBody, false)
	if err != nil {
		return nil, err
	}
	raw, err := p.doJSON(ctx, selected, ChatCompletionsPath, body)
	if err != nil {
		return nil, err
	}
	var source openRouterImageResponse
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &image.GenerateResponse{Created: source.Created, Size: size, OutputFormat: outputFormat}
	for _, choice := range source.Choices {
		for _, generated := range choice.Message.Images {
			if generated.ImageURL.URL != "" {
				result.Data = append(result.Data, image.Data{URL: generated.ImageURL.URL})
			}
		}
	}
	result.Usage.InputTokens = source.Usage.PromptTokens
	result.Usage.OutputTokens = source.Usage.CompletionTokens
	result.Usage.TotalTokens = source.Usage.TotalTokens
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("%w: OpenRouter response contains no images", ErrInvalidRequest)
	}
	if err := normalizeImageResponse(ctx, p.config, ProviderOpenRouter, selected.hint, imageRequestOptions{ResponseFormat: responseFormat, OutputFormat: outputFormat}, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *openRouterProvider) streamImage(ctx context.Context, selected credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image request is nil", ErrInvalidRequest)
	}
	return p.createOpenRouterImageStream(ctx, selected, request.Model, request.Prompt, nil, request.Size, request.OutputFormat, request.ExtraBody)
}

func (p *openRouterProvider) streamImageEdit(ctx context.Context, selected credential, request *image.EditStreamRequest) (image.Stream, error) {
	if request == nil || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image edit request and images are required", ErrInvalidRequest)
	}
	images, err := openRouterInputImages(request.Images)
	if err != nil {
		return nil, err
	}
	return p.createOpenRouterImageStream(ctx, selected, request.Model, request.Prompt, images, request.Size, request.OutputFormat, request.ExtraBody)
}

func (p *openRouterProvider) createOpenRouterImageStream(ctx context.Context, selected credential, model, prompt string, images []string, size, outputFormat string, extraBody map[string]any) (image.Stream, error) {
	body, err := buildOpenRouterImageBody(model, prompt, images, size, extraBody, true)
	if err != nil {
		return nil, err
	}
	streamContext, cancel := context.WithCancel(normalizeContext(ctx))
	response, err := p.doStream(streamContext, selected, ChatCompletionsPath, body)
	if err != nil {
		cancel()
		return nil, err
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &openRouterImageStream{
		ctx: streamContext, scanner: scanner, terminal: newStreamTerminal(cancel, response.Body.Close),
		config: p.config, selected: selected, request: imageRequestOptions{ResponseFormat: imageResponseFormatBase64, OutputFormat: outputFormat},
		size: size,
	}, nil
}

func buildOpenRouterImageBody(model, prompt string, images []string, size string, extraBody map[string]any, stream bool) (map[string]any, error) {
	if strings.TrimSpace(model) == "" || strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("%w: image model and prompt are required", ErrInvalidRequest)
	}
	chatRequest := &chat.CreateRequest{
		Model:    model,
		Messages: []chat.Message{{Role: chat.RoleUser, Content: openRouterImageContent(prompt, images)}},
	}
	body, err := buildOpenAIChatRequest(ProviderOpenRouter, chatRequest, stream, nil)
	if err != nil {
		return nil, err
	}
	body["modalities"] = []string{"image", "text"}
	if size != "" {
		body["image_config"] = map[string]any{"image_size": size}
	}
	return mergeExtraBody(body, extraBody)
}

func openRouterInputImages(inputs []image.Input) ([]string, error) {
	result := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if strings.TrimSpace(input.ImageURL) != "" {
			result = append(result, input.ImageURL)
			continue
		}
		if input.FileID == "" {
			return nil, fmt.Errorf("%w: image URL is empty", ErrInvalidRequest)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("%w: image edit request contains no supported images", ErrInvalidRequest)
	}
	return result, nil
}

type openRouterImageStream struct {
	ctx      context.Context
	scanner  *bufio.Scanner
	terminal *streamTerminal
	config   ClientConfig
	selected credential
	request  imageRequestOptions
	size     string
	pending  []*image.StreamChunk
}

func (s *openRouterImageStream) Recv() (*image.StreamChunk, error) {
	if s.terminal.isDone() {
		return nil, io.EOF
	}
	if len(s.pending) > 0 {
		chunk := s.pending[0]
		s.pending = s.pending[1:]
		return chunk, nil
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 || bytes.HasPrefix(line, []byte(":")) {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if bytes.Equal(line, []byte("[DONE]")) {
			s.terminal.finish()
			return nil, io.EOF
		}
		if err := decodeStreamAPIError(ProviderOpenRouter, s.selected.hint, line); err != nil {
			return nil, s.terminal.fail(s.ctx, err)
		}
		var wire openRouterImageStreamResponse
		if err := json.Unmarshal(line, &wire); err != nil {
			return nil, s.terminal.fail(s.ctx, err)
		}
		usage := image.Usage{}
		if wire.Usage != nil {
			usage.InputTokens = wire.Usage.PromptTokens
			usage.OutputTokens = wire.Usage.CompletionTokens
			usage.TotalTokens = wire.Usage.TotalTokens
		}
		for _, choice := range wire.Choices {
			for _, generated := range choice.Delta.Images {
				if generated.ImageURL.URL == "" {
					continue
				}
				data := image.Data{URL: generated.ImageURL.URL}
				format, err := normalizeImageData(s.ctx, s.config, ProviderOpenRouter, s.selected.hint, s.request, &data)
				if err != nil {
					return nil, s.terminal.fail(s.ctx, err)
				}
				s.pending = append(s.pending, &image.StreamChunk{
					B64JSON: data.B64JSON, PartialImageIndex: len(s.pending), Created: wire.Created,
					OutputFormat: format, Size: s.size,
				})
			}
		}
		if wire.Usage != nil {
			if len(s.pending) == 0 {
				s.pending = append(s.pending, &image.StreamChunk{Created: wire.Created, OutputFormat: normalizeImageFormat(s.request.OutputFormat), Size: s.size})
			}
			s.pending[0].Usage = usage
		}
		if len(s.pending) > 0 {
			chunk := s.pending[0]
			s.pending = s.pending[1:]
			return chunk, nil
		}
	}
	if err := s.scanner.Err(); err != nil {
		return nil, s.terminal.fail(s.ctx, err)
	}
	if s.terminal.isDone() {
		return nil, io.EOF
	}
	if s.ctx != nil && s.ctx.Err() != nil {
		return nil, s.terminal.fail(s.ctx, s.ctx.Err())
	}
	s.terminal.finish()
	return nil, io.EOF
}

func (s *openRouterImageStream) Close() error {
	return s.terminal.close()
}

var _ imageProvider = (*openRouterProvider)(nil)
var _ imageStreamProvider = (*openRouterProvider)(nil)
var _ imageEditProvider = (*openRouterProvider)(nil)
var _ imageEditStreamProvider = (*openRouterProvider)(nil)
var _ image.Stream = (*openRouterImageStream)(nil)
