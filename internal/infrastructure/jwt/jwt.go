package jwt

import (
	"fmt"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	Secret string
}

// compile time check if jwtmanager implements interface
var _ contract.IJWTManager = (*JWTManager)(nil)

func NewJWTManager(scrt string) *JWTManager {
	return &JWTManager{
		Secret: scrt,
	}
}

func (j *JWTManager) GenerateAccessToken(userID, userRole string) (string, error) {
	expirationTime := time.Now().Add(360 * time.Minute)

	claims := contract.CustomClaims{
		Role: userRole,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(j.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}
	return tokenString, nil
}

func (j *JWTManager) GenerateRefreshToken(tokenID, userID string) (string, error) {
	expirationTime := time.Now().Add(168 * time.Hour)
	claims := contract.RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        tokenID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(j.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}
	return tokenString, nil
}

func (j *JWTManager) VerifyToken(tokenString string) (*contract.CustomClaims, error) {
	claims := &contract.CustomClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse or validate token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (j *JWTManager) VerifyRefreshToken(tokenString string) (*contract.RefreshClaims, error) {
	claims := &contract.RefreshClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(j.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse or validate token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
