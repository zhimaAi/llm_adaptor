// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package transport

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
)

type StreamTerminal struct {
	done      atomic.Bool
	closeOnce sync.Once
	closeMu   sync.Mutex
	closeErr  error
	cancel    context.CancelFunc
	closeFunc func() error
}

func NewStreamTerminal(cancel context.CancelFunc, closeFunc func() error) *StreamTerminal {
	return &StreamTerminal{cancel: cancel, closeFunc: closeFunc}
}

func (s *StreamTerminal) IsDone() bool {
	return s != nil && s.done.Load()
}

func (s *StreamTerminal) Finish() {
	_ = s.Close()
}

func (s *StreamTerminal) Fail(ctx context.Context, err error) error {
	if s.IsDone() {
		return io.EOF
	}
	if ctx != nil && ctx.Err() != nil {
		err = ctx.Err()
	}
	_ = s.Close()
	return err
}

func (s *StreamTerminal) Close() error {
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
