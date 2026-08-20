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
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

func (d *Data) UnmarshalJSON(raw []byte) error {
	var value struct {
		Object    string          `json:"object"`
		Embedding json.RawMessage `json:"embedding"`
		Index     int             `json:"index"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	d.Object, d.Index = value.Object, value.Index
	if len(value.Embedding) == 0 || string(value.Embedding) == "null" {
		return fmt.Errorf("embedding value is empty")
	}
	if value.Embedding[0] != '"' {
		if err := json.Unmarshal(value.Embedding, &d.Embedding); err != nil {
			return err
		}
		if len(d.Embedding) == 0 {
			return fmt.Errorf("embedding value is empty")
		}
		return nil
	}
	var encoded string
	if err := json.Unmarshal(value.Embedding, &encoded); err != nil {
		return err
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("decode base64 embedding: %w", err)
	}
	if len(decoded) == 0 || len(decoded)%4 != 0 {
		return fmt.Errorf("base64 embedding byte length must be a positive multiple of 4")
	}
	d.Embedding = make([]float64, len(decoded)/4)
	for index := range d.Embedding {
		bits := binary.LittleEndian.Uint32(decoded[index*4 : index*4+4])
		d.Embedding[index] = float64(math.Float32frombits(bits))
	}
	return nil
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
