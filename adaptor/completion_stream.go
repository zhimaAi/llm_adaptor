// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package adaptor

import (
	"errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencentHunyuan "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/hunyuan/v20230901"
	"github.com/zhimaAi/go_tools/logs"
	"github.com/zhimaAi/go_tools/tool"
	"github.com/zhimaAi/llm_adaptor/api/ali"
	"github.com/zhimaAi/llm_adaptor/api/azure"
	"github.com/zhimaAi/llm_adaptor/api/baichuan"
	"github.com/zhimaAi/llm_adaptor/api/baidu"
	"github.com/zhimaAi/llm_adaptor/api/claude"
	"github.com/zhimaAi/llm_adaptor/api/cohere"
	"github.com/zhimaAi/llm_adaptor/api/deepseek"
	"github.com/zhimaAi/llm_adaptor/api/gemini"
	"github.com/zhimaAi/llm_adaptor/api/hunyuan"
	"github.com/zhimaAi/llm_adaptor/api/lingyiwanwu"
	"github.com/zhimaAi/llm_adaptor/api/minimax"
	"github.com/zhimaAi/llm_adaptor/api/moonshot"
	"github.com/zhimaAi/llm_adaptor/api/ollama"
	"github.com/zhimaAi/llm_adaptor/api/openai"
	openaiagent "github.com/zhimaAi/llm_adaptor/api/openaiAgent"
	"github.com/zhimaAi/llm_adaptor/api/spark"
	"github.com/zhimaAi/llm_adaptor/api/volcenginev3"
	"github.com/zhimaAi/llm_adaptor/api/xinference"
	"github.com/zhimaAi/llm_adaptor/api/zhipu"
)

type ZhimaStreamResult interface {
	Read() (ZhimaChatCompletionResponse, error)
	Close() error
}

type ZhimaChatCompletionStreamResponse struct {
	ZhimaStreamResult
}

