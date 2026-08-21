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
	imageEditPath       string
	rerankPath          string
	rerankBaseURL       string
	rerankDocumentsKey  string
	rerankTopKey        string
}

var embeddingReservedRequestKeys = map[string]struct{}{
	"model": {}, "input": {}, "encoding_format": {}, "dimensions": {}, "user": {},
}

var imageGenerateReservedRequestKeys = map[string]struct{}{
	"model": {}, "prompt": {}, "n": {}, "quality": {}, "response_format": {}, "size": {},
	"user": {}, "output_format": {}, "stream": {},
}

var imageEditReservedRequestKeys = map[string]struct{}{
	"model": {}, "images": {}, "mask": {}, "prompt": {}, "n": {}, "quality": {},
	"response_format": {}, "size": {}, "user": {}, "output_format": {}, "stream": {},
}

var rerankReservedRequestKeys = map[string]struct{}{
	"model": {}, "query": {}, "documents": {}, "passages": {}, "top_n": {}, "top_k": {},
}

type rerankWireResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type rerankWireAPIVersion struct {
	Version        string `json:"version,omitempty"`
	IsDeprecated   *bool  `json:"is_deprecated,omitempty"`
	IsExperimental *bool  `json:"is_experimental,omitempty"`
}

type rerankWireUnits struct {
	InputTokens     *int     `json:"input_tokens,omitempty"`
	OutputTokens    *int     `json:"output_tokens,omitempty"`
	SearchUnits     *float64 `json:"search_units,omitempty"`
	Images          *int     `json:"images,omitempty"`
	Classifications *int     `json:"classifications,omitempty"`
}

type rerankWireTokens struct {
	InputTokens  *int `json:"input_tokens,omitempty"`
	OutputTokens *int `json:"output_tokens,omitempty"`
}

