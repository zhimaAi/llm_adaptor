// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package minimax

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

const (
	speechPath                 = "/t2a_v2"
	miniMaxStreamInitialBuffer = 64 * 1024
	miniMaxStreamMaximumBuffer = 16 * 1024 * 1024
)

type Provider struct {
	*openai.Provider
}

type miniMaxSpeechWireRequest struct {
	Model                   string                              `json:"model"`
	Text                    string                              `json:"text"`
	LanguageBoost           string                              `json:"language_boost,omitempty"`
	VoiceSetting            *miniMaxVoiceSettingWire            `json:"voice_setting,omitempty"`
	AudioSetting            *miniMaxAudioSettingWire            `json:"audio_setting,omitempty"`
	PronunciationDictionary *miniMaxPronunciationDictionaryWire `json:"pronunciation_dict,omitempty"`
	TimbreWeights           []miniMaxTimbreWeightWire           `json:"timbre_weights,omitempty"`
	VoiceModification       *miniMaxVoiceModificationWire       `json:"voice_modify,omitempty"`
	SubtitleEnabled         *bool                               `json:"subtitle_enable,omitempty"`
	SubtitleType            string                              `json:"subtitle_type,omitempty"`
	OutputFormat            string                              `json:"output_format,omitempty"`
	AIGCWatermark           *bool                               `json:"aigc_watermark,omitempty"`
	Stream                  bool                                `json:"stream"`
	StreamOptions           *miniMaxStreamOptionsWire           `json:"stream_options,omitempty"`
}

type miniMaxVoiceSettingWire struct {
	VoiceID              string   `json:"voice_id"`
	Speed                *float64 `json:"speed,omitempty"`
	Volume               *float64 `json:"vol,omitempty"`
	Pitch                *int     `json:"pitch,omitempty"`
	Emotion              string   `json:"emotion,omitempty"`
	EnglishNormalization *bool    `json:"english_normalization,omitempty"`
	LatexRead            *bool    `json:"latex_read,omitempty"`
}

