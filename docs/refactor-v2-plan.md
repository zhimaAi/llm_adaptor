# llm_adaptor v2 重构与 ChatWiki 迁移计划

## 总体方案

- 模块升级为 `github.com/zhimaAi/llm_adaptor/v2`，先发布 `v2.0.0-rc.1`，验证后发布 `v2.0.0`。
- 统一契约采用 OpenAI Chat Completions；供应商存在 OpenAI-compatible 接口时优先使用。
- 一级能力包含 Chat、多模态输入、Embedding、Rerank、Image 和 Speech；Similarity 在 v2 废弃。
- Speech 首版仅实现 MiniMax T2A HTTP 普通与流式生成；不实现转写、翻译、Realtime、WebSocket、异步长文本或音色克隆。
- 不保留旧 `adaptor` API、`Zhima*` 类型和 SecretKey/AKSK/APPID 鉴权链路。

## 公共 API

- 根包使用 `llm`，各能力使用独立包和小接口；供应商使用工厂注册表，移除按供应商分派的大型 `switch` 和连续 `if`。
- 网络接口统一为 `func(ctx context.Context, req *Request) (*Response, error)`；流式接口返回具名 Stream 结构。
- `ClientConfig` 包含 Provider、BaseURL、Credentials、APIVersion、HTTPClient、默认请求头和供应商扩展配置。
- BaseURL 由调用方配置，为空时使用供应商默认值；自建 OpenAI-compatible 服务必须显式配置。
- 自建稳定 DTO，完整保留 OpenAI 标准字段；请求提供 ExtraBody，响应保留 ExtraFields 和原始响应。
- 不支持的能力返回 `UnsupportedCapabilityError`；无法映射的个别参数按 best-effort 调用，并通过 `json:"-"` 的 SDK 元数据报告。
- 所有 Provider、Capability、Role、内容类型、工具类型、结束原因、格式、错误码、资源路径和默认 BaseURL 使用具名常量。

## APIKey 概率池

`CredentialConfig.APIKeys` 支持：

```text
key
key1,key2,key3
key1@10,key2@20,key3@30
```

- `,` 和 `@` 为保留分隔符；未填写权重时默认为 1。
- 空 Key 跳过；重复 Key 合并并累加权重；缺失、非法或小于等于 0 的权重修正为 1。
- 权重使用任意精度整数解析和累计；没有有效 Key 时 `NewClient` 返回 `InvalidAPIKeyConfigError`。
- 每个请求按权重独立随机选择一次；不使用轮询或全局游标。流式请求全生命周期固定同一个 Key。
- 不自动换 Key 重试、熔断或健康检查；日志和错误只记录 Key 序号及不可逆指纹。

## Context、流式与兼容

- 所有 HTTP、供应商 SDK、二次下载和内部子请求都使用调用方 Context，并复用注入的 HTTPClient。
- Stream.Close 幂等，负责取消内部 Context 和关闭响应体；Recv 保留标准 Context 错误。
- 保留 `<think>...</think>` 兼容：无原生 reasoning 时抽取，覆盖跨 chunk、EOF 和未闭合标签。
- 通用 Accumulator 负责合并文本、reasoning、usage 和多个工具调用。

## MiniMax Speech

- 一级接口为 `client.Speech.Create(ctx, *speech.CreateRequest)` 和 `client.Speech.Stream(ctx, *speech.StreamRequest)`。
- 请求完整结构化承载 model、text、language boost、voice setting、audio setting、pronunciation dictionary、voice modification、subtitle 和 MiniMax 扩展字段。
- 非流式响应保留 URL/hex 音频、状态、trace ID、音频元数据和原始响应。
- 流式响应按 chunk 返回 hex 音频、完成状态、字幕/音频元数据和 trace ID；Stream.Close 必须主动中止上游请求。
- 默认 BaseURL 为 `https://api.minimax.io/v1`，资源路径为 `/t2a_v2`；调用方 BaseURL 优先。
- 仅使用 Bearer API Key，并复用统一的 APIKey 概率池。

## ChatWiki 迁移

- `go.mod` 切换到 `/v2`，删除旧模块依赖；模型调用改用结构化 Request/Result，所有入口传递 Context，并删除未实际使用的 Similarity 调用路径。
- 删除流式 watchdog，改由 v2 Context 和 Stream.Close 控制；已输出业务数据后不切换备用模型。
- OpenAI 兼容开放接口复用 v2 Chat DTO；Eino 转换保留完整工具调用、reasoning、usage 和 finish reason。
- 数据库继续使用现有 `api_key` 字段保存概率池字符串，但通过新 goose migration 将 `chat_ai_model_config.api_key` 从 `varchar(1024)` 扩展为 PostgreSQL `text`；迁移只改变字段类型，不改写已有值。保存时只在没有有效 Key 时拒绝，并移除后端及管理端可能存在的 1024 字符限制。
- MiniMax 的 TTS 模型作为与 LLM、Embedding、Rerank、Image 同级的可用模型类型，接入 Speech 请求和计量日志。
- 旧 secret_key、app_id、region 字段不删除、不迁移，但运行时不再读取。

## 测试与发布

- 使用 httptest 覆盖所有公共 DTO、Provider 注册、BaseURL、Context、流式、错误、think 标签和未知字段保留。
- 凭据池覆盖空项、重复项、异常/超大权重、确定性随机区间、并发、流式固定 Key 和脱敏。
- Speech 覆盖 MiniMax 普通 URL/hex、HTTP 流式音频、字幕、业务错误、取消和主动关闭。
- ChatWiki 覆盖结构转换、SSE 生命周期、APIKey text 字段迁移、超长配置保存、逐 Key 配置测试、MiniMax TTS 调用和计量。
- 在当前 v0 基线上创建归档标签，完成 v2 后发布 RC；通过 mock、全量 Go 测试和真实凭据 smoke test 后发布正式版。
