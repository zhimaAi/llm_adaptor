// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package shared

import (
	"regexp"
	"strings"
	"sync"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
)

var modelPatterns sync.Map // map[string]*regexp.Regexp

// IsModel matches IDs or '*' wildcard patterns case-insensitively. Matching is
// anchored to the entire ID; all characters other than '*' are literal.
func IsModel(model string, ids ...string) bool {
	model = strings.TrimSpace(model)
	for _, id := range ids {
		if !strings.Contains(id, "*") {
			if strings.EqualFold(model, id) {
				return true
			}
			continue
		}
		pattern, cached := modelPatterns.Load(id)
		if !cached {
			expression := `(?is)\A` + strings.ReplaceAll(regexp.QuoteMeta(id), `\*`, `.*`) + `\z`
			compiled, err := regexp.Compile(expression)
			if err != nil {
				continue
			}
			pattern, _ = modelPatterns.LoadOrStore(id, compiled)
		}
		if pattern.(*regexp.Regexp).MatchString(model) {
			return true
		}
	}
	return false
}

// ApplyReasoningEffort forwards even unsupported and future values unchanged.
func ApplyReasoningEffort(_ string, effort chat.ReasoningEffort, body map[string]any) {
	if effort != "" {
		body["reasoning_effort"] = string(effort)
	}
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
	applyThinkingType(effort, body, "enabled")
}

// ApplyAdaptiveThinkingType maps the switch to adaptive/disabled. It does not
// send an effort level; callers on a level path must forward that separately.
func ApplyAdaptiveThinkingType(_ string, effort chat.ReasoningEffort, body map[string]any) {
	applyThinkingType(effort, body, "adaptive")
}

func applyThinkingType(effort chat.ReasoningEffort, body map[string]any, enabledType string) {
	if effort == "" {
		return
	}
	thinkingType := "disabled"
	if ReasoningEnabled(effort) {
		thinkingType = enabledType
	}
	delete(body, "reasoning_effort")
	body["thinking"] = map[string]any{"type": thinkingType}
}
