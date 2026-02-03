package randomgenerator

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
)

type RandomGenerator struct{}

// check if RandomGenerator is implementing contract/irandom_generator.go

func NewRandomGenerator() contract.IRandomGenerator {
	return &RandomGenerator{}
}

var _ (contract.IRandomGenerator) = (*RandomGenerator)(nil)

func (rg *RandomGenerator) GenerateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)

	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(b)

	return token, nil
}
