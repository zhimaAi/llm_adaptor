// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"reflect"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
)

func TestPublicRequestContracts(t *testing.T) {
	tests := []struct {
		name   string
		typeOf reflect.Type
		want   []string
	}{
		{
			name: "chat", typeOf: reflect.TypeOf(chat.CreateRequest{}),
			want: []string{"Model", "Messages", "FrequencyPenalty", "MaxTokens", "MaxCompletionTokens", "N", "ParallelToolCalls", "PresencePenalty", "ReasoningEffort", "ResponseFormat", "Seed", "Stop", "Temperature", "ToolChoice", "Tools", "TopP", "User", "ExtraBody"},
		},
		{
			name: "embedding", typeOf: reflect.TypeOf(embedding.CreateRequest{}),
			want: []string{"Model", "Input", "EncodingFormat", "Dimensions", "User", "ExtraBody"},
		},
		{
			name: "image generate", typeOf: reflect.TypeOf(image.GenerateRequest{}),
			want: []string{"Model", "Prompt", "N", "Quality", "ResponseFormat", "Size", "User", "OutputFormat", "ExtraBody"},
		},
		{
			name: "image edit", typeOf: reflect.TypeOf(image.EditRequest{}),
			want: []string{"Model", "Images", "Mask", "Prompt", "N", "Quality", "ResponseFormat", "Size", "User", "OutputFormat", "ExtraBody"},
		},
		{
			name: "rerank", typeOf: reflect.TypeOf(rerank.CreateRequest{}),
			want: []string{"Model", "Query", "Documents", "TopN", "ExtraBody"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := make([]string, test.typeOf.NumField())
			for index := range got {
				got[index] = test.typeOf.Field(index).Name
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("fields = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestTemplateResponsesDoNotExposeAdapterMetadata(t *testing.T) {
	for _, value := range []any{
		chat.CreateResponse{}, chat.StreamChunk{}, embedding.CreateResponse{}, image.GenerateResponse{},
		rerank.CreateResponse{},
	} {
		typeOf := reflect.TypeOf(value)
		for _, forbidden := range []string{"RawResponse", "ExtraFields", "CredentialHint", "Meta"} {
			if forbidden == "Meta" && typeOf == reflect.TypeOf(rerank.CreateResponse{}) {
				continue
			}
			if _, exists := typeOf.FieldByName(forbidden); exists {
				t.Fatalf("%s unexpectedly exposes %s", typeOf, forbidden)
			}
		}
	}
}
