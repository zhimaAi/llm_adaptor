# llm_adaptor v2

Go 语言多模型服务商适配包。v2 将调用方可见的请求和响应收敛为五类稳定公共契约，供应商私有字段只在适配器内部转换。

| 能力 | 公共协议模板 |
|---|---|
| Chat | OpenAI Chat Completions，额外保留 `StreamOptions`、`ReasoningContent` 和 `<think>...</think>` 兼容 |
| Embedding | OpenAI Embeddings |
| Image | OpenAI Images Generate / Edit |
| Rerank | Cohere Rerank v2 核心协议 |
| Speech | MiniMax T2A 与音色管理协议 |

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

所有 Provider 都允许显式传入 `BaseURL`。Gemini、阿里云和 Cohere 同时包含 OpenAI-compatible 与原生能力；需要通过自定义网关覆盖两类接口时，分别传入 `BaseURL` 和 `ServiceBaseURL`，两者都会完整保留自定义子路径。

- `ProviderOpenAIAgent` 是 v2 唯一的通用 OpenAI-compatible 入口，字符串值为 `openaiAgent`。`BaseURL` 和 `APIVersion` 必须由调用方传入，适配器会保留自定义子路径并幂等补齐版本路径；APIKey 可以为空。v2 删除了 `ProviderOpenCompatible`，不提供源码兼容别名。
- Azure 接受资源根地址或已经包含 `/openai/v1` 的地址，适配器幂等补齐 `/openai/v1`；请求使用 Body 中的 `model` 作为 Deployment Name，并通过 `api-key` Header 鉴权。`APIVersion` 对 Azure 不生效。
- `ProviderOpenAIAgent` 和 Xinference 使用 `APIVersion` 幂等补齐版本路径；Ollama 兼容传入服务根地址或已经包含 `/v1` 的地址。

适配器当前注册 25 个 Provider，其中 `ProviderVoyage` 仅提供 Embedding。具体能力以 `Client.ProviderInfo()` 返回的 `Capabilities` 为准。

## 项目结构

- 根包只提供 `Client`、配置、错误、APIKey 选择和五类服务门面。
- `chat`、`embedding`、`image`、`rerank`、`speech` 定义稳定的公共请求与响应。
- 每个模型服务商在 `internal/providers` 下拥有独立目录，并按能力拆分实现文件。
- OpenAI-compatible 协议复用位于 `internal/protocol/openai`；`DefaultSpec` 提供完整 OpenAI 默认路径、字段、Bearer 鉴权和响应处理，各 Provider 目录只声明与默认协议不同的字段、路径、鉴权或转换。HTTP、SSE、URL、Context、ExtraBody 和流状态由内部共享层实现。

新增服务商时应注册新的 Provider Definition，并复用公共协议 Spec；调用应用不需要依赖任何 `internal` 包。

APIKey 支持：

```text
key
key1,key2,key3
key1@10,key2@20,key3@30
```

每个请求按权重独立随机选择一个 Key；流式请求在整个生命周期中固定使用同一个 Key。空 Key 会跳过，重复 Key 的权重会累加，非法或小于等于零的权重按 1 处理。

不通过 `Client` 发起、但仍需复用相同 Key 池规则的代理请求，可以调用 `SelectAPIKey`：

```go
selected, err := llm.SelectAPIKey(llm.SelectAPIKeyRequest{
    Credentials: llm.CredentialConfig{APIKeys: "key1@20,key2@80"},
})
```

## HTTP 日志

初始化时注册全局回调，覆盖已有及新建客户端的所有能力，以及上传、下载等附属 HTTP 请求：

```go
llm.RegisterHTTPLogger(func(ctx context.Context, entry llm.HTTPLog) {
    // 在此接入应用现有日志系统，回调可能并发执行。
    log.Printf("%s %s status=%d duration=%s request=%s response=%s error=%v",
        entry.Method, entry.URL, entry.StatusCode, entry.Duration,
        entry.RequestBody, entry.ResponseBody, entry.Error)
})

// 停止记录新请求；已开始请求仍使用其开始时取得的回调。
llm.RegisterHTTPLogger(nil)
```

