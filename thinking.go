// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"errors"
	"io"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
)

const (
	thinkStartTag = "<think>"
	thinkEndTag   = "</think>"
)

type thinkTagExtractor struct {
	inThink bool
	pending string
}

func (e *thinkTagExtractor) process(content string) (text, reasoning string) {
	e.pending += content
	for {
		tag := thinkStartTag
		destination := &text
		if e.inThink {
			tag = thinkEndTag
			destination = &reasoning
		}
		index := strings.Index(e.pending, tag)
		if index >= 0 {
			*destination += e.pending[:index]
			e.pending = e.pending[index+len(tag):]
			e.inThink = !e.inThink
			continue
		}
		emitLength := len(e.pending) - longestSuffixPrefixLength(e.pending, tag)
		*destination += e.pending[:emitLength]
		e.pending = e.pending[emitLength:]
		return text, reasoning
	}
}

func (e *thinkTagExtractor) flush() (text, reasoning string) {
	if e.inThink {
		reasoning = e.pending
	} else {
		text = e.pending
	}
	e.pending = ""
	e.inThink = false
	return text, reasoning
}

func longestSuffixPrefixLength(value, prefix string) int {
	maximum := len(prefix) - 1
	if len(value) < maximum {
		maximum = len(value)
	}
	for length := maximum; length > 0; length-- {
		if strings.HasSuffix(value, prefix[:length]) {
			return length
		}
	}
	return 0
}

func normalizeThinkTaggedResponse(response *chat.CreateResponse) {
	for index := range response.Choices {
		message := &response.Choices[index].Message
		if message.ReasoningContent != "" || message.Content.Text == nil || !strings.Contains(*message.Content.Text, thinkStartTag) {
			continue
		}
		var extractor thinkTagExtractor
		text, reasoning := extractor.process(*message.Content.Text)
		flushText, flushReasoning := extractor.flush()
		text += flushText
		reasoning += flushReasoning
		message.Content = chat.TextContent(text)
		message.ReasoningContent = reasoning
	}
}

type thinkTagStream struct {
	stream     chat.Stream
	extractors map[int]*thinkTagExtractor
	pending    []*chat.StreamChunk
	finished   bool
}

func newThinkTagStream(stream chat.Stream) chat.Stream {
	return &thinkTagStream{stream: stream, extractors: make(map[int]*thinkTagExtractor)}
}

func (s *thinkTagStream) Recv() (*chat.StreamChunk, error) {
	if len(s.pending) > 0 {
		chunk := s.pending[0]
		s.pending = s.pending[1:]
		return chunk, nil
	}
	if s.finished {
		return nil, io.EOF
	}
	chunk, err := s.stream.Recv()
	if err != nil {
		if !errors.Is(err, io.EOF) {
			s.finished = true
			s.pending = nil
			s.extractors = nil
			return nil, err
		}
		s.finished = true
		for choiceIndex, extractor := range s.extractors {
			text, reasoning := extractor.flush()
			if text == "" && reasoning == "" {
				continue
			}
			choice := chat.ChunkChoice{Index: choiceIndex}
			choice.Delta.Content = chat.TextContent(text)
			choice.Delta.ReasoningContent = reasoning
			s.pending = append(s.pending, &chat.StreamChunk{Choices: []chat.ChunkChoice{choice}})
		}
		if len(s.pending) > 0 {
			return s.Recv()
		}
		return nil, io.EOF
	}
	for index := range chunk.Choices {
		choice := &chunk.Choices[index]
		if choice.Delta.ReasoningContent != "" || choice.Delta.Content.Text == nil {
			continue
		}
		extractor := s.extractors[choice.Index]
		if extractor == nil {
			extractor = &thinkTagExtractor{}
			s.extractors[choice.Index] = extractor
		}
		text, reasoning := extractor.process(*choice.Delta.Content.Text)
		choice.Delta.Content = chat.TextContent(text)
		choice.Delta.ReasoningContent = reasoning
	}
	return chunk, nil
}

func (s *thinkTagStream) Close() error { return s.stream.Close() }

var _ chat.Stream = (*thinkTagStream)(nil)
