// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package embedding_test

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"

	"github.com/zhimaAi/llm_adaptor/v2/embedding"
)

func TestEmbeddingValuePublicFormats(t *testing.T) {
	payload := make([]byte, 8)
	binary.LittleEndian.PutUint32(payload[0:4], math.Float32bits(1.5))
	binary.LittleEndian.PutUint32(payload[4:8], math.Float32bits(-2.25))
	encoded := base64.StdEncoding.EncodeToString(payload)

	tests := []struct {
		name    string
		raw     string
		want    []float64
		wantErr bool
	}{
		{name: "float", raw: `{"embedding":[1.5,-2.25],"index":0}`, want: []float64{1.5, -2.25}},
		{name: "base64", raw: `{"embedding":"` + encoded + `","index":0}`, want: []float64{1.5, -2.25}},
		{name: "invalid base64", raw: `{"embedding":"%%%","index":0}`, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var data embedding.Data
			if err := json.Unmarshal([]byte(test.raw), &data); err != nil {
				t.Fatal(err)
			}
			values, err := data.Embedding.Float64s()
			if test.wantErr {
				if err == nil {
					t.Fatalf("decoded invalid embedding: %#v", values)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(values) != len(test.want) {
				t.Fatalf("values = %#v, want %#v", values, test.want)
			}
			for index := range values {
				if values[index] != test.want[index] {
					t.Fatalf("values = %#v, want %#v", values, test.want)
				}
			}
		})
	}
}
