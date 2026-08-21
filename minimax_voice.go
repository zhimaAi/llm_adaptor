// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"bytes"
	"context"
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

var miniMaxListVoicesReservedRequestKeys = map[string]struct{}{"voice_type": {}}

var miniMaxCloneVoiceReservedRequestKeys = map[string]struct{}{
	"file_id": {}, "voice_id": {}, "clone_prompt": {}, "text": {}, "model": {},
	"language_boost": {}, "text_validation": {}, "accuracy": {}, "need_noise_reduction": {},
	"need_volume_normalization": {}, "aigc_watermark": {},
}

type miniMaxListVoicesWireRequest struct {
	VoiceType string `json:"voice_type"`
}

type miniMaxCloneVoiceWireRequest struct {
	FileID                  int64                   `json:"file_id"`
	VoiceID                 string                  `json:"voice_id"`
	ClonePrompt             *miniMaxClonePromptWire `json:"clone_prompt,omitempty"`
	Text                    string                  `json:"text,omitempty"`
	Model                   string                  `json:"model,omitempty"`
	LanguageBoost           string                  `json:"language_boost,omitempty"`
	TextValidation          string                  `json:"text_validation,omitempty"`
	Accuracy                *float64                `json:"accuracy,omitempty"`
	NeedNoiseReduction      *bool                   `json:"need_noise_reduction,omitempty"`
	NeedVolumeNormalization *bool                   `json:"need_volume_normalization,omitempty"`
	AIGCWatermark           *bool                   `json:"aigc_watermark,omitempty"`
}

type miniMaxClonePromptWire struct {
	PromptAudio int64  `json:"prompt_audio"`
	PromptText  string `json:"prompt_text"`
}

func (p *miniMaxProvider) listVoices(ctx context.Context, selected credential, request *speech.ListVoicesRequest) (*speech.ListVoicesResponse, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	if request == nil {
		return nil, fmt.Errorf("%w: list voices request is nil", ErrInvalidRequest)
	}
	voiceType := request.VoiceType
	if voiceType == "" {
		voiceType = speech.VoiceTypeAll
	}
	body, err := mergeExtraBody(miniMaxListVoicesWireRequest{VoiceType: string(voiceType)}, request.ExtraBody, miniMaxListVoicesReservedRequestKeys)
	if err != nil {
		return nil, err
	}
	httpRequest, err := newJSONRequest(ctx, p.config, bearerPrefix+selected.apiKey, p.config.BaseURL+miniMaxListVoicesPath, body)
	if err != nil {
		return nil, err
	}
	raw, err := p.doMiniMaxVoiceRequest(httpRequest, selected.hint)
	if err != nil {
		return nil, err
	}
	response, err := decodeMiniMaxListVoicesResponse(raw)
	if err != nil {
		return nil, err
	}
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
	response, err := decodeMiniMaxUploadVoiceFileResponse(raw)
	if err != nil {
		return nil, err
	}
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
	wire := miniMaxCloneVoiceWireRequest{
		FileID: request.FileID, VoiceID: request.VoiceID,
		Text: request.Text, Model: request.Model, LanguageBoost: request.LanguageBoost,
		TextValidation: request.TextValidation, Accuracy: request.Accuracy,
		NeedNoiseReduction:      request.NeedNoiseReduction,
		NeedVolumeNormalization: request.NeedVolumeNormalization,
		AIGCWatermark:           request.AIGCWatermark,
	}
	if request.ClonePrompt != nil {
		wire.ClonePrompt = &miniMaxClonePromptWire{
			PromptAudio: request.ClonePrompt.PromptAudio, PromptText: request.ClonePrompt.PromptText,
		}
	}
	body, err := mergeExtraBody(wire, request.ExtraBody, miniMaxCloneVoiceReservedRequestKeys)
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
	response, err := decodeMiniMaxCloneVoiceResponse(raw)
	if err != nil {
		return nil, err
	}
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

var _ speechVoiceProvider = (*miniMaxProvider)(nil)