func (a *Adaptor) CreateChatCompletionStream(req ZhimaChatCompletionRequest) (*ZhimaChatCompletionStreamResponse, error) {
	if len(req.Messages) == 0 {
		return nil, errors.New("messages is required")
	}

	for _, msg := range req.Messages {
		data := "role=" + msg.Role + ",content=" + msg.Content + "\n"
		logs.Debug(data)
	}

	var result *ZhimaChatCompletionStreamResponse

	switch a.meta.Corp {
	case "openai":
		client := openai.NewClient("https://api.openai.com/v1", a.meta.APIKey, &openai.ErrorResponse{})
		var messages []openai.ChatCompletionRequestMessage
		for _, v := range req.Messages {
			messages = append(messages, openai.ChatCompletionRequestMessage{Role: v.Role, Content: v.Content})
		}
		var tools []interface{}
		for _, v := range req.FunctionTools {
			tools = append(tools, map[string]interface{}{
				`type`: `function`,
				`function`: map[string]interface{}{
					`name`:        v.Name,
					`description`: v.Description,
					`parameters`:  v.Parameters,
				},
			})
		}
		req := openai.ChatCompletionRequest{
			Model:       a.meta.Model,
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxToken,
			Tools:       tools,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return nil, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&OpenAIStreamResult{stream},
		}
	case "ali", "baichuan", "moonshot", "lingyiwanwu", "deepseek", "zhipu", "minimax", "openaiAgent":
		var client *openai.Client
		if a.meta.Corp == "ali" {
			client = ali.NewClient(a.meta.APIKey).OpenAIClient
		} else if a.meta.Corp == "baichuan" {
			client = baichuan.NewClient(a.meta.APIKey).OpenAIClient
		} else if a.meta.Corp == "moonshot" {
			client = moonshot.NewClient(a.meta.APIKey).OpenAIClient
		} else if a.meta.Corp == "lingyiwanwu" {
			client = lingyiwanwu.NewClient(a.meta.APIKey).OpenAIClient
		} else if a.meta.Corp == "deepseek" {
			client = deepseek.NewClient(a.meta.APIKey).OpenAIClient
		} else if a.meta.Corp == "zhipu" {
			client = zhipu.NewClient(a.meta.APIKey).OpenAIClient
		} else if a.meta.Corp == "minimax" {
			client = minimax.NewClient(a.meta.APIKey).OpenAIClient
		} else if a.meta.Corp == "openaiAgent" {
			client = openaiagent.NewClient(a.meta.EndPoint, a.meta.APIKey, a.meta.APIVersion).OpenAIClient
		}

		var messages []openai.ChatCompletionRequestMessage
		for _, v := range req.Messages {
			messages = append(messages, openai.ChatCompletionRequestMessage{Role: v.Role, Content: v.Content})
		}
		var tools []interface{}
		for _, v := range req.FunctionTools {
			tools = append(tools, map[string]interface{}{
				`type`: `function`,
				`function`: map[string]interface{}{
					`name`:        v.Name,
					`description`: v.Description,
					`parameters`:  v.Parameters,
				},
			})
		}
		req := openai.ChatCompletionRequest{
			Model:       a.meta.Model,
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxToken,
			Tools:       tools,
		}
		if client == nil {
			return &ZhimaChatCompletionStreamResponse{}, errors.New(`corp not supported`)
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&OpenAIStreamResult{stream},
		}
	case "azure":
		client := azure.NewClient(a.meta.EndPoint, a.meta.APIVersion, a.meta.APIKey, a.meta.Model)
		var messages []azure.ChatCompletionRequestMessage
		for _, v := range req.Messages {
			messages = append(messages, azure.ChatCompletionRequestMessage{Role: v.Role, Content: v.Content})
		}
		var tools []interface{}
		for _, v := range req.FunctionTools {
			tools = append(tools, map[string]interface{}{
				`type`: `function`,
				`function`: map[string]interface{}{
					`name`:        v.Name,
					`description`: v.Description,
					`parameters`:  v.Parameters,
				},
			})
		}
		req := azure.ChatCompletionRequest{
			Model:       a.meta.Model,
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxToken,
			Tools:       tools,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&AzureStreamResult{stream},
		}
	case "baidu":
		client := baidu.NewClient(a.meta.APIKey, a.meta.SecretKey, a.meta.Model)

		var system string
		var messages []baidu.ChatCompletionMessage
		for _, v := range req.Messages {
			if v.Role == "system" {
				system += v.Content
			}
			if v.Role == "user" {
				messages = append(messages, baidu.ChatCompletionMessage{Role: v.Role, Content: v.Content})
			} else if v.Role == "assistant" {
				messages = append(messages, baidu.ChatCompletionMessage{Role: v.Role, Content: v.Content})
			}
		}

		var functions []baidu.Function
		if len(req.FunctionTools) > 0 {
			for _, v := range req.FunctionTools {
				functions = append(functions, baidu.Function{
					Name:        v.Name,
					Description: v.Description,
					Parameters:  v.Parameters,
				})
			}
		}

		req := baidu.ChatCompletionRequest{
			Model:           a.meta.Model,
			Messages:        messages,
			Stream:          true,
			Temperature:     req.Temperature,
			System:          system,
			MaxOutputTokens: req.MaxToken,
			Functions:       functions,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&BaiduStreamResult{stream},
		}
	case "claude":
		client := claude.NewClient(a.meta.APIKey, a.meta.APIVersion)
		var system string
		var messages []claude.Message
		for _, v := range req.Messages {
			if v.Role == "system" {
				system += v.Content
			}
			if v.Role == "user" {
				messages = append(messages, claude.Message{Role: v.Role, Content: v.Content})
			} else if v.Role == "assistant" {
				messages = append(messages, claude.Message{Role: v.Role, Content: v.Content})
			}
		}
		maxTokens := 1024
		if req.MaxToken > 0 {
			maxTokens = req.MaxToken
		}
		var tools []claude.Tool
		if len(req.FunctionTools) > 0 {
			for _, v := range req.FunctionTools {
				tools = append(tools, claude.Tool{
					Name:        v.Name,
					Description: v.Description,
					InputSchema: v.Parameters,
				})
			}
		}
		req := claude.ChatCompletionRequest{
			Model:       a.meta.Model,
			Messages:    messages,
			MaxTokens:   maxTokens,
			Temperature: req.Temperature,
			System:      system,
			//Tools:       tools,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&ClaudeStreamResult{stream},
		}
	case "gemini":
		client := gemini.NewClient(a.meta.APIKey, a.meta.Model)
		var contents []gemini.Content
		for _, v := range req.Messages {
			if v.Role == "user" || v.Role == "system" {
				contents = append(contents, gemini.Content{Role: "user", Parts: []gemini.Part{{Text: v.Content}}})
			} else if v.Role == "assistant" {
				contents = append(contents, gemini.Content{Role: "model", Parts: []gemini.Part{{Text: v.Content}}})
			}

		}
		req := gemini.ChatCompletionRequest{
			Contents:         contents,
			GenerationConfig: gemini.GenerationConfig{Temperature: req.Temperature, MaxOutputTokens: req.MaxToken},
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&GeminiStreamResult{stream},
		}
	case "doubao":
		client := volcenginev3.NewClient("https://ark.cn-beijing.volces.com/api/v3", a.meta.Model, a.meta.APIKey, a.meta.SecretKey, a.meta.Region)
		var messages []openai.ChatCompletionRequestMessage
		for _, v := range req.Messages {
			messages = append(messages, openai.ChatCompletionRequestMessage{Role: v.Role, Content: v.Content})
		}
		req := openai.ChatCompletionRequest{
			Model:       a.meta.Model,
			Messages:    messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxToken,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return nil, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&OpenAIStreamResult{stream},
		}
	case "cohere":
		client := cohere.NewClient(a.meta.APIKey)

		var histories []cohere.ChatHistory
		n := len(req.Messages)
		for _, v := range req.Messages[:n-1] {
			if v.Role == "system" {
				histories = append(histories, cohere.ChatHistory{Role: "SYSTEM", Message: v.Content})
			} else if v.Role == "user" {
				histories = append(histories, cohere.ChatHistory{Role: "USER", Message: v.Content})
			} else if v.Role == "assistant" {
				histories = append(histories, cohere.ChatHistory{Role: "CHATBOT", Message: v.Content})
			}
		}

		req := cohere.ChatCompletionRequest{
			Message:     req.Messages[n-1].Content,
			ChatHistory: histories,
			MaxTokens:   req.MaxToken,
			Temperature: req.Temperature,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return nil, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&CohereStreamResult{stream},
		}
	case "spark":
		client := spark.NewClient(a.meta.APIKey, a.meta.APPID, a.meta.SecretKey, a.meta.Model)
		var messages []spark.ChatCompletionRequestMessage
		for _, v := range req.Messages {
			messages = append(messages, spark.ChatCompletionRequestMessage{Role: v.Role, Content: v.Content})
		}
		var textFunctions []spark.TextFunction
		if len(req.FunctionTools) > 0 && tool.InArrayString(a.meta.Model, []string{`Spark Pro`, `Spark Max`, `Spark4.0 Ultra`}) {
			for _, v := range req.FunctionTools {
				textFunctions = append(textFunctions, spark.TextFunction{
					Name:        v.Name,
					Description: v.Description,
					Parameters:  v.Parameters,
				})
			}
		}
		req := spark.ChatCompletionRequest{
			Parameter: spark.Parameter{
				Chat: spark.Chat{
					Temperature: req.Temperature,
					MaxTokens:   req.MaxToken,
				},
			},
			Payload: spark.RequestPayload{
				Message: spark.RequestMessage{
					Text: messages,
				},
			},
		}
		if len(textFunctions) > 0 {
			//req.Payload.Functions = &spark.Function{Text: textFunctions}
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return nil, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&SparkStreamResult{stream},
		}
	case "hunyuan":
		client := hunyuan.NewClient(a.meta.APIKey, a.meta.SecretKey, a.meta.Region)
		r := tencentHunyuan.NewChatCompletionsRequest()
		r.Model = common.StringPtr(a.meta.Model)
		var systemContent string
		for _, v := range req.Messages {
			if v.Role == "system" {
				systemContent = systemContent + `\n` + v.Content
			}
		}
		r.Messages = append(r.Messages, &tencentHunyuan.Message{
			Role:    common.StringPtr("system"),
			Content: common.StringPtr(systemContent),
		})
		for _, v := range req.Messages {
			if v.Role == "user" || v.Role == "assistant" {
				r.Messages = append(r.Messages, &tencentHunyuan.Message{
					Role:    common.StringPtr(v.Role),
					Content: common.StringPtr(v.Content),
				})
			}
		}
		r.Temperature = common.Float64Ptr(req.Temperature)
		stream, err := client.CreateChatCompletionStream(*r)
		if err != nil {
			return nil, err
		}
		return &ZhimaChatCompletionStreamResponse{
			&TencentStreamResult{stream},
		}, nil
	case "ollama":
		client := ollama.NewClient(a.meta.EndPoint, a.meta.Model)
		var messages []ollama.ChatCompletionMessage
		for _, v := range req.Messages {
			messages = append(messages, ollama.ChatCompletionMessage{Role: v.Role, Content: v.Content})
		}
		req := ollama.ChatCompletionRequest{
			Model:    a.meta.Model,
			Messages: messages,
			Options: map[string]interface{}{
				"temperature": req.Temperature,
				"num_ctx":     req.MaxToken,
			},
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&OllamaStreamResult{stream},
		}
	case "xinference":
		client := xinference.NewClient(a.meta.EndPoint, a.meta.APIVersion, a.meta.Model)
		var messages []xinference.ChatCompletionMessage
		for _, v := range req.Messages {
			messages = append(messages, xinference.ChatCompletionMessage{Role: v.Role, Content: v.Content})
		}
		req := xinference.ChatCompletionRequest{
			Model:       a.meta.Model,
			Messages:    messages,
			MaxTokens:   req.MaxToken,
			Temperature: req.Temperature,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{
			&XinferenceStreamResult{stream},
		}
	}

	return result, nil
}
