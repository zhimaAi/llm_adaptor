// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

type contextCaptureProvider struct {
	contexts []context.Context
}

func (p *contextCaptureProvider) capture(ctx context.Context) {
	p.contexts = append(p.contexts, ctx)
}

func (p *contextCaptureProvider) info() ProviderInfo {
	return ProviderInfo{ID: ProviderOpenAI, Capabilities: []Capability{
		CapabilityChat, CapabilityEmbedding, CapabilityImage, CapabilityRerank, CapabilitySpeech,
	}}
}

func (p *contextCaptureProvider) createChat(ctx context.Context, _ credential, _ *chat.CreateRequest) (*chat.CreateResponse, error) {
	p.capture(ctx)
	return &chat.CreateResponse{}, nil
}

func (p *contextCaptureProvider) streamChat(ctx context.Context, _ credential, _ *chat.StreamRequest) (chat.Stream, error) {
	p.capture(ctx)
	return nil, nil
}

func (p *contextCaptureProvider) createEmbedding(ctx context.Context, _ credential, _ *embedding.CreateRequest) (*embedding.CreateResponse, error) {
	p.capture(ctx)
	return &embedding.CreateResponse{}, nil
}

func (p *contextCaptureProvider) generateImage(ctx context.Context, _ credential, _ *image.GenerateRequest) (*image.GenerateResponse, error) {
	p.capture(ctx)
	return &image.GenerateResponse{}, nil
}

func (p *contextCaptureProvider) streamImage(ctx context.Context, _ credential, _ *image.StreamRequest) (image.Stream, error) {
	p.capture(ctx)
	return nil, nil
}

func (p *contextCaptureProvider) editImage(ctx context.Context, _ credential, _ *image.EditRequest) (*image.GenerateResponse, error) {
	p.capture(ctx)
	return &image.GenerateResponse{}, nil
}

func (p *contextCaptureProvider) streamImageEdit(ctx context.Context, _ credential, _ *image.EditStreamRequest) (image.Stream, error) {
	p.capture(ctx)
	return nil, nil
}

func (p *contextCaptureProvider) createRerank(ctx context.Context, _ credential, _ *rerank.CreateRequest) (*rerank.CreateResponse, error) {
	p.capture(ctx)
	return &rerank.CreateResponse{}, nil
}

func (p *contextCaptureProvider) createSpeech(ctx context.Context, _ credential, _ *speech.CreateRequest) (*speech.CreateResponse, error) {
	p.capture(ctx)
	return &speech.CreateResponse{}, nil
}

func (p *contextCaptureProvider) streamSpeech(ctx context.Context, _ credential, _ *speech.StreamRequest) (speech.Stream, error) {
	p.capture(ctx)
	return nil, nil
}

func (p *contextCaptureProvider) listVoices(ctx context.Context, _ credential, _ *speech.ListVoicesRequest) (*speech.ListVoicesResponse, error) {
	p.capture(ctx)
	return &speech.ListVoicesResponse{}, nil
}

func (p *contextCaptureProvider) uploadVoiceFile(ctx context.Context, _ credential, _ *speech.UploadVoiceFileRequest) (*speech.UploadVoiceFileResponse, error) {
	p.capture(ctx)
	return &speech.UploadVoiceFileResponse{File: speech.UploadedFile{FileID: 1}}, nil
}

func (p *contextCaptureProvider) cloneVoice(ctx context.Context, _ credential, _ *speech.CloneVoiceRequest) (*speech.CloneVoiceResponse, error) {
	p.capture(ctx)
	return &speech.CloneVoiceResponse{}, nil
}

func newContextTestClient(provider *contextCaptureProvider) *Client {
	client := &Client{
		config: ClientConfig{Provider: ProviderOpenAI}, credentials: newAnonymousCredentialPool(), provider: provider,
	}
	client.Chat = ChatService{client: client}
	client.Embeddings = EmbeddingService{client: client}
	client.Images = ImageService{client: client}
	client.Rerank = RerankService{client: client}
	client.Speech = SpeechService{client: client}
	return client
}

func TestPublicServiceEntriesDefaultNilContext(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client) error
	}{
		{name: "chat create", call: func(client *Client) error { _, err := client.Chat.Create(nil, &chat.CreateRequest{}); return err }},
		{name: "chat stream", call: func(client *Client) error { _, err := client.Chat.Stream(nil, &chat.StreamRequest{}); return err }},
		{name: "embedding create", call: func(client *Client) error {
			_, err := client.Embeddings.Create(nil, &embedding.CreateRequest{})
			return err
		}},
		{name: "image generate", call: func(client *Client) error {
			_, err := client.Images.Generate(nil, &image.GenerateRequest{})
			return err
		}},
		{name: "image stream", call: func(client *Client) error { _, err := client.Images.Stream(nil, &image.StreamRequest{}); return err }},
		{name: "image edit", call: func(client *Client) error { _, err := client.Images.Edit(nil, &image.EditRequest{}); return err }},
		{name: "image edit stream", call: func(client *Client) error {
			_, err := client.Images.EditStream(nil, &image.EditStreamRequest{})
			return err
		}},
		{name: "rerank create", call: func(client *Client) error { _, err := client.Rerank.Create(nil, &rerank.CreateRequest{}); return err }},
		{name: "speech create", call: func(client *Client) error { _, err := client.Speech.Create(nil, &speech.CreateRequest{}); return err }},
		{name: "speech stream", call: func(client *Client) error { _, err := client.Speech.Stream(nil, &speech.StreamRequest{}); return err }},
		{name: "speech list voices", call: func(client *Client) error {
			_, err := client.Speech.ListVoices(nil, &speech.ListVoicesRequest{})
			return err
		}},
		{name: "speech upload", call: func(client *Client) error {
			_, err := client.Speech.UploadVoiceFile(nil, &speech.UploadVoiceFileRequest{})
			return err
		}},
		{name: "speech clone", call: func(client *Client) error {
			_, err := client.Speech.CloneVoice(nil, &speech.CloneVoiceRequest{})
			return err
		}},
		{name: "speech clone from files", call: func(client *Client) error {
			_, err := client.Speech.CloneVoiceFromFiles(nil, &speech.CloneVoiceFromFilesRequest{
				SourceFilePath: "source.mp3", CloneRequest: speech.CloneVoiceRequest{VoiceID: "voice"},
			})
			return err
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &contextCaptureProvider{}
			if err := test.call(newContextTestClient(provider)); err != nil {
				t.Fatal(err)
			}
			if len(provider.contexts) == 0 {
				t.Fatal("provider was not called")
			}
			for _, ctx := range provider.contexts {
				if ctx == nil {
					t.Fatal("provider received nil context")
				}
			}
		})
	}
}

func TestPublicServiceEntriesPreserveNonNilContext(t *testing.T) {
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("key"), "value")
	provider := &contextCaptureProvider{}
	client := newContextTestClient(provider)
	if _, err := client.Chat.Create(ctx, &chat.CreateRequest{}); err != nil {
		t.Fatal(err)
	}
	if len(provider.contexts) != 1 || provider.contexts[0] != ctx {
		t.Fatal("non-nil context was not preserved")
	}
}