`HTTPLog` 包含 `RequestTime`、`URL`、`Method`、`RequestHeaders`、`RequestBody`、`ResponseTime`、`EndTime`、`Duration`、`StatusCode`、`ResponseHeaders`、`ResponseBody`、`ResponseBodyEOF` 和 `Error`。`ResponseTime` 为收到响应头的时间，`EndTime` 为正文读完、读失败或关闭的时间，`Duration` 为请求开始至 `EndTime` 的耗时。网络失败时响应时间为零、状态码为 0。

日志保存适配器完成字段转换、`ExtraBody` 覆盖及鉴权设置后的原始输入，以及解码前的 HTTP 响应正文。请求链接、请求头（含鉴权值）、响应头和正文均原样提供，不脱敏、不截断；二进制、multipart 和 SSE 正文使用 `[]byte`。自定义 Transport 或 HTTP 底层自动添加的字段不属于适配器输入快照。

响应边读边收集，不预读或等待完整响应才返回首包。每次 HTTP 往返在 EOF、读取失败或 `Close` 时同步回调一次；重定向产生多条记录。`ResponseBodyEOF` 只表示是否实际读至 HTTP EOF，提前关闭时正文仅包含已读取字节；SSE 收到协议结束标记后关闭但未读至 HTTP EOF，也可能为 `false`。调用方必须消费或关闭流，未关闭且未读完的响应尚不会回调。

默认不记录。未注册时不复制正文；注册后完整正文会保留在内存，存储和脱敏由回调负责。注册及回调替换支持并发，回调取得独立快照，panic 不影响模型调用。适配器复制传入的 `http.Client`，保留其 Transport、Timeout、Cookie Jar 和重定向策略。

## Chat

```go
resp, err := client.Chat.Create(ctx, &chat.CreateRequest{
    Model: "gpt-5",
    Messages: []chat.Message{
        {Role: chat.RoleUser, Content: chat.TextContent("你好")},
    },
})
```

公共常用请求字段包括模型、消息、采样参数、token 上限、停止词、工具、结构化输出、`ReasoningEffort` 和 `StreamOptions`。流式调用使用 `client.Chat.Stream`，返回的 chunk 可交给 `chat.Accumulator` 聚合。`StreamOptions` 未配置时默认请求 usage，只有显式设置 `IncludeUsage=false` 才关闭。

`ReasoningEffort` 采用 OpenAI 的 `none`、`minimal`、`low`、`medium`、`high`、`xhigh`、`max`。类型是可扩展字符串，空值不额外设置思考参数。平台明确接受的等级（包括平台自行映射的兼容值）原样发送，例如 DeepSeek 的 `medium` 和百炼 Qwen3.8 的 `high` 均保持调用方原值。只有明确拒绝的已知档位才向下取可用档位；低于最小可用档位或模型不可关闭时使用最低可用档位，非 `none` 不在适配器内降成关闭。未知非空值在支持等级的协议中透传，在仅支持开关的协议中视为开启。

不同平台按各自协议判断模型能力，不把原生服务商规则套用到托管平台：

