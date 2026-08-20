package llm

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
)

func TestOpenAICompatibleChatUsesSelectedCredentialAndPreservesFields(t *testing.T) {
	var authorizations []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		authorizations = append(authorizations, request.Header.Get(headerAuthorization))
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["provider_option"] != "kept" {
			t.Fatalf("extra body field missing: %#v", body)
		}
		_, _ = io.WriteString(writer, `{"id":"id","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"<think>r</think>a"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3},"provider_field":true}`)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{Provider: ProviderOpenCompatible, BaseURL: server.URL, Credentials: CredentialConfig{APIKeys: "key1,key2"}})
	if err != nil {
		t.Fatal(err)
	}
	values := []*big.Int{big.NewInt(0), big.NewInt(1)}
	client.credentials.randomInt = func(max *big.Int) (*big.Int, error) {
		value := values[0]
		values = values[1:]
		return value, nil
	}
	request := &chat.CreateRequest{
		Model:     "model",
		Messages:  []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent("hello")}},
		ExtraBody: map[string]any{"provider_option": "kept"},
	}
	for range 2 {
		response, err := client.Chat.Create(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		message := response.Choices[0].Message
		if message.Content.Text == nil || *message.Content.Text != "a" || message.ReasoningContent != "r" {
			t.Fatalf("unexpected normalized response: %#v", response)
		}
		if _, exists := response.ExtraFields["provider_field"]; !exists {
			t.Fatalf("missing provider extension: %#v", response.ExtraFields)
		}
	}
	if len(authorizations) != 2 || authorizations[0] != bearerPrefix+"key1" || authorizations[1] != bearerPrefix+"key2" {
		t.Fatalf("unexpected authorizations: %#v", authorizations)
	}
}
