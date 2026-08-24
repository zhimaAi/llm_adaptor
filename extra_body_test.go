// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
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
	body, err := mergeExtraBody(openAIEmbeddingWireRequest{Model: embeddingRequest.Model, Input: input}, embeddingRequest.ExtraBody)
	if err != nil || body["custom"] != "value" {
		t.Fatalf("embedding ExtraBody was not injected: body=%#v err=%v", body, err)
	}
	body, err = mergeExtraBody(openAIEmbeddingWireRequest{Model: "model", Input: "x"}, map[string]any{"model": "override"})
	if err != nil || body["model"] != "override" {
		t.Fatalf("embedding override failed: body=%#v err=%v", body, err)
	}
	body, err = mergeExtraBody(openAIEmbeddingWireRequest{Model: "model", Input: "x"}, map[string]any{"model": nil})
	if err != nil {
		t.Fatal(err)
	}
	if value, exists := body["model"]; !exists || value != nil {
		t.Fatalf("explicit nil was not preserved as a JSON null candidate: %#v", body)
	}

	n := 2
	generate := &image.GenerateRequest{Model: "model", Prompt: "draw", N: &n, ExtraBody: map[string]any{"watermark": true}}
	before := map[string]any{"watermark": true}
	body, err = buildOpenAIImageGenerateRequest(ProviderOpenAI, generate, false)
	if err != nil || body["watermark"] != true || !reflect.DeepEqual(generate.ExtraBody, before) {
		t.Fatalf("image ExtraBody injection or immutability failed: body=%#v request=%#v err=%v", body, generate, err)
	}
	generate.ExtraBody = map[string]any{"stream": true}
	body, err = buildOpenAIImageGenerateRequest(ProviderOpenAI, generate, false)
	if err != nil || body["stream"] != true {
		t.Fatalf("image stream override failed: body=%#v err=%v", body, err)
	}
	edit := &image.EditRequest{
		Model: "model", Prompt: "edit", Images: []image.Input{{ImageURL: "https://example.com/input.png"}},
		ExtraBody: map[string]any{"prompt": "caller prompt", "stream": true},
	}
	body, err = buildOpenAIImageEditRequest(ProviderOpenAI, edit, false)
	if err != nil || body["prompt"] != "caller prompt" || body["stream"] != true {
		t.Fatalf("image edit override failed: body=%#v err=%v", body, err)
	}

	speechRequest := &speech.CreateRequest{Model: "speech-2.8-hd", Text: "hello", ExtraBody: map[string]any{"custom": 1}}
	body, err = buildMiniMaxSpeechRequest(speechRequest, false, nil)
	if err != nil || body["custom"] != float64(1) && body["custom"] != 1 {
		t.Fatalf("speech ExtraBody was not injected: body=%#v err=%v", body, err)
	}
	speechRequest.ExtraBody = map[string]any{"stream": true}
	body, err = buildMiniMaxSpeechRequest(speechRequest, false, nil)
	if err != nil || body["stream"] != true {
		t.Fatalf("speech stream override failed: body=%#v err=%v", body, err)
	}
}
