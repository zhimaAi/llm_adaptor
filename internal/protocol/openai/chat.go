// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
)

func (p *Provider) CreateChat(ctx context.Context, selected provider.Credential, request *chat.CreateRequest) (*chat.CreateResponse, error) {
	if request == nil || strings.TrimSpace(request.Model) == "" || len(request.Messages) == 0 {
		return nil, fmt.Errorf("%w: model and messages are required", provider.ErrInvalidRequest)
	}
	body, err := BuildChatRequest(p.spec, request, false, nil)
	if err != nil {
		return nil, err
	}
	raw, err := p.DoJSON(ctx, selected, p.spec.ChatPath, body)
	if err != nil {
		return nil, err
	}
	result, err := DecodeChatResponse(raw)
	if err != nil {
		return nil, transport.NewResponseDecodeError(p.spec.Info.ID, selected.Hint, raw, err)
	}
	if p.spec.TransformChatResponse != nil {
		if err := p.spec.TransformChatResponse(raw, result); err != nil {
			return nil, transport.NewResponseDecodeError(p.spec.Info.ID, selected.Hint, raw, err)
		}
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("%w: response contains no choices", provider.ErrInvalidRequest)
	}
	shared.NormalizeThinkTaggedResponse(result)
	return result, nil
}

func (p *Provider) StreamChat(ctx context.Context, selected provider.Credential, request *chat.StreamRequest) (chat.Stream, error) {
	if request == nil || strings.TrimSpace(request.Model) == "" || len(request.Messages) == 0 {
		return nil, fmt.Errorf("%w: model and messages are required", provider.ErrInvalidRequest)
	}
	body, err := BuildChatRequest(p.spec, &request.CreateRequest, true, request.StreamOptions)
	if err != nil {
		return nil, err
	}
	streamContext, cancel := context.WithCancel(shared.NormalizeContext(ctx))
	response, err := p.DoStream(streamContext, selected, p.spec.ChatPath, body)
	if err != nil {
		cancel()
		return nil, err
	}
	stream := newChatStream(streamContext, response.Body, cancel, p.spec.Info.ID, selected.Hint, p.spec.TransformStreamResponse)
	return shared.NewThinkTagStream(stream), nil
}

type chatStream struct {
	ctx       context.Context
	scanner   *bufio.Scanner
	terminal  *transport.StreamTerminal
	provider  provider.ID
	hint      string
	transform ChatStreamResponseTransform
}

func newChatStream(ctx context.Context, body io.ReadCloser, cancel context.CancelFunc, providerID provider.ID, hint string, transform ChatStreamResponseTransform) *chatStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, streamInitialBuffer), streamMaximumBuffer)
	return &chatStream{ctx: ctx, scanner: scanner, terminal: transport.NewStreamTerminal(cancel, body.Close), provider: providerID, hint: hint, transform: transform}
}

func (s *chatStream) Recv() (*chat.StreamChunk, error) {
	if s.terminal.IsDone() {
		return nil, io.EOF
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 || bytes.HasPrefix(line, []byte(":")) {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if bytes.Equal(line, []byte("[DONE]")) {
			s.terminal.Finish()
			return nil, io.EOF
		}
		if err := transport.DecodeStreamAPIError(s.provider, s.hint, line); err != nil {
			return nil, s.terminal.Fail(s.ctx, err)
		}
		chunk, err := DecodeChatStreamResponse(line)
		if err != nil {
			return nil, s.terminal.Fail(s.ctx, transport.NewResponseDecodeError(s.provider, s.hint, line, err))
		}
		if s.transform != nil {
			if err := s.transform(line, chunk); err != nil {
				return nil, s.terminal.Fail(s.ctx, transport.NewResponseDecodeError(s.provider, s.hint, line, err))
			}
		}
		return chunk, nil
	}
	if err := s.scanner.Err(); err != nil {
		return nil, s.terminal.Fail(s.ctx, err)
	}
	if s.terminal.IsDone() {
		return nil, io.EOF
	}
	if s.ctx != nil && s.ctx.Err() != nil {
		return nil, s.terminal.Fail(s.ctx, s.ctx.Err())
	}
	s.terminal.Finish()
	return nil, io.EOF
}

func (s *chatStream) Close() error { return s.terminal.Close() }

var _ provider.Chat = (*Provider)(nil)
var _ chat.Stream = (*chatStream)(nil)
