// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package shared

import "github.com/zhimaAi/llm_adaptor/v2/chat"

func ReasoningEnabled(effort chat.ReasoningEffort) bool {
	return effort != "" && effort != chat.ReasoningEffortNone
}

func ApplyReasoningBoolean(field string) func(string, chat.ReasoningEffort, map[string]any) {
	return func(_ string, effort chat.ReasoningEffort, body map[string]any) {
		if effort == "" {
			return
		}
		delete(body, "reasoning_effort")
		body[field] = ReasoningEnabled(effort)
	}
}

func ApplyThinkingType(_ string, effort chat.ReasoningEffort, body map[string]any) {
	if effort == "" {
		return
	}
	thinkingType := "disabled"
	if ReasoningEnabled(effort) {
		thinkingType = "enabled"
	}
	delete(body, "reasoning_effort")
	body["thinking"] = map[string]any{"type": thinkingType}
}
