// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

type streamTerminal struct {
	done      atomic.Bool
	closeOnce sync.Once
	closeMu   sync.Mutex
	closeErr  error
	cancel    context.CancelFunc
	closeFunc func() error
}

func newStreamTerminal(cancel context.CancelFunc, closeFunc func() error) *streamTerminal {
	return &streamTerminal{cancel: cancel, closeFunc: closeFunc}
}

func (s *streamTerminal) isDone() bool {
	return s != nil && s.done.Load()
}

func (s *streamTerminal) finish() {
	_ = s.close()
}

func (s *streamTerminal) fail(ctx context.Context, err error) error {
	if s.isDone() {
		return io.EOF
	}
	if ctx != nil && ctx.Err() != nil {
		err = ctx.Err()
	}
	_ = s.close()
	return err
}

func (s *streamTerminal) close() error {
	if s == nil {
		return nil
	}
	s.done.Store(true)
	s.closeOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
		var err error
		if s.closeFunc != nil {
			err = s.closeFunc()
		}
		s.closeMu.Lock()
		s.closeErr = err
		s.closeMu.Unlock()
	})
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	return s.closeErr
}
