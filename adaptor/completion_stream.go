// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import (
	"errors"
	"strings"

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
	"github.com/zhimaAi/llm_adaptor/api/siliconflow"
	"github.com/zhimaAi/llm_adaptor/api/spark"
	"github.com/zhimaAi/llm_adaptor/api/volcenginev3"
	"github.com/zhimaAi/llm_adaptor/api/xinference"
	"github.com/zhimaAi/llm_adaptor/api/zhipu"
	"github.com/zhimaAi/llm_adaptor/define"
)

type ZhimaStreamResult interface {
	Read() (ZhimaChatCompletionResponse, error)
	Close() error
}

type ZhimaChatCompletionStreamResponse struct {
	ZhimaStreamResult
	extractor  thinkTagExtractor
	eofFlushed bool
}

func (r *ZhimaChatCompletionStreamResponse) Read() (ZhimaChatCompletionResponse, error) {
	return readThinkTaggedStream(r.ZhimaStreamResult, &r.extractor, &r.eofFlushed)
}

func (a *Adaptor) CreateChatCompletionStream(req ZhimaChatCompletionRequest) (*ZhimaChatCompletionStreamResponse, error) {
	if len(req.Messages) == 0 {
		return nil, errors.New("messages is required")
	}

	jsonStr, _ := tool.JsonEncodeIndent(req.Messages, ``, "\t")
	logs.Debug(`messages:%s`, jsonStr)
	jsonStr, _ = tool.JsonEncodeIndent(req.FunctionTools, ``, "\t")
	logs.Debug(`function_tools:%s`, jsonStr)
	a.meta.EndPoint = strings.TrimRight(strings.TrimSpace(a.meta.EndPoint), `/`)

	var result *ZhimaChatCompletionStreamResponse

	switch a.meta.Corp {
	case "openai", "302ai", "openrouter":
		client := openai.NewClient(GenerateOpenAiApiUrl(a), a.meta.APIKey, &openai.ErrorResponse{})
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
			Messages:    req.Messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxToken,
			Tools:       tools,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return nil, err
		}
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &OpenAIStreamResult{stream}}
	case "ali", "baichuan", "moonshot", "lingyiwanwu", "deepseek", "zhipu", "minimax", "openaiAgent", "siliconflow":
		var client *openai.Client
		if a.meta.Corp == "ali" {
			c := ali.NewClient(a.meta.APIKey)
			if len(a.meta.EndPoint) > 0 {
				c.EndPoint, c.OpenAIClient.EndPoint = GenerateClientEndPoint(a)
			}
			client = c.OpenAIClient
		} else if a.meta.Corp == "baichuan" {
			c := baichuan.NewClient(a.meta.APIKey)
			if len(a.meta.EndPoint) > 0 {
				c.EndPoint, c.OpenAIClient.EndPoint = GenerateClientEndPoint(a)
			}
			client = c.OpenAIClient
		} else if a.meta.Corp == "moonshot" {
			c := moonshot.NewClient(a.meta.APIKey)
			if len(a.meta.EndPoint) > 0 {
				c.EndPoint, c.OpenAIClient.EndPoint = GenerateClientEndPoint(a)
			}
			client = c.OpenAIClient
		} else if a.meta.Corp == "lingyiwanwu" {
			c := lingyiwanwu.NewClient(a.meta.APIKey)
			if len(a.meta.EndPoint) > 0 {
				c.EndPoint, c.OpenAIClient.EndPoint = GenerateClientEndPoint(a)
			}
			client = c.OpenAIClient
		} else if a.meta.Corp == "deepseek" {
			c := deepseek.NewClient(a.meta.APIKey)
			if len(a.meta.EndPoint) > 0 {
				c.EndPoint, c.OpenAIClient.EndPoint = GenerateClientEndPoint(a)
			}
			client = c.OpenAIClient
		} else if a.meta.Corp == "zhipu" {
			c := zhipu.NewClient(a.meta.APIKey)
			if len(a.meta.EndPoint) > 0 {
				c.EndPoint, c.OpenAIClient.EndPoint = GenerateClientEndPoint(a)
			}
			client = c.OpenAIClient
		} else if a.meta.Corp == "minimax" {
			c := minimax.NewClient(a.meta.APIKey)
			if len(a.meta.EndPoint) > 0 {
				c.EndPoint, c.OpenAIClient.EndPoint = GenerateClientEndPoint(a)
			}
			client = c.OpenAIClient
		} else if a.meta.Corp == "openaiAgent" {
			client = openaiagent.NewClient(a.meta.EndPoint, a.meta.APIKey, a.meta.APIVersion).OpenAIClient
		} else if a.meta.Corp == "siliconflow" {
			client = siliconflow.NewClient(a.meta.EndPoint, a.meta.APIKey, a.meta.APIVersion).OpenAIClient
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
			Messages:    req.Messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxToken,
			Tools:       tools,
		}
		if tool.InArrayString(a.meta.Corp, []string{`ali`, `siliconflow`}) && a.meta.ChoosableThinking {
			req.EnableThinking = &a.meta.EnabledThinking
		}
		if a.meta.Corp == `deepseek` {
			req.Thinking = &openai.Thinking{Type: openai.ThinkingTypeDisabled}
		}
		if client == nil {
			return &ZhimaChatCompletionStreamResponse{}, errors.New(`corp not supported`)
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &OpenAIStreamResult{stream}}
	case "azure":
		client := azure.NewClient(a.meta.EndPoint, a.meta.APIVersion, a.meta.APIKey, a.meta.Model)
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
			Messages:    req.Messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxToken,
			Tools:       tools,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &AzureStreamResult{stream}}
	case "baidu":
		client := baidu.NewClient(a.meta.APIKey, a.meta.SecretKey, a.meta.Model)
		var functions []baidu.Function
		var tools []interface{}
		if len(req.FunctionTools) > 0 {
			if client.ApiVersion == define.ApiVersionV2 {
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
			} else {
				for _, v := range req.FunctionTools {
					functions = append(functions, baidu.Function{
						Description: v.Description,
						Name:        v.Name,
						Parameters:  v.Parameters,
					})
				}
			}
		}
		messages, system := MessagesPopSystemRole(req.Messages)
		req := baidu.ChatCompletionRequest{
			Model:           client.Model,
			Messages:        messages,
			Stream:          true,
			Temperature:     req.Temperature,
			System:          system,
			MaxOutputTokens: req.MaxToken,
			Functions:       functions,
			Tools:           tools,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &BaiduStreamResult{stream}}
	case "claude":
		client := claude.NewClient(a.meta.APIKey)
		if len(a.meta.EndPoint) > 0 {
			client.EndPoint, _ = GenerateClientEndPoint(a)
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
		messages, system := MessagesPopSystemRole(req.Messages)
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
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &ClaudeStreamResult{stream}}
	case "gemini":
		client := gemini.NewClient(a.meta.APIKey, a.meta.Model)
		if len(a.meta.EndPoint) > 0 {
			client.EndPoint, _ = GenerateClientEndPoint(a)
		}
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
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &GeminiStreamResult{stream}}
	case "doubao":
		baseUrl := "https://ark.cn-beijing.volces.com/api/v3"
		if len(a.meta.EndPoint) > 0 {
			baseUrl, _ = GenerateClientEndPoint(a)
		}
		client := volcenginev3.NewClient(baseUrl, a.meta.Model, a.meta.APIKey, a.meta.SecretKey, a.meta.Region)
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
			Messages:    req.Messages,
			Temperature: req.Temperature,
			MaxTokens:   req.MaxToken,
			Tools:       tools,
		}
		if a.meta.ChoosableThinking {
			thinking := openai.Thinking{Type: openai.ThinkingTypeDisabled}
			if a.meta.EnabledThinking {
				thinking.Type = openai.ThinkingTypeEnabled
			}
			req.Thinking = &thinking
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return nil, err
		}
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &OpenAIStreamResult{stream}}
	case "cohere":
		client := cohere.NewClient(a.meta.APIKey)
		if len(a.meta.EndPoint) > 0 {
			client.EndPoint, _ = GenerateClientEndPoint(a)
		}

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
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &CohereStreamResult{stream}}
	case "spark":
		client := spark.NewClient(a.meta.APIKey, a.meta.APPID, a.meta.SecretKey, a.meta.Model)
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
					Text: req.Messages,
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
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &SparkStreamResult{stream}}
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
		if len(systemContent) > 0 {
			r.Messages = append(r.Messages, &tencentHunyuan.Message{
				Role:    common.StringPtr("system"),
				Content: common.StringPtr(systemContent),
			})
		}
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
		return &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &TencentStreamResult{stream}}, nil
	case "ollama":
		client := ollama.NewClient(a.meta.EndPoint, a.meta.Model)
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
		req := ollama.ChatCompletionRequest{
			Model:    a.meta.Model,
			Messages: req.Messages,
			Options: map[string]interface{}{
				"temperature": req.Temperature,
				"num_ctx":     req.MaxToken,
			},
			Tools: tools,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &OllamaStreamResult{stream}}
	case "xinference":
		client := xinference.NewClient(a.meta.EndPoint, a.meta.APIVersion, a.meta.Model)
		req := xinference.ChatCompletionRequest{
			Model:       a.meta.Model,
			Messages:    req.Messages,
			MaxTokens:   req.MaxToken,
			Temperature: req.Temperature,
		}
		stream, err := client.CreateChatCompletionStream(req)
		if err != nil {
			return &ZhimaChatCompletionStreamResponse{}, err
		}
		result = &ZhimaChatCompletionStreamResponse{ZhimaStreamResult: &XinferenceStreamResult{stream}}
	}

	return result, nil
}
