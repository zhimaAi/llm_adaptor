// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

const (
	DefaultBaseURLMiniMax = "https://api.minimaxi.com/v1"
	MiniMaxSpeechPath     = "/t2a_v2"
	ChatCompletionsPath   = "/chat/completions"
	EmbeddingsPath        = "/embeddings"
	ImageGenerationsPath  = "/images/generations"
)

type Client struct {
	config      ClientConfig
	credentials *credentialPool
	provider    providerImplementation

	Chat       ChatService
	Embeddings EmbeddingService
	Images     ImageService
	Rerank     RerankService
	Speech     SpeechService
}

type providerImplementation interface {
	info() ProviderInfo
}

type speechProvider interface {
	createSpeech(context.Context, credential, *speech.CreateRequest) (*speech.CreateResponse, error)
	streamSpeech(context.Context, credential, *speech.StreamRequest) (speech.Stream, error)
}

type speechVoiceProvider interface {
	listVoices(context.Context, credential, *speech.ListVoicesRequest) (*speech.ListVoicesResponse, error)
	uploadVoiceFile(context.Context, credential, *speech.UploadVoiceFileRequest) (*speech.UploadVoiceFileResponse, error)
	cloneVoice(context.Context, credential, *speech.CloneVoiceRequest) (*speech.CloneVoiceResponse, error)
}

type chatProvider interface {
	createChat(context.Context, credential, *chat.CreateRequest) (*chat.CreateResponse, error)
	streamChat(context.Context, credential, *chat.StreamRequest) (chat.Stream, error)
}

type embeddingProvider interface {
	createEmbedding(context.Context, credential, *embedding.CreateRequest) (*embedding.CreateResponse, error)
}

type imageProvider interface {
	generateImage(context.Context, credential, *image.GenerateRequest) (*image.GenerateResponse, error)
}

type imageStreamProvider interface {
	streamImage(context.Context, credential, *image.StreamRequest) (image.Stream, error)
}

type rerankProvider interface {
	createRerank(context.Context, credential, *rerank.CreateRequest) (*rerank.CreateResponse, error)
}

func NewClient(config ClientConfig) (*Client, error) {
	definition, ok := providerDefinitions[config.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, config.Provider)
	}
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.ServiceBaseURL = strings.TrimRight(strings.TrimSpace(config.ServiceBaseURL), "/")
	if config.BaseURL == "" {
		config.BaseURL = definition.defaultBaseURL
	}
	if config.BaseURL == "" {
		return nil, fmt.Errorf("%w: base_url is required for provider %s", ErrInvalidRequest, config.Provider)
	}
	if config.ServiceBaseURL == "" {
		config.ServiceBaseURL = definition.defaultServiceBaseURL
	}
	switch config.Provider {
	case ProviderOpenAIAgent, ProviderXinference:
		if strings.TrimSpace(config.APIVersion) == "" {
			return nil, fmt.Errorf("%w: api_version is required for provider %s", ErrInvalidRequest, config.Provider)
		}
		config.BaseURL = appendURLSegment(config.BaseURL, config.APIVersion)
	case ProviderOllama:
		config.BaseURL = appendURLSegment(config.BaseURL, "v1")
	}
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}
	var credentials *credentialPool
	var err error
	if strings.TrimSpace(config.Credentials.APIKeys) == "" && definition.credentialsOptional {
		credentials = newAnonymousCredentialPool()
	} else {
		credentials, err = newCredentialPool(config.Credentials)
		if err != nil {
			return nil, err
		}
	}
	implementation := definition.newProvider(config)
	client := &Client{
		config:      config,
		credentials: credentials,
		provider:    implementation,
	}
	client.Chat = ChatService{client: client}
	client.Embeddings = EmbeddingService{client: client}
	client.Images = ImageService{client: client}
	client.Rerank = RerankService{client: client}
	client.Speech = SpeechService{client: client}
	return client, nil
}

type ChatService struct{ client *Client }

func (s ChatService) Create(ctx context.Context, req *chat.CreateRequest) (*chat.CreateResponse, error) {
	if !s.client.supports(CapabilityChat) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityChat}
	}
	provider, ok := s.client.provider.(chatProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityChat}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.createChat(ctx, selected, req)
}

func (s ChatService) Stream(ctx context.Context, req *chat.StreamRequest) (chat.Stream, error) {
	if !s.client.supports(CapabilityChat) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityChat}
	}
	provider, ok := s.client.provider.(chatProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityChat}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.streamChat(ctx, selected, req)
}

type EmbeddingService struct{ client *Client }

