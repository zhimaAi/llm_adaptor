// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openrouter

import (
	"encoding/json"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
)

type reasoningDetail struct {
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type reasoningMessage struct {
	Reasoning        string            `json:"reasoning,omitempty"`
	ReasoningDetails []reasoningDetail `json:"reasoning_details,omitempty"`
}

type reasoningResponse struct {
	Choices []struct {
		Index   int              `json:"index"`
		Message reasoningMessage `json:"message"`
		Delta   reasoningMessage `json:"delta"`
	} `json:"choices"`
}

func configureChat(spec *openai.Spec, _ provider.Config) {
	spec.ApplyReasoning = func(_ string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		delete(body, "reasoning_effort")
		body["reasoning"] = map[string]any{"effort": string(effort)}
	}
	spec.TransformChatResponse = transformChatResponse
	spec.TransformStreamResponse = transformStreamResponse
}

func transformChatResponse(raw []byte, response *chat.CreateResponse) error {
	var source reasoningResponse
	if err := json.Unmarshal(raw, &source); err != nil {
		return err
	}
	for _, sourceChoice := range source.Choices {
		for index := range response.Choices {
			if response.Choices[index].Index != sourceChoice.Index || response.Choices[index].Message.ReasoningContent != "" {
				continue
			}
			response.Choices[index].Message.ReasoningContent = reasoningContent(sourceChoice.Message)
			break
		}
	}
	return nil
}

func transformStreamResponse(raw []byte, response *chat.StreamChunk) error {
	var source reasoningResponse
	if err := json.Unmarshal(raw, &source); err != nil {
		return err
	}
	for _, sourceChoice := range source.Choices {
		for index := range response.Choices {
			if response.Choices[index].Index != sourceChoice.Index || response.Choices[index].Delta.ReasoningContent != "" {
				continue
			}
			response.Choices[index].Delta.ReasoningContent = reasoningContent(sourceChoice.Delta)
			break
		}
	}
	return nil
}

func reasoningContent(message reasoningMessage) string {
	if message.Reasoning != "" {
		return message.Reasoning
	}
	var text strings.Builder
	var summary strings.Builder
	for _, detail := range message.ReasoningDetails {
		switch detail.Type {
		case "reasoning.text":
			text.WriteString(detail.Text)
		case "reasoning.summary":
			summary.WriteString(detail.Summary)
		}
	}
	if text.Len() > 0 {
		return text.String()
	}
	return summary.String()
}
