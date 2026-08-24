// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zhimaAi/llm_adaptor/v2/internal/provider"
	"github.com/zhimaAi/llm_adaptor/v2/internal/transport"
	"github.com/zhimaAi/llm_adaptor/v2/rerank"
)

type rerankWireResult struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}
type rerankWireAPIVersion struct {
	Version        string `json:"version,omitempty"`
	IsDeprecated   *bool  `json:"is_deprecated,omitempty"`
	IsExperimental *bool  `json:"is_experimental,omitempty"`
}
type rerankWireUnits struct {
	InputTokens     *int     `json:"input_tokens,omitempty"`
	OutputTokens    *int     `json:"output_tokens,omitempty"`
	SearchUnits     *float64 `json:"search_units,omitempty"`
	Images          *int     `json:"images,omitempty"`
	Classifications *int     `json:"classifications,omitempty"`
}
type rerankWireTokens struct {
	InputTokens  *int `json:"input_tokens,omitempty"`
	OutputTokens *int `json:"output_tokens,omitempty"`
}
type rerankWireMeta struct {
	APIVersion   *rerankWireAPIVersion `json:"api_version,omitempty"`
	BilledUnits  *rerankWireUnits      `json:"billed_units,omitempty"`
	Tokens       *rerankWireTokens     `json:"tokens,omitempty"`
	CachedTokens *int                  `json:"cached_tokens,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
}
type rerankWireResponse struct {
	ID      string             `json:"id,omitempty"`
	Results []rerankWireResult `json:"results"`
	Meta    *rerankWireMeta    `json:"meta,omitempty"`
}

func (p *Provider) CreateRerank(ctx context.Context, selected provider.Credential, request *rerank.CreateRequest) (*rerank.CreateResponse, error) {
	if p.spec.RerankPath == "" {
		return nil, &provider.UnsupportedCapabilityError{Provider: p.spec.Info.ID, Capability: provider.CapabilityRerank}
	}
	if request == nil || strings.TrimSpace(request.Model) == "" || strings.TrimSpace(request.Query) == "" || len(request.Documents) == 0 {
		return nil, fmt.Errorf("%w: rerank model, query and documents are required", provider.ErrInvalidRequest)
	}
	documentsKey := p.spec.RerankDocumentsKey
	if documentsKey == "" {
		documentsKey = "documents"
	}
	body := map[string]any{"model": request.Model, "query": request.Query, documentsKey: append([]string(nil), request.Documents...)}
	if request.TopN != nil {
		topKey := p.spec.RerankTopKey
		if topKey == "" {
			topKey = "top_n"
		}
		body[topKey] = *request.TopN
	}
	for key, value := range request.ExtraBody {
		body[key] = value
	}
	endpoint := p.spec.RerankPath
	if p.spec.RerankBaseURL != "" {
		endpoint = transport.JoinURLPath(p.spec.RerankBaseURL, p.spec.RerankPath)
	}
	raw, err := p.DoJSON(ctx, selected, endpoint, body)
	if err != nil {
		return nil, err
	}
	var source rerankWireResponse
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, err
	}
	result := &rerank.CreateResponse{ID: source.ID, Results: make([]rerank.Result, len(source.Results))}
	for index, item := range source.Results {
		result.Results[index] = rerank.Result{Index: item.Index, RelevanceScore: item.RelevanceScore}
	}
	result.Meta = mapRerankMeta(source.Meta)
	normalizeRerankMeta(raw, result)
	return result, nil
}

func mapRerankMeta(source *rerankWireMeta) *rerank.Meta {
	if source == nil {
		return nil
	}
	result := &rerank.Meta{CachedTokens: source.CachedTokens, Warnings: append([]string(nil), source.Warnings...)}
	if source.APIVersion != nil {
		result.APIVersion = &rerank.APIVersion{Version: source.APIVersion.Version, IsDeprecated: source.APIVersion.IsDeprecated, IsExperimental: source.APIVersion.IsExperimental}
	}
	if source.BilledUnits != nil {
		result.BilledUnits = &rerank.Units{InputTokens: source.BilledUnits.InputTokens, OutputTokens: source.BilledUnits.OutputTokens, SearchUnits: source.BilledUnits.SearchUnits, Images: source.BilledUnits.Images, Classifications: source.BilledUnits.Classifications}
	}
	if source.Tokens != nil {
		result.Tokens = &rerank.Tokens{InputTokens: source.Tokens.InputTokens, OutputTokens: source.Tokens.OutputTokens}
	}
	return result
}

func normalizeRerankMeta(raw []byte, result *rerank.CreateResponse) {
	if result.Meta != nil {
		return
	}
	var source struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
		Meta struct {
			Tokens int `json:"tokens"`
		} `json:"meta"`
	}
	if json.Unmarshal(raw, &source) != nil {
		return
	}
	inputTokens, outputTokens := source.Usage.InputTokens, source.Usage.OutputTokens
	if inputTokens == 0 && outputTokens == 0 && source.Usage.TotalTokens > 0 {
		inputTokens = source.Usage.TotalTokens
	}
	if inputTokens == 0 && source.Meta.Tokens > 0 {
		inputTokens = source.Meta.Tokens
	}
	if inputTokens == 0 && outputTokens == 0 {
		return
	}
	result.Meta = &rerank.Meta{Tokens: &rerank.Tokens{}}
	if inputTokens > 0 {
		result.Meta.Tokens.InputTokens = &inputTokens
	}
	if outputTokens > 0 {
		result.Meta.Tokens.OutputTokens = &outputTokens
	}
}

var _ provider.Rerank = (*Provider)(nil)
