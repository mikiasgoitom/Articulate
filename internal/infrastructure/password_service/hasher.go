package passwordservice

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"

	"golang.org/x/crypto/bcrypt"
)

type Hasher struct{}

// check if IHasher was implemented at compile time
var _ contract.IHasher = (*Hasher)(nil)

func NewHasher() *Hasher {
	return &Hasher{}
}

func (h *Hasher) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 5)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (h *Hasher) ComparePasswordHash(password, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return fmt.Errorf("password verification failed")
		}
		return fmt.Errorf("failed to check password hash: %w", err)
	}
	return nil
}

func (h *Hasher) HashString(s string) string {
	// Use SHA256 for hashing tokens (not passwords)
	// This is more appropriate for long strings like JWT tokens
	if s == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", hash)
}

func (h *Hasher) CheckHash(s, hash string) bool {
	// Use SHA256 to compare token hashes
	expectedHash := h.HashString(s)
	return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(hash)) == 1
}
