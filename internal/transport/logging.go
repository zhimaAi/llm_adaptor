// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package transport

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// HTTPLog describes one HTTP round trip, before protocol response decoding.
type HTTPLog struct {
	RequestTime     time.Time
	URL             string
	Method          string
	RequestHeaders  http.Header
	RequestBody     []byte
	ResponseTime    time.Time
	EndTime         time.Time
	Duration        time.Duration
	StatusCode      int
	ResponseHeaders http.Header
	ResponseBody    []byte
	ResponseBodyEOF bool
	Error           error
}

type httpLogger struct {
	callback func(context.Context, HTTPLog)
}

var registeredHTTPLogger atomic.Pointer[httpLogger]

func RegisterHTTPLogger(callback func(context.Context, HTTPLog)) {
	if callback == nil {
		registeredHTTPLogger.Store(nil)
		return
	}
	registeredHTTPLogger.Store(&httpLogger{callback: callback})
}

// WithHTTPLogging copies the client without changing its transport or policies.
func WithHTTPLogging(client *http.Client) *http.Client {
	result := *client
	base := client.Transport
	if existing, ok := base.(*loggingTransport); ok {
		base = existing.base
	}
	if base == nil {
		base = http.DefaultTransport
	}
	result.Transport = &loggingTransport{base: base}
	return &result
}

type loggingTransport struct {
	base http.RoundTripper
	// Non-replayable inputs use a cloned request to collect their body. Legacy
	// cancellation must reach the same request pointer passed to the transport.
	requestAliases sync.Map
}

type requestCanceler interface{ CancelRequest(*http.Request) }

func (t *loggingTransport) CancelRequest(request *http.Request) {
	if canceler, ok := t.base.(requestCanceler); ok {
		if alias, found := t.requestAliases.Load(request); found {
			request = alias.(*http.Request)
		}
		canceler.CancelRequest(request)
	}
}

func (t *loggingTransport) CloseIdleConnections() {
	if closer, ok := t.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

func (t *loggingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	logger := registeredHTTPLogger.Load()
	if logger == nil {
		return t.base.RoundTrip(request)
	}
	record := &httpLogRecord{ctx: request.Context(), logger: logger, entry: HTTPLog{
		RequestTime: time.Now(), URL: request.URL.String(), Method: request.Method,
		RequestHeaders: request.Header.Clone(),
	}}
	// All SDK-generated bodies are replayable. Reading the copy also captures the
	// full input when the server rejects a request before consuming its body.
	if request.Body != nil && request.Body != http.NoBody {
		var captured bool
		if request.GetBody != nil {
			if body, err := request.GetBody(); err == nil {
				payload, readErr := io.ReadAll(body)
				_ = body.Close()
				if readErr == nil {
					record.entry.RequestBody = payload
					captured = true
				}
			}
		}
		if !captured {
			clone := request.Clone(request.Context())
			clone.Body = &requestLogBody{ReadCloser: request.Body, record: record}
			if _, ok := t.base.(requestCanceler); ok {
				original := request
				t.requestAliases.Store(original, clone)
				record.onFinish = func() { t.requestAliases.Delete(original) }
			}
			request = clone
		}
	}
	// Capturing a replayable body can outlast the client's deadline. Do not
	// start a legacy transport after its timeout cancellation has already fired.
	if err := request.Context().Err(); err != nil {
		if request.Body != nil {
			_ = request.Body.Close()
		}
		record.finish(err, false)
		return nil, err
	}
	response, err := t.base.RoundTrip(request)
	if response != nil {
		record.entry.ResponseTime = time.Now()
		record.entry.StatusCode = response.StatusCode
		record.entry.ResponseHeaders = response.Header.Clone()
	}
	if err != nil {
		record.finish(err, false)
		return response, err
	}
	if response == nil || response.Body == nil || response.Body == http.NoBody {
		record.finish(nil, true)
		return response, nil
	}
	response.Body = &responseLogBody{ReadCloser: response.Body, record: record}
	return response, nil
}

type httpLogRecord struct {
	ctx      context.Context
	logger   *httpLogger
	mu       sync.Mutex
	once     sync.Once
	entry    HTTPLog
	onFinish func()
}

func (r *httpLogRecord) finish(err error, eof bool) {
	var snapshot *HTTPLog
	r.once.Do(func() {
		if r.onFinish != nil {
			r.onFinish()
		}
		r.mu.Lock()
		r.entry.EndTime = time.Now()
		r.entry.Duration = r.entry.EndTime.Sub(r.entry.RequestTime)
		r.entry.ResponseBodyEOF = eof
		r.entry.Error = err
		entry := r.entry
		entry.RequestBody = bytes.Clone(entry.RequestBody)
		entry.ResponseBody = bytes.Clone(entry.ResponseBody)
		r.mu.Unlock()
		snapshot = &entry
	})
	if snapshot != nil {
		// Run outside sync.Once so completion can be re-entered by a callback.
		func() {
			defer func() { _ = recover() }()
			r.logger.callback(r.ctx, *snapshot)
		}()
	}
}

type requestLogBody struct {
	io.ReadCloser
	record *httpLogRecord
}

func (b *requestLogBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.record.mu.Lock()
	b.record.entry.RequestBody = append(b.record.entry.RequestBody, p[:n]...)
	b.record.mu.Unlock()
	return n, err
}

type responseLogBody struct {
	io.ReadCloser
	record *httpLogRecord
}

func (b *responseLogBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.record.mu.Lock()
	b.record.entry.ResponseBody = append(b.record.entry.ResponseBody, p[:n]...)
	b.record.mu.Unlock()
	if err != nil {
		if err == io.EOF {
			b.record.finish(nil, true)
		} else {
			b.record.finish(err, false)
		}
	}
	return n, err
}

func (b *responseLogBody) Close() error {
	err := b.ReadCloser.Close()
	b.record.finish(err, false)
	return err
}
