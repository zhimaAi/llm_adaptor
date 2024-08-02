// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package adaptor

import (
	"encoding/json"
	"github.com/zhimaAi/go_tools/tool"
	"github.com/zhimaAi/llm_adaptor/api/baidu"
	"regexp"
	"strings"
)

type BaiduStreamResult struct {
	*baidu.ChatCompletionStream
}

func (r *BaiduStreamResult) Read() (ZhimaChatCompletionResponse, error) {
	res, err := r.Recv()
	if err != nil {
		return ZhimaChatCompletionResponse{}, err
	}
	var functionToolCalls []FunctionToolCall
	if len(res.FunctionCall.Name) > 0 || len(res.FunctionCall.Arguments) > 0 {
		functionToolCalls = append(functionToolCalls, FunctionToolCall{
			Name:      res.FunctionCall.Name,
			Arguments: res.FunctionCall.Arguments,
		})
		if strings.Contains(res.FunctionCall.Thoughts, `prompt`) {
			arguments := make(map[string]string)
			err := json.Unmarshal([]byte(res.FunctionCall.Arguments), &arguments)
			if err != nil {
				return ZhimaChatCompletionResponse{}, err
			}
			for k, _ := range arguments {
				arguments[k] = ``
			}
			res.FunctionCall.Arguments, _ = tool.JsonEncode(arguments)

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
	}
	return ZhimaChatCompletionResponse{
		Result:            res.Result,
		FunctionToolCalls: functionToolCalls,
		PromptToken:       res.Usage.PromptTokens,
		CompletionToken:   res.Usage.CompletionTokens,
	}, nil
}
