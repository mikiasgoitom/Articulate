package uuidgen

import (
	"github.com/google/uuid"
	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
)

// Generator implements the usecase.UUIDGenerator interface.
type Generator struct{}

// NewGenerator creates a new UUID generator.
func NewGenerator() contract.IUUIDGenerator {
	return &Generator{}
}

// NewUUID generates a new UUID.
func (g *Generator) NewUUID() string {
	return uuid.New().String()
}

// Ensure Generator implements the contract.IUUIDGenerator interface
var _ contract.IUUIDGenerator = (*Generator)(nil)
