// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package provider

import (
	"context"

	"github.com/zhimaAi/llm_adaptor/v2/chat"
	"github.com/zhimaAi/llm_adaptor/v2/embedding"
	"github.com/zhimaAi/llm_adaptor/v2/image"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
	"github.com/zhimaAi/llm_adaptor/v2/speech"
)

type Implementation interface {
	Info() Info
}

type Chat interface {
	CreateChat(context.Context, Credential, *chat.CreateRequest) (*chat.CreateResponse, error)
	StreamChat(context.Context, Credential, *chat.StreamRequest) (chat.Stream, error)
}

type Embedding interface {
	CreateEmbedding(context.Context, Credential, *embedding.CreateRequest) (*embedding.CreateResponse, error)
}

type Image interface {
	GenerateImage(context.Context, Credential, *image.GenerateRequest) (*image.GenerateResponse, error)
}

type ImageStream interface {
	StreamImage(context.Context, Credential, *image.StreamRequest) (image.Stream, error)
}

type ImageEdit interface {
	EditImage(context.Context, Credential, *image.EditRequest) (*image.GenerateResponse, error)
}

type ImageEditStream interface {
	StreamImageEdit(context.Context, Credential, *image.EditStreamRequest) (image.Stream, error)
}

type Rerank interface {
	CreateRerank(context.Context, Credential, *rerank.CreateRequest) (*rerank.CreateResponse, error)
}

type Speech interface {
	CreateSpeech(context.Context, Credential, *speech.CreateRequest) (*speech.CreateResponse, error)
	StreamSpeech(context.Context, Credential, *speech.StreamRequest) (speech.Stream, error)
}

type SpeechVoice interface {
	ListVoices(context.Context, Credential, *speech.ListVoicesRequest) (*speech.ListVoicesResponse, error)
	UploadVoiceFile(context.Context, Credential, *speech.UploadVoiceFileRequest) (*speech.UploadVoiceFileResponse, error)
	CloneVoice(context.Context, Credential, *speech.CloneVoiceRequest) (*speech.CloneVoiceResponse, error)
}
