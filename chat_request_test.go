// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
)

func TestReasoningEffortValidation(t *testing.T) {
	valid := []chat.ReasoningEffort{
		"", chat.ReasoningEffortNone, chat.ReasoningEffortMinimal, chat.ReasoningEffortLow,
		chat.ReasoningEffortMedium, chat.ReasoningEffortHigh, chat.ReasoningEffortXHigh,
		chat.ReasoningEffortMax,
	}
	for _, effort := range valid {
		if err := validateReasoningEffort(effort); err != nil {
			t.Fatalf("expected %q to be valid: %v", effort, err)
		}
	}
	if err := validateReasoningEffort("typo"); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}

func TestProviderReasoningRequestMapping(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		model    string
		effort   chat.ReasoningEffort
		assert   func(*testing.T, map[string]any)
	}{
		{name: "openai", provider: ProviderOpenAI, model: "o3", effort: chat.ReasoningEffortMedium, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "reasoning_effort", "medium")
			assertBodyValue(t, body, "max_completion_tokens", float64(1024))
			assertBodyMissing(t, body, "max_tokens", "temperature", "thinking")
		}},
		{name: "gemini level", provider: ProviderGemini, model: "gemini-2.5-pro", effort: chat.ReasoningEffortMedium, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "reasoning_effort", "medium")
		}},
		{name: "gemini 3 disabled", provider: ProviderGemini, model: "gemini-3-pro", effort: chat.ReasoningEffortNone, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "reasoning_effort", "minimal")
		}},
		{name: "ali", provider: ProviderAli, model: "qwen", effort: chat.ReasoningEffortLow, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "enable_thinking", true)
			assertBodyMissing(t, body, "reasoning_effort")
		}},
		{name: "ollama disabled", provider: ProviderOllama, model: "qwen", effort: chat.ReasoningEffortNone, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "think", false)
		}},
		{name: "baidu boolean model", provider: ProviderBaidu, model: "qwen3-32b", effort: chat.ReasoningEffortHigh, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "enable_thinking", true)
		}},
		{name: "baidu typed model disabled", provider: ProviderBaidu, model: "ernie-x1", effort: chat.ReasoningEffortNone, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "thinking", map[string]any{"type": "disabled"})
		}},
		{name: "deepseek", provider: ProviderDeepSeek, model: "deepseek-reasoner", effort: chat.ReasoningEffortMedium, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "thinking", map[string]any{"type": "enabled"})
		}},
		{name: "openrouter disabled", provider: ProviderOpenRouter, model: "model", effort: chat.ReasoningEffortNone, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "reasoning", map[string]any{"enabled": false})
		}},
		{name: "minimax m3", provider: ProviderMiniMax, model: "MiniMax-M3", effort: chat.ReasoningEffortMedium, assert: func(t *testing.T, body map[string]any) {
			assertBodyValue(t, body, "thinking", map[string]any{"type": "adaptive"})
			assertBodyValue(t, body, "reasoning_split", true)
			assertBodyValue(t, body, "max_completion_tokens", float64(1024))
			assertBodyMissing(t, body, "max_tokens", "reasoning_effort")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			maxTokens := 1024
			temperature := 0.7
			extra := map[string]any{"custom_extension": "kept"}
			request := &chat.CreateRequest{
				Model: test.model, Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}},
				MaxTokens: &maxTokens, Temperature: &temperature, ReasoningEffort: test.effort, ExtraBody: extra,
			}
			body, err := buildOpenAIChatRequest(test.provider, request, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			assertBodyValue(t, body, "custom_extension", "kept")
			test.assert(t, body)
			if request.MaxTokens != &maxTokens || request.Temperature != &temperature || request.ExtraBody["custom_extension"] != "kept" {
				t.Fatalf("caller request was mutated: %#v", request)
			}
		})
	}
}

