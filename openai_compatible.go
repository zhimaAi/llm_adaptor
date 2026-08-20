// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
)

const (
	streamInitialBuffer = 64 * 1024
	streamMaximumBuffer = 16 * 1024 * 1024
	imagePartialFailed  = "image_generation.partial_failed"
)

type openAICompatibleProvider struct {
	config              ClientConfig
	providerInfo        ProviderInfo
	authorizationPrefix *string
	chatPath            string
	embeddingPath       string
	imagePath           string
	rerankPath          string
	rerankBaseURL       string
	rerankDocumentsKey  string
	rerankTopKey        string
}

func newOpenAICompatibleProvider(config ClientConfig, info ProviderInfo) *openAICompatibleProvider {
	prefix := bearerPrefix
	return &openAICompatibleProvider{config: config, providerInfo: info, authorizationPrefix: &prefix}
}

func (p *openAICompatibleProvider) info() ProviderInfo { return p.providerInfo }

func (p *openAICompatibleProvider) createChat(ctx context.Context, selected credential, request *chat.CreateRequest) (*chat.CreateResponse, error) {
	if request == nil || strings.TrimSpace(request.Model) == "" || len(request.Messages) == 0 {
		return nil, fmt.Errorf("%w: model and messages are required", ErrInvalidRequest)
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	body["stream"] = false
	path := p.chatPath
	if path == "" {
		path = ChatCompletionsPath
	}
	raw, err := p.doJSON(ctx, selected, path, body)
	if err != nil {
		return nil, err
	}
	result := &chat.CreateResponse{}
	if err := json.Unmarshal(raw, result); err != nil {
		return nil, err
	}
	result.RawResponse = append(result.RawResponse[:0], raw...)
	result.ExtraFields = extractExtraFields(raw, "id", "object", "created", "model", "system_fingerprint", "service_tier", "choices", "usage")
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("%w: response contains no choices", ErrInvalidRequest)
	}
	normalizeThinkTaggedResponse(result)
	return result, nil
}

