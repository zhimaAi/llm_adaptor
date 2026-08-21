// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package embedding

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"testing"
)

func TestDataUnmarshalBase64Float32(t *testing.T) {
	payload := make([]byte, 8)
	binary.LittleEndian.PutUint32(payload[0:4], math.Float32bits(1.5))
	binary.LittleEndian.PutUint32(payload[4:8], math.Float32bits(-2.25))
	raw, _ := json.Marshal(map[string]any{"object": "embedding", "index": 0, "embedding": base64.StdEncoding.EncodeToString(payload)})
	var data Data
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	values, err := data.Embedding.Float64s()
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0] != 1.5 || values[1] != -2.25 {
		t.Fatalf("unexpected embedding: %#v", values)
	}
}

func TestDataRejectsInvalidBase64Embedding(t *testing.T) {
	for _, raw := range []string{
		`{"embedding":"%%%","index":0}`,
		`{"embedding":"AQ==","index":0}`,
	} {
		var data Data
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			t.Fatalf("failed to preserve base64 embedding %s: %v", raw, err)
		}
		if _, err := data.Embedding.Float64s(); err == nil {
			t.Fatalf("decoded invalid embedding %s", raw)
		}
	}
}
