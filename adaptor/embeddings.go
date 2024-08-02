// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package adaptor

import (
	"errors"
	"github.com/zhimaAi/llm_adaptor/api/ali"
	"github.com/zhimaAi/llm_adaptor/api/azure"
	"github.com/zhimaAi/llm_adaptor/api/baai"
	"github.com/zhimaAi/llm_adaptor/api/baichuan"
	"github.com/zhimaAi/llm_adaptor/api/baidu"
	"github.com/zhimaAi/llm_adaptor/api/cohere"
	"github.com/zhimaAi/llm_adaptor/api/gemini"
	"github.com/zhimaAi/llm_adaptor/api/hunyuan"
	"github.com/zhimaAi/llm_adaptor/api/jina"
	"github.com/zhimaAi/llm_adaptor/api/ollama"
	"github.com/zhimaAi/llm_adaptor/api/openai"
	openaiagent "github.com/zhimaAi/llm_adaptor/api/openaiAgent"
	"github.com/zhimaAi/llm_adaptor/api/volcenginev2"
	"github.com/zhimaAi/llm_adaptor/api/voyage"
	"github.com/zhimaAi/llm_adaptor/api/xinference"
	"github.com/zhimaAi/llm_adaptor/api/zhipu"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencentHunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
)

type ZhimaEmbeddingRequest struct {
	Input string `json:"input"`
}

type ZhimaEmbeddingResponse struct {
	Result          []float64 `json:"result"`
	PromptToken     int       `json:"prompt_token"`
	CompletionToken int       `json:"completion_token"`
}