func (p *openAICompatibleProvider) streamChat(ctx context.Context, selected credential, request *chat.StreamRequest) (chat.Stream, error) {
	if request == nil || strings.TrimSpace(request.Model) == "" || len(request.Messages) == 0 {
		return nil, fmt.Errorf("%w: model and messages are required", ErrInvalidRequest)
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	body["stream"] = true
	path := p.chatPath
	if path == "" {
		path = ChatCompletionsPath
	}
	streamContext, cancel := context.WithCancel(ctx)
	response, err := p.doStream(streamContext, selected, path, body)
	if err != nil {
		cancel()
		return nil, err
	}
	return newThinkTagStream(newOpenAIChatStream(response.Body, cancel, p.providerInfo.ID, selected.hint)), nil
}

func (p *openAICompatibleProvider) createEmbedding(ctx context.Context, selected credential, request *embedding.CreateRequest) (*embedding.CreateResponse, error) {
	if request == nil || strings.TrimSpace(request.Model) == "" {
		return nil, fmt.Errorf("%w: embedding model is required", ErrInvalidRequest)
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	path := p.embeddingPath
	if path == "" {
		path = EmbeddingsPath
	}
	raw, err := p.doJSON(ctx, selected, path, body)
	if err != nil {
		return nil, err
	}
	result := &embedding.CreateResponse{}
	if err := json.Unmarshal(raw, result); err != nil {
		return nil, err
	}
	result.RawResponse = append(result.RawResponse[:0], raw...)
	result.ExtraFields = extractExtraFields(raw, "object", "data", "model", "usage")
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("%w: response contains no embeddings", ErrInvalidRequest)
	}
	return result, nil
}

func (p *openAICompatibleProvider) generateImage(ctx context.Context, selected credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image prompt is required", ErrInvalidRequest)
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	path := p.imagePath
	if path == "" {
		path = ImageGenerationsPath
	}
	raw, err := p.doJSON(ctx, selected, path, body)
	if err != nil {
		return nil, err
	}
	result := &image.GenerateResponse{}
	if err := json.Unmarshal(raw, result); err != nil {
		return nil, err
	}
	result.RawResponse = append(result.RawResponse[:0], raw...)
	result.ExtraFields = extractExtraFields(raw, "created", "data", "usage")
	if err := normalizeImageResponse(ctx, p.config, p.providerInfo.ID, selected.hint, request, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *openAICompatibleProvider) streamImage(ctx context.Context, selected credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image prompt is required", ErrInvalidRequest)
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	body["stream"] = true
	path := p.imagePath
	if path == "" {
		path = ImageGenerationsPath
	}
	streamContext, cancel := context.WithCancel(ctx)
	response, err := p.doStream(streamContext, selected, path, body)
	if err != nil {
		cancel()
		return nil, err
	}
	return newOpenAIImageStream(streamContext, response.Body, cancel, p.config, p.providerInfo.ID, selected.hint, request.GenerateRequest), nil
}

func (p *openAICompatibleProvider) createRerank(ctx context.Context, selected credential, request *rerank.CreateRequest) (*rerank.CreateResponse, error) {
	if p.rerankPath == "" {
		return nil, &UnsupportedCapabilityError{Provider: p.providerInfo.ID, Capability: CapabilityRerank}
	}
	if request == nil || strings.TrimSpace(request.Model) == "" || strings.TrimSpace(request.Query) == "" || len(request.Documents) == 0 {
		return nil, fmt.Errorf("%w: rerank model, query and documents are required", ErrInvalidRequest)
	}
	documents := make([]string, len(request.Documents))
	for index, document := range request.Documents {
		documents[index] = document.Text
	}
	documentsKey := p.rerankDocumentsKey
	if documentsKey == "" {
		documentsKey = "documents"
	}
	body := map[string]any{"model": request.Model, "query": request.Query, documentsKey: documents}
	if request.TopN != nil {
		topKey := p.rerankTopKey
		if topKey == "" {
			topKey = "top_n"
		}
		body[topKey] = *request.TopN
	}
	if request.ReturnDocuments != nil {
		body["return_documents"] = *request.ReturnDocuments
	}
	if request.MaxChunksPerDoc != nil {
		body["max_chunks_per_doc"] = *request.MaxChunksPerDoc
	}
	for key, value := range request.ExtraBody {
		body[key] = value
	}
	rerankEndpoint := p.rerankPath
	if p.rerankBaseURL != "" {
		rerankEndpoint = joinURLPath(p.rerankBaseURL, p.rerankPath)
	}
	raw, err := p.doJSON(ctx, selected, rerankEndpoint, body)
	if err != nil {
		return nil, err
	}
	result := &rerank.CreateResponse{}
	if err := json.Unmarshal(raw, result); err != nil {
		return nil, err
	}
	result.RawResponse = append(result.RawResponse[:0], raw...)
	result.ExtraFields = extractExtraFields(raw, "id", "results", "usage")
	for index := range result.Results {
		item := &result.Results[index]
		if item.Document == nil && item.Index >= 0 && item.Index < len(request.Documents) && request.ReturnDocuments != nil && *request.ReturnDocuments {
			document := request.Documents[item.Index]
			item.Document = &document
		}
	}
	return result, nil
}

func (p *openAICompatibleProvider) doJSON(ctx context.Context, selected credential, path string, body any) ([]byte, error) {
	response, err := p.doStream(ctx, selected, path, body)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return io.ReadAll(response.Body)
}

func (p *openAICompatibleProvider) doStream(ctx context.Context, selected credential, path string, body any) (*http.Response, error) {
	endpoint := path
	if !strings.HasPrefix(path, "https://") && !strings.HasPrefix(path, "http://") {
		endpoint = joinURLPath(p.config.BaseURL, path)
	}
	authorization := ""
	if p.authorizationPrefix != nil && selected.apiKey != "" {
		authorization = *p.authorizationPrefix + selected.apiKey
	}
	request, err := newJSONRequest(ctx, p.config, authorization, endpoint, body)
	if err != nil {
		return nil, err
	}
	response, err := p.config.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	if err := checkHTTPResponse(p.providerInfo.ID, selected.hint, response); err != nil {
		response.Body.Close()
		return nil, err
	}
	return response, nil
}

type openAIChatStream struct {
	body      io.ReadCloser
	scanner   *bufio.Scanner
	closeOnce sync.Once
	finished  bool
	cancel    context.CancelFunc
	provider  Provider
	hint      string
}

type openAIImageStream struct {
	ctx       context.Context
	body      io.ReadCloser
	scanner   *bufio.Scanner
	closeOnce sync.Once
	finished  bool
	cancel    context.CancelFunc
	provider  Provider
	hint      string
	config    ClientConfig
	request   image.GenerateRequest
}

func newOpenAIImageStream(ctx context.Context, body io.ReadCloser, cancel context.CancelFunc, config ClientConfig, provider Provider, hint string, request image.GenerateRequest) *openAIImageStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &openAIImageStream{ctx: ctx, body: body, scanner: scanner, cancel: cancel, config: config, provider: provider, hint: hint, request: request}
}

func (s *openAIImageStream) Recv() (*image.StreamChunk, error) {
	if s.finished {
		return nil, io.EOF
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if bytes.Equal(line, []byte("[DONE]")) {
			s.finished = true
			return nil, io.EOF
		}
		if !isImagePartialFailure(line) {
			if err := decodeStreamAPIError(s.provider, s.hint, line); err != nil {
				s.finished = true
				return nil, err
			}
		}
		chunk := &image.StreamChunk{}
		if err := json.Unmarshal(line, chunk); err != nil {
			return nil, err
		}
		if chunk.URL != "" || chunk.B64JSON != "" {
			data := image.Data{URL: chunk.URL, B64JSON: chunk.B64JSON}
			if err := normalizeImageData(s.ctx, s.config, s.provider, s.hint, &s.request, &data); err != nil {
				return nil, err
			}
			chunk.URL, chunk.B64JSON, chunk.Format, chunk.MIMEType = data.URL, data.B64JSON, data.Format, data.MIMEType
		}
		chunk.RawResponse = append(chunk.RawResponse[:0], line...)
		chunk.ExtraFields = extractExtraFields(line, "type", "model", "created", "image_index", "url", "b64_json", "size", "error", "usage")
		return chunk, nil
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	s.finished = true
	return nil, io.EOF
}

func isImagePartialFailure(raw []byte) bool {
	var event struct {
		Type string `json:"type"`
	}
	return json.Unmarshal(raw, &event) == nil && event.Type == imagePartialFailed
}

func (s *openAIImageStream) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.finished = true
		if s.cancel != nil {
			s.cancel()
		}
		err = s.body.Close()
	})
	return err
}

