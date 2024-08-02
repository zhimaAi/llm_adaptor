// Copyright © 2016- 2024 Sesame Network Technology all right reserved

package zhipu

import "github.com/zhimaAi/llm_adaptor/api/openai"

type Client struct {
	APIKey       string
	EndPoint     string
	OpenAIClient *openai.Client
}

func NewClient(APIKey string) *Client {
	return &Client{
		APIKey:   APIKey,
		EndPoint: "https://open.bigmodel.cn/api/paas/v4",
		OpenAIClient: &openai.Client{
			EndPoint: "https://open.bigmodel.cn/api/paas/v4",
			APIKey:   APIKey,
			ErrResp:  &openai.ErrorResponse{},
		},
	}
}
