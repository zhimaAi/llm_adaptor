// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"io"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
)

func TestNormalizeThinkTaggedResponse(t *testing.T) {
	response := &chat.CreateResponse{Choices: []chat.Choice{{
		Message: chat.Message{Content: chat.TextContent("<think>reason</think>answer")},
	}}}
	normalizeThinkTaggedResponse(response)
	message := response.Choices[0].Message
	if message.Content.Text == nil || *message.Content.Text != "answer" || message.ReasoningContent != "reason" {
		t.Fatalf("unexpected normalized message: %#v", message)
	}
}

func TestThinkTagStreamAcrossChunks(t *testing.T) {
	stream := newThinkTagStream(&sliceChatStream{chunks: []*chat.StreamChunk{
		{Choices: []chat.ChunkChoice{{Index: 0, Delta: chat.Message{Content: chat.TextContent("<thi")}}}},
		{Choices: []chat.ChunkChoice{{Index: 0, Delta: chat.Message{Content: chat.TextContent("nk>reason</th")}}}},
		{Choices: []chat.ChunkChoice{{Index: 0, Delta: chat.Message{Content: chat.TextContent("ink>answer")}}}},
	}})
	accumulator := chat.NewAccumulator()
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := accumulator.Add(chunk); err != nil {
			t.Fatal(err)
		}
	}
	message := accumulator.Response().Choices[0].Message
	if message.Content.Text == nil || *message.Content.Text != "answer" || message.ReasoningContent != "reason" {
		t.Fatalf("unexpected accumulated message: %#v", message)
	}
}

type sliceChatStream struct {
	chunks []*chat.StreamChunk
	index  int
}

func (s *sliceChatStream) Recv() (*chat.StreamChunk, error) {
	if s.index >= len(s.chunks) {
		return nil, io.EOF
	}
	chunk := s.chunks[s.index]
	s.index++
	return chunk, nil
}

func (s *sliceChatStream) Close() error { return nil }