func newOpenAIChatStream(body io.ReadCloser, cancel context.CancelFunc, provider Provider, hint string) *openAIChatStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &openAIChatStream{body: body, scanner: scanner, cancel: cancel, provider: provider, hint: hint}
}

func (s *openAIChatStream) Recv() (*chat.StreamChunk, error) {
	if s.finished {
		return nil, io.EOF
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 || bytes.HasPrefix(line, []byte(":")) {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if bytes.Equal(line, []byte("[DONE]")) {
			s.finished = true
			return nil, io.EOF
		}
		if err := decodeStreamAPIError(s.provider, s.hint, line); err != nil {
			s.finished = true
			return nil, err
		}
		chunk := &chat.StreamChunk{}
		if err := json.Unmarshal(line, chunk); err != nil {
			return nil, err
		}
		chunk.RawResponse = append(chunk.RawResponse[:0], line...)
		chunk.ExtraFields = extractExtraFields(line, "id", "object", "created", "model", "system_fingerprint", "service_tier", "choices", "usage")
		return chunk, nil
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	s.finished = true
	return nil, io.EOF
}

func (s *openAIChatStream) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.finished = true
		if s.cancel != nil {
			s.cancel()
		}
		err = s.body.Close()
	})
	return err
}

var _ chatProvider = (*openAICompatibleProvider)(nil)
var _ embeddingProvider = (*openAICompatibleProvider)(nil)
var _ imageProvider = (*openAICompatibleProvider)(nil)
var _ imageStreamProvider = (*openAICompatibleProvider)(nil)
var _ rerankProvider = (*openAICompatibleProvider)(nil)
var _ chat.Stream = (*openAIChatStream)(nil)
var _ image.Stream = (*openAIImageStream)(nil)
