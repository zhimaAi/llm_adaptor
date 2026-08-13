// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package adaptor

import (
	"encoding/json"
	"errors"
	"regexp"
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
	"github.com/zhimaAi/llm_adaptor/basics"
	"github.com/zhimaAi/llm_adaptor/define"
)

type ZhimaChatCompletionMessage struct {
	Role             basics.RoleType `form:"role" json:"role"`
	Content          string          `form:"content" json:"content"`
	questionMultiple QuestionMultiple
	basics.MessageOther
}

func (m *ZhimaChatCompletionMessage) SetQuestionMultiple(questionMultiple QuestionMultiple) {
	m.Content = tool.JsonEncodeNoError(questionMultiple)
	m.questionMultiple = questionMultiple
}

type zhimaChatCompletionMessageReal struct {
	Role    basics.RoleType `form:"role" json:"role"`
	Content any             `form:"content" json:"content"`
	basics.MessageOther
}

func (m *ZhimaChatCompletionMessage) MarshalJSON() ([]byte, error) {
	message := zhimaChatCompletionMessageReal{
		Role:         m.Role,
		Content:      m.Content,
		MessageOther: m.MessageOther,
	}
	if len(m.questionMultiple) > 0 {
		message.Content = m.questionMultiple
	}
	return json.Marshal(message)
}

func MessagesPopSystemRole(messages []ZhimaChatCompletionMessage) ([]ZhimaChatCompletionMessage, string) {
	newMsgs, system := make([]ZhimaChatCompletionMessage, 0), ``
	for i := range messages {
		if messages[i].Role == basics.System {
			system += messages[i].Content
		} else {
			newMsgs = append(newMsgs, messages[i])
		}
	}
	return newMsgs, system
}

func (m *ZhimaChatCompletionMessage) UnmarshalJSON(data []byte) error {
	message := zhimaChatCompletionMessageReal{}
	if err := json.Unmarshal(data, &message); err != nil {
		return err
	}
	m.Role = message.Role
	if content, ok := message.Content.(string); ok {
		m.Content = content
	} else {
		bs, err := json.Marshal(message.Content)
		if err != nil {
			return err
		}
		m.Content = string(bs)
	}
	m.MessageOther = message.MessageOther
	return nil
}

type ZhimaChatCompletionRequest struct {
	Messages      []ZhimaChatCompletionMessage `json:"messages"`
	MaxToken      int                          `json:"max_token,omitzero"`
	Temperature   float64                      `json:"temperature,omitzero"`
	FunctionTools []FunctionTool               `json:"function_tools,omitzero"`
}

type FunctionTool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
}

type Parameters struct {
	Type       string      `json:"type"`
	Properties interface{} `json:"properties"`
	Required   []string    `json:"required"`
}

type ZhimaChatCompletionResponse struct {
	Result          string           `json:"result"`
	PromptToken     int              `json:"prompt_token"`
	CompletionToken int              `json:"completion_token"`
	ToolCalls       basics.ToolCalls `json:"tool_calls,omitempty"`
	// Deprecated: use ToolCalls directly for full metadata or ToolCalls.FunctionToolCalls for the legacy projection.
	FunctionToolCalls   []FunctionToolCall `json:"function_tool_calls"`
	IsValidFunctionCall bool               `json:"is_valid_function_call"`
	ReasoningContent    string             `json:"reasoning_content"`
}

type FunctionToolCall = basics.FunctionToolCall

