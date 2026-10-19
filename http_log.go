// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"

	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

// HTTPLog contains the original HTTP input and output of one round trip.
// Headers and bodies are not redacted or truncated. ResponseBodyEOF reports
// whether the consumer read to EOF; Close without EOF records the bytes read.
type HTTPLog = transport.HTTPLog

// RegisterHTTPLogger replaces the process-wide callback for all SDK clients.
// A nil callback disables logging. Each request snapshots the current callback
// and calls it synchronously once on EOF, read failure, Close, or network error.
// Callbacks may run concurrently and must arrange their own storage and locking.
// Callback panics are isolated from model calls. Close streams to finish logs.
func RegisterHTTPLogger(callback func(context.Context, HTTPLog)) {
	transport.RegisterHTTPLogger(callback)
}
