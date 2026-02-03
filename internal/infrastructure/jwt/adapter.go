package jwt

import (
	"github.com/google/uuid"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"github.com/mikiasgoitom/Articulate/internal/usecase"
)

// JWTServiceAdapter adapts JWTManager to the usecase.JWTService interface.
// It wraps JWTManager methods into the usecase-friendly interface.
type JWTServiceAdapter struct {
	mgr *JWTManager
}

// NewJWTService creates a new usecase.JWTService from JWTManager
func NewJWTService(mgr *JWTManager) usecase.JWTService {
	return &JWTServiceAdapter{mgr: mgr}
}

// GenerateAccessToken issues an access token for a user.
func (a *JWTServiceAdapter) GenerateAccessToken(userID string, role entity.UserRole) (string, error) {
	return a.mgr.GenerateAccessToken(userID, string(role))
}

// GenerateRefreshToken issues a refresh token for a user.
func (a *JWTServiceAdapter) GenerateRefreshToken(userID string, role entity.UserRole) (string, error) {
	tokenID := uuid.New().String()
	return a.mgr.GenerateRefreshToken(tokenID, userID)
}

// ParseAccessToken validates an access token and returns Claims.
func (a *JWTServiceAdapter) ParseAccessToken(tokenStr string) (*entity.Claims, error) {
	customClaims, err := a.mgr.VerifyToken(tokenStr)
	if err != nil {
		return nil, err
	}
	return &entity.Claims{
		UserID:           customClaims.Subject,
		Role:             entity.UserRole(customClaims.Role),
		RegisteredClaims: customClaims.RegisteredClaims,
	}, nil
}

// ParseRefreshToken validates a refresh token and returns Claims.
func (a *JWTServiceAdapter) ParseRefreshToken(tokenStr string) (*entity.Claims, error) {
	customClaims, err := a.mgr.VerifyRefreshToken(tokenStr)
	if err != nil {
		return nil, err
	}
	return &entity.Claims{
		UserID:           customClaims.Subject,
		RegisteredClaims: customClaims.RegisteredClaims,
	}, nil
}

// GeneratePasswordResetToken issues a password reset token.
func (a *JWTServiceAdapter) GeneratePasswordResetToken(userID string) (string, error) {
	return a.GenerateRefreshToken(userID, "")
}

// ParsePasswordResetToken validates a password reset token.
func (a *JWTServiceAdapter) ParsePasswordResetToken(tokenStr string) (*entity.Claims, error) {
	return a.ParseRefreshToken(tokenStr)
}

// GenerateEmailVerificationToken issues an email verification token.
func (a *JWTServiceAdapter) GenerateEmailVerificationToken(userID string) (string, error) {
	return a.GenerateRefreshToken(userID, "")
}

// ParseEmailVerificationToken validates an email verification token.
func (a *JWTServiceAdapter) ParseEmailVerificationToken(tokenStr string) (*entity.Claims, error) {
	return a.ParseRefreshToken(tokenStr)
}