| 平台及模型 | 最终参数 |
|---|---|
| DeepSeek | `thinking.type` 与原值 `reasoning_effort` |
| 百炼 Qwen3.8、DeepSeek V4、GLM 5 系列 | `enable_thinking` 与平台接受的原值 `reasoning_effort`；旧版托管 DeepSeek/GLM 的 minimal 使用最低可用档，none 仅关闭开关；GLM 5.3 仅接受 low/high/max，不能关闭 |
| 百炼 Kimi K3 | 阿里云直供使用 low/high/max（可关闭）；`kimi/kimi-k3` 仅支持 max |
| 百炼 MiniMax M3 | `thinking.type=adaptive/disabled`；其他旧模型维持 `enable_thinking` |
| Kimi K3 | 仅 `reasoning_effort`，low/high/max；不可关闭；K2 保留 `thinking.type` |
| 智谱 GLM 5.2 | `thinking.type` 与原值 `reasoning_effort`；5.3 仅接受 low/high/max，不能关闭；旧模型仅设置开关 |
| 火山 Seed 2、Seed 1.8、Seed 1.6-251015、DeepSeek V4、GLM 5 等支持等级的模型 | `thinking.type` 与原值 `reasoning_effort`；`ep-*` 端点由平台解析模型能力 |
| SiliconFlow DeepSeek V4、GLM 5.2 | `enable_thinking` 与 `reasoning_effort`；保留 low/medium/xhigh 等平台接受的别名，minimal 使用 high；none 仅关闭开关 |
| OpenRouter | 原值 `reasoning.effort` |
| OpenAI | `reasoning_effort`；已知不支持的档位按模型降档 |
| Claude | `thinking` 与 `output_config.effort`，按模型支持范围适配 |

只有发生档位变化或等级折叠的传参如下；未列出的组合保持原等级：

| 模型或协议能力 | 调用方值 | 最终传参 |
|---|---|---|
| Gemini 3.1 Pro | `none`、`minimal` | `reasoning_effort=low` |
| Gemini 3.1 Pro | `xhigh`、`max` | `reasoning_effort=high` |
| Gemini 3、Gemini 2.5 Pro | `none` | `reasoning_effort=minimal` |
| Gemini 2.5/3 | `xhigh`、`max` | `reasoning_effort=high` |
| Claude effort 模型 | `minimal` | `output_config.effort=low` |
| Claude 最高只支持 high | `xhigh`、`max` | `output_config.effort=high` |
| Claude 支持 max、但不支持 xhigh | `xhigh` | `output_config.effort=high` |
| 不可关闭 Claude | `none` | adaptive thinking，`output_config.effort=low` |
| Claude Sonnet 5.5 | `none` | `thinking.type=between_tools`，`output_config.effort=low` |
| 旧版 Claude | 任意非 `none` | enabled thinking，`budget_tokens=1024` |
| 仅支持开关的模型、Xinference、Ollama、百度等 | 任意非 `none` | 对应思考开关或 enabled 模式 |
| 混元 | `none` | `thinking.type=disabled` |
| 混元 | `minimal`、`xhigh`、`max` | 分别降档为 `low`、`high`、`high`；`low/medium/high` 与未知非空值原样发送为 `reasoning_effort` |
| MiniMax M3 | `none` / 其他非空值 | disabled / adaptive，并设置 `reasoning_split` |
| MiniMax M3.1 | `none`、`minimal` | adaptive，`reasoning_effort=low`，`reasoning_split=true`；其他等级原样发送 |
| 非 M3 MiniMax | `none` / 其他非空值 | 仅设置 `reasoning_split=false/true` |

MiniMax 的 `reasoning_split` 只控制思考内容拆分到 `reasoning_content` 或保留在 `<think>` 中，不代表是否关闭思考；M2 不可通过这个参数关闭思考。M3 可使用 `thinking.type=disabled`，M3.1 不可关闭。`ExtraBody` 仍可最终覆盖所有自动生成的字段。

DeepSeek、Kimi、百炼、智谱、火山和 SiliconFlow 的历史 assistant 消息会保留 `ReasoningContent`，用于要求回传思考内容的多轮及工具调用场景。

响应完整保留 OpenAI Chat Completions 的 ID、对象类型、模型、choices、工具调用、音频、引用、logprobs、usage、service tier 和 system fingerprint。`ReasoningContent` 是标准化兼容字段：供应商有原生 reasoning 时优先使用，否则从完整或跨 chunk 的 `<think>...</think>` 中抽取。OpenRouter 的 `reasoning`、`reasoning_content` 及 `reasoning_details` 文本或摘要会在普通和流式响应中统一映射到该字段。

## Embedding

