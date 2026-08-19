// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package speech

import (
	"encoding/json"
)

type AudioFormat string

const (
	AudioFormatMP3  AudioFormat = "mp3"
	AudioFormatPCM  AudioFormat = "pcm"
	AudioFormatFLAC AudioFormat = "flac"
	AudioFormatWAV  AudioFormat = "wav"
)

type OutputFormat string

const (
	OutputFormatHex OutputFormat = "hex"
	OutputFormatURL OutputFormat = "url"
)

type SubtitleType string

const (
	SubtitleTypeSentence      SubtitleType = "sentence"
	SubtitleTypeWord          SubtitleType = "word"
	SubtitleTypeWordStreaming SubtitleType = "word_streaming"
)

type VoiceSetting struct {
	VoiceID              string   `json:"voice_id"`
	Speed                *float64 `json:"speed,omitempty"`
	Volume               *float64 `json:"vol,omitempty"`
	Pitch                *int     `json:"pitch,omitempty"`
	Emotion              string   `json:"emotion,omitempty"`
	EnglishNormalization *bool    `json:"english_normalization,omitempty"`
	LatexRead            *bool    `json:"latex_read,omitempty"`
}

type AudioSetting struct {
	SampleRate int         `json:"sample_rate,omitempty"`
	Bitrate    int         `json:"bitrate,omitempty"`
	Format     AudioFormat `json:"format,omitempty"`
	Channel    int         `json:"channel,omitempty"`
	ForceCBR   *bool       `json:"force_cbr,omitempty"`
}

type PronunciationDictionary struct {
	Tone []string `json:"tone,omitempty"`
}

type TimbreWeight struct {
	VoiceID string `json:"voice_id"`
	Weight  int    `json:"weight"`
}

type VoiceModification struct {
	Pitch        *int   `json:"pitch,omitempty"`
	Intensity    *int   `json:"intensity,omitempty"`
	Timbre       *int   `json:"timbre,omitempty"`
	SoundEffects string `json:"sound_effects,omitempty"`
}

type StreamOptions struct {
	ExcludeAggregatorAudio *bool `json:"exclude_aggregator_audio,omitempty"`
}

type CreateRequest struct {
	Model                   string                   `json:"model"`
	Text                    string                   `json:"text"`
	LanguageBoost           string                   `json:"language_boost,omitempty"`
	VoiceSetting            *VoiceSetting            `json:"voice_setting,omitempty"`
	AudioSetting            *AudioSetting            `json:"audio_setting,omitempty"`
	PronunciationDictionary *PronunciationDictionary `json:"pronunciation_dict,omitempty"`
	TimbreWeights           []TimbreWeight           `json:"timbre_weights,omitempty"`
	VoiceModification       *VoiceModification       `json:"voice_modify,omitempty"`
	SubtitleEnabled         *bool                    `json:"subtitle_enable,omitempty"`
	SubtitleType            SubtitleType             `json:"subtitle_type,omitempty"`
	OutputFormat            OutputFormat             `json:"output_format,omitempty"`
	ExtraBody               map[string]any           `json:"-"`
}

type StreamRequest struct {
	CreateRequest
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
}

type AudioData struct {
	Audio  string `json:"audio,omitempty"`
	Status int    `json:"status,omitempty"`
}

type AudioInfo struct {
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

type BaseResponse struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type ResponseMeta struct {
	Provider          string   `json:"-"`
	CredentialHint    string   `json:"-"`
	IgnoredParameters []string `json:"-"`
}

type CreateResponse struct {
	Data         *AudioData                 `json:"data,omitempty"`
	ExtraInfo    *AudioInfo                 `json:"extra_info,omitempty"`
	TraceID      string                     `json:"trace_id,omitempty"`
	BaseResponse BaseResponse               `json:"base_resp"`
	ExtraFields  map[string]json.RawMessage `json:"-"`
	RawResponse  json.RawMessage            `json:"-"`
	Meta         ResponseMeta               `json:"-"`
}

type StreamChunk struct {
	Data         *AudioData                 `json:"data,omitempty"`
	ExtraInfo    *AudioInfo                 `json:"extra_info,omitempty"`
	TraceID      string                     `json:"trace_id,omitempty"`
	BaseResponse BaseResponse               `json:"base_resp"`
	ExtraFields  map[string]json.RawMessage `json:"-"`
	RawResponse  json.RawMessage            `json:"-"`
	Meta         ResponseMeta               `json:"-"`
}

type Stream interface {
	Recv() (*StreamChunk, error)
	Close() error
}
