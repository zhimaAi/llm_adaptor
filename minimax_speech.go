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

	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

const (
	miniMaxSpeechStatusStreaming = 1
	miniMaxSpeechStatusComplete  = 2
	miniMaxStreamInitialBuffer   = 64 * 1024
	miniMaxStreamMaximumBuffer   = 16 * 1024 * 1024
)

type miniMaxProvider struct {
	*openAICompatibleProvider
}

func newMiniMaxProvider(config ClientConfig) *miniMaxProvider {
	return &miniMaxProvider{openAICompatibleProvider: newOpenAICompatibleProvider(
		config,
		ProviderInfo{ID: ProviderMiniMax, DefaultBaseURL: DefaultBaseURLMiniMax, Capabilities: []Capability{CapabilityChat, CapabilitySpeech}},
	)}
}

func (p *miniMaxProvider) info() ProviderInfo {
	return p.providerInfo
}

func (p *miniMaxProvider) createSpeech(ctx context.Context, credential credential, request *speech.CreateRequest) (*speech.CreateResponse, error) {
	if err := validateSpeechRequest(ctx, request); err != nil {
		return nil, err
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	body["stream"] = false
	httpRequest, err := newJSONRequest(ctx, p.config, bearerPrefix+credential.apiKey, p.config.BaseURL+MiniMaxSpeechPath, body)
	if err != nil {
		return nil, err
	}
	response, err := p.config.HTTPClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if err := checkHTTPResponse(ProviderMiniMax, credential.hint, response); err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	result := &speech.CreateResponse{}
	if err := decodeSpeechResponse(raw, result); err != nil {
		return nil, err
	}
	result.Meta.Provider = string(ProviderMiniMax)
	result.Meta.CredentialHint = credential.hint
	if result.BaseResponse.StatusCode != 0 {
		return nil, miniMaxBusinessError(result.BaseResponse, result.TraceID, credential.hint, raw)
	}
	return result, nil
}

func (p *miniMaxProvider) streamSpeech(ctx context.Context, credential credential, request *speech.StreamRequest) (speech.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: speech request is nil", ErrInvalidRequest)
	}
	if err := validateSpeechRequest(ctx, &request.CreateRequest); err != nil {
		return nil, err
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	body["stream"] = true
	body["output_format"] = string(speech.OutputFormatHex)
	streamContext, cancel := context.WithCancel(ctx)
	httpRequest, err := newJSONRequest(streamContext, p.config, bearerPrefix+credential.apiKey, p.config.BaseURL+MiniMaxSpeechPath, body)
	if err != nil {
		cancel()
		return nil, err
	}
	httpRequest.Header.Set("Accept", "text/event-stream")
	response, err := p.config.HTTPClient.Do(httpRequest)
	if err != nil {
		cancel()
		return nil, err
	}
	if err := checkHTTPResponse(ProviderMiniMax, credential.hint, response); err != nil {
		cancel()
		response.Body.Close()
		return nil, err
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, miniMaxStreamInitialBuffer), miniMaxStreamMaximumBuffer)
	return &miniMaxSpeechStream{
		body:           response.Body,
		scanner:        scanner,
		credentialHint: credential.hint,
		cancel:         cancel,
	}, nil
}

func validateSpeechRequest(ctx context.Context, request *speech.CreateRequest) error {
	if ctx == nil {
		return ErrNilContext
	}
	if request == nil {
		return fmt.Errorf("%w: speech request is nil", ErrInvalidRequest)
	}
	if strings.TrimSpace(request.Model) == "" {
		return fmt.Errorf("%w: model is required", ErrInvalidRequest)
	}
	if strings.TrimSpace(request.Text) == "" {
		return fmt.Errorf("%w: text is required", ErrInvalidRequest)
	}
	return nil
}

type miniMaxSpeechStream struct {
	body           io.ReadCloser
	scanner        *bufio.Scanner
	credentialHint string
	finished       bool
	closeOnce      sync.Once
	cancel         context.CancelFunc
}

func (s *miniMaxSpeechStream) Recv() (*speech.StreamChunk, error) {
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
		chunk := &speech.StreamChunk{}
		if err := decodeSpeechChunk(line, chunk); err != nil {
			return nil, err
		}
		chunk.Meta.Provider = string(ProviderMiniMax)
		chunk.Meta.CredentialHint = s.credentialHint
		if chunk.BaseResponse.StatusCode != 0 {
			return nil, miniMaxBusinessError(chunk.BaseResponse, chunk.TraceID, s.credentialHint, line)
		}
		if chunk.Data != nil && chunk.Data.Status == miniMaxSpeechStatusComplete {
			s.finished = true
		}
		return chunk, nil
	}
	if err := s.scanner.Err(); err != nil {
		return nil, err
	}
	s.finished = true
	return nil, io.EOF
}

func (s *miniMaxSpeechStream) Close() error {
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

func decodeSpeechResponse(raw []byte, response *speech.CreateResponse) error {
	if err := json.Unmarshal(raw, response); err != nil {
		return err
	}
	response.RawResponse = append(response.RawResponse[:0], raw...)
	response.ExtraFields = extractExtraFields(raw, "data", "extra_info", "trace_id", "base_resp")
	return nil
}

func decodeSpeechChunk(raw []byte, response *speech.StreamChunk) error {
	if err := json.Unmarshal(raw, response); err != nil {
		return err
	}
	response.RawResponse = append(response.RawResponse[:0], raw...)
	response.ExtraFields = extractExtraFields(raw, "data", "extra_info", "trace_id", "base_resp")
	return nil
}

func extractExtraFields(raw []byte, knownFields ...string) map[string]json.RawMessage {
	fields := make(map[string]json.RawMessage)
	if json.Unmarshal(raw, &fields) != nil {
		return nil
	}
	for _, field := range knownFields {
		delete(fields, field)
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

func miniMaxBusinessError(baseResponse speech.BaseResponse, traceID, credentialHint string, raw []byte) error {
	return &APIError{
		Provider:       ProviderMiniMax,
		StatusCode:     http.StatusOK,
		Code:           fmt.Sprint(baseResponse.StatusCode),
		Message:        baseResponse.StatusMsg,
		RequestID:      traceID,
		CredentialHint: credentialHint,
		Raw:            append([]byte(nil), raw...),
	}
}

var _ speechProvider = (*miniMaxProvider)(nil)
var _ speech.Stream = (*miniMaxSpeechStream)(nil)