请求字段为 `Model`、`Input`、`EncodingFormat`、`Dimensions` 和 `User`。响应遵循 OpenAI Embeddings 的 `Object`、`Data`、`Model` 和 `Usage`。

`Data.Embedding` 能原样保存浮点数组或 Base64 字符串；需要数值向量时显式调用：

```go
values, err := resp.Data[0].Embedding.Float64s()
```

Base64 按小端 `float32` 解码并转换为 `[]float64`。

## Image

纯文本生图使用 Generate：

```go
resp, err := client.Images.Generate(ctx, &image.GenerateRequest{
    Model:          "gpt-image-1.5",
    Prompt:         "一只在窗边晒太阳的橘猫",
    ResponseFormat: "b64_json",
    OutputFormat:   "png",
})
```

带参考图的编辑使用统一的文件结构；OpenAI multipart、供应商 data URL 等格式由 Provider 内部转换：

```go
file, err := os.Open("input.png")
if err != nil {
    return err
}
defer file.Close()

resp, err := client.Images.Edit(ctx, &image.EditRequest{
    Model:  "gpt-image-1.5",
    Prompt: "将背景替换为雪山",
    Images: []image.File{
        {
            Filename:    "input.png",
            ContentType: "image/png",
            Reader:      file,
        },
    },
})
```

调用方负责关闭文件 Reader；适配器读取但不关闭或回卷。Generate/Edit 分别提供 `Stream`/`EditStream`。响应完整保留 OpenAI Images 的 `Created`、`Background`、`Data`、`OutputFormat`、`Quality`、`Size` 和 token usage。请求 `b64_json` 而供应商只返回 URL 时，适配器会使用调用 Context 下载并转换；无法识别格式时，`OutputFormat` 为 `jpeg`。

## Rerank

公共请求只包含 Cohere v2 核心字段：`Model`、`Query`、`Documents []string` 和 `TopN`。响应包含 `ID`、`Results` 及完整 `Meta`；结果不携带文档副本，调用方通过 `Result.Index` 关联原始文档。

```go
resp, err := client.Rerank.Create(ctx, &rerank.CreateRequest{
    Model:     "rerank-v3.5",
    Query:     "退款需要多久",
    Documents: []string{"退款将在三天内到账", "如何修改头像"},
})
```

阿里、BAAI、Cohere、Jina、SiliconFlow 和 Xinference 的私有字段与响应均在 Provider 内部转换。

BAAI 的 APIKey 可选：不传时不发送鉴权 Header；传入时 Embedding 和 Rerank 都使用标准 `Authorization: Bearer <key>`。

## 不支持参数

公共请求允许调用方始终使用同一套字段。Provider 不支持的公共字段会被静默忽略，不会阻断请求；支持字段的非法值仍返回 `ErrInvalidRequest`。多模态 Part、Tool 或编辑图片中只有部分输入可用时，适配器保留可转换部分；过滤后没有必要输入时返回 `ErrInvalidRequest`，避免发送空消息或空编辑请求。

Chat、Embedding 和 Image 的通用 Builder 通过各 Provider 目录提供的 Spec 提取字段。OpenAI、Azure v1、OpenAI Agent、302.AI 和 OpenRouter 使用完整的 OpenAI 常用字段；Cohere、MiniMax、星火、Ollama 等兼容协议只发送其明确支持的字段。原生协议 Provider 继续使用自己的内部请求 DTO。所有 Provider 都不会直接把公共请求结构序列化为请求 Body。

`DefaultSpec` 的标准路径为 `/chat/completions`、`/embeddings`、`/images/generations` 和 `/images/edits`。Provider 只覆盖差异路径，例如 302.AI 使用 `/v1/chat/completions`、`/302/images/generations` 和 `/302/images/edits`；Image Generate/Edit 路径通过同一个成组配置入口设置，避免只覆盖其中一个操作。

所有带 `context.Context` 的公开服务入口都接受 `nil`，并通过公开的 `llm.NormalizeContext` 将其视为 `context.Background()`。调用方在进入适配器前还需要使用 Context 时，也可以显式调用该方法；需要主动取消或设置超时时，仍应传入自己的 Context。

