// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

const (
	miniMaxListVoicesPath       = "/get_voice"
	miniMaxUploadVoiceFilePath  = "/files/upload"
	miniMaxCloneVoicePath       = "/voice_clone"
	miniMaxUploadFileField      = "file"
	miniMaxUploadPurposeField   = "purpose"
	miniMaxVoiceClonePurpose    = "voice_clone"
	miniMaxPromptAudioPurpose   = "prompt_audio"
	miniMaxMaximumVoiceFileSize = int64(20 * 1024 * 1024)
)

func (p *miniMaxProvider) listVoices(ctx context.Context, selected credential, request *speech.ListVoicesRequest) (*speech.ListVoicesResponse, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	if request == nil {
		return nil, fmt.Errorf("%w: list voices request is nil", ErrInvalidRequest)
	}
	if request.VoiceType == "" {
		request = &speech.ListVoicesRequest{VoiceType: speech.VoiceTypeAll}
	}
	httpRequest, err := newJSONRequest(ctx, p.config, bearerPrefix+selected.apiKey, p.config.BaseURL+miniMaxListVoicesPath, request)
	if err != nil {
		return nil, err
	}
	raw, err := p.doMiniMaxVoiceRequest(httpRequest, selected.hint)
	if err != nil {
		return nil, err
	}
	response := &speech.ListVoicesResponse{}
	if err := json.Unmarshal(raw, response); err != nil {
		return nil, err
	}
	setMiniMaxListVoicesMeta(response, raw, selected.hint)
	if response.BaseResponse.StatusCode != 0 {
		return nil, miniMaxBusinessError(response.BaseResponse, "", selected.hint, raw)
	}
	return response, nil
}

func (p *miniMaxProvider) uploadVoiceFile(ctx context.Context, selected credential, request *speech.UploadVoiceFileRequest) (*speech.UploadVoiceFileResponse, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	if request == nil {
		return nil, fmt.Errorf("%w: upload voice file request is nil", ErrInvalidRequest)
	}
	purpose := strings.TrimSpace(request.Purpose)
	if purpose != miniMaxVoiceClonePurpose && purpose != miniMaxPromptAudioPurpose {
		return nil, fmt.Errorf("%w: unsupported MiniMax voice file purpose %q", ErrInvalidRequest, purpose)
	}
	file, err := os.Open(request.FilePath)
	if err != nil {
		return nil, fmt.Errorf("open MiniMax voice file: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat MiniMax voice file: %w", err)
	}
	if info.Size() <= 0 || info.Size() > miniMaxMaximumVoiceFileSize {
		return nil, fmt.Errorf("%w: MiniMax voice file size must be between 1 and %d bytes", ErrInvalidRequest, miniMaxMaximumVoiceFileSize)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField(miniMaxUploadPurposeField, purpose); err != nil {
		return nil, err
	}
	part, err := writer.CreateFormFile(miniMaxUploadFileField, filepath.Base(request.FilePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.BaseURL+miniMaxUploadVoiceFilePath, &body)
	if err != nil {
		return nil, err
	}
	for key, values := range p.config.DefaultHeaders {
		for _, value := range values {
			httpRequest.Header.Add(key, value)
		}
	}
	httpRequest.Header.Set(headerAuthorization, bearerPrefix+selected.apiKey)
	httpRequest.Header.Set(headerContentType, writer.FormDataContentType())
	raw, err := p.doMiniMaxVoiceRequest(httpRequest, selected.hint)
	if err != nil {
		return nil, err
	}
	response := &speech.UploadVoiceFileResponse{}
	if err := json.Unmarshal(raw, response); err != nil {
		return nil, err
	}
	response.RawResponse = append(response.RawResponse[:0], raw...)
	response.ExtraFields = extractExtraFields(raw, "file", "base_resp")
	response.Meta.Provider = string(ProviderMiniMax)
	response.Meta.CredentialHint = selected.hint
	if response.BaseResponse.StatusCode != 0 {
		return nil, miniMaxBusinessError(response.BaseResponse, "", selected.hint, raw)
	}
	return response, nil
}

func (p *miniMaxProvider) cloneVoice(ctx context.Context, selected credential, request *speech.CloneVoiceRequest) (*speech.CloneVoiceResponse, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	if request == nil || request.FileID <= 0 || strings.TrimSpace(request.VoiceID) == "" {
		return nil, fmt.Errorf("%w: MiniMax file_id and voice_id are required", ErrInvalidRequest)
	}
	body, err := mergeExtraBody(request, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	httpRequest, err := newJSONRequest(ctx, p.config, bearerPrefix+selected.apiKey, p.config.BaseURL+miniMaxCloneVoicePath, body)
	if err != nil {
		return nil, err
	}
	raw, err := p.doMiniMaxVoiceRequest(httpRequest, selected.hint)
	if err != nil {
		return nil, err
	}
	response := &speech.CloneVoiceResponse{}
	if err := json.Unmarshal(raw, response); err != nil {
		return nil, err
	}
	response.RawResponse = append(response.RawResponse[:0], raw...)
	response.ExtraFields = extractExtraFields(raw, "input_sensitive", "input_sensitive_type", "demo_audio", "base_resp")
	response.Meta.Provider = string(ProviderMiniMax)
	response.Meta.CredentialHint = selected.hint
	if response.BaseResponse.StatusCode != 0 {
		return nil, miniMaxBusinessError(response.BaseResponse, "", selected.hint, raw)
	}
	return response, nil
}

func (p *miniMaxProvider) doMiniMaxVoiceRequest(request *http.Request, credentialHint string) ([]byte, error) {
	response, err := p.config.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if err := checkHTTPResponse(ProviderMiniMax, credentialHint, response); err != nil {
		return nil, err
	}
	return io.ReadAll(response.Body)
}

func setMiniMaxListVoicesMeta(response *speech.ListVoicesResponse, raw []byte, credentialHint string) {
	response.RawResponse = append(response.RawResponse[:0], raw...)
	response.ExtraFields = extractExtraFields(raw, "system_voice", "voice_cloning", "voice_generation", "base_resp")
	response.Meta.Provider = string(ProviderMiniMax)
	response.Meta.CredentialHint = credentialHint
}

var _ speechVoiceProvider = (*miniMaxProvider)(nil)
