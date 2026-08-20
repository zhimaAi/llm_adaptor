// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestCredentialPoolNormalizesConfiguration(t *testing.T) {
	pool, err := newCredentialPoolWithRandom(CredentialConfig{
		APIKeys: " key1@2, ,key2@0,key1@3,key3@invalid ",
	}, func(max *big.Int) (*big.Int, error) {
		return big.NewInt(0), nil
	})
	if err != nil {
		t.Fatal(err)
	}

	keys := make([]string, 0, len(pool.items))
	weights := make([]string, 0, len(pool.items))
	for _, item := range pool.items {
		keys = append(keys, item.apiKey)
		weights = append(weights, item.weight.String())
	}
	if !reflect.DeepEqual(keys, []string{"key1", "key2", "key3"}) {
		t.Fatalf("unexpected keys: %#v", keys)
	}
	if !reflect.DeepEqual(weights, []string{"5", "1", "1"}) {
		t.Fatalf("unexpected weights: %#v", weights)
	}
	if pool.totalWeight.String() != "7" {
		t.Fatalf("unexpected total weight: %s", pool.totalWeight)
	}
}

func TestCredentialPoolRejectsEmptyConfiguration(t *testing.T) {
	_, err := newCredentialPool(CredentialConfig{APIKeys: " , , "})
	if !errors.Is(err, ErrInvalidAPIKeyConfig) {
		t.Fatalf("expected ErrInvalidAPIKeyConfig, got %v", err)
	}
}

func TestCredentialPoolSupportsArbitraryPrecisionWeight(t *testing.T) {
	const hugeWeight = "999999999999999999999999999999999999999999999999"
	pool, err := newCredentialPoolWithRandom(CredentialConfig{
		APIKeys: "key1@" + hugeWeight + ",key2",
	}, func(max *big.Int) (*big.Int, error) {
		return new(big.Int).Sub(max, big.NewInt(1)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := pool.selectCredential()
	if err != nil {
		t.Fatal(err)
	}
	if selected.apiKey != "key2" {
		t.Fatalf("expected key2 at the final interval, got %q", selected.apiKey)
	}
}

func TestCredentialPoolWeightedIntervals(t *testing.T) {
	tests := []struct {
		value int64
		want  string
	}{
		{value: 0, want: "key1"},
		{value: 1, want: "key1"},
		{value: 2, want: "key2"},
		{value: 4, want: "key2"},
		{value: 5, want: "key3"},
		{value: 9, want: "key3"},
	}
	for _, test := range tests {
		t.Run(test.want+big.NewInt(test.value).String(), func(t *testing.T) {
			pool, err := newCredentialPoolWithRandom(CredentialConfig{
				APIKeys: "key1@2,key2@3,key3@5",
			}, func(max *big.Int) (*big.Int, error) {
				return big.NewInt(test.value), nil
			})
			if err != nil {
				t.Fatal(err)
			}
			selected, err := pool.selectCredential()
			if err != nil {
				t.Fatal(err)
			}
			if selected.apiKey != test.want {
				t.Fatalf("value %d selected %q, want %q", test.value, selected.apiKey, test.want)
			}
		})
	}
}

func TestCredentialPoolPropagatesRandomFailure(t *testing.T) {
	wantErr := errors.New("random unavailable")
	pool, err := newCredentialPoolWithRandom(CredentialConfig{APIKeys: "key1,key2"}, func(max *big.Int) (*big.Int, error) {
		return nil, wantErr
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.selectCredential()
	if !errors.Is(err, ErrCredentialSelection) || !strings.Contains(err.Error(), wantErr.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
}
