package tests

import (
	"errors"
	"fmt"
	"github.com/zhimaAi/llm_adaptor/adaptor"
	"io"
	"os"
	"testing"
)

func testChatCompletionStream(Meta adaptor.Meta) {
	client := &adaptor.Adaptor{}
	client.Init(Meta)
	req := adaptor.ZhimaChatCompletionRequest{
		Messages:    []adaptor.ZhimaChatCompletionMessage{{Role: "user", Content: "你好"}},
		Temperature: 0.1,
		MaxToken:    10,
	}
	stream, err := client.CreateChatCompletionStream(req)
	if err != nil {
		panic(err.Error())
	}
	defer stream.Close()
	for {
		response, err := stream.Read()
		if errors.Is(err, io.EOF) {
			fmt.Println("\nStream finished")
			return
		}
		if err != nil {
			fmt.Printf("\nStream error: %v", err)
			return
		}
		fmt.Print(response.Result)
	}
}

func TestOpenAIChatCompletionStream(t *testing.T) {
	testChatCompletionStream(adaptor.Meta{
		Corp:   "openai",
		Model:  `gpt-3.5-turbo`,
		APIKey: os.Getenv(`OPENAI_KEY`),
	})
}

func TestMinimaxiChatCompletionStream(t *testing.T) {
	testChatCompletionStream(adaptor.Meta{
		Corp:   "minimax",
		Model:  `abab6.5s-chat`,
		APIKey: os.Getenv(`MINIMAX_KEY`),
	})
}
