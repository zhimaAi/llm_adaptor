// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package rerank

type CreateRequest struct {
	Model     string         `json:"model"`
	Query     string         `json:"query"`
	Documents []string       `json:"documents"`
	TopN      *int           `json:"top_n,omitempty"`
	ExtraBody map[string]any `json:"-"`
}

type Result struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type APIVersion struct {
	Version        string `json:"version,omitempty"`
	IsDeprecated   *bool  `json:"is_deprecated,omitempty"`
	IsExperimental *bool  `json:"is_experimental,omitempty"`
}

type Units struct {
	InputTokens     *int     `json:"input_tokens,omitempty"`
	OutputTokens    *int     `json:"output_tokens,omitempty"`
	SearchUnits     *float64 `json:"search_units,omitempty"`
	Images          *int     `json:"images,omitempty"`
	Classifications *int     `json:"classifications,omitempty"`
}

type Tokens struct {
	InputTokens  *int `json:"input_tokens,omitempty"`
	OutputTokens *int `json:"output_tokens,omitempty"`
}

type Meta struct {
	APIVersion   *APIVersion `json:"api_version,omitempty"`
	BilledUnits  *Units      `json:"billed_units,omitempty"`
	Tokens       *Tokens     `json:"tokens,omitempty"`
	CachedTokens *int        `json:"cached_tokens,omitempty"`
	Warnings     []string    `json:"warnings,omitempty"`
}

type CreateResponse struct {
	ID      string   `json:"id,omitempty"`
	Results []Result `json:"results"`
	Meta    *Meta    `json:"meta,omitempty"`
}
