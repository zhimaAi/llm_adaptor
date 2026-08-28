// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ali

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

const imagePath = "/api/v1/services/aigc/multimodal-generation/generation"

func configureImage(spec *openai.Spec, _ provider.Config) {
	spec.ImageFields = openai.Fields("n", "response_format", "size", "output_format")
}

func (p *Provider) GenerateImage(ctx context.Context, selected provider.Credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil || request.Model == "" || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image model and prompt are required", provider.ErrInvalidRequest)
	}
	return p.createImage(ctx, selected, request.Model, request.Prompt, nil, request.N, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func (p *Provider) EditImage(ctx context.Context, selected provider.Credential, request *image.EditRequest) (*image.GenerateResponse, error) {
	if request == nil || request.Model == "" || strings.TrimSpace(request.Prompt) == "" || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image model, prompt and images are required", provider.ErrInvalidRequest)
	}
	images, err := shared.ImageFilesDataURL(request.Images)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("%w: image edit request contains no supported images", provider.ErrInvalidRequest)
	}
	return p.createImage(ctx, selected, request.Model, request.Prompt, images, request.N, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func (p *Provider) createImage(ctx context.Context, selected provider.Credential, model, prompt string, images []string, n *int, size, responseFormat, outputFormat string, extraBody map[string]any) (*image.GenerateResponse, error) {
	parameters := map[string]any{}
	if size != "" {
		parameters["size"] = size
	}
	if n != nil {
		if *n <= 0 {
			return nil, fmt.Errorf("%w: image count must be greater than zero", provider.ErrInvalidRequest)
		}
		parameters["n"] = *n
	}
	content := []any{map[string]any{"text": prompt}}
	for _, inputImage := range images {
		content = append(content, map[string]any{"image": inputImage})
	}
	body := map[string]any{
		"model":      model,
		"input":      map[string]any{"messages": []any{map[string]any{"role": "user", "content": content}}},
		"parameters": parameters,
	}
	parameters, err := shared.MergeExtraBody(parameters, extraBody)
	if err != nil {
		return nil, err
	}
	body["parameters"] = parameters
	raw, err := p.DoJSON(ctx, selected, transport.JoinURLPath(p.Config().ServiceBaseURL, imagePath), body)
	if err != nil {
		return nil, err
	}
	var source struct {
		RequestID string `json:"request_id"`
		Code      string `json:"code"`
		Message   string `json:"message"`
		Output    struct {
			Choices []struct {
				Message struct {
					Content []struct {
						Image string `json:"image"`
					} `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		} `json:"output"`
	}
	if err := transport.DecodeJSONResponse(provider.IDAli, selected.Hint, raw, &source); err != nil {
		return nil, err
	}
	if source.Code != "" {
		return nil, &provider.APIError{Provider: provider.IDAli, Code: source.Code, Message: source.Message, RequestID: source.RequestID, CredentialHint: selected.Hint, Raw: raw}
	}
	result := &image.GenerateResponse{Size: size, OutputFormat: outputFormat}
	for _, choice := range source.Output.Choices {
		for _, content := range choice.Message.Content {
			if content.Image != "" {
				result.Data = append(result.Data, image.Data{URL: content.Image})
			}
		}
	}
	options := shared.ImageRequestOptions{ResponseFormat: responseFormat, OutputFormat: outputFormat}
	if err := shared.NormalizeImageResponse(ctx, p.Config(), provider.IDAli, selected.Hint, options, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *Provider) StreamImage(ctx context.Context, selected provider.Credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image request is nil", provider.ErrInvalidRequest)
	}
	copyRequest := request.GenerateRequest
	copyRequest.ResponseFormat = "b64_json"
	response, err := p.GenerateImage(ctx, selected, &copyRequest)
	if err != nil {
		return nil, err
	}
	return &singleImageStream{response: response, terminal: transport.NewStreamTerminal(nil, nil)}, nil
}

func (p *Provider) StreamImageEdit(ctx context.Context, selected provider.Credential, request *image.EditStreamRequest) (image.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image edit request is nil", provider.ErrInvalidRequest)
	}
	copyRequest := request.EditRequest
	copyRequest.ResponseFormat = "b64_json"
	response, err := p.EditImage(ctx, selected, &copyRequest)
	if err != nil {
		return nil, err
	}
	return &singleImageStream{response: response, terminal: transport.NewStreamTerminal(nil, nil)}, nil
}

type singleImageStream struct {
	response *image.GenerateResponse
	index    int
	terminal *transport.StreamTerminal
}

func (s *singleImageStream) Recv() (*image.StreamChunk, error) {
	if s.terminal.IsDone() {
		return nil, io.EOF
	}
	chunk := &image.StreamChunk{}
	if s.index == 0 {
		chunk.Usage = s.response.Usage
		chunk.Created = s.response.Created
		chunk.Background = s.response.Background
		chunk.OutputFormat = s.response.OutputFormat
		chunk.Quality = s.response.Quality
		chunk.Size = s.response.Size
	}
	if len(s.response.Data) == 0 {
		s.terminal.Finish()
		return chunk, nil
	}
	item := s.response.Data[s.index]
	chunk.B64JSON = item.B64JSON
	chunk.PartialImageIndex = s.index
	s.index++
	if s.index == len(s.response.Data) {
		s.terminal.Finish()
	}
	return chunk, nil
}

func (s *singleImageStream) Close() error { return s.terminal.Close() }

var _ provider.Image = (*Provider)(nil)
var _ provider.ImageStream = (*Provider)(nil)
var _ provider.ImageEdit = (*Provider)(nil)
var _ provider.ImageEditStream = (*Provider)(nil)
var _ image.Stream = (*singleImageStream)(nil)
