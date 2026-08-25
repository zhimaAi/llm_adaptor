// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openrouter

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
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

const (
	streamInitialBuffer = 64 * 1024
	streamMaximumBuffer = 16 * 1024 * 1024
)

type Provider struct{ *openai.Provider }

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

func (p *Provider) GenerateImage(ctx context.Context, selected provider.Credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image request is nil", provider.ErrInvalidRequest)
	}
	return p.createImage(ctx, selected, request.Model, request.Prompt, nil, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func (p *Provider) EditImage(ctx context.Context, selected provider.Credential, request *image.EditRequest) (*image.GenerateResponse, error) {
	if request == nil || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image edit request and images are required", provider.ErrInvalidRequest)
	}
	images, err := openRouterInputImages(request.Images)
	if err != nil {
		return nil, err
	}
	return p.createImage(ctx, selected, request.Model, request.Prompt, images, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func (p *Provider) createImage(ctx context.Context, selected provider.Credential, model, prompt string, images []string, size, responseFormat, outputFormat string, extraBody map[string]any) (*image.GenerateResponse, error) {
	body, err := buildOpenRouterImageBody(model, prompt, images, size, extraBody, false)
	if err != nil {
		return nil, err
	}
	raw, err := p.DoJSON(ctx, selected, openai.ChatPath, body)
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
		return nil, fmt.Errorf("%w: OpenRouter response contains no images", provider.ErrInvalidRequest)
	}
	options := shared.ImageRequestOptions{ResponseFormat: responseFormat, OutputFormat: outputFormat}
	if err := shared.NormalizeImageResponse(ctx, p.Config(), provider.IDOpenRouter, selected.Hint, options, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Provider) StreamImage(ctx context.Context, selected provider.Credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image request is nil", provider.ErrInvalidRequest)
	}
	return p.createImageStream(ctx, selected, request.Model, request.Prompt, nil, request.Size, request.OutputFormat, request.ExtraBody)
}

func (p *Provider) StreamImageEdit(ctx context.Context, selected provider.Credential, request *image.EditStreamRequest) (image.Stream, error) {
	if request == nil || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image edit request and images are required", provider.ErrInvalidRequest)
	}
	images, err := openRouterInputImages(request.Images)
	if err != nil {
		return nil, err
	}
	return p.createImageStream(ctx, selected, request.Model, request.Prompt, images, request.Size, request.OutputFormat, request.ExtraBody)
}

func (p *Provider) createImageStream(ctx context.Context, selected provider.Credential, model, prompt string, images []string, size, outputFormat string, extraBody map[string]any) (image.Stream, error) {
	body, err := buildOpenRouterImageBody(model, prompt, images, size, extraBody, true)
	if err != nil {
		return nil, err
	}
	streamContext, cancel := context.WithCancel(shared.NormalizeContext(ctx))
	response, err := p.DoStream(streamContext, selected, openai.ChatPath, body)
	if err != nil {
		cancel()
		return nil, err
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &openRouterImageStream{
		ctx: streamContext, scanner: scanner, terminal: transport.NewStreamTerminal(cancel, response.Body.Close),
		config: p.Config(), selected: selected, request: shared.ImageRequestOptions{ResponseFormat: "b64_json", OutputFormat: outputFormat},
		size: size,
	}, nil
}

func buildOpenRouterImageBody(model, prompt string, images []string, size string, extraBody map[string]any, stream bool) (map[string]any, error) {
	if strings.TrimSpace(model) == "" || strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("%w: image model and prompt are required", provider.ErrInvalidRequest)
	}
	chatRequest := &chat.CreateRequest{
		Model:    model,
		Messages: []chat.Message{{Role: chat.RoleUser, Content: openRouterImageContent(prompt, images)}},
	}
	body, err := openai.BuildChatRequest(imageChatSpec(), chatRequest, stream, nil)
	if err != nil {
		return nil, err
	}
	body["modalities"] = []string{"image", "text"}
	if size != "" {
		body["image_config"] = map[string]any{"image_size": size}
	}
	return shared.MergeExtraBody(body, extraBody)
}

func openRouterInputImages(inputs []image.File) ([]string, error) {
	result, err := shared.ImageFilesDataURL(inputs)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("%w: image edit request contains no supported images", provider.ErrInvalidRequest)
	}
	return result, nil
}

type openRouterImageStream struct {
	ctx            context.Context
	scanner        *bufio.Scanner
	terminal       *transport.StreamTerminal
	config         provider.Config
	selected       provider.Credential
	request        shared.ImageRequestOptions
	size           string
	pending        []*image.StreamChunk
	nextImageIndex int
}

func (s *openRouterImageStream) Recv() (*image.StreamChunk, error) {
	if s.terminal.IsDone() {
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
			s.terminal.Finish()
			return nil, io.EOF
		}
		if err := transport.DecodeStreamAPIError(provider.IDOpenRouter, s.selected.Hint, line); err != nil {
			return nil, s.terminal.Fail(s.ctx, err)
		}
		var wire openRouterImageStreamResponse
		if err := json.Unmarshal(line, &wire); err != nil {
			return nil, s.terminal.Fail(s.ctx, err)
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
				format, err := shared.NormalizeImageData(s.ctx, s.config, provider.IDOpenRouter, s.selected.Hint, s.request, &data)
				if err != nil {
					return nil, s.terminal.Fail(s.ctx, err)
				}
				s.pending = append(s.pending, &image.StreamChunk{
					B64JSON: data.B64JSON, PartialImageIndex: s.nextImageIndex, Created: wire.Created,
					OutputFormat: format, Size: s.size,
				})
				s.nextImageIndex++
			}
		}
		if wire.Usage != nil {
			if len(s.pending) == 0 {
				s.pending = append(s.pending, &image.StreamChunk{Created: wire.Created, OutputFormat: shared.NormalizeImageFormat(s.request.OutputFormat), Size: s.size})
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
		return nil, s.terminal.Fail(s.ctx, err)
	}
	if s.terminal.IsDone() {
		return nil, io.EOF
	}
	if s.ctx != nil && s.ctx.Err() != nil {
		return nil, s.terminal.Fail(s.ctx, s.ctx.Err())
	}
	s.terminal.Finish()
	return nil, io.EOF
}

func (s *openRouterImageStream) Close() error {
	return s.terminal.Close()
}

var _ provider.Image = (*Provider)(nil)
var _ provider.ImageStream = (*Provider)(nil)
var _ provider.ImageEdit = (*Provider)(nil)
var _ provider.ImageEditStream = (*Provider)(nil)
var _ image.Stream = (*openRouterImageStream)(nil)
