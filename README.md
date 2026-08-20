# llm_adaptor v2

Go 语言多模型服务商适配包。v2 使用 OpenAI Chat Completions 作为主要统一契约，并将 Chat、Embedding、Rerank、Image 与 MiniMax Speech 作为一级能力。

## 安装

```bash
go get github.com/zhimaAi/llm_adaptor/v2
```

## 创建客户端

```go
client, err := llm.NewClient(llm.ClientConfig{
    Provider: llm.ProviderOpenAI,
    BaseURL:  "", // 留空使用供应商默认地址
    Credentials: llm.CredentialConfig{
        APIKeys: "key1@20,key2@80",
    },
})
```

所有 Provider 都允许显式传入 `BaseURL`。Gemini、阿里云和 Cohere 同时包含 OpenAI-compatible 与原生能力；需要通过自定义网关覆盖两类接口时，分别传入 `BaseURL` 和 `ServiceBaseURL`，两者都会完整保留自定义子路径。OpenAI Agent、Xinference 使用 `APIVersion` 补齐版本路径，Ollama 兼容传入服务根地址或已经包含 `/v1` 的地址。

APIKey 支持：

```text
key
key1,key2,key3
key1@10,key2@20,key3@30
```

每个请求按权重独立随机选择一个 Key；流式请求在整个生命周期中固定使用同一个 Key。空 Key 会跳过，重复 Key 的权重会累加，非法或小于等于零的权重按 1 处理。

## Chat

```go
resp, err := client.Chat.Create(ctx, &chat.CreateRequest{
    Model: "gpt-5",
    Messages: []chat.Message{
        {Role: chat.RoleUser, Content: chat.TextContent("你好")},
    },
})
```

流式调用使用 `client.Chat.Stream`，返回的 chunk 可交给 `chat.Accumulator` 聚合。没有原生 reasoning 字段时，v2 会兼容抽取 `<think>...</think>`。

## MiniMax Speech

```go
client, err := llm.NewClient(llm.ClientConfig{
    Provider: llm.ProviderMiniMax,
    Credentials: llm.CredentialConfig{APIKeys: "MINIMAX_API_KEY"},
})

resp, err := client.Speech.Create(ctx, &speech.CreateRequest{
    Model: "speech-2.8-hd",
    Text:  "欢迎使用 ChatWiki",
    VoiceSetting: &speech.VoiceSetting{
        VoiceID: "male-qn-qingse",
    },
    AudioSetting: &speech.AudioSetting{
        Format: speech.AudioFormatMP3,
    },
})
```

`client.Speech.Stream` 使用 MiniMax HTTP T2A 流式协议。v2 不提供语音转写、语音翻译、WebSocket T2A、异步长文本或音色克隆。

## 能力说明

- OpenAI-compatible 驱动：OpenAI、302.AI、阿里百炼、百川、百度千帆、DeepSeek、豆包、Gemini、混元、零一万物、MiniMax、Moonshot、Ollama、OpenRouter、SiliconFlow、讯飞星火、Xinference、智谱、自建兼容服务等。
- 原生适配：Azure OpenAI、Anthropic Claude、阿里 Rerank/Image、Gemini Embedding、OpenRouter Image、MiniMax Speech。
- Rerank：阿里、BAAI、Cohere、Jina、SiliconFlow、Xinference。
- Similarity 已在 v2 删除。
- v2 不支持 SecretKey、AK/SK 或 APPID 鉴权。

具体供应商可用能力以 `Client.ProviderInfo()` 返回的 `Capabilities` 为准。
