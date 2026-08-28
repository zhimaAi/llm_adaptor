// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package embedding

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
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
	Object    string         `json:"object"`
	Embedding EmbeddingValue `json:"embedding"`
	Index     int            `json:"index"`
}

type EmbeddingValue struct {
	Floats []float64
	Base64 *string
}

func FloatEmbedding(values []float64) EmbeddingValue {
	return EmbeddingValue{Floats: append([]float64(nil), values...)}
}

func Base64Embedding(value string) EmbeddingValue {
	return EmbeddingValue{Base64: &value}
}

func (v EmbeddingValue) MarshalJSON() ([]byte, error) {
	if v.Base64 != nil {
		return json.Marshal(*v.Base64)
	}
	return json.Marshal(v.Floats)
}

func (v *EmbeddingValue) UnmarshalJSON(raw []byte) error {
	if len(raw) == 0 || string(raw) == "null" {
		return fmt.Errorf("embedding value is empty")
	}
	if raw[0] != '"' {
		if err := json.Unmarshal(raw, &v.Floats); err != nil {
			return err
		}
		if len(v.Floats) == 0 {
			return fmt.Errorf("embedding value is empty")
		}
		v.Base64 = nil
		return nil
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return err
	}
	if encoded == "" {
		return fmt.Errorf("embedding value is empty")
	}
	v.Base64 = &encoded
	v.Floats = nil
	return nil
}

func (v EmbeddingValue) Float64s() ([]float64, error) {
	if len(v.Floats) > 0 {
		return append([]float64(nil), v.Floats...), nil
	}
	if v.Base64 == nil || *v.Base64 == "" {
		return nil, fmt.Errorf("embedding value is empty")
	}
	encoded := *v.Base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode base64 embedding: %w", err)
	}
	if len(decoded) == 0 || len(decoded)%4 != 0 {
		return nil, fmt.Errorf("base64 embedding byte length must be a positive multiple of 4")
	}
	values := make([]float64, len(decoded)/4)
	for index := range values {
		bits := binary.LittleEndian.Uint32(decoded[index*4 : index*4+4])
		values[index] = float64(math.Float32frombits(bits))
	}
	return values, nil
}

type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type CreateResponse struct {
	Object string `json:"object"`
	Data   []Data `json:"data"`
	Model  string `json:"model"`
	Usage  Usage  `json:"usage"`
}