func (a *Adaptor) CreateChatCompletion(req ZhimaChatCompletionRequest) (resp ZhimaChatCompletionResponse, err error) {
	defer func() {
		if err == nil {
			resp = normalizeThinkTaggedResponse(resp)
		}
	}()

	if len(req.Messages) == 0 {
		return ZhimaChatCompletionResponse{}, errors.New("messages is required")
	}

	jsonStr, _ := tool.JsonEncodeIndent(req.Messages, ``, "\t")
	logs.Debug(`messages:%s`, jsonStr)
	jsonStr, _ = tool.JsonEncodeIndent(req.FunctionTools, ``, "\t")
	logs.Debug(`function_tools:%s`, jsonStr)
	a.meta.EndPoint = strings.TrimRight(strings.TrimSpace(a.meta.EndPoint), `/`)
	switch a.meta.Corp {
	case "openai", "302ai", "openrouter":
		var tools []interface{}
		client := openai.NewClient(GenerateOpenAiApiUrl(a), a.meta.APIKey, &openai.ErrorResponse{})
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
		applyThinking(a.meta, &req)
		res, err := client.CreateChatCompletion(req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		toolCalls := res.Choices[0].Message.ToolCalls
		return ZhimaChatCompletionResponse{
			Result:            res.Choices[0].Message.Content,
			ReasoningContent:  res.Choices[0].Message.GetReasoningContent(),
			ToolCalls:         toolCalls,
			FunctionToolCalls: toolCalls.FunctionToolCalls(),
			PromptToken:       res.Usage.PromptTokens,
			CompletionToken:   res.Usage.CompletionTokens,
		}, nil
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
		applyThinking(a.meta, &req)
		if client == nil {
			return ZhimaChatCompletionResponse{}, errors.New(`corp not supported`)
		}
		res, err := client.CreateChatCompletion(req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		toolCalls := res.Choices[0].Message.ToolCalls
		return ZhimaChatCompletionResponse{
			Result:            res.Choices[0].Message.Content,
			ReasoningContent:  res.Choices[0].Message.GetReasoningContent(),
			ToolCalls:         toolCalls,
			FunctionToolCalls: toolCalls.FunctionToolCalls(),
			PromptToken:       res.Usage.PromptTokens,
			CompletionToken:   res.Usage.CompletionTokens,
		}, nil
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
		applyThinking(a.meta, &req)
		res, err := client.CreateChatCompletion(req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		toolCalls := res.Choices[0].Message.ToolCalls
		return ZhimaChatCompletionResponse{
			Result:            res.Choices[0].Message.Content,
			ReasoningContent:  res.Choices[0].Message.ReasoningContent,
			ToolCalls:         toolCalls,
			FunctionToolCalls: toolCalls.FunctionToolCalls(),
			PromptToken:       res.Usage.PromptTokens,
			CompletionToken:   res.Usage.CompletionTokens,
		}, nil
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
			Stream:          false,
			Temperature:     req.Temperature,
			System:          system,
			MaxOutputTokens: req.MaxToken,
			Functions:       functions,
			Tools:           tools,
		}
		applyThinking(a.meta, &req, client.ApiVersion)
		res, err := client.CreateChatCompletion(req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		var toolCalls basics.ToolCalls
		if client.ApiVersion == define.ApiVersionV2 && len(res.Choices) > 0 {
			res.Result = res.Choices[0].Message.Content
			res.ReasoningContent = res.Choices[0].Message.ReasoningContent
			if len(tools) > 0 && res.Result == "" {
				toolCalls = res.Choices[0].Message.ToolCalls
			}
		} else if strings.Contains(res.FunctionCall.Thoughts, `prompt`) {
			arguments := make(map[string]string)
			err := basics.JsonDecode([]byte(res.FunctionCall.Arguments), &arguments)
			if err != nil {
				return ZhimaChatCompletionResponse{}, err
			}
			for k := range arguments {
				arguments[k] = ``
			}
			res.FunctionCall.Arguments, _ = basics.JsonEncodeStr(arguments)

			re := regexp.MustCompile(`"prompt":\s*"([^"]*)"`)
			matches := re.FindStringSubmatch(res.FunctionCall.Thoughts)
			if len(matches) > 1 {
				res.Result = matches[1]
			} else {
				var argumentKeys []string
				for _, argumentKey := range arguments {
					argumentKeys = append(argumentKeys, argumentKey)
				}
				res.Result = `请提供必须参数: ` + strings.Join(argumentKeys, `, `)
			}
		}
		return ZhimaChatCompletionResponse{
			Result:            res.Result,
			ReasoningContent:  res.ReasoningContent,
			ToolCalls:         toolCalls,
			FunctionToolCalls: toolCalls.FunctionToolCalls(),
			PromptToken:       res.Usage.PromptTokens,
			CompletionToken:   res.Usage.CompletionTokens,
		}, nil
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
		applyThinking(a.meta, &req)
		res, err := client.CreateChatCompletion(req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		var toolCalls basics.ToolCalls
		for _, content := range res.Content {
			if content.Type != `tool_use` {
				continue
			}
			arguments, encodeErr := basics.JsonEncodeStr(content.Input)
			if encodeErr != nil {
				return ZhimaChatCompletionResponse{}, encodeErr
			}
			toolCalls = append(toolCalls, basics.NewFunctionToolCall(content.Id, content.Name, arguments))
		}
		result, reasoningContent := claudeTextAndThinking(res.Content)
		return ZhimaChatCompletionResponse{
			Result:            result,
			ReasoningContent:  reasoningContent,
			ToolCalls:         toolCalls,
			FunctionToolCalls: toolCalls.FunctionToolCalls(),
			PromptToken:       res.Usage.InputTokens,
			CompletionToken:   res.Usage.OutputTokens,
		}, nil
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
		applyThinking(a.meta, &req)
		res, err := client.CreateChatCompletion(req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		result, reasoningContent := geminiTextAndThinking(res.Candidates[0].Content.Parts)
		return ZhimaChatCompletionResponse{
			Result:           result,
			ReasoningContent: reasoningContent,
			PromptToken:      res.UsageMetadata.PromptTokenCount,
			CompletionToken:  geminiCompletionTokens(res.UsageMetadata),
		}, nil
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
		applyThinking(a.meta, &req)
		res, err := client.CreateChatCompletion(req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		toolCalls := res.Choices[0].Message.ToolCalls
		return ZhimaChatCompletionResponse{
			Result:            res.Choices[0].Message.Content,
			ReasoningContent:  res.Choices[0].Message.ReasoningContent,
			ToolCalls:         toolCalls,
			FunctionToolCalls: toolCalls.FunctionToolCalls(),
			PromptToken:       res.Usage.PromptTokens,
			CompletionToken:   res.Usage.CompletionTokens,
		}, nil
	case "cohere":
		client := cohere.NewClient(a.meta.APIKey)
		if len(a.meta.EndPoint) > 0 {
			client.EndPoint, _ = GenerateClientEndPoint(a)
		}

		cohereReq, useV2, err := buildCohereChatCompletionRequest(a.meta, req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		res, err := client.CreateChatCompletion(cohereReq)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		if !useV2 {
			return ZhimaChatCompletionResponse{
				Result:          res.Text,
				PromptToken:     res.Meta.Tokens.InputTokens,
				CompletionToken: res.Meta.Tokens.OutputTokens,
			}, nil
		}
		result, reasoningContent := cohereTextAndThinking(res.Message.Content)
		return ZhimaChatCompletionResponse{
			Result:           result,
			ReasoningContent: reasoningContent,
			PromptToken:      res.Usage.Tokens.InputTokens,
			CompletionToken:  res.Usage.Tokens.OutputTokens,
		}, nil
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
		applyThinking(a.meta, &req)
		if len(textFunctions) > 0 {
			//req.Payload.Functions = &spark.Function{Text: textFunctions}
		}
		res, err := client.CreateChatCompletion(req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		var toolCalls basics.ToolCalls
		if len(res.Payload.Choices.Text[0].FunctionCall.Name) > 0 {
			toolCalls = append(toolCalls, basics.NewFunctionToolCall("", res.Payload.Choices.Text[0].FunctionCall.Name, res.Payload.Choices.Text[0].FunctionCall.Arguments))
		}
		return ZhimaChatCompletionResponse{
			Result:            res.Payload.Choices.Text[0].Content,
			ReasoningContent:  res.Payload.Choices.Text[0].ReasoningContent,
			ToolCalls:         toolCalls,
			FunctionToolCalls: toolCalls.FunctionToolCalls(),
			PromptToken:       res.Payload.Usage.Text.PromptTokens,
			CompletionToken:   res.Payload.Usage.Text.CompletionTokens,
		}, nil
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
		applyThinking(a.meta, r)
		res, err := client.CreateChatCompletion(*r)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		return ZhimaChatCompletionResponse{
			Result:           *res.Choices[0].Message.Content,
			ReasoningContent: stringValue(res.Choices[0].Message.ReasoningContent),
			PromptToken:      int(*res.Usage.PromptTokens),
			CompletionToken:  int(*res.Usage.CompletionTokens),
		}, nil
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
			Tools:    tools,
			Options: map[string]interface{}{
				"temperature": req.Temperature,
				"num_ctx":     req.MaxToken,
			},
		}
		applyThinking(a.meta, &req)
		res, err := client.CreateChatCompletion(req)
		logs.Info("CreateChatCompletionStream:req:%v,res:%v,%v", req, res, err)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		if len(res.Message.ReasoningContent) == 0 { //兼容处理
			res.Message.ReasoningContent = res.Message.Thinking
		}
		toolCalls := res.Message.ToolCalls
		return ZhimaChatCompletionResponse{
			Result:            res.Message.Content,
			ReasoningContent:  res.Message.ReasoningContent,
			ToolCalls:         toolCalls,
			FunctionToolCalls: toolCalls.FunctionToolCalls(),
			PromptToken:       res.PromptEvalCount,
			CompletionToken:   res.EvalCount,
		}, nil
	case "xinference":
		client := xinference.NewClient(a.meta.EndPoint, a.meta.APIVersion, a.meta.Model)
		req := xinference.ChatCompletionRequest{
			Model:       a.meta.Model,
			Messages:    req.Messages,
			MaxTokens:   req.MaxToken,
			Temperature: req.Temperature,
		}
		applyThinking(a.meta, &req)
		res, err := client.CreateChatCompletion(req)
		if err != nil {
			return ZhimaChatCompletionResponse{}, err
		}
		return ZhimaChatCompletionResponse{
			Result:           res.Choices[0].Message.Content,
			ReasoningContent: res.Choices[0].Message.ReasoningContent,
			PromptToken:      res.Usage.PromptTokens,
			CompletionToken:  res.Usage.CompletionTokens,
		}, nil
	}

	return ZhimaChatCompletionResponse{}, nil
}

func GenerateOpenAiApiUrl(a *Adaptor) string {
	endPoint := "https://api.openai.com"
	switch a.meta.Corp {
	case "302ai":
		endPoint = "https://api.302ai.cn"
	case "openrouter":
		endPoint = "https://openrouter.ai/api"
	}
	if len(a.meta.EndPoint) > 0 {
		endPoint = a.meta.EndPoint
	}
	return endPoint + "/v1"
}

func GenerateClientEndPoint(a *Adaptor) (string, string) {
	switch a.meta.Corp {
	case "ali":
		return a.meta.EndPoint, a.meta.EndPoint + `/compatible-mode/v1`
	case "openai", "302ai", "baichuan", "moonshot", "lingyiwanwu", "gemini", "jina":
		return a.meta.EndPoint + `/v1`, a.meta.EndPoint + `/v1`
	case "zhipu":
		return a.meta.EndPoint + `/api/paas/v4`, a.meta.EndPoint + `/api/paas/v4`
	case "deepseek", "minimax", "baidu", "claude", "cohere", "ollama":
		return a.meta.EndPoint, a.meta.EndPoint
	case "doubao":
		return a.meta.EndPoint + `/api/v3`, ``
	}
	return ``, ``
}
