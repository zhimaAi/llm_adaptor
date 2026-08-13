// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package cohere

import (
	"bytes"
	"fmt"
	"io"

	"github.com/zhimaAi/llm_adaptor/basics"
	"github.com/zhimaAi/llm_adaptor/common"
)

type ChatCompletionRequest struct {
	Model             string        `json:"model,omitempty"`
	Messages          []ChatMessage `json:"messages,omitempty"`
	Message           string        `json:"message,omitempty"`
	Stream            bool          `json:"stream,omitempty"`
	Preamble          string        `json:"preamble,omitempty"`
	ChatHistory       []ChatHistory `json:"chat_history,omitempty"`
	ConversationID    string        `json:"conversation_id,omitempty"`
	PromptTruncation  string        `json:"prompt_truncation,omitempty"`
	Connectors        []Connector   `json:"connectors,omitempty"`
	SearchQueriesOnly bool          `json:"search_queries_only,omitempty"`
	Documents         []Document    `json:"documents,omitempty"`
	CitationQuality   string        `json:"citation_quality,omitempty"`
	Temperature       float64       `json:"temperature,omitempty"`
	MaxTokens         int           `json:"max_tokens,omitempty"`
	MaxInputTokens    int           `json:"max_input_tokens,omitempty"`
	K                 int           `json:"k,omitempty"`
	P                 int           `json:"p,omitempty"`
	Seed              int           `json:"seed,omitempty"`
	StopSequences     []string      `json:"stop_sequences,omitempty"`
	FrequencyPenalty  int           `json:"frequency_penalty,omitempty"`
	PresencePenalty   int           `json:"presence_penalty,omitempty"`
	Thinking          *Thinking     `json:"thinking,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatHistory and Connector retain the V1 Chat API request schema for callers
// that use the api/cohere package directly.
type ChatHistory struct {
	Role    string `json:"role"`
	Message string `json:"message"`
}

type Connector struct {
	ID              string `json:"id"`
	UserAccessToken string `json:"access_token,omitempty"`
}

type Thinking struct {
	Type        string `json:"type,omitempty"`
	TokenBudget int    `json:"token_budget,omitempty"`
}

type ChatContent struct {
	Type     string `json:"type,omitempty"`
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
}

type ChatResponseMessage struct {
	Role    string        `json:"role"`
	Content []ChatContent `json:"content"`
}

type ChatUsage struct {
	BilledUnits BilledUnits `json:"billed_units"`
	Tokens      Tokens      `json:"tokens"`
}

// ResponseMeta is retained for V1 response schemas.
type ResponseMeta struct {
	APIVersion  APIVersion  `json:"api_version"`
	BilledUnits BilledUnits `json:"billed_units"`
	Tokens      Tokens      `json:"tokens"`
	Warnings    []string    `json:"warnings"`
}

type ChatCompletionResponse struct {
	ID           string              `json:"id"`
	FinishReason string              `json:"finish_reason"`
	Message      ChatResponseMessage `json:"message"`
	Usage        ChatUsage           `json:"usage"`
	Text         string              `json:"text"`
	GenerationId string              `json:"generation_id"`
	Documents    []Document          `json:"documents"`
	Meta         ResponseMeta        `json:"meta"`
}

type ChatCompletionStreamResponse struct {
	Type         string   `json:"type"`
	Index        int      `json:"index,omitempty"`
	IsFinished   bool     `json:"is_finished"`
	EventType    string   `json:"event_type"`
	Text         string   `json:"text"`
	Response     Response `json:"response"`
	FinishReason string   `json:"finish_reason"`
	Delta        struct {
		Message struct {
			Role    string      `json:"role,omitempty"`
			Content ChatContent `json:"content,omitempty"`
		} `json:"message,omitempty"`
		FinishReason string    `json:"finish_reason,omitempty"`
		Usage        ChatUsage `json:"usage,omitempty"`
	} `json:"delta"`
}

type Response struct {
	ResponseID   string        `json:"response_id"`
	Text         string        `json:"text"`
	GenerationID string        `json:"generation_id"`
	ChatHistory  []ChatHistory `json:"chat_history"`
	FinishReason string        `json:"finish_reason"`
	Meta         ResponseMeta  `json:"meta"`
}

type ChatCompletionStream struct {
	*common.StreamReader[ChatCompletionStreamResponse]
}

func (c *ChatCompletionStream) Recv() (ChatCompletionStreamResponse, error) {
	if c.StreamReader.IsFinished {
		return ChatCompletionStreamResponse{}, io.EOF
	}

	for {
		rawLine, readErr := c.StreamReader.Reader.ReadBytes('\n')
		if readErr != nil {
			if readErr == io.EOF {
				c.StreamReader.IsFinished = true
				return ChatCompletionStreamResponse{}, io.EOF
			}
			c.StreamReader.UnmarshalError()
			if c.StreamReader.ErrorResponse != nil {
				return ChatCompletionStreamResponse{}, fmt.Errorf("unmarshal error, %w", c.StreamReader.ErrorResponse.Error())
			}
			return ChatCompletionStreamResponse{}, readErr
		}

		line := bytes.TrimSpace(rawLine)
		if len(line) == 0 || bytes.HasPrefix(line, []byte("event:")) || bytes.HasPrefix(line, []byte(":")) {
			continue
		}
		if bytes.HasPrefix(line, []byte("data:")) {
			line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		}
		if bytes.Equal(line, []byte("[DONE]")) {
			c.StreamReader.IsFinished = true
			return ChatCompletionStreamResponse{}, io.EOF
		}

		var response ChatCompletionStreamResponse
		if err := basics.JsonDecode(line, &response); err != nil {
			return ChatCompletionStreamResponse{}, err
		}
		if response.Type == "message-end" || response.EventType == "stream-end" || response.IsFinished {
			c.StreamReader.IsFinished = true
		}
		return response, nil
	}
}