func (s EmbeddingService) Create(ctx context.Context, req *embedding.CreateRequest) (*embedding.CreateResponse, error) {
	if !s.client.supports(CapabilityEmbedding) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityEmbedding}
	}
	provider, ok := s.client.provider.(embeddingProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityEmbedding}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.createEmbedding(ctx, selected, req)
}

type ImageService struct{ client *Client }

func (s ImageService) Generate(ctx context.Context, req *image.GenerateRequest) (*image.GenerateResponse, error) {
	if !s.client.supports(CapabilityImage) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	provider, ok := s.client.provider.(imageProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.generateImage(ctx, selected, req)
}

func (s ImageService) Stream(ctx context.Context, req *image.StreamRequest) (image.Stream, error) {
	if !s.client.supports(CapabilityImage) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	provider, ok := s.client.provider.(imageStreamProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityImage}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.streamImage(ctx, selected, req)
}

type RerankService struct{ client *Client }

func (s RerankService) Create(ctx context.Context, req *rerank.CreateRequest) (*rerank.CreateResponse, error) {
	if !s.client.supports(CapabilityRerank) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityRerank}
	}
	provider, ok := s.client.provider.(rerankProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilityRerank}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.createRerank(ctx, selected, req)
}

func (c *Client) ProviderInfo() ProviderInfo {
	return c.provider.info()
}

func (c *Client) supports(capability Capability) bool {
	for _, supported := range c.provider.info().Capabilities {
		if supported == capability {
			return true
		}
	}
	return false
}

type SpeechService struct {
	client *Client
}

