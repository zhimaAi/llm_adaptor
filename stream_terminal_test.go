// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
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
	stream := newOpenAIImageStream(ctx, body, cancel, ClientConfig{}, ProviderOpenAI, "hint", imageRequestOptions{})
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

func TestOpenRouterImageStreamErrorTerminates(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("data: {invalid}\n"))
	stream := &openRouterImageStream{
		ctx: context.Background(), scanner: scanner, terminal: newStreamTerminal(nil, nil),
	}
	if _, err := stream.Recv(); err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("expected parse error, got %v", err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF after error, got %v", err)
	}
}
