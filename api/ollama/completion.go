// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ollama

import (
	"bytes"
	"io"
	"time"

	"github.com/zhimaAi/llm_adaptor/basics"
	"github.com/zhimaAi/llm_adaptor/common"
)

type ImageData []byte

type ChatCompletionMessage struct {
	Role             string           `json:"role"`
	Content          string           `json:"content"`
	Thinking         string           `json:"thinking"`
	ToolCalls        basics.ToolCalls `json:"tool_calls"`
	ReasoningContent string           `json:"reasoning_content"`
	Images           []ImageData      `json:"images,omitempty"`
}

type ChatCompletionRequest struct {
	// Model is the model name, as in [GenerateRequest].
	Model string `json:"model"`

	// Tools
	Tools []interface{} `json:"tools"`

	// Messages is the messages of the chat - can be used to keep a chat memory.
	Messages any `json:"messages"`

	// Stream enable streaming of returned response; true by default.
	Stream *bool `json:"stream,omitempty"`

	// Think controls whether supported models return a separate thinking trace.
	Think *bool `json:"think,omitempty"`

	// Format is the format to return the response in (e.g. "json").
	Format string `json:"format"`

	// KeepAlive controls how long the model will stay loaded into memory
	// followin the request.
	KeepAlive *Duration `json:"keep_alive,omitempty"`

	// Options lists model-specific options.
	Options map[string]interface{} `json:"options"`
}

type ChatCompletionChoice struct {
	Message ChatCompletionMessage `json:"message"`
}
type ChatCompletionStreamChoice struct {
	Index        int                   `json:"index"`
	Delta        ChatCompletionMessage `json:"delta"`
	FinishReason string                `json:"finish_reason"`
}

type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type ChatCompletionResponse struct {
	Model      string                `json:"model"`
	CreatedAt  time.Time             `json:"created_at"`
	Message    ChatCompletionMessage `json:"message"`
	DoneReason string                `json:"done_reason,omitempty"`

	Done bool `json:"done"`

	Metrics
}

type Metrics struct {
	TotalDuration      time.Duration `json:"total_duration,omitempty"`
	LoadDuration       time.Duration `json:"load_duration,omitempty"`
	PromptEvalCount    int           `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration time.Duration `json:"prompt_eval_duration,omitempty"`
	EvalCount          int           `json:"eval_count,omitempty"`
	EvalDuration       time.Duration `json:"eval_duration,omitempty"`
}

type ChatCompletionStreamResponse struct {
	Model      string                `json:"model"`
	CreatedAt  time.Time             `json:"created_at"`
	Message    ChatCompletionMessage `json:"message"`
	DoneReason string                `json:"done_reason,omitempty"`

	Done bool `json:"done"`

	Metrics
}

type ChatCompletionStream struct {
	*common.StreamReader[ChatCompletionStreamResponse]
}

func (c *ChatCompletionStream) Recv() (ChatCompletionStreamResponse, error) {
	if c.StreamReader.IsFinished {
		return ChatCompletionStreamResponse{}, io.EOF
	}

	for {
		rawLine, readErr := c.StreamReader.Reader.ReadBytes(byte('\n'))
		if readErr != nil && readErr == io.EOF {
			return *new(ChatCompletionStreamResponse), io.EOF
		}
		noSpaceLine := bytes.TrimSpace(rawLine)
		var response ChatCompletionStreamResponse
		unmarshalErr := basics.JsonDecode(noSpaceLine, &response)
		if unmarshalErr != nil {
			return *new(ChatCompletionStreamResponse), unmarshalErr
		}
		if response.Done {
			c.StreamReader.IsFinished = true
			return *new(ChatCompletionStreamResponse), io.EOF
		}
		if response.Message.Content == "" && response.Message.ReasoningContent == "" && len(response.Message.ToolCalls) == 0 {
			continue
		}
		return response, nil
	}
}
