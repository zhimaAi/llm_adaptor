// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package registry

import (
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/ai302"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/ali"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/azure"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/baai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/baichuan"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/baidu"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/claude"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/cohere"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/deepseek"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/doubao"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/gemini"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/hunyuan"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/jina"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/lingyiwanwu"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/minimax"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/moonshot"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/ollama"
	openaiProvider "github.com/zhimaAi/llm_adaptor/v2/internal/providers/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/openaiagent"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/openrouter"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/siliconflow"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/spark"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/voyage"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/xinference"
	"github.com/zhimaAi/llm_adaptor/v2/internal/providers/zhipu"
)

var definitions = map[provider.ID]provider.Definition{
	provider.ID302AI:       ai302.Definition(),
	provider.IDAli:         ali.Definition(),
	provider.IDAzure:       azure.Definition(),
	provider.IDBAAI:        baai.Definition(),
	provider.IDBaichuan:    baichuan.Definition(),
	provider.IDBaidu:       baidu.Definition(),
	provider.IDClaude:      claude.Definition(),
	provider.IDCohere:      cohere.Definition(),
	provider.IDDeepSeek:    deepseek.Definition(),
	provider.IDDoubao:      doubao.Definition(),
	provider.IDGemini:      gemini.Definition(),
	provider.IDHunyuan:     hunyuan.Definition(),
	provider.IDJina:        jina.Definition(),
	provider.IDLingYiWanWu: lingyiwanwu.Definition(),
	provider.IDMiniMax:     minimax.Definition(),
	provider.IDMoonshot:    moonshot.Definition(),
	provider.IDOllama:      ollama.Definition(),
	provider.IDOpenAI:      openaiProvider.Definition(),
	provider.IDOpenAIAgent: openaiagent.Definition(),
	provider.IDOpenRouter:  openrouter.Definition(),
	provider.IDSiliconFlow: siliconflow.Definition(),
	provider.IDSpark:       spark.Definition(),
	provider.IDVoyage:      voyage.Definition(),
	provider.IDXinference:  xinference.Definition(),
	provider.IDZhipu:       zhipu.Definition(),
}

func Lookup(id provider.ID) (provider.Definition, bool) {
	definition, ok := definitions[id]
	return definition, ok
}

func IDs() []provider.ID {
	result := make([]provider.ID, 0, len(definitions))
	for id := range definitions {
		result = append(result, id)
	}
	return result
}
