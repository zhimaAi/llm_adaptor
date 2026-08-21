// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
)

func TestBuildClaudeRequestConvertsToolHistory(t *testing.T) {
	body, err := buildClaudeRequest(&chat.CreateRequest{
		Model: "claude-test",
		Messages: []chat.Message{
			{Role: chat.RoleUser, Content: chat.TextContent("weather")},
			{
				Role:    chat.RoleAssistant,
				Content: chat.TextContent(""),
				ToolCalls: []chat.ToolCall{{
					ID: "tool-1", Type: "function",
					Function: chat.FunctionCall{Name: "get_weather", Arguments: `{"city":"Wuhan"}`},
				}},
			},
			{Role: chat.RoleTool, ToolCallID: "tool-1", Content: chat.TextContent(`{"temperature":30}`)},
		},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	messages, ok := body["messages"].([]claudeRequestMessage)
	if !ok || len(messages) != 3 {
		t.Fatalf("unexpected messages: %#v", body["messages"])
	}
	toolUse := messages[1].Content[0]
	if messages[1].Role != claudeRoleAssistant || toolUse.Type != claudeContentToolUse || toolUse.ID != "tool-1" || toolUse.Name != "get_weather" {
		t.Fatalf("unexpected tool_use: %#v", messages[1])
	}
	toolResult := messages[2].Content[0]
	if messages[2].Role != claudeRoleUser || toolResult.Type != claudeContentToolResult || toolResult.ToolUseID != "tool-1" {
		t.Fatalf("unexpected tool_result: %#v", messages[2])
	}
}

func TestBuildClaudeRequestConvertsImageContent(t *testing.T) {
	body, err := buildClaudeRequest(&chat.CreateRequest{
		Model: "claude-test",
		Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.PartsContent(
			chat.ContentPart{Type: chat.ContentPartText, Text: "describe"},
			chat.ContentPart{Type: chat.ContentPartImageURL, ImageURL: &chat.ImageURL{URL: "https://example.com/image.png"}},
		)}},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	messages := body["messages"].([]claudeRequestMessage)
	if source := messages[0].Content[1].Source; source == nil || source.Type != claudeSourceURL || source.URL != "https://example.com/image.png" {
		t.Fatalf("unexpected image source: %#v", source)
	}
}

func TestClaudeStreamUsesDenseToolCallIndexes(t *testing.T) {
	raw := strings.Join([]string{
		`data: {"type":"message_start","message":{"id":"msg-1","model":"claude-test","usage":{"input_tokens":3}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text"}}`,
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"tool-1","name":"first"}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"a\":"}}`,
		`data: {"type":"content_block_start","index":2,"content_block":{"type":"text"}}`,
		`data: {"type":"content_block_start","index":3,"content_block":{"type":"tool_use","id":"tool-2","name":"second"}}`,
		`data: {"type":"content_block_delta","index":3,"delta":{"type":"input_json_delta","partial_json":"{\"b\":"}}`,
		`data: {"type":"message_stop"}`,
	}, "\n")
	body := io.NopCloser(strings.NewReader(raw))
	stream := newClaudeStream(context.Background(), body, func() {}, "hint")
	var indexes []int
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		for _, choice := range chunk.Choices {
			for _, toolCall := range choice.Delta.ToolCalls {
				indexes = append(indexes, *toolCall.Index)
			}
		}
	}
	want := []int{0, 0, 1, 1}
	if len(indexes) != len(want) {
		t.Fatalf("unexpected indexes: %#v", indexes)
	}
	for index := range want {
		if indexes[index] != want[index] {
			t.Fatalf("unexpected indexes: %#v", indexes)
		}
	}
}

func TestClaudeStreamUnknownToolDeltaTerminates(t *testing.T) {
	body := io.NopCloser(strings.NewReader(`data: {"type":"content_block_delta","index":4,"delta":{"partial_json":"{}"}}`))
	stream := newClaudeStream(context.Background(), body, func() {}, "hint")
	if _, err := stream.Recv(); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after protocol error, got %v", err)
	}
}
