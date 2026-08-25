// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package chat_test

import (
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
)

func TestAccumulatorMergesPublicStreamData(t *testing.T) {
	toolIndex := 0
	accumulator := chat.NewAccumulator()
	chunks := []*chat.StreamChunk{
		{
			ID: "id", Model: "model", Usage: &chat.Usage{PromptTokens: 7, TotalTokens: 7},
			Choices: []chat.ChunkChoice{{
				Index: 0,
				Delta: chat.Message{
					Role: chat.RoleAssistant, Content: chat.TextContent("hel"), ReasoningContent: "plan ",
					ToolCalls: []chat.ToolCall{{Index: &toolIndex, ID: "call-1", Type: "function", Function: chat.FunctionCall{Name: "weather", Arguments: `{"city":`}}},
				},
			}},
		},
		{
			Usage: &chat.Usage{CompletionTokens: 5},
			Choices: []chat.ChunkChoice{{
				Index: 0, FinishReason: "tool_calls",
				Delta: chat.Message{
					Content: chat.TextContent("lo"), ReasoningContent: "done",
					ToolCalls: []chat.ToolCall{{Index: &toolIndex, Function: chat.FunctionCall{Arguments: `"Wuhan"}`}}},
				},
			}},
		},
	}
	for _, chunk := range chunks {
		if err := accumulator.Add(chunk); err != nil {
			t.Fatal(err)
		}
	}

	response := accumulator.Response()
	if len(response.Choices) != 1 {
		t.Fatalf("unexpected response: %#v", response)
	}
	choice := response.Choices[0]
	if choice.Message.Content.Text == nil || *choice.Message.Content.Text != "hello" || choice.Message.ReasoningContent != "plan done" {
		t.Fatalf("unexpected message: %#v", choice.Message)
	}
	if len(choice.Message.ToolCalls) != 1 || choice.Message.ToolCalls[0].Function.Arguments != `{"city":"Wuhan"}` {
		t.Fatalf("unexpected tool calls: %#v", choice.Message.ToolCalls)
	}
	if response.Usage.PromptTokens != 7 || response.Usage.CompletionTokens != 5 || response.Usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %#v", response.Usage)
	}
}