## ExtraBody

五类 JSON 请求均保留 `ExtraBody map[string]any`，用于尚未纳入公共常用字段的高级参数。适配器先构造供应商内部请求，再注入 ExtraBody：

- ExtraBody 是请求 Body 的最终、最高优先级浅层覆盖；同名字段直接使用 ExtraBody 的值。
- 可覆盖公共字段、`stream` 以及适配器生成的 `enable_thinking`、`think`、`thinking`、`reasoning`、`reasoning_split`。
- 覆盖仅作用于 JSON Body，不影响 URL、Header 或鉴权；嵌套对象不会递归合并。
- 显式的 `nil` 会写入 JSON `null`。
- 请求、slice、map 和 ExtraBody 均不会被适配器修改。
- 供应商不接受扩展字段时返回供应商 API 错误。

公共字段被 Provider 忽略后，调用方仍可通过 ExtraBody 显式注入同名供应商字段；ExtraBody 不参与“不支持参数”的过滤。

原生协议可能约定不同的扩展层级：阿里图片的 ExtraBody 注入最终 `parameters`，可覆盖适配器从 `N` 等公共字段推导出的参数；其他当前请求默认覆盖最终 Body 顶层。

## MiniMax Speech

```go
client, err := llm.NewClient(llm.ClientConfig{
    Provider: llm.ProviderMiniMax,
    Credentials: llm.CredentialConfig{APIKeys: "MINIMAX_API_KEY"},
})

resp, err := client.Speech.Create(ctx, &speech.CreateRequest{
    Model: "speech-2.8-hd",
    Text:  "欢迎使用语音合成服务",
    VoiceSetting: &speech.VoiceSetting{
        VoiceID: "male-qn-qingse",
    },
    AudioSetting: &speech.AudioSetting{
        Format: speech.AudioFormatMP3,
    },
})
```

`client.Speech.Stream` 使用 MiniMax HTTP T2A 流式协议，可通过 `StreamOptions.ExcludeAggregatedAudio` 控制最后一个 chunk 是否省略聚合后的完整音频。MiniMax 音色管理通过 `client.Speech.ListVoices`、`UploadVoiceFile` 和 `CloneVoice` 调用，并与 T2A 一样按请求从 APIKey 池中随机选择 Key。完整的上传并克隆流程应使用 `CloneVoiceFromFiles`，它会固定同一个 Key，避免账号级 `file_id` 在后续克隆请求中失效。Speech 请求和响应直接覆盖 MiniMax 官方字段，包括 `aigc_watermark`、克隆校验参数、`TraceID`、`BaseResponse` 和 `ExtraInfo`。v2 不提供语音转写、语音翻译、WebSocket T2A 或异步长文本。

流式响应可通过 `speech.AudioStatusStreaming` 和 `speech.AudioStatusComplete` 判断合成状态。直接调用 `UploadVoiceFile` 时，`Purpose` 使用 `speech.VoiceFilePurposeVoiceClone` 或 `speech.VoiceFilePurposePromptAudio`。

## 能力说明

- OpenAI-compatible 驱动：OpenAI、302.AI、阿里百炼、百川、百度千帆、DeepSeek、豆包、Gemini、混元、零一万物、MiniMax、Moonshot、Ollama、OpenRouter、SiliconFlow、讯飞星火、Xinference、智谱、自建兼容服务等。
- 原生适配：Azure OpenAI、Anthropic Claude、阿里 Rerank/Image、Gemini Embedding、OpenRouter Image、MiniMax Speech。
- Rerank：阿里、BAAI、Cohere、Jina、SiliconFlow、Xinference。
- Similarity 已在 v2 删除。
- v2 不支持 SecretKey、AK/SK 或 APPID 鉴权。

具体供应商可用能力以 `Client.ProviderInfo()` 返回的 `Capabilities` 为准。