type rerankWireMeta struct {
	APIVersion   *rerankWireAPIVersion `json:"api_version,omitempty"`
	BilledUnits  *rerankWireUnits      `json:"billed_units,omitempty"`
	Tokens       *rerankWireTokens     `json:"tokens,omitempty"`
	CachedTokens *int                  `json:"cached_tokens,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}

type rerankWireResponse struct {
	ID      string             `json:"id,omitempty"`
	Results []rerankWireResult `json:"results"`
	Meta    *rerankWireMeta    `json:"meta,omitempty"`
}

type openAIEmbeddingWireRequest struct {
	Model          string `json:"model"`
	Input          any    `json:"input"`
	EncodingFormat string `json:"encoding_format,omitempty"`
	Dimensions     *int   `json:"dimensions,omitempty"`
	User           string `json:"user,omitempty"`
}

type openAIImageInputWire struct {
	FileID   string `json:"file_id,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

type openAIImageGenerateWireRequest struct {
	Model          string `json:"model,omitempty"`
	Prompt         string `json:"prompt"`
	N              *int   `json:"n,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	Size           string `json:"size,omitempty"`
	User           string `json:"user,omitempty"`
	OutputFormat   string `json:"output_format,omitempty"`
}

type openAIImageEditWireRequest struct {
	Model          string                 `json:"model,omitempty"`
	Images         []openAIImageInputWire `json:"images"`
	Mask           *openAIImageInputWire  `json:"mask,omitempty"`
	Prompt         string                 `json:"prompt"`
	N              *int                   `json:"n,omitempty"`
	Quality        string                 `json:"quality,omitempty"`
	ResponseFormat string                 `json:"response_format,omitempty"`
	Size           string                 `json:"size,omitempty"`
	User           string                 `json:"user,omitempty"`
	OutputFormat   string                 `json:"output_format,omitempty"`
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
	body, err := buildOpenAIChatRequest(p.providerInfo.ID, request, false, nil)
	if err != nil {
		return nil, err
	}
	path := p.chatPath
	if path == "" {
		path = ChatCompletionsPath
	}
	raw, err := p.doJSON(ctx, selected, path, body)
	if err != nil {
		return nil, err
	}
	result, err := decodeOpenAIChatResponse(raw)
	if err != nil {
		return nil, err
	}
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
	body, err := buildOpenAIChatRequest(p.providerInfo.ID, &request.CreateRequest, true, request.StreamOptions)
	if err != nil {
		return nil, err
	}
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
	return newThinkTagStream(newOpenAIChatStream(streamContext, response.Body, cancel, p.providerInfo.ID, selected.hint)), nil
}

func (p *openAICompatibleProvider) createEmbedding(ctx context.Context, selected credential, request *embedding.CreateRequest) (*embedding.CreateResponse, error) {
	if request == nil || strings.TrimSpace(request.Model) == "" {
		return nil, fmt.Errorf("%w: embedding model is required", ErrInvalidRequest)
	}
	input, err := openAIEmbeddingInput(request.Input)
	if err != nil {
		return nil, err
	}
	wire := openAIEmbeddingWireRequest{
		Model: request.Model, Input: input, EncodingFormat: request.EncodingFormat,
		Dimensions: request.Dimensions, User: request.User,
	}
	body, err := mergeExtraBody(wire, request.ExtraBody, embeddingReservedRequestKeys)
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
	result, err := decodeOpenAIEmbeddingResponse(raw)
	if err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("%w: response contains no embeddings", ErrInvalidRequest)
	}
	return result, nil
}

func openAIEmbeddingInput(input embedding.Input) (any, error) {
	count := 0
	if input.Text != nil {
		count++
	}
	if input.Texts != nil {
		count++
	}
	if input.Tokens != nil {
		count++
	}
	if input.TokenBatches != nil {
		count++
	}
	if count != 1 {
		return nil, fmt.Errorf("%w: embedding input must contain exactly one input representation", ErrInvalidRequest)
	}
	switch {
	case input.Text != nil:
		return *input.Text, nil
	case input.Texts != nil:
		return append([]string(nil), input.Texts...), nil
	case input.Tokens != nil:
		return append([]int(nil), input.Tokens...), nil
	default:
		batches := make([][]int, len(input.TokenBatches))
		for index := range input.TokenBatches {
			batches[index] = append([]int(nil), input.TokenBatches[index]...)
		}
		return batches, nil
	}
}

func (p *openAICompatibleProvider) generateImage(ctx context.Context, selected credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image prompt is required", ErrInvalidRequest)
	}
	body, err := buildOpenAIImageGenerateRequest(request, false)
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
	result, err := decodeOpenAIImageResponse(raw)
	if err != nil {
		return nil, err
	}
	if err := normalizeImageResponse(ctx, p.config, p.providerInfo.ID, selected.hint, imageRequestOptions{
		ResponseFormat: request.ResponseFormat, OutputFormat: request.OutputFormat,
	}, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *openAICompatibleProvider) streamImage(ctx context.Context, selected credential, request *image.StreamRequest) (image.Stream, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image prompt is required", ErrInvalidRequest)
	}
	body, err := buildOpenAIImageGenerateRequest(&request.GenerateRequest, true)
	if err != nil {
		return nil, err
	}
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
	return newOpenAIImageStream(streamContext, response.Body, cancel, p.config, p.providerInfo.ID, selected.hint, imageRequestOptions{
		ResponseFormat: request.ResponseFormat, OutputFormat: request.OutputFormat,
	}), nil
}

func buildOpenAIImageGenerateRequest(request *image.GenerateRequest, stream bool) (map[string]any, error) {
	wire := openAIImageGenerateWireRequest{
		Model: request.Model, Prompt: request.Prompt, N: request.N, Quality: request.Quality,
		ResponseFormat: request.ResponseFormat, Size: request.Size, User: request.User, OutputFormat: request.OutputFormat,
	}
	body, err := mergeExtraBody(wire, request.ExtraBody, imageGenerateReservedRequestKeys)
	if err != nil {
		return nil, err
	}
	body["stream"] = stream
	return body, nil
}

func buildOpenAIImageEditRequest(request *image.EditRequest, stream bool) (map[string]any, error) {
	images, err := openAIImageInputs(request.Images)
	if err != nil {
		return nil, err
	}
	var mask *openAIImageInputWire
	if request.Mask != nil {
		converted, err := openAIImageInput(*request.Mask)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid image mask: %v", ErrInvalidRequest, err)
		}
		mask = &converted
	}
	wire := openAIImageEditWireRequest{
		Model: request.Model, Images: images, Mask: mask, Prompt: request.Prompt, N: request.N,
		Quality: request.Quality, ResponseFormat: request.ResponseFormat, Size: request.Size, User: request.User,
		OutputFormat: request.OutputFormat,
	}
	body, err := mergeExtraBody(wire, request.ExtraBody, imageEditReservedRequestKeys)
	if err != nil {
		return nil, err
	}
	body["stream"] = stream
	return body, nil
}

func openAIImageInputs(inputs []image.Input) ([]openAIImageInputWire, error) {
	result := make([]openAIImageInputWire, len(inputs))
	for index, input := range inputs {
		converted, err := openAIImageInput(input)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid image input at index %d: %v", ErrInvalidRequest, index, err)
		}
		result[index] = converted
	}
	return result, nil
}

func openAIImageInput(input image.Input) (openAIImageInputWire, error) {
	fileID := strings.TrimSpace(input.FileID)
	imageURL := strings.TrimSpace(input.ImageURL)
	if (fileID == "") == (imageURL == "") {
		return openAIImageInputWire{}, fmt.Errorf("exactly one of file_id or image_url is required")
	}
	return openAIImageInputWire{FileID: fileID, ImageURL: imageURL}, nil
}

func (p *openAICompatibleProvider) editImage(ctx context.Context, selected credential, request *image.EditRequest) (*image.GenerateResponse, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image edit prompt and images are required", ErrInvalidRequest)
	}
	body, err := buildOpenAIImageEditRequest(request, false)
	if err != nil {
		return nil, err
	}
	path := p.imageEditPath
	if path == "" {
		path = ImageEditsPath
	}
	raw, err := p.doJSON(ctx, selected, path, body)
	if err != nil {
		return nil, err
	}
	result, err := decodeOpenAIImageResponse(raw)
	if err != nil {
		return nil, err
	}
	if err := normalizeImageResponse(ctx, p.config, p.providerInfo.ID, selected.hint, imageRequestOptions{
		ResponseFormat: request.ResponseFormat, OutputFormat: request.OutputFormat,
	}, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *openAICompatibleProvider) streamImageEdit(ctx context.Context, selected credential, request *image.EditStreamRequest) (image.Stream, error) {
	if request == nil || strings.TrimSpace(request.Prompt) == "" || len(request.Images) == 0 {
		return nil, fmt.Errorf("%w: image edit prompt and images are required", ErrInvalidRequest)
	}
	body, err := buildOpenAIImageEditRequest(&request.EditRequest, true)
	if err != nil {
		return nil, err
	}
	path := p.imageEditPath
	if path == "" {
		path = ImageEditsPath
	}
	streamContext, cancel := context.WithCancel(ctx)
	response, err := p.doStream(streamContext, selected, path, body)
	if err != nil {
		cancel()
		return nil, err
	}
	return newOpenAIImageStream(streamContext, response.Body, cancel, p.config, p.providerInfo.ID, selected.hint, imageRequestOptions{
		ResponseFormat: request.ResponseFormat, OutputFormat: request.OutputFormat,
	}), nil
}

func (p *openAICompatibleProvider) createRerank(ctx context.Context, selected credential, request *rerank.CreateRequest) (*rerank.CreateResponse, error) {
	if p.rerankPath == "" {
		return nil, &UnsupportedCapabilityError{Provider: p.providerInfo.ID, Capability: CapabilityRerank}
	}
	if request == nil || strings.TrimSpace(request.Model) == "" || strings.TrimSpace(request.Query) == "" || len(request.Documents) == 0 {
		return nil, fmt.Errorf("%w: rerank model, query and documents are required", ErrInvalidRequest)
	}
	documentsKey := p.rerankDocumentsKey
	if documentsKey == "" {
		documentsKey = "documents"
	}
	body := map[string]any{"model": request.Model, "query": request.Query, documentsKey: append([]string(nil), request.Documents...)}
	if request.TopN != nil {
		topKey := p.rerankTopKey
		if topKey == "" {
			topKey = "top_n"
		}
		body[topKey] = *request.TopN
	}
	for key, value := range request.ExtraBody {
		if _, reserved := rerankReservedRequestKeys[key]; reserved {
			return nil, fmt.Errorf("%w: extra_body field %q conflicts with a reserved rerank field", ErrInvalidRequest, key)
		}
		if _, exists := body[key]; exists {
			return nil, fmt.Errorf("%w: extra_body field %q conflicts with a rerank field", ErrInvalidRequest, key)
		}
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
	var source rerankWireResponse
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &rerank.CreateResponse{ID: source.ID, Results: make([]rerank.Result, len(source.Results))}
	for index, item := range source.Results {
		result.Results[index] = rerank.Result{Index: item.Index, RelevanceScore: item.RelevanceScore}
	}
	result.Meta = mapRerankMeta(source.Meta)
	normalizeRerankMeta(raw, result)
	return result, nil
}

func mapRerankMeta(source *rerankWireMeta) *rerank.Meta {
	if source == nil {
		return nil
	}
	result := &rerank.Meta{CachedTokens: source.CachedTokens, Warnings: append([]string(nil), source.Warnings...)}
	if source.APIVersion != nil {
		result.APIVersion = &rerank.APIVersion{
			Version: source.APIVersion.Version, IsDeprecated: source.APIVersion.IsDeprecated,
			IsExperimental: source.APIVersion.IsExperimental,
		}
	}
	if source.BilledUnits != nil {
		result.BilledUnits = &rerank.Units{
			InputTokens: source.BilledUnits.InputTokens, OutputTokens: source.BilledUnits.OutputTokens,
			SearchUnits: source.BilledUnits.SearchUnits, Images: source.BilledUnits.Images,
			Classifications: source.BilledUnits.Classifications,
		}
	}
	if source.Tokens != nil {
		result.Tokens = &rerank.Tokens{InputTokens: source.Tokens.InputTokens, OutputTokens: source.Tokens.OutputTokens}
	}
	return result
}

func normalizeRerankMeta(raw []byte, result *rerank.CreateResponse) {
	if result.Meta != nil {
		return
	}
	var source struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
		Meta struct {
			Tokens int `json:"tokens"`
		} `json:"meta"`
	}
	if json.Unmarshal(raw, &source) != nil {
		return
	}
	inputTokens := source.Usage.InputTokens
	outputTokens := source.Usage.OutputTokens
	if inputTokens == 0 && outputTokens == 0 && source.Usage.TotalTokens > 0 {
		inputTokens = source.Usage.TotalTokens
	}
	if inputTokens == 0 && source.Meta.Tokens > 0 {
		inputTokens = source.Meta.Tokens
	}
	if inputTokens == 0 && outputTokens == 0 {
		return
	}
	result.Meta = &rerank.Meta{Tokens: &rerank.Tokens{}}
	if inputTokens > 0 {
		result.Meta.Tokens.InputTokens = &inputTokens
	}
	if outputTokens > 0 {
		result.Meta.Tokens.OutputTokens = &outputTokens
	}
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
	ctx      context.Context
	scanner  *bufio.Scanner
	terminal *streamTerminal
	provider Provider
	hint     string
}

type openAIImageStream struct {
	ctx      context.Context
	scanner  *bufio.Scanner
	terminal *streamTerminal
	provider Provider
	hint     string
	config   ClientConfig
	request  imageRequestOptions
}

func newOpenAIImageStream(ctx context.Context, body io.ReadCloser, cancel context.CancelFunc, config ClientConfig, provider Provider, hint string, request imageRequestOptions) *openAIImageStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &openAIImageStream{ctx: ctx, scanner: scanner, terminal: newStreamTerminal(cancel, body.Close), config: config, provider: provider, hint: hint, request: request}
}

func (s *openAIImageStream) Recv() (*image.StreamChunk, error) {
	if s.terminal.isDone() {
		return nil, io.EOF
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if bytes.Equal(line, []byte("[DONE]")) {
			s.terminal.finish()
			return nil, io.EOF
		}
		if err := decodeStreamAPIError(s.provider, s.hint, line); err != nil {
			return nil, s.terminal.fail(s.ctx, err)
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
			return nil, s.terminal.fail(s.ctx, err)
		}
		chunk := &image.StreamChunk{
			Type: wire.Type, B64JSON: wire.B64JSON, PartialImageIndex: wire.PartialImageIndex,
			Created: wire.Created, Background: wire.Background, OutputFormat: wire.OutputFormat,
			Quality: wire.Quality, Size: wire.Size, Usage: wire.Usage,
		}
		if chunk.PartialImageIndex == 0 && wire.ImageIndex != 0 {
			chunk.PartialImageIndex = wire.ImageIndex
		}
		if wire.URL != "" || wire.B64JSON != "" {
			data := image.Data{URL: wire.URL, B64JSON: wire.B64JSON}
			format, err := normalizeImageData(s.ctx, s.config, s.provider, s.hint, s.request, &data)
			if err != nil {
				return nil, s.terminal.fail(s.ctx, err)
			}
			chunk.B64JSON = data.B64JSON
			if chunk.OutputFormat == "" {
				chunk.OutputFormat = format
			}
		}
		return chunk, nil
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

func (s *openAIImageStream) Close() error {
	return s.terminal.close()
}

func newOpenAIChatStream(ctx context.Context, body io.ReadCloser, cancel context.CancelFunc, provider Provider, hint string) *openAIChatStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &openAIChatStream{ctx: ctx, scanner: scanner, terminal: newStreamTerminal(cancel, body.Close), provider: provider, hint: hint}
}

func (s *openAIChatStream) Recv() (*chat.StreamChunk, error) {
	if s.terminal.isDone() {
		return nil, io.EOF
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
		if err := decodeStreamAPIError(s.provider, s.hint, line); err != nil {
			return nil, s.terminal.fail(s.ctx, err)
		}
		chunk, err := decodeOpenAIChatStreamResponse(line)
		if err != nil {
			return nil, s.terminal.fail(s.ctx, err)
		}
		return chunk, nil
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

func (s *openAIChatStream) Close() error {
	return s.terminal.close()
}

var _ chatProvider = (*openAICompatibleProvider)(nil)
var _ embeddingProvider = (*openAICompatibleProvider)(nil)
var _ imageProvider = (*openAICompatibleProvider)(nil)
var _ imageStreamProvider = (*openAICompatibleProvider)(nil)
var _ imageEditProvider = (*openAICompatibleProvider)(nil)
var _ imageEditStreamProvider = (*openAICompatibleProvider)(nil)
var _ rerankProvider = (*openAICompatibleProvider)(nil)
var _ chat.Stream = (*openAIChatStream)(nil)
var _ image.Stream = (*openAIImageStream)(nil)
