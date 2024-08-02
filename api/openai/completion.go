// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zhimaAi/llm_adaptor/common"
	"io"
	"strings"
)

type ChatCompletionRequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type ChatCompletionResponseMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type ToolCall struct {
	Id       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatCompletionRequest struct {
	Model            string                         `json:"model"`
	Messages         []ChatCompletionRequestMessage `json:"messages"`
	Stream           bool                           `json:"stream,omitempty"`
	StreamOptions    *StreamOptions                 `json:"stream_options,omitempty"`
	FrequencyPenalty int                            `json:"frequency_penalty,omitempty"`
	MaxTokens        int                            `json:"max_tokens,omitempty"`
	N                int                            `json:"n,omitempty"`
	PresencePenalty  int                            `json:"presence_penalty,omitempty"`
	ResponseFormat   string                         `json:"response_format,omitempty"`
	Seed             int                            `json:"seed,omitempty"`
	Temperature      float64                        `json:"temperature,omitempty"`
	TopP             int                            `json:"top_p,omitempty"`
	User             string                         `json:"user,omitempty"`
	Tools            []interface{}                  `json:"tools"`
}
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type ChatCompletionChoice struct {
	Message ChatCompletionResponseMessage `json:"message"`
}
type ChatCompletionStreamChoice struct {
	Index        int                           `json:"index"`
	Delta        ChatCompletionResponseMessage `json:"delta"`
	FinishReason string                        `json:"finish_reason"`
	Usage        ChatCompletionUsage           `json:"usage"`
}

type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type ChatCompletionResponse struct {
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   ChatCompletionUsage    `json:"usage"`
}
type ChatCompletionStreamResponse struct {
	ID      string                       `json:"id"`
	Choices []ChatCompletionStreamChoice `json:"choices,omitempty"`
	Usage   ChatCompletionUsage          `json:"usage"`
}

type ChatCompletionStream struct {
	*common.StreamReader[ChatCompletionStreamResponse]
}

var (
	headerData  = []byte("data:")
	errorPrefix = []byte(`data: {"error":`)
)

func (c *ChatCompletionStream) Recv() (ChatCompletionStreamResponse, error) {
	if c.StreamReader.IsFinished {
		return ChatCompletionStreamResponse{}, io.EOF
	}

	var (
		emptyMessagesCount uint
		hasErrorPrefix     bool
	)

	for {
		rawLine, readErr := c.StreamReader.Reader.ReadBytes('\n')
		if readErr != nil || hasErrorPrefix {
			if readErr != io.EOF {
				c.StreamReader.UnmarshalError()
				if c.StreamReader.ErrorResponse != nil {
					return *new(ChatCompletionStreamResponse), fmt.Errorf("unmarshal error, %w", c.StreamReader.ErrorResponse.Error())
				}
				return *new(ChatCompletionStreamResponse), readErr
			} else {
				c.StreamReader.IsFinished = true
				return *new(ChatCompletionStreamResponse), io.EOF
			}
		}

		noSpaceLine := bytes.TrimSpace(rawLine)
		if bytes.HasPrefix(noSpaceLine, errorPrefix) {
			hasErrorPrefix = true
		}
		if !bytes.HasPrefix(noSpaceLine, headerData) || hasErrorPrefix {
			if hasErrorPrefix {
				noSpaceLine = bytes.TrimPrefix(noSpaceLine, headerData)
			}
			writeErr := c.StreamReader.ErrAccumulator.Write(noSpaceLine)
			if writeErr != nil {
				return *new(ChatCompletionStreamResponse), writeErr
			}
			emptyMessagesCount++
			if emptyMessagesCount > c.StreamReader.EmptyMessagesLimit {
				return *new(ChatCompletionStreamResponse), errors.New("stream has sent too many empty messages")
			}

			continue
		}

		noPrefixLine := bytes.TrimPrefix(noSpaceLine, headerData)
		if strings.TrimSpace(string(noPrefixLine)) == "[DONE]" {
			c.StreamReader.IsFinished = true
			return *new(ChatCompletionStreamResponse), io.EOF
		}

		var response ChatCompletionStreamResponse
		unmarshalErr := json.Unmarshal(noPrefixLine, &response)
		if unmarshalErr != nil {
			return *new(ChatCompletionStreamResponse), unmarshalErr
		}

		return response, nil
	}
}
