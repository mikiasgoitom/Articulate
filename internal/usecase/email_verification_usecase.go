package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"golang.org/x/crypto/bcrypt"
)

type EmailVerificationUseCase struct {
	tokenRepository contract.ITokenRepository
	userRepository  contract.IUserRepository
	emailService    contract.IEmailService
	RandomGenerator contract.IRandomGenerator
	UUIDGenerator   contract.IUUIDGenerator
	baseURL         string // Add baseURL for config
}

func NewEmailVerificationUseCase(tr contract.ITokenRepository, ur contract.IUserRepository, es contract.IEmailService, rg contract.IRandomGenerator, uuidgen contract.IUUIDGenerator, baseURL string) *EmailVerificationUseCase {
	return &EmailVerificationUseCase{
		tokenRepository: tr,
		userRepository:  ur,
		emailService:    es,
		RandomGenerator: rg,
		UUIDGenerator:   uuidgen,
		baseURL:         baseURL,
	}
}

func (eu *EmailVerificationUseCase) RequestVerificationEmail(ctx context.Context, user *entity.User) error {
	if err := eu.tokenRepository.RevokeAllTokensForUser(ctx, user.ID, entity.TokenTypeEmailVerification); err != nil {
		return fmt.Errorf("failed to revoke old tokens: %w", err)
	}

	plainToken, err := eu.RandomGenerator.GenerateRandomToken(32)
	if err != nil {
		return fmt.Errorf("failed to creating token: %w", err)
	}
	tokenHash, err := bcrypt.GenerateFromPassword([]byte(plainToken), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash token: %w", err)
	}
	verifier, err := eu.RandomGenerator.GenerateRandomToken(16)
	if err != nil {
		return fmt.Errorf("failed to creating verifier: %w", err)
	}
	newToken := entity.Token{
		ID:        eu.UUIDGenerator.NewUUID(),
		UserID:    user.ID,
		TokenType: entity.TokenTypeEmailVerification,
		TokenHash: string(tokenHash),
		Verifier:  verifier,
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
		CreatedAt: time.Now().UTC(),
		Revoke:    false,
	}
	if err = eu.tokenRepository.CreateToken(ctx, &newToken); err != nil {
		return fmt.Errorf("failed to create token in db: %w", err)
	}
	verificationLink := fmt.Sprintf("%s/api/v1/auth/verify-email?verifier=%s&token=%s", eu.baseURL, verifier, plainToken)
	emailSubject := "Verify your email address"
	emailBody := fmt.Sprintf("Hello %s\n, please click the following link to verify your email address: %s", user.Username, verificationLink)
	if err = eu.emailService.SendEmail(ctx, user.Email, emailSubject, emailBody); err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}
	return nil
}

func (eu *EmailVerificationUseCase) VerifyEmailToken(ctx context.Context, verifier, plainToken string) (*entity.User, error) {
	token, err := eu.tokenRepository.GetTokenByVerifier(ctx, verifier)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token or invalid token: %w", err)
	}
	// check it token isnt expired
	if time.Now().After(token.ExpiresAt) {
		eu.tokenRepository.RevokeToken(ctx, token.ID)
		return nil, fmt.Errorf("expired token")
	}
	// check if token is revoked
	if token.Revoke {
		return nil, fmt.Errorf("token has been revoked")
	}
	// chech if the plaintoken and the hashed token match
	if err = bcrypt.CompareHashAndPassword([]byte(token.TokenHash), []byte(plainToken)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return nil, fmt.Errorf("token doesnt match: %w", err)
		}
		return nil, fmt.Errorf("failed to match the hashed tokens with plain token: %w", err)
	}
	// get user by id
	user, err := eu.userRepository.GetUserByID(ctx, token.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	// check if user is already verified
	if user.IsVerified {
		return nil, fmt.Errorf("user is already verified")
	}
	if user.IsActive {
		return nil, fmt.Errorf("user is already verified")
	}
	user.IsActive = true
	user.IsVerified = true
	// update user
	if _, err = eu.userRepository.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user verification status: %w", err)
	}
	// revoke token
	if err = eu.tokenRepository.RevokeToken(ctx, token.ID); err != nil {
		return nil, fmt.Errorf("failed to revoke token after user is verified: %w", err)
	}
	return user, nil
}
