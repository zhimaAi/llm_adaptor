// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/image"
)

func TestOpenAIChatStreamParseErrorTerminates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	body := io.NopCloser(strings.NewReader("data: {invalid}\n"))
	stream := newOpenAIChatStream(ctx, body, cancel, ProviderOpenAI, "hint")
	if _, err := stream.Recv(); err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("expected parse error, got %v", err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after parse error, got %v", err)
	}
}

func TestOpenAIImageStreamParseErrorTerminates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	body := io.NopCloser(strings.NewReader("data: {invalid}\n"))
	stream := newOpenAIImageStream(ctx, body, cancel, ClientConfig{}, ProviderOpenAI, "hint", image.GenerateRequest{})
	if _, err := stream.Recv(); err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("expected parse error, got %v", err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after parse error, got %v", err)
	}
}

func TestOpenAIChatStreamConcurrentCloseUnblocksRecv(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader, writer := io.Pipe()
	defer writer.Close()
	stream := newOpenAIChatStream(ctx, reader, cancel, ProviderOpenAI, "hint")
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(started)
		_, err := stream.Recv()
		result <- err
	}()
	<-started
	if err := stream.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("expected EOF after Close, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Recv remained blocked after Close")
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

type errorChatStream struct {
	err       error
	closeOnce sync.Once
}

func (s *errorChatStream) Recv() (*chat.StreamChunk, error) { return nil, s.err }
func (s *errorChatStream) Close() error {
	s.closeOnce.Do(func() {})
	return nil
}

func TestOpenRouterImageStreamErrorTerminates(t *testing.T) {
	wantErr := errors.New("stream failed")
	inner := &errorChatStream{err: wantErr}
	stream := &openRouterImageStream{
		ctx: context.Background(), stream: inner, terminal: newStreamTerminal(nil, inner.Close),
	}
	if _, err := stream.Recv(); !errors.Is(err, wantErr) {
		t.Fatalf("expected original error, got %v", err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after error, got %v", err)
	}
}
