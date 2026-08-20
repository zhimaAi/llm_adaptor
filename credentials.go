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

type SelectAPIKeyRequest struct {
	Credentials CredentialConfig `json:"credentials"`
}

type SelectAPIKeyResponse struct {
	APIKey         string `json:"api_key"`
	CredentialHint string `json:"credential_hint"`
}

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

// SelectAPIKey parses an APIKey pool and selects one key with the same weighted
// random strategy used by Client requests.
func SelectAPIKey(request SelectAPIKeyRequest) (SelectAPIKeyResponse, error) {
	pool, err := newCredentialPool(request.Credentials)
	if err != nil {
		return SelectAPIKeyResponse{}, err
	}
	selected, err := pool.selectCredential()
	if err != nil {
		return SelectAPIKeyResponse{}, err
	}
	return SelectAPIKeyResponse{APIKey: selected.apiKey, CredentialHint: selected.hint}, nil
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
