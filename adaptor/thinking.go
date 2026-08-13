// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import (
	"errors"
	"strings"

	tencentHunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
	"github.com/zhimaAi/llm_adaptor/api/azure"
	"github.com/zhimaAi/llm_adaptor/api/baidu"
	"github.com/zhimaAi/llm_adaptor/api/claude"
	"github.com/zhimaAi/llm_adaptor/api/cohere"
	"github.com/zhimaAi/llm_adaptor/api/gemini"
	"github.com/zhimaAi/llm_adaptor/api/ollama"
	"github.com/zhimaAi/llm_adaptor/api/openai"
	"github.com/zhimaAi/llm_adaptor/api/spark"
	"github.com/zhimaAi/llm_adaptor/api/xinference"
	"github.com/zhimaAi/llm_adaptor/define"
)

type thinkingRequest interface {
	*openai.ChatCompletionRequest |
		*azure.ChatCompletionRequest |
		*baidu.ChatCompletionRequest |
		*claude.ChatCompletionRequest |
		*cohere.ChatCompletionRequest |
		*gemini.ChatCompletionRequest |
		*ollama.ChatCompletionRequest |
		*spark.ChatCompletionRequest |
		*xinference.ChatCompletionRequest |
		*tencentHunyuan.ChatCompletionsRequest
}

func applyThinking[T thinkingRequest](meta Meta, request T, apiVersion ...string) {
	if !meta.ChoosableThinking {
		return
	}

	switch req := any(request).(type) {
	case *openai.ChatCompletionRequest:
		switch meta.Corp {
		case "openai", "openaiAgent":
			if meta.EnabledThinking {
				req.ReasoningEffort = openai.ReasoningEffortMedium
			} else {
				req.ReasoningEffort = openai.ReasoningEffortNone
			}
			// Reasoning models use max_completion_tokens instead of max_tokens.
			req.MaxCompletionTokens = req.MaxTokens
			req.MaxTokens = 0
			// Reasoning models do not support sampling temperature.
			req.Temperature = 0
		case "ali", "siliconflow":
			req.EnableThinking = &meta.EnabledThinking
		case "deepseek", "moonshot", "zhipu", "doubao":
			req.Thinking = enabledOrDisabledThinking(meta.EnabledThinking)
		case "minimax":
			if miniMaxUsesThinkingToggle(meta.Model) {
				thinkingType := openai.ThinkingTypeDisabled
				if meta.EnabledThinking {
					thinkingType = openai.ThinkingTypeAdaptive
				}
				req.Thinking = &openai.Thinking{Type: thinkingType}
			}
			// MiniMax reasoning models use max_completion_tokens; max_tokens is deprecated.
			req.MaxCompletionTokens = req.MaxTokens
			req.MaxTokens = 0
			// Keep thinking outside content so it can be returned as ReasoningContent.
			req.ReasoningSplit = &meta.EnabledThinking
		case "openrouter":
			req.Reasoning = &openai.Reasoning{Enabled: meta.EnabledThinking}
		}
	case *azure.ChatCompletionRequest:
		if meta.EnabledThinking {
			req.ReasoningEffort = string(openai.ReasoningEffortMedium)
		} else {
			req.ReasoningEffort = string(openai.ReasoningEffortNone)
		}
		req.MaxCompletionTokens = req.MaxTokens
		req.MaxTokens = 0
		req.Temperature = 0
	case *baidu.ChatCompletionRequest:
		if len(apiVersion) == 0 || apiVersion[0] != define.ApiVersionV2 {
			return
		}
		if baiduUsesEnableThinking(meta.Model) {
			req.EnableThinking = &meta.EnabledThinking
			return
		}
		typeValue := "disabled"
		if meta.EnabledThinking {
			typeValue = "enabled"
		}
		req.Thinking = &baidu.Thinking{Type: typeValue}
	case *claude.ChatCompletionRequest:
		adaptive := claudeUsesAdaptiveThinking(meta.Model)
		if !meta.EnabledThinking {
			// Fable and Mythos models always use adaptive thinking and reject
			// thinking.type=disabled. Omitting thinking keeps the request valid;
			// these models already default to hiding the thinking content.
			if claudeCannotDisableThinking(meta.Model) {
				req.Thinking = nil
				req.Temperature = 0
				return
			}
			req.Thinking = &claude.Thinking{Type: "disabled"}
			if claudeRequiresDefaultSampling(meta.Model) {
				req.Temperature = 0
			}
			return
		}

		// Claude 4.6 and later use adaptive thinking. Older thinking-capable
		// models require the legacy enabled mode and a token budget.
		if adaptive {
			req.Thinking = &claude.Thinking{Type: "adaptive", Display: "summarized"}
		} else {
			req.Thinking = &claude.Thinking{Type: "enabled", BudgetTokens: 1024}
			if req.MaxTokens <= 1024 {
				req.MaxTokens = 2048
			}
		}
		// Thinking requests only accept the default temperature.
		req.Temperature = 0
	case *gemini.ChatCompletionRequest:
		thinkingConfig := &gemini.ThinkingConfig{IncludeThoughts: meta.EnabledThinking}
		if geminiUsesThinkingLevel(meta.Model) {
			thinkingConfig.ThinkingLevel = "minimal"
			if meta.EnabledThinking {
				thinkingConfig.ThinkingLevel = "high"
			}
		} else {
			budget := 0
			if meta.EnabledThinking {
				budget = -1
			}
			thinkingConfig.ThinkingBudget = &budget
		}
		req.GenerationConfig.ThinkingConfig = thinkingConfig
	case *cohere.ChatCompletionRequest:
		typeValue := "disabled"
		if meta.EnabledThinking {
			typeValue = "enabled"
		}
		req.Thinking = &cohere.Thinking{Type: typeValue}
	case *spark.ChatCompletionRequest:
		typeValue := "disabled"
		if meta.EnabledThinking {
			typeValue = "enabled"
		}
		req.Parameter.Chat.Thinking = &spark.Thinking{Type: typeValue}
	case *ollama.ChatCompletionRequest:
		req.Think = &meta.EnabledThinking
	case *xinference.ChatCompletionRequest:
		req.EnableThinking = &meta.EnabledThinking
	case *tencentHunyuan.ChatCompletionsRequest:
		req.EnableThinking = &meta.EnabledThinking
	}
}

