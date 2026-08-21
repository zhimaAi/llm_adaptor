// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

func TestCommonRequestExtraBodyRules(t *testing.T) {
	text := "hello"
	embeddingRequest := &embedding.CreateRequest{
		Model: "model", Input: embedding.Input{Text: &text}, ExtraBody: map[string]any{"custom": "value"},
	}
	input, err := openAIEmbeddingInput(embeddingRequest.Input)
	if err != nil {
		t.Fatal(err)
	}
	body, err := mergeExtraBody(openAIEmbeddingWireRequest{Model: embeddingRequest.Model, Input: input}, embeddingRequest.ExtraBody, embeddingReservedRequestKeys)
	if err != nil || body["custom"] != "value" {
		t.Fatalf("embedding ExtraBody was not injected: body=%#v err=%v", body, err)
	}
	if _, err := mergeExtraBody(openAIEmbeddingWireRequest{Model: "model", Input: "x"}, map[string]any{"model": "override"}, embeddingReservedRequestKeys); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("embedding conflict error = %v", err)
	}

	n := 2
	generate := &image.GenerateRequest{Model: "model", Prompt: "draw", N: &n, ExtraBody: map[string]any{"watermark": true}}
	before := map[string]any{"watermark": true}
	body, err = buildOpenAIImageGenerateRequest(generate, false)
	if err != nil || body["watermark"] != true || !reflect.DeepEqual(generate.ExtraBody, before) {
		t.Fatalf("image ExtraBody injection or immutability failed: body=%#v request=%#v err=%v", body, generate, err)
	}
	generate.ExtraBody = map[string]any{"stream": true}
	if _, err := buildOpenAIImageGenerateRequest(generate, false); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("image stream conflict error = %v", err)
	}

	speechRequest := &speech.CreateRequest{Model: "speech-2.8-hd", Text: "hello", ExtraBody: map[string]any{"custom": 1}}
	body, err = buildMiniMaxSpeechRequest(speechRequest, false, nil)
	if err != nil || body["custom"] != float64(1) && body["custom"] != 1 {
		t.Fatalf("speech ExtraBody was not injected: body=%#v err=%v", body, err)
	}
	speechRequest.ExtraBody = map[string]any{"stream": true}
	if _, err := buildMiniMaxSpeechRequest(speechRequest, false, nil); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("speech stream conflict error = %v", err)
	}
}
