// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
)

const (
	aliDefaultServiceBaseURL = "https://dashscope.aliyuncs.com"
	aliRerankPath            = "/api/v1/services/rerank/text-rerank/text-rerank"
	aliCompatibleRerankPath  = "/compatible-api/v1/reranks"
	aliImagePath             = "/api/v1/services/aigc/multimodal-generation/generation"
)

const aliQwen3RerankModelPrefix = "qwen3-rerank"

type aliRerankWireResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type aliProvider struct{ *openAICompatibleProvider }

func newAliProvider(config ClientConfig) *aliProvider {
	return &aliProvider{newOpenAICompatibleProvider(config, ProviderInfo{
		ID: ProviderAli, DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", DefaultServiceBaseURL: aliDefaultServiceBaseURL,
		Capabilities: []Capability{CapabilityChat, CapabilityEmbedding, CapabilityRerank, CapabilityImage},
	})}
}

func (p *aliProvider) serviceBaseURL() string {
	return p.config.ServiceBaseURL
}

func (p *aliProvider) createRerank(ctx context.Context, selected credential, request *rerank.CreateRequest) (*rerank.CreateResponse, error) {
	if request == nil || request.Model == "" || request.Query == "" || len(request.Documents) == 0 {
		return nil, fmt.Errorf("%w: rerank model, query and documents are required", ErrInvalidRequest)
	}
	parameters := map[string]any{}
	body := map[string]any{"model": request.Model}
	if hasModelPrefix(request.Model, aliQwen3RerankModelPrefix) {
		body["query"] = request.Query
		body["documents"] = append([]string(nil), request.Documents...)
		if request.TopN != nil {
			body["top_n"] = *request.TopN
		}
	} else {
		if request.TopN != nil {
			parameters["top_n"] = *request.TopN
		}
		body["input"] = map[string]any{"query": request.Query, "documents": append([]string(nil), request.Documents...)}
		body["parameters"] = parameters
	}
	body, err := mergeExtraBody(body, request.ExtraBody, rerankReservedRequestKeys)
	if err != nil {
		return nil, err
	}
	rerankPath := aliRerankPath
	if hasModelPrefix(request.Model, aliQwen3RerankModelPrefix) {
		rerankPath = aliCompatibleRerankPath
	}
	raw, err := p.doJSON(ctx, selected, joinURLPath(p.serviceBaseURL(), rerankPath), body)
	if err != nil {
		return nil, err
	}
	var source struct {
		ID        string `json:"id"`
		RequestID string `json:"request_id"`
		Code      string `json:"code"`
		Message   string `json:"message"`
		Output    struct {
			Results []aliRerankWireResult `json:"results"`
		} `json:"output"`
		Results []aliRerankWireResult `json:"results"`
		Usage   struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	wireResults := source.Output.Results
	if len(wireResults) == 0 {
		wireResults = source.Results
	}
	if source.Code != "" && len(wireResults) == 0 {
		return nil, &APIError{Provider: ProviderAli, Code: source.Code, Message: source.Message, RequestID: source.RequestID, CredentialHint: selected.hint, Raw: raw}
	}
	responseID := source.RequestID
	if responseID == "" {
		responseID = source.ID
	}
	response := &rerank.CreateResponse{ID: responseID, Results: make([]rerank.Result, len(wireResults))}
	for index, item := range wireResults {
		response.Results[index] = rerank.Result{Index: item.Index, RelevanceScore: item.RelevanceScore}
	}
	if source.Usage.TotalTokens != 0 {
		tokens := source.Usage.TotalTokens
		response.Meta = &rerank.Meta{Tokens: &rerank.Tokens{InputTokens: &tokens}}
	}
	return response, nil
}

func (p *aliProvider) generateImage(ctx context.Context, selected credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil || request.Model == "" || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image model and prompt are required", ErrInvalidRequest)
	}
	return p.createAliImage(ctx, selected, request.Model, request.Prompt, nil, request.N, request.Quality, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func (p *aliProvider) editImage(ctx context.Context, selected credential, request *image.EditRequest) (*image.GenerateResponse, error) {
	if request == nil || request.Model == "" || strings.TrimSpace(request.Prompt) == "" || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image model, prompt and images are required", ErrInvalidRequest)
	}
	if request.Mask != nil {
		return nil, &UnsupportedParameterError{Provider: ProviderAli, Capability: CapabilityImage, Parameter: "mask"}
	}
	images := make([]string, 0, len(request.Images))
	for _, input := range request.Images {
		if input.FileID != "" || strings.TrimSpace(input.ImageURL) == "" {
			return nil, &UnsupportedParameterError{Provider: ProviderAli, Capability: CapabilityImage, Parameter: "images.file_id"}
		}
		images = append(images, input.ImageURL)
	}
	return p.createAliImage(ctx, selected, request.Model, request.Prompt, images, request.N, request.Quality, request.Size, request.ResponseFormat, request.OutputFormat, request.ExtraBody)
}

func (p *aliProvider) createAliImage(ctx context.Context, selected credential, model, prompt string, images []string, n *int, quality, size, responseFormat, outputFormat string, extraBody map[string]any) (*image.GenerateResponse, error) {
	parameters := map[string]any{}
	if size != "" {
		parameters["size"] = size
	}
	if n != nil {
		parameters["max_images"] = *n
		parameters["sequential_image_generation"] = *n > 1
	}
	if quality != "" {
		parameters["quality"] = quality
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
	parameters, err := mergeExtraBody(parameters, extraBody, imageGenerateReservedRequestKeys, imageEditReservedRequestKeys, map[string]struct{}{
		"max_images": {}, "sequential_image_generation": {},
	})
	if err != nil {
		return nil, err
	}
	body["parameters"] = parameters
	raw, err := p.doJSON(ctx, selected, joinURLPath(p.serviceBaseURL(), aliImagePath), body)
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
		Usage struct {
			ImageCount int `json:"image_count"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	if source.Code != "" {
		return nil, &APIError{Provider: ProviderAli, Code: source.Code, Message: source.Message, RequestID: source.RequestID, CredentialHint: selected.hint, Raw: raw}
	}
	result := &image.GenerateResponse{Quality: quality, Size: size, OutputFormat: outputFormat}
	for _, choice := range source.Output.Choices {
		for _, content := range choice.Message.Content {
			if content.Image != "" {
				result.Data = append(result.Data, image.Data{URL: content.Image})
			}
		}
	}
	if err := normalizeImageResponse(ctx, p.config, ProviderAli, selected.hint, imageRequestOptions{ResponseFormat: responseFormat, OutputFormat: outputFormat}, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *aliProvider) streamImage(ctx context.Context, selected credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image request is nil", ErrInvalidRequest)
	}
	copyRequest := request.GenerateRequest
	copyRequest.ResponseFormat = imageResponseFormatBase64
	response, err := p.generateImage(ctx, selected, &copyRequest)
	if err != nil {
		return nil, err
	}
	return &singleImageStream{response: response, terminal: newStreamTerminal(nil, nil)}, nil
}

func (p *aliProvider) streamImageEdit(ctx context.Context, selected credential, request *image.EditStreamRequest) (image.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image edit request is nil", ErrInvalidRequest)
	}
	copyRequest := request.EditRequest
	copyRequest.ResponseFormat = imageResponseFormatBase64
	response, err := p.editImage(ctx, selected, &copyRequest)
	if err != nil {
		return nil, err
	}
	return &singleImageStream{response: response, terminal: newStreamTerminal(nil, nil)}, nil
}

type singleImageStream struct {
	response *image.GenerateResponse
	index    int
	terminal *streamTerminal
}

func (s *singleImageStream) Recv() (*image.StreamChunk, error) {
	if s.terminal.isDone() {
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
		s.terminal.finish()
		return chunk, nil
	}
	item := s.response.Data[s.index]
	chunk.B64JSON = item.B64JSON
	chunk.PartialImageIndex = s.index
	s.index++
	if s.index == len(s.response.Data) {
		s.terminal.finish()
	}
	return chunk, nil
}
func (s *singleImageStream) Close() error { return s.terminal.close() }

var _ rerankProvider = (*aliProvider)(nil)
var _ imageProvider = (*aliProvider)(nil)
var _ imageStreamProvider = (*aliProvider)(nil)
var _ imageEditProvider = (*aliProvider)(nil)
var _ imageEditStreamProvider = (*aliProvider)(nil)
var _ image.Stream = (*singleImageStream)(nil)