type miniMaxAudioSettingWire struct {
	SampleRate int    `json:"sample_rate,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	Format     string `json:"format,omitempty"`
	Channel    int    `json:"channel,omitempty"`
	ForceCBR   *bool  `json:"force_cbr,omitempty"`
}

type miniMaxPronunciationDictionaryWire struct {
	Tone []string `json:"tone,omitempty"`
}

type miniMaxTimbreWeightWire struct {
	VoiceID string `json:"voice_id"`
	Weight  int    `json:"weight"`
}

type miniMaxVoiceModificationWire struct {
	Pitch        *int   `json:"pitch,omitempty"`
	Intensity    *int   `json:"intensity,omitempty"`
	Timbre       *int   `json:"timbre,omitempty"`
	SoundEffects string `json:"sound_effects,omitempty"`
}

type miniMaxStreamOptionsWire struct {
	ExcludeAggregatedAudio *bool `json:"exclude_aggregated_audio,omitempty"`
}

func (p *Provider) CreateSpeech(ctx context.Context, credential provider.Credential, request *speech.CreateRequest) (*speech.CreateResponse, error) {
	if err := validateSpeechRequest(request); err != nil {
		return nil, err
	}
	body, err := buildMiniMaxSpeechRequest(request, false, nil)
	if err != nil {
		return nil, err
	}
	config := p.Config()
	httpRequest, err := transport.NewJSONRequest(ctx, config, transport.BearerPrefix+credential.APIKey, config.BaseURL+speechPath, body)
	if err != nil {
		return nil, err
	}
	response, err := config.HTTPClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if err := transport.CheckHTTPResponse(provider.IDMiniMax, credential.Hint, response); err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	result, err := decodeMiniMaxSpeechResponse(raw)
	if err != nil {
		return nil, transport.NewResponseDecodeError(provider.IDMiniMax, credential.Hint, raw, err)
	}
	if result.BaseResponse.StatusCode != 0 {
		return nil, miniMaxBusinessError(result.BaseResponse, result.TraceID, credential.Hint, raw)
	}
	return result, nil
}

func (p *Provider) StreamSpeech(ctx context.Context, credential provider.Credential, request *speech.StreamRequest) (speech.Stream, error) {
	if request == nil {
		return nil, fmt.Errorf("%w: speech request is nil", provider.ErrInvalidRequest)
	}
	if err := validateSpeechRequest(&request.CreateRequest); err != nil {
		return nil, err
	}
	body, err := buildMiniMaxSpeechRequest(&request.CreateRequest, true, request.StreamOptions)
	if err != nil {
		return nil, err
	}
	streamContext, cancel := context.WithCancel(shared.NormalizeContext(ctx))
	config := p.Config()
	httpRequest, err := transport.NewJSONRequest(streamContext, config, transport.BearerPrefix+credential.APIKey, config.BaseURL+speechPath, body)
	if err != nil {
		cancel()
		return nil, err
	}
	httpRequest.Header.Set("Accept", "text/event-stream")
	response, err := config.HTTPClient.Do(httpRequest)
	if err != nil {
		cancel()
		return nil, err
	}
	if err := transport.CheckHTTPResponse(provider.IDMiniMax, credential.Hint, response); err != nil {
		cancel()
		response.Body.Close()
		return nil, err
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, miniMaxStreamInitialBuffer), miniMaxStreamMaximumBuffer)
	return &miniMaxSpeechStream{
		ctx:            streamContext,
		scanner:        scanner,
		credentialHint: credential.Hint,
		terminal:       transport.NewStreamTerminal(cancel, response.Body.Close),
	}, nil
}

func validateSpeechRequest(request *speech.CreateRequest) error {
	if request == nil {
		return fmt.Errorf("%w: speech request is nil", provider.ErrInvalidRequest)
	}
	if strings.TrimSpace(request.Model) == "" {
		return fmt.Errorf("%w: model is required", provider.ErrInvalidRequest)
	}
	if strings.TrimSpace(request.Text) == "" {
		return fmt.Errorf("%w: text is required", provider.ErrInvalidRequest)
	}
	return nil
}

type miniMaxSpeechStream struct {
	ctx            context.Context
	scanner        *bufio.Scanner
	credentialHint string
	terminal       *transport.StreamTerminal
}

func (s *miniMaxSpeechStream) Recv() (*speech.StreamChunk, error) {
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
		chunk, err := decodeMiniMaxSpeechChunk(line)
		if err != nil {
			return s.fail(transport.NewResponseDecodeError(provider.IDMiniMax, s.credentialHint, line, err))
		}
		if chunk.BaseResponse.StatusCode != 0 {
			return s.fail(miniMaxBusinessError(chunk.BaseResponse, chunk.TraceID, s.credentialHint, line))
		}
		if chunk.Data != nil && chunk.Data.Status == speech.AudioStatusComplete {
			s.terminal.Finish()
		}
		return chunk, nil
	}
	if err := s.scanner.Err(); err != nil {
		return s.fail(err)
	}
	if s.terminal.IsDone() {
		return nil, io.EOF
	}
	if s.ctx != nil && s.ctx.Err() != nil {
		return s.fail(s.ctx.Err())
	}
	s.terminal.Finish()
	return nil, io.EOF
}

func (s *miniMaxSpeechStream) fail(err error) (*speech.StreamChunk, error) {
	return nil, s.terminal.Fail(s.ctx, err)
}

func (s *miniMaxSpeechStream) Close() error {
	return s.terminal.Close()
}

func buildMiniMaxSpeechRequest(request *speech.CreateRequest, stream bool, streamOptions *speech.StreamOptions) (map[string]any, error) {
	outputFormat := request.OutputFormat
	if stream {
		outputFormat = speech.OutputFormatHex
	}
	wire := miniMaxSpeechWireRequest{
		Model: request.Model, Text: request.Text, LanguageBoost: request.LanguageBoost,
		SubtitleEnabled: request.SubtitleEnabled, SubtitleType: string(request.SubtitleType),
		OutputFormat: string(outputFormat), AIGCWatermark: request.AIGCWatermark, Stream: stream,
	}
	if request.VoiceSetting != nil {
		wire.VoiceSetting = &miniMaxVoiceSettingWire{
			VoiceID: request.VoiceSetting.VoiceID, Speed: request.VoiceSetting.Speed,
			Volume: request.VoiceSetting.Volume, Pitch: request.VoiceSetting.Pitch,
			Emotion: request.VoiceSetting.Emotion, EnglishNormalization: request.VoiceSetting.EnglishNormalization,
			LatexRead: request.VoiceSetting.LatexRead,
		}
	}
	if request.AudioSetting != nil {
		wire.AudioSetting = &miniMaxAudioSettingWire{
			SampleRate: request.AudioSetting.SampleRate, Bitrate: request.AudioSetting.Bitrate,
			Format: string(request.AudioSetting.Format), Channel: request.AudioSetting.Channel,
			ForceCBR: request.AudioSetting.ForceCBR,
		}
	}
	if request.PronunciationDictionary != nil {
		wire.PronunciationDictionary = &miniMaxPronunciationDictionaryWire{
			Tone: append([]string(nil), request.PronunciationDictionary.Tone...),
		}
	}
	wire.TimbreWeights = make([]miniMaxTimbreWeightWire, len(request.TimbreWeights))
	for index, weight := range request.TimbreWeights {
		wire.TimbreWeights[index] = miniMaxTimbreWeightWire{VoiceID: weight.VoiceID, Weight: weight.Weight}
	}
	if request.VoiceModification != nil {
		wire.VoiceModification = &miniMaxVoiceModificationWire{
			Pitch: request.VoiceModification.Pitch, Intensity: request.VoiceModification.Intensity,
			Timbre: request.VoiceModification.Timbre, SoundEffects: request.VoiceModification.SoundEffects,
		}
	}
	if streamOptions != nil {
		wire.StreamOptions = &miniMaxStreamOptionsWire{ExcludeAggregatedAudio: streamOptions.ExcludeAggregatedAudio}
	}
	return shared.MergeExtraBody(wire, request.ExtraBody)
}

func miniMaxBusinessError(baseResponse speech.BaseResponse, traceID, credentialHint string, raw []byte) error {
	return &provider.APIError{
		Provider:       provider.IDMiniMax,
		StatusCode:     http.StatusOK,
		Code:           fmt.Sprint(baseResponse.StatusCode),
		Message:        baseResponse.StatusMsg,
		RequestID:      traceID,
		CredentialHint: credentialHint,
		Raw:            append([]byte(nil), raw...),
	}
}

var _ provider.Speech = (*Provider)(nil)
var _ speech.Stream = (*miniMaxSpeechStream)(nil)
