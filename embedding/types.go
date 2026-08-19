package embedding

import (
	"bytes"
	"encoding/json"
)

type Input struct {
	Text         *string
	Texts        []string
	Tokens       []int
	TokenBatches [][]int
}

func (i Input) MarshalJSON() ([]byte, error) {
	switch {
	case i.Text != nil:
		return json.Marshal(*i.Text)
	case i.Texts != nil:
		return json.Marshal(i.Texts)
	case i.Tokens != nil:
		return json.Marshal(i.Tokens)
	default:
		return json.Marshal(i.TokenBatches)
	}
}

func (i *Input) UnmarshalJSON(data []byte) error {
	var text string
	if json.Unmarshal(data, &text) == nil {
		i.Text = &text
		return nil
	}
	var texts []string
	if json.Unmarshal(data, &texts) == nil {
		i.Texts = texts
		return nil
	}
	var tokens []int
	if json.Unmarshal(data, &tokens) == nil {
		i.Tokens = tokens
		return nil
	}
	var batches [][]int
	if err := json.Unmarshal(bytes.TrimSpace(data), &batches); err != nil {
		return err
	}
	i.TokenBatches = batches
	return nil
}

type CreateRequest struct {
	Model          string         `json:"model"`
	Input          Input          `json:"input"`
	EncodingFormat string         `json:"encoding_format,omitempty"`
	Dimensions     *int           `json:"dimensions,omitempty"`
	User           string         `json:"user,omitempty"`
	ExtraBody      map[string]any `json:"-"`
}

type Data struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type CreateResponse struct {
	Object      string                     `json:"object"`
	Data        []Data                     `json:"data"`
	Model       string                     `json:"model"`
	Usage       Usage                      `json:"usage"`
	ExtraFields map[string]json.RawMessage `json:"-"`
	RawResponse json.RawMessage            `json:"-"`
}