func (a *Adaptor) CreateEmbeddings(req ZhimaEmbeddingRequest) (ZhimaEmbeddingResponse, error) {
	if req.Input == "" {
		return ZhimaEmbeddingResponse{}, errors.New("input empty")
	}

	switch a.meta.Corp {
	case "openai":
		client := openai.NewClient("https://api.openai.com/v1", a.meta.APIKey, &openai.ErrorResponse{})
		r := openai.EmbeddingRequest{
			Model: a.meta.Model,
			Input: []string{req.Input},
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Data[0].Embedding,
			PromptToken:     res.Usage.PromptTokens,
			CompletionToken: res.Usage.TotalTokens - res.Usage.PromptTokens,
		}, nil
	case "baichuan", "zhipu", "openaiAgent":
		var client *openai.Client
		if a.meta.Corp == "baichuan" {
			client = baichuan.NewClient(a.meta.APIKey).OpenAIClient
		} else if a.meta.Corp == "zhipu" {
			client = zhipu.NewClient(a.meta.APIKey).OpenAIClient
		} else if a.meta.Corp == "openaiAgent" {
			client = openaiagent.NewClient(a.meta.EndPoint, a.meta.APIKey, a.meta.APIVersion).OpenAIClient
		}
		r := openai.EmbeddingRequest{
			Model: a.meta.Model,
			Input: []string{req.Input},
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Data[0].Embedding,
			PromptToken:     res.Usage.PromptTokens,
			CompletionToken: res.Usage.TotalTokens - res.Usage.PromptTokens,
		}, nil
	case "azure":
		client := azure.NewClient(
			a.meta.EndPoint,
			a.meta.APIVersion,
			a.meta.APIKey,
			a.meta.Model,
		)
		r := azure.EmbeddingRequest{
			Input: []string{req.Input},
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Data[0].Embedding,
			PromptToken:     res.Usage.PromptTokens,
			CompletionToken: res.Usage.TotalTokens - res.Usage.PromptTokens,
		}, nil
	case "baidu":
		client := baidu.NewClient(
			a.meta.APIKey,
			a.meta.SecretKey,
			a.meta.Model,
		)
		r := baidu.EmbeddingRequest{
			Input: []string{req.Input},
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Data[0].Embedding,
			PromptToken:     res.Usage.PromptTokens,
			CompletionToken: res.Usage.CompletionTokens,
		}, nil
	case "ali":
		client := ali.NewClient(a.meta.APIKey)
		r := ali.EmbeddingRequest{
			Input:      ali.Texts{Texts: []string{req.Input}},
			Model:      a.meta.Model,
			Parameters: ali.TextType{TextType: "document"},
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Output.Embeddings[0].Embedding,
			PromptToken:     0,
			CompletionToken: res.Usage.TotalTokens,
		}, nil
	case "voyage":
		client := voyage.NewClient(
			a.meta.APIKey,
		)
		r := voyage.EmbeddingRequest{
			Input: []string{req.Input},
			Model: a.meta.Model,
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Data[0].Embedding,
			PromptToken:     res.Usage.PromptTokens,
			CompletionToken: res.Usage.TotalTokens - res.Usage.PromptTokens,
		}, nil
	case "gemini":
		client := gemini.NewClient(
			a.meta.APIKey,
			a.meta.Model,
		)
		r := gemini.EmbeddingRequest{
			Content: gemini.Content{Parts: []gemini.Part{{Text: req.Input}}},
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result: res.Embedding.Values,
		}, nil

	case "baai":
		client := baai.NewClient(a.meta.EndPoint, a.meta.Model, a.meta.APIKey)
		r := baai.EmbeddingRequest{
			Model: a.meta.Model,
			Input: []string{req.Input},
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Data[0].Embedding,
			PromptToken:     res.Usage.PromptTokens,
			CompletionToken: res.Usage.TotalTokens - res.Usage.PromptTokens,
		}, nil
	case "doubao":
		client := volcenginev2.NewClient(`maas-api.ml-platform-cn-beijing.volces.com`, a.meta.Model, a.meta.APIKey, a.meta.SecretKey, a.meta.Region)
		r := volcenginev2.EmbeddingRequest{
			Input: []string{req.Input},
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Data[0].Embedding,
			PromptToken:     res.Usage.PromptTokens,
			CompletionToken: res.Usage.CompletionTokens,
		}, nil
	case "cohere":
		client := cohere.NewClient(a.meta.APIKey)
		r := cohere.EmbeddingRequest{
			Texts:     []string{req.Input},
			Model:     a.meta.Model,
			InputType: "classification",
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Embeddings[0],
			PromptToken:     res.Meta.Tokens.InputTokens,
			CompletionToken: res.Meta.Tokens.OutputTokens,
		}, nil
	case "hunyuan":
		client := hunyuan.NewClient(a.meta.APIKey, a.meta.SecretKey, a.meta.Region)
		r := tencentHunyuan.NewGetEmbeddingRequest()
		r.Input = common.StringPtr(req.Input)
		res, err := client.CreateEmbeddings(*r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		var result []float64
		for _, v := range res.Data[0].Embedding {
			result = append(result, *v)
		}
		return ZhimaEmbeddingResponse{
			Result:          result,
			PromptToken:     int(*res.Usage.PromptTokens),
			CompletionToken: int(*res.Usage.TotalTokens) - int(*res.Usage.PromptTokens),
		}, nil
	case "jina":
		client := jina.NewClient(a.meta.APIKey)
		r := jina.EmbeddingRequest{
			Input:        []string{req.Input},
			Model:        a.meta.Model,
			EncodingType: "float",
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result:          res.Data[0].Embedding,
			PromptToken:     res.Usage.PromptTokens,
			CompletionToken: res.Usage.TotalTokens - res.Usage.PromptTokens,
		}, nil
	case "ollama":
		client := ollama.NewClient(a.meta.EndPoint, a.meta.Model)
		r := ollama.EmbeddingRequest{
			Prompt: req.Input,
			Model:  a.meta.Model,
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result: res.Embedding,
		}, nil
	case "xinference":
		client := xinference.NewClient(a.meta.EndPoint, a.meta.APIVersion, a.meta.Model)
		r := xinference.EmbeddingRequest{
			Input: []string{req.Input},
			Model: a.meta.Model,
		}
		res, err := client.CreateEmbeddings(r)
		if err != nil {
			return ZhimaEmbeddingResponse{}, err
		}
		return ZhimaEmbeddingResponse{
			Result: res.Data[0].Embedding,
		}, nil
	}
	return ZhimaEmbeddingResponse{}, nil
}
