// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
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
