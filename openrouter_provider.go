package llm

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/image"
)

type openRouterProvider struct{ *openAICompatibleProvider }

func newOpenRouterProvider(config ClientConfig) *openRouterProvider {
	return &openRouterProvider{newOpenAICompatibleProvider(config, ProviderInfo{
		ID: ProviderOpenRouter, DefaultBaseURL: "https://openrouter.ai/api/v1", Capabilities: []Capability{CapabilityChat, CapabilityImage},
	})}
}

func (p *openRouterProvider) generateImage(ctx context.Context, selected credential, request *image.GenerateRequest) (*image.GenerateResponse, error) {
	if request == nil || request.Model == "" || strings.TrimSpace(request.Prompt) == "" {
		return nil, fmt.Errorf("%w: image model and prompt are required", ErrInvalidRequest)
	}
	extra := make(map[string]any, len(request.ExtraBody)+2)
	for key, value := range request.ExtraBody {
		extra[key] = value
	}
	extra["modalities"] = []string{"image", "text"}
	imageConfig := map[string]any{}
	if request.Size != "" {
		imageConfig["image_size"] = request.Size
	}
	if len(imageConfig) > 0 {
		extra["image_config"] = imageConfig
	}
	chatResponse, err := p.createChat(ctx, selected, &chat.CreateRequest{
		Model:     request.Model,
		Messages:  []chat.Message{{Role: chat.RoleUser, Content: chat.TextContent(request.Prompt)}},
		ExtraBody: extra,
	})
	if err != nil {
		return nil, err
	}
	result := &image.GenerateResponse{RawResponse: chatResponse.RawResponse}
	for _, choice := range chatResponse.Choices {
		for _, generated := range choice.Message.Images {
			if generated.ImageURL == nil {
				continue
			}
			item := image.Data{URL: generated.ImageURL.URL}
			if strings.HasPrefix(item.URL, "data:") {
				if comma := strings.IndexByte(item.URL, ','); comma >= 0 {
					encoded := item.URL[comma+1:]
					if _, decodeErr := base64.StdEncoding.DecodeString(encoded); decodeErr == nil {
						item.B64JSON = encoded
						item.URL = ""
					}
				}
			}
			result.Data = append(result.Data, item)
		}
	}
	result.Usage.InputTokens = chatResponse.Usage.PromptTokens
	result.Usage.OutputTokens = chatResponse.Usage.CompletionTokens
	result.Usage.TotalTokens = chatResponse.Usage.TotalTokens
	return result, nil
}

func (p *openRouterProvider) streamImage(context.Context, credential, *image.StreamRequest) (image.Stream, error) {
	return nil, &UnsupportedCapabilityError{Provider: ProviderOpenRouter, Capability: CapabilityImage}
}

var _ imageProvider = (*openRouterProvider)(nil)
var _ imageStreamProvider = (*openRouterProvider)(nil)