func (s SpeechService) Create(ctx context.Context, req *speech.CreateRequest) (*speech.CreateResponse, error) {
	if !s.client.supports(CapabilitySpeech) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	provider, ok := s.client.provider.(speechProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	credential, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.createSpeech(ctx, credential, req)
}

func (s SpeechService) Stream(ctx context.Context, req *speech.StreamRequest) (speech.Stream, error) {
	if !s.client.supports(CapabilitySpeech) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	provider, ok := s.client.provider.(speechProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	credential, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.streamSpeech(ctx, credential, req)
}

func (s SpeechService) ListVoices(ctx context.Context, req *speech.ListVoicesRequest) (*speech.ListVoicesResponse, error) {
	if !s.client.supports(CapabilitySpeech) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	provider, ok := s.client.provider.(speechVoiceProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.listVoices(ctx, selected, req)
}

func (s SpeechService) UploadVoiceFile(ctx context.Context, req *speech.UploadVoiceFileRequest) (*speech.UploadVoiceFileResponse, error) {
	if !s.client.supports(CapabilitySpeech) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	provider, ok := s.client.provider.(speechVoiceProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.uploadVoiceFile(ctx, selected, req)
}

func (s SpeechService) CloneVoice(ctx context.Context, req *speech.CloneVoiceRequest) (*speech.CloneVoiceResponse, error) {
	if !s.client.supports(CapabilitySpeech) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	provider, ok := s.client.provider.(speechVoiceProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	return provider.cloneVoice(ctx, selected, req)
}

// CloneVoiceFromFiles uploads the source and optional prompt audio, then clones
// the voice with one pinned credential so account-scoped file IDs stay valid.
func (s SpeechService) CloneVoiceFromFiles(ctx context.Context, req *speech.CloneVoiceFromFilesRequest) (*speech.CloneVoiceFromFilesResponse, error) {
	if !s.client.supports(CapabilitySpeech) {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	provider, ok := s.client.provider.(speechVoiceProvider)
	if !ok {
		return nil, &UnsupportedCapabilityError{Provider: s.client.config.Provider, Capability: CapabilitySpeech}
	}
	if req == nil || strings.TrimSpace(req.SourceFilePath) == "" {
		return nil, fmt.Errorf("%w: MiniMax source_file_path is required", ErrInvalidRequest)
	}
	if strings.TrimSpace(req.CloneRequest.VoiceID) == "" {
		return nil, fmt.Errorf("%w: MiniMax voice_id is required", ErrInvalidRequest)
	}
	if strings.TrimSpace(req.PromptFilePath) != "" && (req.CloneRequest.ClonePrompt == nil || strings.TrimSpace(req.CloneRequest.ClonePrompt.PromptText) == "") {
		return nil, fmt.Errorf("%w: MiniMax clone_prompt is required with prompt_file_path", ErrInvalidRequest)
	}
	selected, err := s.client.credentials.selectCredential()
	if err != nil {
		return nil, err
	}
	result := &speech.CloneVoiceFromFilesResponse{}
	result.SourceUpload, err = provider.uploadVoiceFile(ctx, selected, &speech.UploadVoiceFileRequest{
		Purpose: miniMaxVoiceClonePurpose, FilePath: req.SourceFilePath,
	})
	if err != nil {
		return nil, err
	}
	cloneRequest := req.CloneRequest
	if cloneRequest.ClonePrompt != nil {
		clonePrompt := *cloneRequest.ClonePrompt
		cloneRequest.ClonePrompt = &clonePrompt
	}
	cloneRequest.FileID = result.SourceUpload.File.FileID
	if strings.TrimSpace(req.PromptFilePath) != "" {
		result.PromptUpload, err = provider.uploadVoiceFile(ctx, selected, &speech.UploadVoiceFileRequest{
			Purpose: miniMaxPromptAudioPurpose, FilePath: req.PromptFilePath,
		})
		if err != nil {
			return nil, err
		}
		cloneRequest.ClonePrompt.PromptAudio = result.PromptUpload.File.FileID
	}
	result.Clone, err = provider.cloneVoice(ctx, selected, &cloneRequest)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type providerDefinition struct {
	defaultBaseURL        string
	defaultServiceBaseURL string
	credentialsOptional   bool
	newProvider           func(ClientConfig) providerImplementation
}

var providerDefinitions = map[Provider]providerDefinition{
	Provider302AI: new302AIProviderDefinition(),
	ProviderAli: {
		defaultBaseURL:        "https://dashscope.aliyuncs.com/compatible-mode/v1",
		defaultServiceBaseURL: aliDefaultServiceBaseURL,
		newProvider:           func(config ClientConfig) providerImplementation { return newAliProvider(config) },
	},
	ProviderBaichuan: newGenericProviderDefinition("https://api.baichuan-ai.com/v1", ProviderBaichuan, false, CapabilityChat, CapabilityEmbedding),
	ProviderBaidu:    newGenericProviderDefinition("https://qianfan.baidubce.com/v2", ProviderBaidu, false, CapabilityChat, CapabilityEmbedding),
	ProviderDeepSeek: newGenericProviderDefinition("https://api.deepseek.com", ProviderDeepSeek, false, CapabilityChat),
	ProviderDoubao:   newGenericProviderDefinition("https://ark.cn-beijing.volces.com/api/v3", ProviderDoubao, false, CapabilityChat, CapabilityEmbedding, CapabilityImage),
	ProviderGemini: {
		defaultBaseURL:        "https://generativelanguage.googleapis.com/v1beta/openai",
		defaultServiceBaseURL: geminiDefaultServiceBaseURL,
		newProvider:           func(config ClientConfig) providerImplementation { return newGeminiProvider(config) },
	},
	ProviderHunyuan:        newGenericProviderDefinition("https://api.hunyuan.cloud.tencent.com/v1", ProviderHunyuan, false, CapabilityChat, CapabilityEmbedding),
	ProviderLingYiWanWu:    newGenericProviderDefinition("https://api.lingyiwanwu.com/v1", ProviderLingYiWanWu, false, CapabilityChat),
	ProviderMoonshot:       newGenericProviderDefinition("https://api.moonshot.cn/v1", ProviderMoonshot, false, CapabilityChat),
	ProviderOllama:         newGenericProviderDefinition("http://localhost:11434/v1", ProviderOllama, true, CapabilityChat, CapabilityEmbedding),
	ProviderOpenAI:         newGenericProviderDefinition("https://api.openai.com/v1", ProviderOpenAI, false, CapabilityChat, CapabilityEmbedding, CapabilityImage),
	ProviderOpenAIAgent:    newGenericProviderDefinition("", ProviderOpenAIAgent, false, CapabilityChat, CapabilityEmbedding),
	ProviderOpenCompatible: newGenericProviderDefinition("", ProviderOpenCompatible, true, CapabilityChat, CapabilityEmbedding, CapabilityImage),
	ProviderOpenRouter: {
		defaultBaseURL: "https://openrouter.ai/api/v1",
		newProvider:    func(config ClientConfig) providerImplementation { return newOpenRouterProvider(config) },
	},
	ProviderSiliconFlow: newRerankProviderDefinition("https://api.siliconflow.cn/v1", ProviderSiliconFlow, false, "/rerank", "documents", "top_k", CapabilityChat, CapabilityEmbedding, CapabilityRerank),
	ProviderSpark:       newGenericProviderDefinition("https://spark-api-open.xf-yun.com/v1", ProviderSpark, false, CapabilityChat),
	ProviderXinference:  newRerankProviderDefinition("", ProviderXinference, true, "/rerank", "documents", "top_n", CapabilityChat, CapabilityEmbedding, CapabilityRerank),
	ProviderZhipu:       newGenericProviderDefinition("https://open.bigmodel.cn/api/paas/v4", ProviderZhipu, false, CapabilityChat, CapabilityEmbedding),
	ProviderAzure: {
		newProvider: func(config ClientConfig) providerImplementation { return &azureProvider{config: config} },
	},
	ProviderClaude: {
		defaultBaseURL: "https://api.anthropic.com/v1",
		newProvider:    func(config ClientConfig) providerImplementation { return &claudeProvider{config: config} },
	},
	ProviderBAAI:   newBAAIProviderDefinition(),
	ProviderCohere: newCohereProviderDefinition(),
	ProviderJina:   newRerankProviderDefinition("https://api.jina.ai/v1", ProviderJina, false, "/rerank", "documents", "top_n", CapabilityEmbedding, CapabilityRerank),
	ProviderVoyage: newGenericProviderDefinition("https://api.voyageai.com/v1", ProviderVoyage, false, CapabilityEmbedding),
	ProviderMiniMax: {
		defaultBaseURL: DefaultBaseURLMiniMax,
		newProvider: func(config ClientConfig) providerImplementation {
			return newMiniMaxProvider(config)
		},
	},
}

func newGenericProviderDefinition(baseURL string, id Provider, credentialsOptional bool, capabilities ...Capability) providerDefinition {
	return providerDefinition{
		defaultBaseURL:      baseURL,
		credentialsOptional: credentialsOptional,
		newProvider: func(config ClientConfig) providerImplementation {
			return newOpenAICompatibleProvider(config, ProviderInfo{ID: id, DefaultBaseURL: baseURL, Capabilities: capabilities})
		},
	}
}

func newRerankProviderDefinition(baseURL string, id Provider, credentialsOptional bool, path, documentsKey, topKey string, capabilities ...Capability) providerDefinition {
	definition := newGenericProviderDefinition(baseURL, id, credentialsOptional, capabilities...)
	definition.newProvider = func(config ClientConfig) providerImplementation {
		provider := newOpenAICompatibleProvider(config, ProviderInfo{ID: id, DefaultBaseURL: baseURL, Capabilities: capabilities})
		provider.rerankPath = path
		provider.rerankDocumentsKey = documentsKey
		provider.rerankTopKey = topKey
		return provider
	}
	return definition
}

func newCohereProviderDefinition() providerDefinition {
	const (
		baseURL        = "https://api.cohere.ai/compatibility/v1"
		serviceBaseURL = "https://api.cohere.com"
	)
	return providerDefinition{
		defaultBaseURL:        baseURL,
		defaultServiceBaseURL: serviceBaseURL,
		newProvider: func(config ClientConfig) providerImplementation {
			provider := newOpenAICompatibleProvider(config, ProviderInfo{
				ID: ProviderCohere, DefaultBaseURL: baseURL, DefaultServiceBaseURL: serviceBaseURL,
				Capabilities: []Capability{CapabilityChat, CapabilityEmbedding, CapabilityRerank},
			})
			provider.rerankPath = "/v2/rerank"
			provider.rerankBaseURL = config.ServiceBaseURL
			provider.rerankDocumentsKey = "documents"
			provider.rerankTopKey = "top_n"
			return provider
		},
	}
}

func newBAAIProviderDefinition() providerDefinition {
	definition := newRerankProviderDefinition("", ProviderBAAI, true, "/v1/rerank", "passages", "top_k", CapabilityEmbedding, CapabilityRerank)
	definition.newProvider = func(config ClientConfig) providerImplementation {
		info := ProviderInfo{ID: ProviderBAAI, Capabilities: []Capability{CapabilityEmbedding, CapabilityRerank}}
		provider := newOpenAICompatibleProvider(config, info)
		rawPrefix := ""
		provider.authorizationPrefix = &rawPrefix
		provider.embeddingPath = "/v1/embeddings"
		provider.rerankPath = "/v1/rerank"
		provider.rerankDocumentsKey = "passages"
		provider.rerankTopKey = "top_k"
		return provider
	}
	return definition
}

func new302AIProviderDefinition() providerDefinition {
	definition := newGenericProviderDefinition("https://api.302ai.cn", Provider302AI, false, CapabilityChat, CapabilityImage)
	definition.newProvider = func(config ClientConfig) providerImplementation {
		provider := newOpenAICompatibleProvider(config, ProviderInfo{ID: Provider302AI, DefaultBaseURL: "https://api.302ai.cn", Capabilities: []Capability{CapabilityChat, CapabilityImage}})
		provider.chatPath = "/v1/chat/completions"
		provider.imagePath = "/302/images/generations"
		return provider
	}
	return definition
}
