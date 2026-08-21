// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package chat

import (
	"bytes"
	"encoding/json"
)

type Role string

const (
	RoleDeveloper Role = "developer"
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type ContentPartType string

const (
	ContentPartText       ContentPartType = "text"
	ContentPartImageURL   ContentPartType = "image_url"
	ContentPartInputAudio ContentPartType = "input_audio"
	ContentPartVideoURL   ContentPartType = "video_url"
)

type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type InputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"`
}

type VideoURL struct {
	URL string `json:"url"`
}

type ContentPart struct {
	Type       ContentPartType `json:"type"`
	Text       string          `json:"text,omitempty"`
	ImageURL   *ImageURL       `json:"image_url,omitempty"`
	InputAudio *InputAudio     `json:"input_audio,omitempty"`
	VideoURL   *VideoURL       `json:"video_url,omitempty"`
}

type MessageContent struct {
	Text  *string
	Parts []ContentPart
}

func TextContent(value string) MessageContent {
	return MessageContent{Text: &value}
}

func PartsContent(value ...ContentPart) MessageContent {
	return MessageContent{Parts: value}
}

func (c MessageContent) MarshalJSON() ([]byte, error) {
	if c.Text != nil {
		return json.Marshal(*c.Text)
	}
	if c.Parts == nil {
		return []byte("null"), nil
	}
	return json.Marshal(c.Parts)
}

func (c *MessageContent) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*c = MessageContent{}
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*c = TextContent(text)
		return nil
	}
	var parts []ContentPart
	if err := json.Unmarshal(data, &parts); err != nil {
		return err
	}
	*c = PartsContent(parts...)
	return nil
}

type FunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type ToolCall struct {
	Index    *int         `json:"index,omitempty"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function FunctionCall `json:"function"`
}

type Message struct {
	Role             Role           `json:"role"`
	Content          MessageContent `json:"content"`
	Name             string         `json:"name,omitempty"`
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	FunctionCall     *FunctionCall  `json:"function_call,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	Refusal          string         `json:"refusal,omitempty"`
	Audio            *Audio         `json:"audio,omitempty"`
	Annotations      []Annotation   `json:"annotations,omitempty"`
}

type Audio struct {
	ID         string `json:"id"`
	Data       string `json:"data,omitempty"`
	ExpiresAt  int64  `json:"expires_at,omitempty"`
	Transcript string `json:"transcript,omitempty"`
}

type Annotation struct {
	Type        string       `json:"type"`
	URLCitation *URLCitation `json:"url_citation,omitempty"`
}

type URLCitation struct {
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	URL        string `json:"url"`
	Title      string `json:"title"`
}

type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      *bool           `json:"strict,omitempty"`
}

type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type ResponseFormat struct {
	Type       string          `json:"type"`
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

type StreamOptions struct {
	IncludeUsage *bool `json:"include_usage,omitempty"`
}

// ReasoningEffort constrains how much reasoning-capable models should perform.
type ReasoningEffort string

const (
	ReasoningEffortNone    ReasoningEffort = "none"
	ReasoningEffortMinimal ReasoningEffort = "minimal"
	ReasoningEffortLow     ReasoningEffort = "low"
	ReasoningEffortMedium  ReasoningEffort = "medium"
	ReasoningEffortHigh    ReasoningEffort = "high"
	ReasoningEffortXHigh   ReasoningEffort = "xhigh"
	ReasoningEffortMax     ReasoningEffort = "max"
)

type CreateRequest struct {
	Model               string          `json:"model"`
	Messages            []Message       `json:"messages"`
	FrequencyPenalty    *float64        `json:"frequency_penalty,omitempty"`
	MaxTokens           *int            `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int            `json:"max_completion_tokens,omitempty"`
	N                   *int            `json:"n,omitempty"`
	ParallelToolCalls   *bool           `json:"parallel_tool_calls,omitempty"`
	PresencePenalty     *float64        `json:"presence_penalty,omitempty"`
	ReasoningEffort     ReasoningEffort `json:"reasoning_effort,omitempty"`
	ResponseFormat      *ResponseFormat `json:"response_format,omitempty"`
	Seed                *int64          `json:"seed,omitempty"`
	Stop                []string        `json:"stop,omitempty"`
	Temperature         *float64        `json:"temperature,omitempty"`
	ToolChoice          any             `json:"tool_choice,omitempty"`
	Tools               []Tool          `json:"tools,omitempty"`
	TopP                *float64        `json:"top_p,omitempty"`
	User                string          `json:"user,omitempty"`
	ExtraBody           map[string]any  `json:"-"`
}

type StreamRequest struct {
	CreateRequest
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
}

type PromptTokensDetails struct {
	AudioTokens  int `json:"audio_tokens,omitempty"`
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type CompletionTokensDetails struct {
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`
	AudioTokens              int `json:"audio_tokens,omitempty"`
	ReasoningTokens          int `json:"reasoning_tokens,omitempty"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

type Usage struct {
	PromptTokens            int                     `json:"prompt_tokens"`
	CompletionTokens        int                     `json:"completion_tokens"`
	TotalTokens             int                     `json:"total_tokens"`
	PromptTokensDetails     PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type TopLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []int   `json:"bytes,omitempty"`
}

type TokenLogProb struct {
	Token       string       `json:"token"`
	LogProb     float64      `json:"logprob"`
	Bytes       []int        `json:"bytes,omitempty"`
	TopLogProbs []TopLogProb `json:"top_logprobs,omitempty"`
}

type LogProbs struct {
	Content []TokenLogProb `json:"content,omitempty"`
	Refusal []TokenLogProb `json:"refusal,omitempty"`
}

type Choice struct {
	Index        int       `json:"index"`
	Message      Message   `json:"message"`
	FinishReason string    `json:"finish_reason,omitempty"`
	LogProbs     *LogProbs `json:"logprobs,omitempty"`
}

type CreateResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
	ServiceTier       string   `json:"service_tier,omitempty"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
}

type ChunkChoice struct {
	Index        int       `json:"index"`
	Delta        Message   `json:"delta"`
	FinishReason string    `json:"finish_reason,omitempty"`
	LogProbs     *LogProbs `json:"logprobs,omitempty"`
}

type StreamChunk struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	SystemFingerprint string        `json:"system_fingerprint,omitempty"`
	ServiceTier       string        `json:"service_tier,omitempty"`
	Choices           []ChunkChoice `json:"choices"`
	Usage             *Usage        `json:"usage,omitempty"`
}

type Stream interface {
	Recv() (*StreamChunk, error)
	Close() error
}
