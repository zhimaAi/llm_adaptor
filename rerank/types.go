// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package rerank

import "encoding/json"

type Document struct {
	Text  string         `json:"text"`
	Extra map[string]any `json:"extra,omitempty"`
}

type CreateRequest struct {
	Model           string         `json:"model"`
	Query           string         `json:"query"`
	Documents       []Document     `json:"documents"`
	TopN            *int           `json:"top_n,omitempty"`
	ReturnDocuments *bool          `json:"return_documents,omitempty"`
	MaxChunksPerDoc *int           `json:"max_chunks_per_doc,omitempty"`
	ExtraBody       map[string]any `json:"-"`
}

type Result struct {
	Index          int       `json:"index"`
	RelevanceScore float64   `json:"relevance_score"`
	Document       *Document `json:"document,omitempty"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

type CreateResponse struct {
	ID          string                     `json:"id,omitempty"`
	Results     []Result                   `json:"results"`
	Usage       Usage                      `json:"usage,omitempty"`
	ExtraFields map[string]json.RawMessage `json:"-"`
	RawResponse json.RawMessage            `json:"-"`
}
