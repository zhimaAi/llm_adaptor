// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package basics

type MessageOther struct {
	Name             string         `json:"name,omitempty"`
	ToolCalls        ToolCalls      `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
	ToolName         string         `json:"tool_name,omitempty"`
	ResponseMeta     *ResponseMeta  `json:"response_meta,omitempty"`
	ReasoningContent string         `json:"reasoning_content"`
	Extra            map[string]any `json:"extra,omitempty"`
}

type RoleType = string

const (
	Assistant RoleType = "assistant"
	User      RoleType = "user"
	System    RoleType = "system"
	Tool      RoleType = "tool"
)

type ResponseMeta struct {
	FinishReason string      `json:"finish_reason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
	LogProbs     *LogProbs   `json:"logprobs,omitempty"`
}

type TokenUsage struct {
	PromptTokens            int                     `json:"prompt_tokens"`
	PromptTokenDetails      PromptTokenDetails      `json:"prompt_token_details"`
	CompletionTokens        int                     `json:"completion_tokens"`
	TotalTokens             int                     `json:"total_tokens"`
	CompletionTokensDetails CompletionTokensDetails `json:"completion_token_details"`
}

type PromptTokenDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type LogProbs struct {
	Content []LogProb `json:"content"`
}

type LogProb struct {
	Token       string       `json:"token"`
	LogProb     float64      `json:"logprob"`
	Bytes       []int64      `json:"bytes,omitempty"`
	TopLogProbs []TopLogProb `json:"top_logprobs"`
}

type TopLogProb struct {
	Token   string  `json:"token"`
	LogProb float64 `json:"logprob"`
	Bytes   []int64 `json:"bytes,omitempty"`
}
