// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

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

type imageInputWire struct {
	FileID   string `json:"file_id,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type imageGenerateWireRequest struct {
	Model          string `json:"model,omitempty"`
	Prompt         string `json:"prompt"`
	N              *int   `json:"n,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Size           string `json:"size,omitempty"`
	User           string `json:"user,omitempty"`
	OutputFormat   string `json:"output_format,omitempty"`
}

type imageEditWireRequest struct {
	Model          string           `json:"model,omitempty"`
	Images         []imageInputWire `json:"images"`
	Mask           *imageInputWire  `json:"mask,omitempty"`
	Prompt         string           `json:"prompt"`
	N              *int             `json:"n,omitempty"`
	Quality        string           `json:"quality,omitempty"`
	ResponseFormat string           `json:"response_format,omitempty"`
	Size           string           `json:"size,omitempty"`
	User           string           `json:"user,omitempty"`
	OutputFormat   string           `json:"output_format,omitempty"`
}

func (p *Provider) GenerateImage(ctx context.Context, selected provider.Credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image prompt is required", provider.ErrInvalidRequest)
	}
	body, err := buildImageGenerateRequest(p.spec, request, false)
	if err != nil {
		return nil, err
	}
	path := p.spec.ImagePath
	if path == "" {
		path = ImagePath
	}
	raw, err := p.DoJSON(ctx, selected, path, body)
	if err != nil {
		return nil, err
	}
	result, err := DecodeImageResponse(raw)
	if err != nil {
		return nil, err
	}
	options := shared.ImageRequestOptions{ResponseFormat: request.ResponseFormat, OutputFormat: request.OutputFormat}
	if err := shared.NormalizeImageResponse(ctx, p.config, p.spec.Info.ID, selected.Hint, options, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Provider) StreamImage(ctx context.Context, selected provider.Credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image prompt is required", provider.ErrInvalidRequest)
	}
	body, err := buildImageGenerateRequest(p.spec, &request.GenerateRequest, true)
	if err != nil {
		return nil, err
	}
	path := p.spec.ImagePath
	if path == "" {
		path = ImagePath
	}
	streamContext, cancel := context.WithCancel(shared.NormalizeContext(ctx))
	response, err := p.DoStream(streamContext, selected, path, body)
	if err != nil {
		cancel()
		return nil, err
	}
	options := shared.ImageRequestOptions{ResponseFormat: request.ResponseFormat, OutputFormat: request.OutputFormat}
	return newImageStream(streamContext, response.Body, cancel, p.config, p.spec.Info.ID, selected.Hint, options), nil
}

func (p *Provider) EditImage(ctx context.Context, selected provider.Credential, request *image.EditRequest) (*image.GenerateResponse, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image edit prompt and images are required", provider.ErrInvalidRequest)
	}
	body, err := buildImageEditRequest(p.spec, request, false)
	if err != nil {
		return nil, err
	}
	path := p.spec.ImageEditPath
	if path == "" {
		path = ImageEditPath
	}
	raw, err := p.DoJSON(ctx, selected, path, body)
	if err != nil {
		return nil, err
	}
	result, err := DecodeImageResponse(raw)
	if err != nil {
		return nil, err
	}
	options := shared.ImageRequestOptions{ResponseFormat: request.ResponseFormat, OutputFormat: request.OutputFormat}
	if err := shared.NormalizeImageResponse(ctx, p.config, p.spec.Info.ID, selected.Hint, options, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Provider) StreamImageEdit(ctx context.Context, selected provider.Credential, request *image.EditStreamRequest) (image.Stream, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image edit prompt and images are required", provider.ErrInvalidRequest)
	}
	body, err := buildImageEditRequest(p.spec, &request.EditRequest, true)
	if err != nil {
		return nil, err
	}
	path := p.spec.ImageEditPath
	if path == "" {
		path = ImageEditPath
	}
	streamContext, cancel := context.WithCancel(shared.NormalizeContext(ctx))
	response, err := p.DoStream(streamContext, selected, path, body)
	if err != nil {
		cancel()
		return nil, err
	}
	options := shared.ImageRequestOptions{ResponseFormat: request.ResponseFormat, OutputFormat: request.OutputFormat}
	return newImageStream(streamContext, response.Body, cancel, p.config, p.spec.Info.ID, selected.Hint, options), nil
}

func buildImageGenerateRequest(spec Spec, request *image.GenerateRequest, stream bool) (map[string]any, error) {
	wire := imageGenerateWireRequest{Model: request.Model, Prompt: request.Prompt, N: request.N, Quality: request.Quality, ResponseFormat: request.ResponseFormat, Size: request.Size, User: request.User, OutputFormat: request.OutputFormat}
	body, err := shared.MergeExtraBody(wire, nil)
	if err != nil {
		return nil, err
	}
	body["stream"] = stream
	filterFields(body, spec.ImageFields, AllImageFields)
	return shared.MergeExtraBody(body, request.ExtraBody)
}

