// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package ali

import (
	"context"
	"fmt"

	"github.com/zhimaAi/llm_adaptor/v2/internal/protocol/openai"
	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/shared"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
)

const (
	rerankPath           = "/api/v1/services/rerank/text-rerank/text-rerank"
	compatibleRerankPath = "/compatible-api/v1/reranks"
	qwen3RerankPrefix    = "qwen3-rerank"
)

type rerankWireResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

func (p *Provider) CreateRerank(ctx context.Context, selected provider.Credential, request *rerank.CreateRequest) (*rerank.CreateResponse, error) {
	if request == nil || request.Model == "" || request.Query == "" || len(request.Documents) == 0 {
		return nil, fmt.Errorf("%w: rerank model, query and documents are required", provider.ErrInvalidRequest)
	}
	parameters := map[string]any{}
	body := map[string]any{"model": request.Model}
	if openai.HasModelPrefix(request.Model, qwen3RerankPrefix) {
		body["query"] = request.Query
		body["documents"] = append([]string(nil), request.Documents...)
		if request.TopN != nil {
			body["top_n"] = *request.TopN
		}
	} else {
		if request.TopN != nil {
			parameters["top_n"] = *request.TopN
		}
		body["input"] = map[string]any{"query": request.Query, "documents": append([]string(nil), request.Documents...)}
		body["parameters"] = parameters
	}
	body, err := shared.MergeExtraBody(body, request.ExtraBody)
	if err != nil {
		return nil, err
	}
	path := rerankPath
	if openai.HasModelPrefix(request.Model, qwen3RerankPrefix) {
		path = compatibleRerankPath
	}
	raw, err := p.DoJSON(ctx, selected, transport.JoinURLPath(p.Config().ServiceBaseURL, path), body)
	if err != nil {
		return nil, err
	}
	var source struct {
		ID        string `json:"id"`
		RequestID string `json:"request_id"`
		Code      string `json:"code"`
		Message   string `json:"message"`
		Output    struct {
			Results []rerankWireResult `json:"results"`
		} `json:"output"`
		Results []rerankWireResult `json:"results"`
		Usage   struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := transport.DecodeJSONResponse(provider.IDAli, selected.Hint, raw, &source); err != nil {
		return nil, err
	}
	wireResults := source.Output.Results
	if len(wireResults) == 0 {
		wireResults = source.Results
	}
	if source.Code != "" && len(wireResults) == 0 {
		return nil, &provider.APIError{Provider: provider.IDAli, Code: source.Code, Message: source.Message, RequestID: source.RequestID, CredentialHint: selected.Hint, Raw: raw}
	}
	responseID := source.RequestID
	if responseID == "" {
		responseID = source.ID
	}
	response := &rerank.CreateResponse{ID: responseID, Results: make([]rerank.Result, len(wireResults))}
	for index, item := range wireResults {
		response.Results[index] = rerank.Result{Index: item.Index, RelevanceScore: item.RelevanceScore}
	}
	if source.Usage.TotalTokens != 0 {
		tokens := source.Usage.TotalTokens
		response.Meta = &rerank.Meta{Tokens: &rerank.Tokens{InputTokens: &tokens}}
	}
	return response, nil
}

var _ provider.Rerank = (*Provider)(nil)
