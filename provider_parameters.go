// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

type requestFieldSet map[string]struct{}

func requestFields(fields ...string) requestFieldSet {
	result := make(requestFieldSet, len(fields))
	for _, field := range fields {
		result[field] = struct{}{}
	}
	return result
}

func requestFieldsWithout(source requestFieldSet, fields ...string) requestFieldSet {
	result := make(requestFieldSet, len(source))
	for field := range source {
		result[field] = struct{}{}
	}
	for _, field := range fields {
		delete(result, field)
	}
	return result
}

var allOpenAIChatFields = requestFields(
	"frequency_penalty", "max_tokens", "max_completion_tokens", "n", "parallel_tool_calls",
	"presence_penalty", "response_format", "seed", "stop", "temperature", "tool_choice",
	"tools", "top_p", "user", "stream_options",
)

var chatProviderParameterFields = map[Provider]requestFieldSet{
	Provider302AI:          allOpenAIChatFields,
	ProviderAli:            allOpenAIChatFields,
	ProviderAzure:          allOpenAIChatFields,
	ProviderBaichuan:       requestFields("max_tokens", "stop", "temperature", "tools", "top_p", "stream_options"),
	ProviderBaidu:          requestFields("frequency_penalty", "max_tokens", "max_completion_tokens", "presence_penalty", "response_format", "stop", "temperature", "tool_choice", "tools", "top_p", "user", "stream_options"),
	ProviderClaude:         requestFields("max_tokens", "stop", "temperature", "tool_choice", "tools", "top_p"),
	ProviderCohere:         requestFields("frequency_penalty", "max_tokens", "presence_penalty", "response_format", "seed", "stop", "temperature", "tools", "top_p", "stream_options"),
	ProviderDeepSeek:       requestFields("frequency_penalty", "max_tokens", "presence_penalty", "response_format", "seed", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options"),
	ProviderDoubao:         allOpenAIChatFields,
	ProviderGemini:         allOpenAIChatFields,
	ProviderHunyuan:        requestFields("frequency_penalty", "max_tokens", "presence_penalty", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options"),
	ProviderLingYiWanWu:    requestFields("frequency_penalty", "max_tokens", "presence_penalty", "response_format", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options"),
	ProviderMiniMax:        requestFields("max_tokens", "max_completion_tokens", "temperature", "tools", "top_p", "stream_options"),
	ProviderMoonshot:       requestFields("frequency_penalty", "max_tokens", "n", "presence_penalty", "response_format", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options"),
	ProviderOllama:         requestFieldsWithout(allOpenAIChatFields, "max_completion_tokens", "parallel_tool_calls"),
	ProviderOpenAI:         allOpenAIChatFields,
	ProviderOpenCompatible: allOpenAIChatFields,
	ProviderOpenRouter:     allOpenAIChatFields,
	ProviderSiliconFlow:    requestFields("frequency_penalty", "max_tokens", "n", "presence_penalty", "response_format", "seed", "stop", "temperature", "tool_choice", "tools", "top_p", "stream_options"),
	ProviderSpark:          requestFields("frequency_penalty", "max_tokens", "presence_penalty", "temperature", "tools", "user"),
	ProviderXinference:     allOpenAIChatFields,
	ProviderZhipu:          allOpenAIChatFields,
}

var allOpenAIEmbeddingFields = requestFields("encoding_format", "dimensions", "user")

var embeddingProviderParameterFields = map[Provider]requestFieldSet{
	ProviderAli:            allOpenAIEmbeddingFields,
	ProviderAzure:          allOpenAIEmbeddingFields,
	ProviderBAAI:           requestFields(),
	ProviderBaichuan:       requestFields("encoding_format"),
	ProviderBaidu:          allOpenAIEmbeddingFields,
	ProviderCohere:         requestFields("encoding_format"),
	ProviderDoubao:         requestFields("encoding_format", "dimensions"),
	ProviderGemini:         requestFields("dimensions"),
	ProviderHunyuan:        requestFields("encoding_format"),
	ProviderJina:           requestFields("dimensions"),
	ProviderOllama:         allOpenAIEmbeddingFields,
	ProviderOpenAI:         allOpenAIEmbeddingFields,
	ProviderOpenCompatible: allOpenAIEmbeddingFields,
	ProviderSiliconFlow:    requestFields("encoding_format", "dimensions"),
	ProviderVoyage:         requestFields("encoding_format", "dimensions"),
	ProviderXinference:     allOpenAIEmbeddingFields,
	ProviderZhipu:          requestFields("dimensions"),
}

var allOpenAIImageFields = requestFields("n", "quality", "response_format", "size", "user", "output_format", "mask")

var imageProviderParameterFields = map[Provider]requestFieldSet{
	Provider302AI:          allOpenAIImageFields,
	ProviderAli:            requestFields("n", "response_format", "size", "output_format"),
	ProviderDoubao:         requestFields("n", "response_format", "size", "output_format"),
	ProviderOpenAI:         allOpenAIImageFields,
	ProviderOpenCompatible: allOpenAIImageFields,
	ProviderOpenRouter:     requestFields("size"),
}

func filterRequestFields(body map[string]any, supported, known requestFieldSet) {
	for field := range known {
		if _, ok := supported[field]; !ok {
			delete(body, field)
		}
	}
}
