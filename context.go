// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"

	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
)

// NormalizeContext returns context.Background when ctx is nil.
func NormalizeContext(ctx context.Context) context.Context {
	return shared.NormalizeContext(ctx)
}
