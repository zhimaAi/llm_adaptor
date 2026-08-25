// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package chat

import "testing"

func TestAccumulatorMergesMultipleToolCalls(t *testing.T) {
	firstIndex, secondIndex := 0, 1
	accumulator := NewAccumulator()
	chunks := []*StreamChunk{
		{ID: "id", Model: "model", Choices: []ChunkChoice{{Index: 0, Delta: Message{Role: RoleAssistant, ToolCalls: []ToolCall{
			{Index: &firstIndex, ID: "call-1", Type: "function", Function: FunctionCall{Name: "weather", Arguments: `{"city":`}},
			{Index: &secondIndex, ID: "call-2", Type: "function", Function: FunctionCall{Name: "time", Arguments: `{"zone":`}},
		}}}}},
		{Choices: []ChunkChoice{{Index: 0, Delta: Message{ToolCalls: []ToolCall{
			{Index: &firstIndex, Function: FunctionCall{Arguments: `"Wuhan"}`}},
			{Index: &secondIndex, Function: FunctionCall{Arguments: `"UTC+8"}`}},
		}}, FinishReason: "tool_calls"}}},
	}
	for _, chunk := range chunks {
		if err := accumulator.Add(chunk); err != nil {
			t.Fatal(err)
		}
	}
	response := accumulator.Response()
	if len(response.Choices) != 1 || len(response.Choices[0].Message.ToolCalls) != 2 {
		t.Fatalf("unexpected response: %#v", response)
	}
	if got := response.Choices[0].Message.ToolCalls[0].Function.Arguments; got != `{"city":"Wuhan"}` {
		t.Fatalf("unexpected first arguments: %s", got)
	}
	if got := response.Choices[0].Message.ToolCalls[1].Function.Arguments; got != `{"zone":"UTC+8"}` {
		t.Fatalf("unexpected second arguments: %s", got)
	}
}

func TestAccumulatorMergesPartialUsage(t *testing.T) {
	accumulator := NewAccumulator()
	if err := accumulator.Add(&StreamChunk{Usage: &Usage{PromptTokens: 7, TotalTokens: 7}}); err != nil {
		t.Fatal(err)
	}
	if err := accumulator.Add(&StreamChunk{Usage: &Usage{CompletionTokens: 5}}); err != nil {
		t.Fatal(err)
	}
	usage := accumulator.Response().Usage
	if usage.PromptTokens != 7 || usage.CompletionTokens != 5 || usage.TotalTokens != 12 {
		t.Fatalf("unexpected merged usage: %#v", usage)
	}
}

func TestAccumulatorMergesStandardMessageDeltas(t *testing.T) {
	accumulator := NewAccumulator()
	chunks := []*StreamChunk{
		{Choices: []ChunkChoice{{Index: 0, Delta: Message{
			FunctionCall: &FunctionCall{Name: "weather", Arguments: `{"city":`},
			Audio:        &Audio{ID: "audio-id", Data: "first", Transcript: "hello "},
			Annotations:  []Annotation{{Type: "url_citation", URLCitation: &URLCitation{URL: "https://example.com/one"}}},
		}}}},
		{Choices: []ChunkChoice{{Index: 0, Delta: Message{
			FunctionCall: &FunctionCall{Arguments: `"Wuhan"}`},
			Audio:        &Audio{Data: "second", ExpiresAt: 123, Transcript: "world"},
			Annotations:  []Annotation{{Type: "url_citation", URLCitation: &URLCitation{URL: "https://example.com/two"}}},
		}}}},
	}
	for _, chunk := range chunks {
		if err := accumulator.Add(chunk); err != nil {
			t.Fatal(err)
		}
	}
	message := accumulator.Response().Choices[0].Message
	if message.FunctionCall == nil || message.FunctionCall.Name != "weather" || message.FunctionCall.Arguments != `{"city":"Wuhan"}` {
		t.Fatalf("unexpected function call: %#v", message.FunctionCall)
	}
	if message.Audio == nil || message.Audio.ID != "audio-id" || message.Audio.Data != "firstsecond" || message.Audio.ExpiresAt != 123 || message.Audio.Transcript != "hello world" {
		t.Fatalf("unexpected audio: %#v", message.Audio)
	}
	if len(message.Annotations) != 2 || message.Annotations[0].URLCitation.URL != "https://example.com/one" || message.Annotations[1].URLCitation.URL != "https://example.com/two" {
		t.Fatalf("unexpected annotations: %#v", message.Annotations)
	}
}
