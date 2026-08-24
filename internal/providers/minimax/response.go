// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package minimax

import (
	"encoding/json"

	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

type miniMaxBaseResponseWire struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type miniMaxAudioDataWire struct {
	Audio  string `json:"audio,omitempty"`
	Status int    `json:"status,omitempty"`
}

type miniMaxAudioInfoWire struct {
	AudioLength             int64   `json:"audio_length,omitempty"`
	AudioSampleRate         int     `json:"audio_sample_rate,omitempty"`
	AudioSize               int64   `json:"audio_size,omitempty"`
	Bitrate                 int     `json:"bitrate,omitempty"`
	WordCount               int     `json:"word_count,omitempty"`
	InvisibleCharacterRatio float64 `json:"invisible_character_ratio,omitempty"`
	UsageCharacters         int     `json:"usage_characters,omitempty"`
	AudioFormat             string  `json:"audio_format,omitempty"`
	AudioChannel            int     `json:"audio_channel,omitempty"`
}

type miniMaxSpeechResponseWire struct {
	Data         *miniMaxAudioDataWire   `json:"data,omitempty"`
	ExtraInfo    *miniMaxAudioInfoWire   `json:"extra_info,omitempty"`
	TraceID      string                  `json:"trace_id,omitempty"`
	BaseResponse miniMaxBaseResponseWire `json:"base_resp"`
}

type miniMaxVoiceWire struct {
	VoiceID     string   `json:"voice_id"`
	Description []string `json:"description,omitempty"`
	VoiceName   string   `json:"voice_name,omitempty"`
	CreatedTime string   `json:"created_time,omitempty"`
}

type miniMaxListVoicesResponseWire struct {
	SystemVoices    []miniMaxVoiceWire      `json:"system_voice,omitempty"`
	ClonedVoices    []miniMaxVoiceWire      `json:"voice_cloning,omitempty"`
	GeneratedVoices []miniMaxVoiceWire      `json:"voice_generation,omitempty"`
	BaseResponse    miniMaxBaseResponseWire `json:"base_resp"`
}

type miniMaxUploadedFileWire struct {
	FileID    int64  `json:"file_id"`
	Bytes     int64  `json:"bytes,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	Filename  string `json:"filename,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
}

type miniMaxUploadVoiceFileResponseWire struct {
	File         miniMaxUploadedFileWire `json:"file"`
	BaseResponse miniMaxBaseResponseWire `json:"base_resp"`
}

type miniMaxCloneVoiceResponseWire struct {
	InputSensitive     bool                    `json:"input_sensitive,omitempty"`
	InputSensitiveType int                     `json:"input_sensitive_type,omitempty"`
	DemoAudio          string                  `json:"demo_audio,omitempty"`
	ExtraInfo          *miniMaxAudioInfoWire   `json:"extra_info,omitempty"`
	BaseResponse       miniMaxBaseResponseWire `json:"base_resp"`
}

func decodeMiniMaxSpeechResponse(raw []byte) (*speech.CreateResponse, error) {
	var source miniMaxSpeechResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &speech.CreateResponse{
		TraceID: source.TraceID, BaseResponse: mapMiniMaxBaseResponse(source.BaseResponse),
		ExtraInfo: mapMiniMaxAudioInfo(source.ExtraInfo),
	}
	if source.Data != nil {
		result.Data = &speech.AudioData{Audio: source.Data.Audio, Status: source.Data.Status}
	}
	return result, nil
}

func decodeMiniMaxSpeechChunk(raw []byte) (*speech.StreamChunk, error) {
	var source miniMaxSpeechResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &speech.StreamChunk{
		TraceID: source.TraceID, BaseResponse: mapMiniMaxBaseResponse(source.BaseResponse),
		ExtraInfo: mapMiniMaxAudioInfo(source.ExtraInfo),
	}
	if source.Data != nil {
		result.Data = &speech.AudioData{Audio: source.Data.Audio, Status: source.Data.Status}
	}
	return result, nil
}

func decodeMiniMaxListVoicesResponse(raw []byte) (*speech.ListVoicesResponse, error) {
	var source miniMaxListVoicesResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	return &speech.ListVoicesResponse{
		SystemVoices: mapMiniMaxVoices(source.SystemVoices), ClonedVoices: mapMiniMaxVoices(source.ClonedVoices),
		GeneratedVoices: mapMiniMaxVoices(source.GeneratedVoices), BaseResponse: mapMiniMaxBaseResponse(source.BaseResponse),
	}, nil
}

func decodeMiniMaxUploadVoiceFileResponse(raw []byte) (*speech.UploadVoiceFileResponse, error) {
	var source miniMaxUploadVoiceFileResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	return &speech.UploadVoiceFileResponse{
		File: speech.UploadedFile{
			FileID: source.File.FileID, Bytes: source.File.Bytes, CreatedAt: source.File.CreatedAt,
			Filename: source.File.Filename, Purpose: source.File.Purpose,
		},
		BaseResponse: mapMiniMaxBaseResponse(source.BaseResponse),
	}, nil
}

func decodeMiniMaxCloneVoiceResponse(raw []byte) (*speech.CloneVoiceResponse, error) {
	var source miniMaxCloneVoiceResponseWire
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	return &speech.CloneVoiceResponse{
		InputSensitive: source.InputSensitive, InputSensitiveType: source.InputSensitiveType,
		DemoAudio: source.DemoAudio, ExtraInfo: mapMiniMaxAudioInfo(source.ExtraInfo),
		BaseResponse: mapMiniMaxBaseResponse(source.BaseResponse),
	}, nil
}

func mapMiniMaxBaseResponse(source miniMaxBaseResponseWire) speech.BaseResponse {
	return speech.BaseResponse{StatusCode: source.StatusCode, StatusMsg: source.StatusMsg}
}

func mapMiniMaxAudioInfo(source *miniMaxAudioInfoWire) *speech.AudioInfo {
	if source == nil {
		return nil
	}
	return &speech.AudioInfo{
		AudioLength: source.AudioLength, AudioSampleRate: source.AudioSampleRate, AudioSize: source.AudioSize,
		Bitrate: source.Bitrate, WordCount: source.WordCount,
		InvisibleCharacterRatio: source.InvisibleCharacterRatio, UsageCharacters: source.UsageCharacters,
		AudioFormat: source.AudioFormat, AudioChannel: source.AudioChannel,
	}
}

func mapMiniMaxVoices(source []miniMaxVoiceWire) []speech.Voice {
	result := make([]speech.Voice, len(source))
	for index, voice := range source {
		result[index] = speech.Voice{
			VoiceID: voice.VoiceID, Description: append([]string(nil), voice.Description...),
			VoiceName: voice.VoiceName, CreatedTime: voice.CreatedTime,
		}
	}
	return result
}