func buildImageEditRequest(spec Spec, request *image.EditRequest, stream bool) (map[string]any, error) {
	images, err := imageInputs(request.Images)
	if err != nil {
		return nil, err
	}
	var mask *imageInputWire
	if request.Mask != nil {
		converted, err := imageInput(*request.Mask)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid image mask: %v", provider.ErrInvalidRequest, err)
		}
		mask = &converted
	}
	wire := imageEditWireRequest{Model: request.Model, Images: images, Mask: mask, Prompt: request.Prompt, N: request.N, Quality: request.Quality, ResponseFormat: request.ResponseFormat, Size: request.Size, User: request.User, OutputFormat: request.OutputFormat}
	body, err := shared.MergeExtraBody(wire, nil)
	if err != nil {
		return nil, err
	}
	body["stream"] = stream
	filterFields(body, spec.ImageFields, AllImageFields)
	return shared.MergeExtraBody(body, request.ExtraBody)
}

func imageInputs(inputs []image.Input) ([]imageInputWire, error) {
	result := make([]imageInputWire, len(inputs))
	for index, input := range inputs {
		converted, err := imageInput(input)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid image input at index %d: %v", provider.ErrInvalidRequest, index, err)
		}
		result[index] = converted
	}
	return result, nil
}

func imageInput(input image.Input) (imageInputWire, error) {
	fileID := strings.TrimSpace(input.FileID)
	imageURL := strings.TrimSpace(input.ImageURL)
	if (fileID == "") == (imageURL == "") {
		return imageInputWire{}, fmt.Errorf("exactly one of file_id or image_url is required")
	}
	return imageInputWire{FileID: fileID, ImageURL: imageURL}, nil
}

type imageStream struct {
	ctx      context.Context
	scanner  *bufio.Scanner
	terminal *transport.StreamTerminal
	provider provider.ID
	hint     string
	config   provider.Config
	request  shared.ImageRequestOptions
}

func newImageStream(ctx context.Context, body io.ReadCloser, cancel context.CancelFunc, config provider.Config, providerID provider.ID, hint string, request shared.ImageRequestOptions) *imageStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &imageStream{ctx: ctx, scanner: scanner, terminal: transport.NewStreamTerminal(cancel, body.Close), config: config, provider: providerID, hint: hint, request: request}
}

func (s *imageStream) Recv() (*image.StreamChunk, error) {
	if s.terminal.IsDone() {
		return nil, io.EOF
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if bytes.Equal(line, []byte("[DONE]")) {
			s.terminal.Finish()
			return nil, io.EOF
		}
		if err := transport.DecodeStreamAPIError(s.provider, s.hint, line); err != nil {
			return nil, s.terminal.Fail(s.ctx, err)
		}
		var wire struct {
			Type              string      `json:"type"`
			URL               string      `json:"url"`
			B64JSON           string      `json:"b64_json"`
			PartialImageIndex int         `json:"partial_image_index"`
			ImageIndex        int         `json:"image_index"`
			Created           int64       `json:"created"`
			Background        string      `json:"background"`
			OutputFormat      string      `json:"output_format"`
			Quality           string      `json:"quality"`
			Size              string      `json:"size"`
			Usage             image.Usage `json:"usage"`
		}
		if err := json.Unmarshal(line, &wire); err != nil {
			return nil, s.terminal.Fail(s.ctx, err)
		}
		chunk := &image.StreamChunk{Type: wire.Type, B64JSON: wire.B64JSON, PartialImageIndex: wire.PartialImageIndex, Created: wire.Created, Background: wire.Background, OutputFormat: wire.OutputFormat, Quality: wire.Quality, Size: wire.Size, Usage: wire.Usage}
		if chunk.PartialImageIndex == 0 && wire.ImageIndex != 0 {
			chunk.PartialImageIndex = wire.ImageIndex
		}
		if wire.URL != "" || wire.B64JSON != "" {
			data := image.Data{URL: wire.URL, B64JSON: wire.B64JSON}
			format, err := shared.NormalizeImageData(s.ctx, s.config, s.provider, s.hint, s.request, &data)
			if err != nil {
				return nil, s.terminal.Fail(s.ctx, err)
			}
			chunk.B64JSON = data.B64JSON
			if chunk.OutputFormat == "" {
				chunk.OutputFormat = format
			}
		}
		return chunk, nil
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

func (s *imageStream) Close() error { return s.terminal.Close() }

var _ provider.Image = (*Provider)(nil)
var _ provider.ImageStream = (*Provider)(nil)
var _ provider.ImageEdit = (*Provider)(nil)
var _ provider.ImageEditStream = (*Provider)(nil)
var _ image.Stream = (*imageStream)(nil)
