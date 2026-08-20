# llm_adaptor v2 与 ChatWiki 完整修复计划

## 总体方案

- v2 一级能力为 Chat（含多模态输入）、Embedding、Rerank、Image 和 Speech。
- Speech 首版仅实现 MiniMax T2A 普通与流式生成；Similarity 在 v2 废弃。
- 不保留旧 `adaptor` API，不支持 SecretKey、APPID、AK/SK 或旧签名认证。
- 公共调用统一使用结构体请求/响应与 `context.Context`，Provider 通过注册表和小能力接口分派。
- ChatWiki 仅为 OpenAI Agent、Azure、BAAI、Xinference、Ollama 传入用户 Endpoint；OpenRouter 仅允许内部系统代理。

## 公共 API 与 Endpoint

- 根包为 `llm`，能力包为 `chat`、`embedding`、`image`、`rerank`、`speech`。
- `ClientConfig.BaseURL` 对所有 Provider 开放，为空时使用 Provider 默认值。
- 混合协议供应商使用独立 `ServiceBaseURL`：
  - Gemini Chat 使用 BaseURL，Embedding 使用 ServiceBaseURL。
  - 阿里云 Chat/Embedding 使用 BaseURL，Rerank/Image 使用 ServiceBaseURL。
  - Cohere Chat/Embedding 使用 BaseURL，Rerank 使用 ServiceBaseURL。
- URL 拼接保留自定义子路径并清理重复斜杠；OpenAI Agent、Xinference 幂等补 APIVersion，Ollama 幂等补 `/v1`。
- 默认地址、版本、资源路径、Provider、Capability、Role、格式和错误码均使用常量。

## APIKey 概率池

`CredentialConfig.APIKeys` 支持：

```text
key
key1,key2,key3
key1@10,key2@20,key3@30
```

- 空 Key 跳过；重复 Key 合并并累加权重。
- 未填写、非法或小于等于 0 的权重修正为 1；权重使用任意精度整数且不设上限。
- 没有有效 Key 时返回 `ErrInvalidAPIKeyConfig`。
- 每个请求独立概率随机一次，流式生命周期固定同一 Key；不轮询、不自动换 Key 重试。
- 日志和错误只记录不可逆 Key 指纹。

## Context、流式与兼容修复

- 所有 HTTP、图片下载和内部请求使用调用方 Context 与注入的 HTTPClient。
- Stream.Close 幂等并取消内部 Context；Recv 保留标准 Context 错误。
- 流式解析在普通 chunk 前识别 OpenAI、Claude、图片和 Speech 错误事件并返回 `APIError`；错误只返回一次。
- 保留 `<think>...</think>` 抽取，原生 reasoning 优先，覆盖跨 chunk、EOF 和未闭合标签。
- Accumulator 按字段合并 usage，避免 Claude 的输入、输出 token 相互覆盖。
- Thinking 恢复 v1 的供应商及模型规则，扩展参数合并到现有 ExtraBody，不覆盖调用方字段。

## Embedding、Image 与 Speech

- Embedding 响应同时支持浮点数组和 Base64；Base64 按小端 float32 解码成 `[]float64`。
- Image 请求使用具名图片输入字段，不再依赖 ExtraBody 传递标准图片参数。
- Image 响应保留 Format/MIME；请求 b64_json 时，URL 响应使用 Context 下载并转换。
- 图片格式按 data URL MIME、HTTP Content-Type、URL 后缀、请求 OutputFormat 推导。
- OpenRouter 恢复通过 Chat Stream 读取 `delta.images` 的流式图片能力。
- MiniMax Speech 使用统一 APIKey 概率池，支持普通/流式 T2A、业务错误、取消和主动关闭。

## ChatWiki Endpoint 策略

| 供应商 | 新版行为 | 旧数据 |
|---|---|---|
| OpenAI Agent | 传入 E 和 APIVersion，最终为 `E/version/operation` | 完全兼容 |
| Azure | 保持 Azure deployment 路径和 api-version 查询参数 | 完全兼容 |
| BAAI | 保持 `E/v1/embeddings`、`E/v1/rerank` | 完全兼容 |
| Xinference | 传入 E 和 APIVersion | 完全兼容 |
| Ollama | 传入服务根地址，适配器补 `/v1` | Endpoint 兼容，服务需支持 OpenAI v1 |
| OpenRouter | 忽略用户 Endpoint，仅使用内部代理或适配器默认值 | 自定义地址不再生效 |

- 其他内置供应商不再从 ChatWiki 传 Endpoint，统一使用适配器默认值。
- 数据库旧 Endpoint 不删除，运行时和界面忽略，便于回滚。
- 配置测试与真实调用使用相同 Endpoint 策略。
- `ProviderOpenCompatible` 仍要求公共包调用方显式传 BaseURL，ChatWiki 暂不开放。

## ChatWiki 数据与展示

- `chat_ai_model_config.api_key` 由 `202608191900_expand_model_api_key.sql` 扩展为 text。
- 新增 `202608201100_expand_self_owned_model_api_key.sql`，将 `self_owned_model_config.api_key` 扩展为 text。
- 已提交迁移不修改；新增迁移不提供降回 varchar(1024) 的 Down，避免长数据丢失。
- 后端和管理端移除 1024 字符限制；保存时只有完全没有有效 Key 才拒绝。
- 配置测试逐个验证规范化后的有效 Key，错误只返回 Key 序号而不泄漏明文。
- secret_key、app_id、Region 等历史字段不删除、不迁移、不读取。
- `SupportList` 仅用于产品展示，`SupportedType` 表示真实调用能力；MiniMax Tts 同时进入两者。

## 验收与发布

- llm_adaptor 测试覆盖 Provider 注册、Endpoint、ServiceBaseURL、APIKey、Context、SSE 错误、Thinking、Embedding Base64、图片转换、OpenRouter 图片流、Claude usage 和 MiniMax Speech。
- ChatWiki 验证 Endpoint 白名单、OpenRouter 系统代理、历史 Endpoint 隐藏、两张表的 APIKey text 迁移、逐 Key 配置测试、图片及 Speech 调用。
- 删除 llm_adaptor `.gitignore` 中的 `*_test.go`，将测试正式纳入 Git。
- 两个仓库执行 `go test ./...` 和必要的 `go vet ./...`；原有非本任务 vet 问题单独记录。
- 修复完成后发布下一个 v2 RC，ChatWiki 升级并完成回归，再发布 `v2.0.0`。
