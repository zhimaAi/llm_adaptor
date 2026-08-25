// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package doubao

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

const (
	imagePath            = "/images/generations"
	imageStreamBuffer    = 64 * 1024
	imageStreamMaxBuffer = 16 * 1024 * 1024
)

type imageWireResponse struct {
	Created int64 `json:"created,omitempty"`
	Data    []struct {
		URL     string `json:"url,omitempty"`
		B64JSON string `json:"b64_json,omitempty"`
		Size    string `json:"size,omitempty"`
	} `json:"data"`
	Usage struct {
		OutputTokens int `json:"output_tokens,omitempty"`
		TotalTokens  int `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

type imageStreamWireResponse struct {
	Type       string `json:"type,omitempty"`
	Created    int64  `json:"created,omitempty"`
	ImageIndex *int   `json:"image_index,omitempty"`
	URL        string `json:"url,omitempty"`
	B64JSON    string `json:"b64_json,omitempty"`
	Size       string `json:"size,omitempty"`
	Usage      struct {
		OutputTokens int `json:"output_tokens,omitempty"`
		TotalTokens  int `json:"total_tokens,omitempty"`
	} `json:"usage,omitempty"`
}

func (p *Provider) GenerateImage(ctx context.Context, selected provider.Credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image request is nil", provider.ErrInvalidRequest)
	}
	body, err := buildImageBody(request.Model, request.Prompt, nil, request.N, request.Size, request.ResponseFormat, request.OutputFormat, false, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	raw, err := p.DoJSON(ctx, selected, imagePath, body)
	if err != nil {
		return nil, err
	}
	return p.decodeImageResponse(ctx, selected, raw, request.ResponseFormat, request.OutputFormat, request.Size)
}

func (p *Provider) EditImage(ctx context.Context, selected provider.Credential, request *image.EditRequest) (*image.GenerateResponse, error) {
	if request == nil || (len(request.Images) == 0 && request.ExtraBody["image"] == nil) {
		return nil, fmt.Errorf("%w: image edit request and images are required", provider.ErrInvalidRequest)
	}
	images, err := shared.ImageFilesDataURL(request.Images)
	if err != nil {
		return nil, err
	}
	body, err := buildImageBody(request.Model, request.Prompt, images, request.N, request.Size, request.ResponseFormat, request.OutputFormat, false, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	raw, err := p.DoJSON(ctx, selected, imagePath, body)
	if err != nil {
		return nil, err
	}
	return p.decodeImageResponse(ctx, selected, raw, request.ResponseFormat, request.OutputFormat, request.Size)
}

func (p *Provider) StreamImage(ctx context.Context, selected provider.Credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image request is nil", provider.ErrInvalidRequest)
	}
	return p.createImageStream(ctx, selected, request.Model, request.Prompt, nil, request.N, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func (p *Provider) StreamImageEdit(ctx context.Context, selected provider.Credential, request *image.EditStreamRequest) (image.Stream, error) {
	if request == nil || (len(request.Images) == 0 && request.ExtraBody["image"] == nil) {
		return nil, fmt.Errorf("%w: image edit request and images are required", provider.ErrInvalidRequest)
	}
	images, err := shared.ImageFilesDataURL(request.Images)
	if err != nil {
		return nil, err
	}
	return p.createImageStream(ctx, selected, request.Model, request.Prompt, images, request.N, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func buildImageBody(model, prompt string, images []string, n *int, size, responseFormat, outputFormat string, stream bool, extraBody map[string]any) (map[string]any, error) {
	if strings.TrimSpace(model) == "" || strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("%w: image model and prompt are required", provider.ErrInvalidRequest)
	}
	body := map[string]any{"model": model, "prompt": prompt, "stream": stream}
	if len(images) > 0 {
		body["image"] = append([]string(nil), images...)
	}
	if size != "" {
		body["size"] = size
	}
	if responseFormat != "" {
		body["response_format"] = responseFormat
	}
	if outputFormat != "" {
		body["output_format"] = outputFormat
	}
	if n != nil {
		if *n <= 0 {
			return nil, fmt.Errorf("%w: image count must be greater than zero", provider.ErrInvalidRequest)
		}
		if *n == 1 {
			body["sequential_image_generation"] = "disabled"
		} else {
			body["sequential_image_generation"] = "auto"
			body["sequential_image_generation_options"] = map[string]any{"max_images": *n}
		}
	}
	for key, value := range extraBody {
		body[key] = value
	}
	return body, nil
}

func (p *Provider) decodeImageResponse(ctx context.Context, selected provider.Credential, raw []byte, responseFormat, outputFormat, size string) (*image.GenerateResponse, error) {
	if err := transport.DecodeStreamAPIError(provider.IDDoubao, selected.Hint, raw); err != nil {
		return nil, err
	}
	var source imageWireResponse
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &image.GenerateResponse{Created: source.Created, OutputFormat: outputFormat, Size: size}
	result.Usage.OutputTokens = source.Usage.OutputTokens
	result.Usage.TotalTokens = source.Usage.TotalTokens
	for _, item := range source.Data {
		result.Data = append(result.Data, image.Data{URL: item.URL, B64JSON: item.B64JSON})
		if result.Size == "" && item.Size != "" {
			result.Size = item.Size
		}
	}
	options := shared.ImageRequestOptions{ResponseFormat: responseFormat, OutputFormat: outputFormat}
	if err := shared.NormalizeImageResponse(ctx, p.Config(), provider.IDDoubao, selected.Hint, options, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Provider) createImageStream(ctx context.Context, selected provider.Credential, model, prompt string, images []string, n *int, size, responseFormat, outputFormat string, extraBody map[string]any) (image.Stream, error) {
	body, err := buildImageBody(model, prompt, images, n, size, responseFormat, outputFormat, true, extraBody)
	if err != nil {
		return nil, err
	}
	streamContext, cancel := context.WithCancel(shared.NormalizeContext(ctx))
	response, err := p.DoStream(streamContext, selected, imagePath, body)
	if err != nil {
		cancel()
		return nil, err
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, imageStreamBuffer), imageStreamMaxBuffer)
	return &imageStream{
		ctx: streamContext, scanner: scanner, terminal: transport.NewStreamTerminal(cancel, response.Body.Close),
		config: p.Config(), selected: selected, request: shared.ImageRequestOptions{ResponseFormat: responseFormat, OutputFormat: outputFormat},
		outputFormat: outputFormat, size: size,
	}, nil
}

type imageStream struct {
	ctx          context.Context
	scanner      *bufio.Scanner
	terminal     *transport.StreamTerminal
	config       provider.Config
	selected     provider.Credential
	request      shared.ImageRequestOptions
	outputFormat string
	size         string
	index        int
	usageSent    bool
	pending      []*image.StreamChunk
}

func (s *imageStream) Recv() (*image.StreamChunk, error) {
	if s.terminal.IsDone() {
		return nil, io.EOF
	}
	if len(s.pending) > 0 {
		return s.pop(), nil
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 || bytes.HasPrefix(line, []byte("event:")) || bytes.HasPrefix(line, []byte(":")) {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if bytes.Equal(line, []byte("[DONE]")) {
			s.terminal.Finish()
			return nil, io.EOF
		}
		if err := transport.DecodeStreamAPIError(provider.IDDoubao, s.selected.Hint, line); err != nil {
			return nil, s.terminal.Fail(s.ctx, err)
		}
		var wire imageStreamWireResponse
		if err := json.Unmarshal(line, &wire); err != nil {
			return nil, s.terminal.Fail(s.ctx, err)
		}
		if wire.URL != "" || wire.B64JSON != "" {
			data := image.Data{URL: wire.URL, B64JSON: wire.B64JSON}
			format, err := shared.NormalizeImageData(s.ctx, s.config, provider.IDDoubao, s.selected.Hint, s.request, &data)
			if err != nil {
				return nil, s.terminal.Fail(s.ctx, err)
			}
			chunkSize := s.size
			if wire.Size != "" {
				chunkSize = wire.Size
			}
			imageIndex := s.index
			if wire.ImageIndex != nil {
				imageIndex = *wire.ImageIndex
			}
			s.pending = append(s.pending, &image.StreamChunk{Type: wire.Type, B64JSON: data.B64JSON, PartialImageIndex: imageIndex, Created: wire.Created, OutputFormat: format, Size: chunkSize})
			s.index++
		}
		if !s.usageSent && (wire.Usage.OutputTokens != 0 || wire.Usage.TotalTokens != 0) {
			usage := image.Usage{OutputTokens: wire.Usage.OutputTokens, TotalTokens: wire.Usage.TotalTokens}
			if len(s.pending) == 0 {
				s.pending = append(s.pending, &image.StreamChunk{Type: wire.Type, Created: wire.Created, OutputFormat: shared.NormalizeImageFormat(s.outputFormat), Size: s.size})
			}
			s.pending[0].Usage = usage
			s.usageSent = true
		}
		if len(s.pending) > 0 {
			return s.pop(), nil
		}
	}
	if err := s.scanner.Err(); err != nil {
		return nil, s.terminal.Fail(s.ctx, err)
	}
	if s.ctx.Err() != nil {
		return nil, s.terminal.Fail(s.ctx, s.ctx.Err())
	}
	s.terminal.Finish()
	return nil, io.EOF
}

func (s *imageStream) pop() *image.StreamChunk {
	chunk := s.pending[0]
	s.pending = s.pending[1:]
	return chunk
}

func (s *imageStream) Close() error { return s.terminal.Close() }

var _ provider.Image = (*Provider)(nil)
var _ provider.ImageStream = (*Provider)(nil)
var _ provider.ImageEdit = (*Provider)(nil)
var _ provider.ImageEditStream = (*Provider)(nil)
var _ image.Stream = (*imageStream)(nil)
