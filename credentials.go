// Copyright © 2016- 2025 Wuhan Sesame Small Customer Service Network Technology Co., Ltd.

package llm

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
)

const (
	APIKeyListSeparator   = ","
	APIKeyWeightSeparator = "@"
	DefaultAPIKeyWeight   = 1
)

type randomIntFunc func(max *big.Int) (*big.Int, error)

type credential struct {
	apiKey string
	weight *big.Int
	hint   string
}

type credentialPool struct {
	items       []credential
	totalWeight *big.Int
	randomInt   randomIntFunc
}

// CredentialEntry is a normalized API key and its arbitrary-precision weight.
type CredentialEntry struct {
	APIKey string `json:"api_key"`
	Weight string `json:"weight"`
}

// ParseCredentialConfig applies the same normalization used by NewClient.
func ParseCredentialConfig(config CredentialConfig) ([]CredentialEntry, error) {
	pool, err := newCredentialPoolWithRandom(config, func(max *big.Int) (*big.Int, error) {
		return new(big.Int), nil
	})
	if err != nil {
		return nil, err
	}
	entries := make([]CredentialEntry, 0, len(pool.items))
	for _, item := range pool.items {
		entries = append(entries, CredentialEntry{APIKey: item.apiKey, Weight: item.weight.String()})
	}
	return entries, nil
}

func newAnonymousCredentialPool() *credentialPool {
	return &credentialPool{
		items:       []credential{{weight: big.NewInt(DefaultAPIKeyWeight)}},
		totalWeight: big.NewInt(DefaultAPIKeyWeight),
		randomInt: func(max *big.Int) (*big.Int, error) {
			return new(big.Int), nil
		},
	}
}

func newCredentialPool(config CredentialConfig) (*credentialPool, error) {
	return newCredentialPoolWithRandom(config, func(max *big.Int) (*big.Int, error) {
		return rand.Int(rand.Reader, max)
	})
}

func newCredentialPoolWithRandom(config CredentialConfig, randomInt randomIntFunc) (*credentialPool, error) {
	weights := make(map[string]*big.Int)
	order := make([]string, 0)
	for _, rawItem := range strings.Split(config.APIKeys, APIKeyListSeparator) {
		apiKey, weight := parseCredential(strings.TrimSpace(rawItem))
		if apiKey == "" {
			continue
		}
		if current, exists := weights[apiKey]; exists {
			current.Add(current, weight)
			continue
		}
		weights[apiKey] = new(big.Int).Set(weight)
		order = append(order, apiKey)
	}
	if len(order) == 0 {
		return nil, ErrInvalidAPIKeyConfig
	}
	if randomInt == nil {
		return nil, fmt.Errorf("%w: random source is nil", ErrCredentialSelection)
	}

	pool := &credentialPool{
		items:       make([]credential, 0, len(order)),
		totalWeight: new(big.Int),
		randomInt:   randomInt,
	}
	for _, apiKey := range order {
		weight := weights[apiKey]
		pool.items = append(pool.items, credential{
			apiKey: apiKey,
			weight: new(big.Int).Set(weight),
			hint:   credentialHint(apiKey),
		})
		pool.totalWeight.Add(pool.totalWeight, weight)
	}
	return pool, nil
}

func parseCredential(value string) (string, *big.Int) {
	if value == "" {
		return "", nil
	}
	apiKey, weightValue, hasWeight := strings.Cut(value, APIKeyWeightSeparator)
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", nil
	}
	weight := big.NewInt(DefaultAPIKeyWeight)
	if !hasWeight {
		return apiKey, weight
	}
	parsedWeight, ok := new(big.Int).SetString(strings.TrimSpace(weightValue), 10)
	if !ok || parsedWeight.Sign() <= 0 {
		return apiKey, weight
	}
	return apiKey, parsedWeight
}

func (p *credentialPool) selectCredential() (credential, error) {
	if len(p.items) == 1 {
		return p.items[0], nil
	}
	value, err := p.randomInt(new(big.Int).Set(p.totalWeight))
	if err != nil {
		return credential{}, fmt.Errorf("%w: %v", ErrCredentialSelection, err)
	}
	if value == nil || value.Sign() < 0 || value.Cmp(p.totalWeight) >= 0 {
		return credential{}, fmt.Errorf("%w: random value outside range", ErrCredentialSelection)
	}
	for _, item := range p.items {
		if value.Cmp(item.weight) < 0 {
			return item, nil
		}
		value.Sub(value, item.weight)
	}
	return credential{}, fmt.Errorf("%w: no credential selected", ErrCredentialSelection)
}

func credentialHint(apiKey string) string {
	digest := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("sha256:%x", digest[:6])
}