func TestChatExtraBodyRejectsPublicAndGeneratedFields(t *testing.T) {
	for _, field := range []string{"temperature", "stream", "reasoning_effort", "enable_thinking", "think", "thinking", "reasoning", "reasoning_split"} {
		t.Run(field, func(t *testing.T) {
			request := &chat.CreateRequest{
				Model: "model", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}},
				ReasoningEffort: chat.ReasoningEffortMedium, ExtraBody: map[string]any{field: true},
			}
			if _, err := buildOpenAIChatRequest(ProviderAli, request, false, nil); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("field %q error = %v, want ErrInvalidRequest", field, err)
			}
		})
	}
}

func TestChatStreamOptionsDefaultAndExplicitDisable(t *testing.T) {
	falseValue := false
	trueValue := true
	tests := []struct {
		name    string
		options *chat.StreamOptions
		want    bool
	}{
		{name: "nil defaults enabled", want: true},
		{name: "nil include usage defaults enabled", options: &chat.StreamOptions{}, want: true},
		{name: "explicit enabled", options: &chat.StreamOptions{IncludeUsage: &trueValue}, want: true},
		{name: "explicit disabled", options: &chat.StreamOptions{IncludeUsage: &falseValue}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := &chat.CreateRequest{Model: "model", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}}}
			body, err := buildOpenAIChatRequest(ProviderOpenAI, request, true, test.options)
			if err != nil {
				t.Fatal(err)
			}
			options := body["stream_options"].(map[string]any)
			assertBodyValue(t, options, "include_usage", test.want)
			if test.options == nil && request.ExtraBody != nil {
				t.Fatal("request unexpectedly mutated")
			}
		})
	}
}

func TestNormalizeChatStreamRequestDoesNotMutateCaller(t *testing.T) {
	request := &chat.StreamRequest{CreateRequest: chat.CreateRequest{Model: "model"}}
	normalized := normalizeChatStreamRequest(request)
	if request.StreamOptions != nil {
		t.Fatal("caller request was mutated")
	}
	if normalized == request || normalized.StreamOptions == nil || normalized.StreamOptions.IncludeUsage == nil || !*normalized.StreamOptions.IncludeUsage {
		t.Fatalf("unexpected normalized request: %#v", normalized)
	}
}

func TestBuildClaudeRequestAppliesProviderReasoning(t *testing.T) {
	maxTokens := 1024
	temperature := 0.8
	request := &chat.CreateRequest{
		Model: "claude-3-7-sonnet", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}},
		MaxTokens: &maxTokens, Temperature: &temperature, ReasoningEffort: chat.ReasoningEffortMedium,
	}
	body, err := buildClaudeRequest(request, false)
	if err != nil {
		t.Fatal(err)
	}
	assertBodyValue(t, body, "thinking", map[string]any{"type": "enabled", "budget_tokens": claudeLegacyThinkingBudget})
	assertBodyValue(t, body, "max_tokens", claudeLegacyMinimumTokens)
	assertBodyMissing(t, body, "temperature", "reasoning_effort")
	if *request.MaxTokens != 1024 || *request.Temperature != 0.8 {
		t.Fatal("caller request was mutated")
	}
}

func TestBuildClaudeRequestRejectsGeneratedThinkingConflict(t *testing.T) {
	request := &chat.CreateRequest{
		Model: "claude-3-7-sonnet", Messages: []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}},
		ReasoningEffort: chat.ReasoningEffortMedium,
		ExtraBody:       map[string]any{"thinking": map[string]any{"type": "caller"}},
	}
	if _, err := buildClaudeRequest(request, false); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("error = %v, want ErrInvalidRequest", err)
	}
}

func assertBodyValue(t *testing.T, body map[string]any, key string, want any) {
	t.Helper()
	if got, exists := body[key]; !exists || !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected %s: got %#v, want %#v in %#v", key, got, want, body)
	}
}

func assertBodyMissing(t *testing.T, body map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if value, exists := body[key]; exists {
			t.Fatalf("expected %s to be absent, got %#v in %#v", key, value, body)
		}
	}
}
