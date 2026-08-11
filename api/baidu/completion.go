// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package baidu

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/zhimaAi/llm_adaptor/basics"
	"github.com/zhimaAi/llm_adaptor/common"
	"github.com/zhimaAi/llm_adaptor/define"
)

type ChatCompletionRequest struct {
	Model           string        `json:"model"`
	Messages        any           `json:"messages"`
	Stream          bool          `json:"stream,omitempty"`
	Temperature     float64       `json:"temperature,omitempty"`
	TopP            float32       `json:"top_p,omitempty"`
	PenaltyScore    float32       `json:"penalty_score,omitempty"`
	System          string        `json:"system,omitempty"`
	Stop            []string      `json:"stop,omitempty"`
	DisableSearch   bool          `json:"disable_search,omitempty"`
	EnableCitation  bool          `json:"enable_citation,omitempty"`
	EnableTrace     bool          `json:"enable_trace,omitempty"`
	MaxOutputTokens int           `json:"max_output_tokens,omitempty"`
	ResponseFormat  string        `json:"response_format,omitempty"`
	UserId          string        `json:"user_id,omitempty"`
	Functions       []Function    `json:"functions,omitempty"`
	Tools           []interface{} `json:"tools,omitempty"`
	Thinking        *Thinking     `json:"thinking,omitempty"`
	EnableThinking  *bool         `json:"enable_thinking,omitempty"`
}

type Thinking struct {
	Type string `json:"type"`
}
type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ChatCompletionResponse struct {
	ID               string                                 `json:"id"`
	Object           string                                 `json:"object"`
	Created          int64                                  `json:"created"`
	IsTruncated      bool                                   `json:"is_truncated"`
	NeedClearHistory bool                                   `json:"need_clear_history"`
	FinishReason     string                                 `json:"finish_reason"`
	Usage            Usage                                  `json:"usage"`
	Result           string                                 `json:"result"`
	ReasoningContent string                                 `json:"reasoning_content"`
	FunctionCall     FunctionCall                           `json:"function_call"`
	Choices          []define.CommonChatCompletionChoiceRes `json:"choices"`
}

type ChatCompletionChoice struct {
	Message ChatCompletionResponseMessage `json:"message,omitempty"`
	Delta   ChatCompletionResponseMessage `json:"delta,omitempty"`
}
type ChatCompletionResponseMessage struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content"`
}

type ChatCompletionStreamResponse struct {
	ID               string                                 `json:"id"`
	Object           string                                 `json:"object"`
	Created          int64                                  `json:"created"`
	IsTruncated      bool                                   `json:"is_truncated"`
	IsEnd            bool                                   `json:"is_end"`
	NeedClearHistory bool                                   `json:"need_clear_history"`
	FinishReason     string                                 `json:"finish_reason"`
	Usage            Usage                                  `json:"usage"`
	Result           string                                 `json:"result"`
	ReasoningContent string                                 `json:"reasoning_content"`
	FunctionCall     FunctionCall                           `json:"function_call"`
	Choices          []define.CommonChatCompletionChoiceRes `json:"choices"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Thoughts  string `json:"thoughts"`
	Arguments string `json:"arguments"`
}

type ChatCompletionStream struct {
	*common.StreamReader[ChatCompletionStreamResponse]
}

func (c *ChatCompletionStream) Recv() (ChatCompletionStreamResponse, error) {
	if c.StreamReader.IsFinished {
		return ChatCompletionStreamResponse{}, io.EOF
	}

	var emptyMessagesCount uint
	var headerData = []byte("data: ")
	var errorPrefix = []byte(`{"error`)

	for {
		rawLine, readErr := c.StreamReader.Reader.ReadBytes('\n')
		if readErr != nil {
			if readErr != io.EOF {
				c.StreamReader.UnmarshalError()
				if c.StreamReader.ErrorResponse != nil {
					return *new(ChatCompletionStreamResponse), fmt.Errorf("unmarshal error, %w", c.StreamReader.ErrorResponse.Error())
				}
				return *new(ChatCompletionStreamResponse), readErr
			}
		}

		noSpaceLine := bytes.TrimSpace(rawLine)
		if !bytes.HasPrefix(noSpaceLine, headerData) {
			if bytes.HasPrefix(noSpaceLine, errorPrefix) {
				var errResp ErrorResponse
				err := basics.JsonDecode(noSpaceLine, &errResp)
				if err != nil {
					return *new(ChatCompletionStreamResponse), fmt.Errorf("unmarshal error, %w", c.StreamReader.ErrorResponse.Error())
				} else {
					errResp.SetHTTPStatusCode(c.Response.StatusCode)
					return *new(ChatCompletionStreamResponse), errResp.Error()
				}
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

		var response ChatCompletionStreamResponse
		if strings.TrimSpace(string(noPrefixLine)) == "[DONE]" {
			c.StreamReader.IsFinished = true
			return *new(ChatCompletionStreamResponse), io.EOF
		}
		unmarshalErr := basics.JsonDecode(noPrefixLine, &response)
		if unmarshalErr != nil {
			return *new(ChatCompletionStreamResponse), unmarshalErr
		}
		if response.IsEnd {
			c.StreamReader.IsFinished = true
			return response, nil
		}

		return response, nil
	}
}
