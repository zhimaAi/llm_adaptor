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
	aliImagePath             = "/api/v1/services/aigc/multimodal-generation/generation"
)

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
	documents := make([]string, len(request.Documents))
	for index, document := range request.Documents {
		documents[index] = document.Text
	}
	parameters := map[string]any{}
	if request.TopN != nil {
		parameters["top_n"] = *request.TopN
	}
	if request.ReturnDocuments != nil {
		parameters["return_documents"] = *request.ReturnDocuments
	}
	body := map[string]any{
		"model":      request.Model,
		"input":      map[string]any{"query": request.Query, "documents": documents},
		"parameters": parameters,
	}
	for key, value := range request.ExtraBody {
		body[key] = value
	}
	raw, err := p.doJSON(ctx, selected, joinURLPath(p.serviceBaseURL(), aliRerankPath), body)
	if err != nil {
		return nil, err
	}
	var source struct {
		RequestID string `json:"request_id"`
		Code      string `json:"code"`
		Message   string `json:"message"`
		Output    struct {
			Results []rerank.Result `json:"results"`
		} `json:"output"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	if source.Code != "" && len(source.Output.Results) == 0 {
		return nil, &APIError{Provider: ProviderAli, Code: source.Code, Message: source.Message, RequestID: source.RequestID, CredentialHint: selected.hint, Raw: raw}
	}
	return &rerank.CreateResponse{ID: source.RequestID, Results: source.Output.Results, Usage: rerank.Usage{TotalTokens: source.Usage.TotalTokens}, RawResponse: raw}, nil
}

func (p *aliProvider) generateImage(ctx context.Context, selected credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil || request.Model == "" || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image model and prompt are required", ErrInvalidRequest)
	}
	parameters := map[string]any{}
	if request.Size != "" {
		parameters["size"] = request.Size
	}
	for key, value := range request.ExtraBody {
		parameters[key] = value
	}
	content := []any{map[string]any{"text": request.Prompt}}
	for _, inputImage := range request.Image {
		content = append(content, map[string]any{"image": inputImage})
	}
	body := map[string]any{
		"model":      request.Model,
		"input":      map[string]any{"messages": []any{map[string]any{"role": "user", "content": content}}},
		"parameters": parameters,
	}
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
	result := &image.GenerateResponse{RawResponse: raw, Usage: image.Usage{GeneratedImages: source.Usage.ImageCount}}
	for _, choice := range source.Output.Choices {
		for _, content := range choice.Message.Content {
			if content.Image != "" {
				result.Data = append(result.Data, image.Data{URL: content.Image})
			}
		}
	}
	if err := normalizeImageResponse(ctx, p.config, ProviderAli, selected.hint, request, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *aliProvider) streamImage(ctx context.Context, selected credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: image request is nil", ErrInvalidRequest)
	}
	response, err := p.generateImage(ctx, selected, &request.GenerateRequest)
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
		chunk.RawResponse = s.response.RawResponse
	}
	if len(s.response.Data) == 0 {
		s.terminal.finish()
		return chunk, nil
	}
	item := s.response.Data[s.index]
	chunk.URL = item.URL
	chunk.B64JSON = item.B64JSON
	chunk.Format = item.Format
	chunk.MIMEType = item.MIMEType
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
var _ image.Stream = (*singleImageStream)(nil)
