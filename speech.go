// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"fmt"
	"strings"

	internalprovider "github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

type SpeechService struct{ client *Client }

func (s SpeechService) Create(ctx context.Context, request *speech.CreateRequest) (*speech.CreateResponse, error) {
	ctx = shared.NormalizeContext(ctx)
	implementation, selected, err := s.speechProvider()
	if err != nil {
		return nil, err
	}
	response, err := implementation.CreateSpeech(ctx, internalCredential(selected), request)
	return response, normalizeProviderError(err)
}

func (s SpeechService) Stream(ctx context.Context, request *speech.StreamRequest) (speech.Stream, error) {
	ctx = shared.NormalizeContext(ctx)
	implementation, selected, err := s.speechProvider()
	if err != nil {
		return nil, err
	}
	stream, err := implementation.StreamSpeech(ctx, internalCredential(selected), request)
	if err != nil {
		return nil, normalizeProviderError(err)
	}
	return &speechErrorStream{Stream: stream}, nil
}

func (s SpeechService) ListVoices(ctx context.Context, request *speech.ListVoicesRequest) (*speech.ListVoicesResponse, error) {
	ctx = shared.NormalizeContext(ctx)
	implementation, selected, err := s.voiceProvider()
	if err != nil {
		return nil, err
	}
	response, err := implementation.ListVoices(ctx, internalCredential(selected), request)
	return response, normalizeProviderError(err)
}

func (s SpeechService) UploadVoiceFile(ctx context.Context, request *speech.UploadVoiceFileRequest) (*speech.UploadVoiceFileResponse, error) {
	ctx = shared.NormalizeContext(ctx)
	implementation, selected, err := s.voiceProvider()
	if err != nil {
		return nil, err
	}
	response, err := implementation.UploadVoiceFile(ctx, internalCredential(selected), request)
	return response, normalizeProviderError(err)
}

func (s SpeechService) CloneVoice(ctx context.Context, request *speech.CloneVoiceRequest) (*speech.CloneVoiceResponse, error) {
	ctx = shared.NormalizeContext(ctx)
	implementation, selected, err := s.voiceProvider()
	if err != nil {
		return nil, err
	}
	response, err := implementation.CloneVoice(ctx, internalCredential(selected), request)
	return response, normalizeProviderError(err)
}

func (s SpeechService) CloneVoiceFromFiles(ctx context.Context, request *speech.CloneVoiceFromFilesRequest) (*speech.CloneVoiceFromFilesResponse, error) {
	ctx = shared.NormalizeContext(ctx)
	implementation, selected, err := s.voiceProvider()
	if err != nil {
		return nil, normalizeProviderError(err)
	}
	if request == nil || strings.TrimSpace(request.SourceFilePath) == "" {
		return nil, fmt.Errorf("%w: MiniMax source_file_path is required", ErrInvalidRequest)
	}
	if strings.TrimSpace(request.CloneRequest.VoiceID) == "" {
		return nil, fmt.Errorf("%w: MiniMax voice_id is required", ErrInvalidRequest)
	}
	if strings.TrimSpace(request.PromptFilePath) != "" && (request.CloneRequest.ClonePrompt == nil || strings.TrimSpace(request.CloneRequest.ClonePrompt.PromptText) == "") {
		return nil, fmt.Errorf("%w: MiniMax clone_prompt is required with prompt_file_path", ErrInvalidRequest)
	}
	credential := internalCredential(selected)
	result := &speech.CloneVoiceFromFilesResponse{}
	result.SourceUpload, err = implementation.UploadVoiceFile(ctx, credential, &speech.UploadVoiceFileRequest{Purpose: speech.VoiceFilePurposeVoiceClone, FilePath: request.SourceFilePath})
	if err != nil {
		return nil, normalizeProviderError(err)
	}
	cloneRequest := request.CloneRequest
	if cloneRequest.ClonePrompt != nil {
		clonePrompt := *cloneRequest.ClonePrompt
		cloneRequest.ClonePrompt = &clonePrompt
	}
	cloneRequest.FileID = result.SourceUpload.File.FileID
	if strings.TrimSpace(request.PromptFilePath) != "" {
		result.PromptUpload, err = implementation.UploadVoiceFile(ctx, credential, &speech.UploadVoiceFileRequest{Purpose: speech.VoiceFilePurposePromptAudio, FilePath: request.PromptFilePath})
		if err != nil {
			return nil, normalizeProviderError(err)
		}
		cloneRequest.ClonePrompt.PromptAudio = result.PromptUpload.File.FileID
	}
	result.Clone, err = implementation.CloneVoice(ctx, credential, &cloneRequest)
	if err != nil {
		return nil, normalizeProviderError(err)
	}
	return result, nil
}

type speechErrorStream struct{ speech.Stream }

func (s *speechErrorStream) Recv() (*speech.StreamChunk, error) {
	chunk, err := s.Stream.Recv()
	return chunk, normalizeProviderError(err)
}

func (s *speechErrorStream) Close() error { return normalizeProviderError(s.Stream.Close()) }

func (s SpeechService) speechProvider() (internalprovider.Speech, credential, error) {
	if !s.client.supports(CapabilitySpeech) {
		return nil, credential{}, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	implementation, ok := s.client.provider.(internalprovider.Speech)
	if !ok {
		return nil, credential{}, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	selected, err := s.client.credentials.selectCredential()
	return implementation, selected, err
}

func (s SpeechService) voiceProvider() (internalprovider.SpeechVoice, credential, error) {
	if !s.client.supports(CapabilitySpeech) {
		return nil, credential{}, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	implementation, ok := s.client.provider.(internalprovider.SpeechVoice)
	if !ok {
		return nil, credential{}, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	selected, err := s.client.credentials.selectCredential()
	return implementation, selected, err
}
