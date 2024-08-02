package adaptor_tests

import (
	"fmt"
	"github.com/zhimaAi/llm_adaptor/adaptor"
	"testing"
)

func testChatCompletion(Meta adaptor.Meta) {
	client := &adaptor.Adaptor{}
	client.Init(Meta)
	req := adaptor.ZhimaChatCompletionRequest{
		Messages: []adaptor.ZhimaChatCompletionMessage{
			{
				Role:    "system",
				Content: "你现在是一个可爱的机器人",
			},
			{
				Role:    "user",
				Content: "给我讲个故事",
			},
		},
		Temperature: 0.1,
		MaxToken:    10,
	}
	res, err := client.CreateChatCompletion(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(res.Result)
}

func TestOpenAIChatCompletion(t *testing.T) {
	testChatCompletion(adaptor.Meta{
		Corp:   "openai",
		Model:  `gpt-3.5-turbo`,
		APIKey: `abcdefg`,
	})
}
