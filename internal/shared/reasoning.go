// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package shared

import "github.com/zhimaAi/llm_adaptor/v2/chat"

// SupportedReasoningEffort preserves accepted values (including server-side
// aliases) and custom values. Only known, rejected levels are lowered.
func SupportedReasoningEffort(effort chat.ReasoningEffort, accepted ...chat.ReasoningEffort) chat.ReasoningEffort {
	levels := []chat.ReasoningEffort{chat.ReasoningEffortNone, chat.ReasoningEffortMinimal, chat.ReasoningEffortLow, chat.ReasoningEffortMedium, chat.ReasoningEffortHigh, chat.ReasoningEffortXHigh, chat.ReasoningEffortMax}
	index := -1
	for i, level := range levels {
		if level == effort {
			index = i
		}
	}
	if index < 0 || len(accepted) == 0 {
		return effort
	}
	for _, level := range accepted {
		if effort == level {
			return effort
		}
	}
	for i := index - 1; i >= 0; i-- {
		if levels[i] == chat.ReasoningEffortNone {
			continue
		}
		for _, level := range accepted {
			if levels[i] == level {
				return level
			}
		}
	}
	for _, candidate := range levels[1:] {
		for _, level := range accepted {
			if candidate == level {
				return level
			}
		}
	}
	return effort
}

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