func miniMaxUsesThinkingToggle(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "minimax-m3")
}

func geminiUsesThinkingLevel(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "gemini-3")
}

func baiduUsesEnableThinking(model string) bool {
	model = strings.ToLower(model)
	return strings.HasPrefix(model, "qwen3-") ||
		strings.HasPrefix(model, "ernie-4.5-turbo-vl") ||
		strings.HasPrefix(model, "ernie-4.5-vl-28b-a3b") ||
		strings.HasPrefix(model, "ernie-5.0-thinking-preview")
}

func claudeUsesAdaptiveThinking(model string) bool {
	model = strings.ToLower(model)
	adaptivePrefixes := []string{
		"claude-opus-4-6", "claude-sonnet-4-6",
		"claude-opus-4-7", "claude-opus-4-8",
		"claude-opus-5", "claude-sonnet-5",
		"claude-fable-5", "claude-mythos-5", "claude-mythos-preview",
	}
	for _, prefix := range adaptivePrefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

func claudeRequiresDefaultSampling(model string) bool {
	model = strings.ToLower(model)
	prefixes := []string{
		"claude-opus-4-7", "claude-opus-4-8",
		"claude-opus-5", "claude-sonnet-5",
		"claude-fable-5", "claude-mythos-5", "claude-mythos-preview",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

func claudeCannotDisableThinking(model string) bool {
	model = strings.ToLower(model)
	prefixes := []string{
		"claude-fable-5", "claude-mythos-5", "claude-mythos-preview",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

func enabledOrDisabledThinking(enabled bool) *openai.Thinking {
	thinkingType := openai.ThinkingTypeDisabled
	if enabled {
		thinkingType = openai.ThinkingTypeEnabled
	}
	return &openai.Thinking{Type: thinkingType}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func claudeTextAndThinking(contents []claude.Content) (string, string) {
	var textBuilder, thinkingBuilder strings.Builder
	for _, content := range contents {
		switch content.Type {
		case "text":
			textBuilder.WriteString(content.Text)
		case "thinking":
			thinkingBuilder.WriteString(content.Thinking)
		default:
			if content.Thinking != "" {
				thinkingBuilder.WriteString(content.Thinking)
			} else {
				textBuilder.WriteString(content.Text)
			}
		}
	}
	return textBuilder.String(), thinkingBuilder.String()
}

func geminiTextAndThinking(parts []gemini.Part) (string, string) {
	var textBuilder, thinkingBuilder strings.Builder
	for _, part := range parts {
		if part.Thought {
			thinkingBuilder.WriteString(part.Text)
		} else {
			textBuilder.WriteString(part.Text)
		}
	}
	return textBuilder.String(), thinkingBuilder.String()
}

func geminiCompletionTokens(usage gemini.UsageMetadata) int {
	return usage.CandidatesTokenCount + usage.ThoughtsTokenCount
}

func cohereTextAndThinking(contents []cohere.ChatContent) (string, string) {
	var textBuilder, thinkingBuilder strings.Builder
	for _, content := range contents {
		switch content.Type {
		case "text":
			textBuilder.WriteString(content.Text)
		case "thinking":
			thinkingBuilder.WriteString(content.Thinking)
		default:
			if content.Thinking != "" {
				thinkingBuilder.WriteString(content.Thinking)
			} else {
				textBuilder.WriteString(content.Text)
			}
		}
	}
	return textBuilder.String(), thinkingBuilder.String()
}

func buildCohereChatCompletionRequest(meta Meta, req ZhimaChatCompletionRequest) (cohere.ChatCompletionRequest, bool, error) {
	request := cohere.ChatCompletionRequest{
		MaxTokens:   req.MaxToken,
		Temperature: req.Temperature,
	}
	if !meta.ChoosableThinking {
		n := len(req.Messages)
		for _, message := range req.Messages[:n-1] {
			switch message.Role {
			case "system":
				request.ChatHistory = append(request.ChatHistory, cohere.ChatHistory{Role: "SYSTEM", Message: message.Content})
			case "user":
				request.ChatHistory = append(request.ChatHistory, cohere.ChatHistory{Role: "USER", Message: message.Content})
			case "assistant":
				request.ChatHistory = append(request.ChatHistory, cohere.ChatHistory{Role: "CHATBOT", Message: message.Content})
			}
		}
		request.Message = req.Messages[n-1].Content
		return request, false, nil
	}

	if strings.TrimSpace(meta.Model) == "" {
		return cohere.ChatCompletionRequest{}, true, errors.New("model is required for cohere v2 chat")
	}
	request.Model = meta.Model
	request.Messages = make([]cohere.ChatMessage, 0, len(req.Messages))
	for _, message := range req.Messages {
		request.Messages = append(request.Messages, cohere.ChatMessage{Role: message.Role, Content: message.Content})
	}
	applyThinking(meta, &request)
	return request, true, nil
}
