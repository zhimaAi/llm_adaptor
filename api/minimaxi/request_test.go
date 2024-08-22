package minimaxi

import (
	"errors"
	"fmt"
	"github.com/zhimaAi/llm_adaptor/api/openai"
	"io"
	"os"
	"testing"
)

func TestCompletion(t *testing.T) {
	key := os.Getenv("MINIMAXI_KEY")
	client := NewClient(key)
	req := openai.ChatCompletionRequest{
		Model:    `abab6.5s-chat`,
		Messages: []openai.ChatCompletionRequestMessage{{Role: "user", Content: "你好"}},
	}
	res, err := client.OpenAIClient.CreateChatCompletion(req)
	if err != nil {
		panic(err.Error())
	}
	println(res.Choices[0].Message.Content)
}

func TestCompletionStream(t *testing.T) {
	key := os.Getenv("MINIMAXI_KEY")
	client := NewClient(key)
	req := openai.ChatCompletionRequest{
		Model:    `abab6.5s-chat`,
		Messages: []openai.ChatCompletionRequestMessage{{Role: "user", Content: "你好,给我讲一个300字的小故事吧"}},
	}
	stream, err := client.OpenAIClient.CreateChatCompletionStream(req)
	if err != nil {
		panic(err.Error())
	}
	defer stream.Close()
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("\nStream finished")
			return
		}
		if err != nil {
			fmt.Printf("\nStream error: %v", err)
			return
		}
		fmt.Print(response.Choices[0].Delta.Content)
	}
}
