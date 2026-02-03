package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
	"golang.org/x/crypto/bcrypt"
)

// Constants for common error messages
const (
	errUserNotFound   = "user not found"
	errTokenNotFound  = "token not found"
	errInternalServer = "internal server error"
)

// UserUsecase implements the UserUseCase interface.
type UserUsecase struct {
	userRepo        contract.IUserRepository
	tokenRepo       contract.ITokenRepository
	emailUsecase    usecasecontract.IEmailVerificationUC
	hasher          contract.IHasher
	jwtService      JWTService
	mailService     contract.IEmailService
	logger          usecasecontract.IAppLogger
	config          usecasecontract.IConfigProvider
	validator       usecasecontract.IValidator
	uuidGenerator   contract.IUUIDGenerator
	randomGenerator contract.IRandomGenerator
}

// NewUserUsecase creates a new UserUsecase instance.
func NewUserUsecase(
	userRepo contract.IUserRepository,
	tokenRepo contract.ITokenRepository,
	emailUC usecasecontract.IEmailVerificationUC,
	hasher contract.IHasher,
	jwtService JWTService,
	mailService contract.IEmailService,
	logger usecasecontract.IAppLogger,
	cfg usecasecontract.IConfigProvider,
	validator usecasecontract.IValidator,
	uuidGenerator contract.IUUIDGenerator,
	randomgen contract.IRandomGenerator,
) *UserUsecase {
	return &UserUsecase{
		userRepo:        userRepo,
		tokenRepo:       tokenRepo,
		emailUsecase:    emailUC,
		hasher:          hasher,
		jwtService:      jwtService,
		mailService:     mailService,
		logger:          logger,
		config:          cfg,
		validator:       validator,
		uuidGenerator:   uuidGenerator,
		randomGenerator: randomgen,
	}
}

// check if UserUseCase implements the IUserUseCase
var _ usecasecontract.IUserUseCase = (*UserUsecase)(nil)

// Register handles user registration.
func (uc *UserUsecase) Register(ctx context.Context, username, email, password, firstName, lastName string) (*entity.User, error) {
	// Validate input fields using the injected validator
	if err := uc.validator.ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email format: %w", err)
	}
	if err := uc.validator.ValidatePasswordStrength(password); err != nil {
		return nil, fmt.Errorf("weak password: %w", err)
	}

	// Check if user with same username or email already exists
	existingUserByEmail, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil && err.Error() != errUserNotFound {
		uc.logger.Errorf("failed to check for existing user by email: %v", err)
		return nil, errors.New(errInternalServer)
	}
	if existingUserByEmail != nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	existingUserByUsername, err := uc.userRepo.GetUserByUsername(ctx, username)
	if err != nil && err.Error() != errUserNotFound {
		uc.logger.Errorf("failed to check for existing user by username: %v", err)
		return nil, errors.New(errInternalServer)
	}
	if existingUserByUsername != nil {
		return nil, fmt.Errorf("user with username %s already exists", username)
	}

	// Hash the password
	hashedPassword, err := uc.hasher.HashPassword(password)
	if err != nil {
		uc.logger.Errorf("failed to hash password: %v", err)
		return nil, fmt.Errorf("failed to process password")
	}

	// Initialize firstName and lastName as pointers, setting to nil if empty
	var pFirstName *string
	if firstName != "" {
		pFirstName = &firstName
	}
	var pLastName *string
	if lastName != "" {
		pLastName = &lastName
	}

	// Create new user entity, initializing new fields to their zero values or nil
	user := &entity.User{
		ID:           uc.uuidGenerator.NewUUID(),
		Username:     username,
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         entity.UserRoleUser,
		IsActive:     !uc.config.GetSendActivationEmail(), // Activate user immediately if email verification is off
		AvatarURL:    nil,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		FirstName:    pFirstName,
		LastName:     pLastName,
	}

	// Save user to database
	if err := uc.userRepo.CreateUser(ctx, user); err != nil {
		uc.logger.Errorf("failed to create user: %v", err)
		return nil, fmt.Errorf("failed to register user")
	}

	// Send activation email if required, using config from injected ConfigProvider
	if uc.config.GetSendActivationEmail() {
		// Generate email verification token
		if err = uc.emailUsecase.RequestVerificationEmail(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to send verification email")
		}
	}

	return user, nil
}

// Login handles user login and token generation.
func (uc *UserUsecase) Login(ctx context.Context, email, password string) (*entity.User, string, string, error) {
	// Retrieve user by username or email
	var user *entity.User
	var err error

	if uc.validator.ValidateEmail(email) == nil {
		user, err = uc.userRepo.GetUserByEmail(ctx, email)
	} else {
		user, err = uc.userRepo.GetUserByUsername(ctx, email)
	}

	if err != nil {
		if err.Error() == errUserNotFound {
			return nil, "", "", errors.New("invalid credentials")
		}
		uc.logger.Errorf("failed to retrieve user for login: %v", err)
		return nil, "", "", errors.New(errInternalServer)
	}

	// Check if the user's email is active/verified
	if !user.IsActive {
		return nil, "", "", errors.New("account not active. Please verify your email")
	}

	// Verify password
	if err := uc.hasher.ComparePasswordHash(password, user.PasswordHash); err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	// Generate access and refresh tokens
	accessToken, err := uc.jwtService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		uc.logger.Errorf("failed to generate access token: %v", err)
		return nil, "", "", errors.New("failed to generate token")
	}

	refreshToken, err := uc.jwtService.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		uc.logger.Errorf("failed to generate refresh token: %v", err)
		return nil, "", "", errors.New("failed to generate token")
	}

	refreshTokenExpiry := uc.config.GetRefreshTokenExpiry()
	if refreshTokenExpiry <= 0 {
		uc.logger.Errorf("invalid refresh token expiry configuration: %v", refreshTokenExpiry)
		return nil, "", "", errors.New("invalid refresh token expiry configuration")
	}

	// Create token entity with all fields from the schema
	tokenEntity := &entity.Token{
		ID:        uc.uuidGenerator.NewUUID(),
		UserID:    user.ID,
		TokenType: entity.TokenTypeRefresh,
		TokenHash: uc.hasher.HashString(refreshToken),
		ExpiresAt: time.Now().Add(refreshTokenExpiry),
		CreatedAt: time.Now(),
		Revoke:    false,
	}
	if err := uc.tokenRepo.CreateToken(ctx, tokenEntity); err != nil {
		uc.logger.Errorf("failed to store refresh token for user %s: %v", user.ID, err)
		return nil, "", "", errors.New("failed to store token")
	}

	return user, accessToken, refreshToken, nil
}

// Authenticate handles user authentication using access tokens.
func (uc *UserUsecase) Authenticate(ctx context.Context, accessToken string) (*entity.User, error) {
	claims, err := uc.jwtService.ParseAccessToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}

	user, err := uc.userRepo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		if err.Error() == errUserNotFound {
			return nil, errors.New("user not found")
		}
		uc.logger.Errorf("failed to retrieve user during authentication: %v", err)
		return nil, errors.New(errInternalServer)
	}

	return user, nil
}

// RefreshToken handles refreshing expired access tokens using refresh tokens.
func (uc *UserUsecase) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// Parse the refresh token to get the user claims.
	uc.logger.Infof("Debug: Attempting to parse refresh token")
	claims, err := uc.jwtService.ParseRefreshToken(refreshToken)
	if err != nil {
		uc.logger.Errorf("Debug: Failed to parse refresh token: %v", err)
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}
	uc.logger.Infof("Debug: Successfully parsed token for user: %s", claims.UserID)

	// The UserID from claims is already a string, so we can use it directly.
	// The UserID from claims is already a string, so we can use it directly.
	userID := claims.UserID

	// Retrieve the stored token using the parsed UUID.
	uc.logger.Infof("Debug: Looking up stored token for user: %s", userID)
	storedToken, err := uc.tokenRepo.GetTokenByUserID(ctx, userID)
	if err != nil {
		uc.logger.Errorf("Debug: Failed to retrieve stored token: %v", err)
		if err.Error() == "token not found" {
			return "", "", errors.New("refresh token not found or invalidated, please log in again")
		}
		uc.logger.Errorf("failed to retrieve stored refresh token: %v", err)
		return "", "", errors.New(errInternalServer)
	}
	uc.logger.Infof("Debug: Found stored token with hash length: %d", len(storedToken.TokenHash))

	// Check if the token has been revoked.
	if storedToken.Revoke {
		return "", "", errors.New("refresh token has been revoked, please log in again")
	}

	// Validate refresh token against the stored hash.
	uc.logger.Infof("Debug: Comparing tokens - provided token length: %d, stored hash length: %d", len(refreshToken), len(storedToken.TokenHash))
	if !uc.hasher.CheckHash(refreshToken, storedToken.TokenHash) {
		uc.logger.Warnf("refresh token mismatch for user %s", claims.UserID)
		uc.logger.Errorf("Debug: Token hash comparison failed")
		_ = uc.tokenRepo.RevokeToken(ctx, storedToken.ID) // Invalidate the stored token by revoking it
		return "", "", errors.New("invalid refresh token")
	}
	uc.logger.Infof("Debug: Token hash comparison successful")

	if storedToken.ExpiresAt.Before(time.Now()) {
		// Refresh token expired
		_ = uc.tokenRepo.RevokeToken(ctx, storedToken.ID) // revoke the expired token
		return "", "", errors.New("refresh token expired, please log in again")
	}

	// Generate new access token
	newAccessToken, err := uc.jwtService.GenerateAccessToken(claims.UserID, claims.Role)
	if err != nil {
		uc.logger.Errorf("failed to generate new access token during refresh: %v", err)
		return "", "", errors.New("failed to generate new access token")
	}

	// Generate a new refresh token
	newRefreshToken, err := uc.jwtService.GenerateRefreshToken(claims.UserID, claims.Role)
	if err != nil {
		uc.logger.Errorf("failed to generate new refresh token during refresh: %v", err)
		return "", "", errors.New("failed to generate new refresh token")
	}

	// Hash the new refresh token before storing it in the database.
	newHashedRefreshToken := uc.hasher.HashString(newRefreshToken)

	// Update the stored refresh token with the new hash and expiry.
	err = uc.tokenRepo.UpdateToken(ctx, storedToken.ID, newHashedRefreshToken, time.Now().Add(uc.config.GetRefreshTokenExpiry()))
	if err != nil {
		uc.logger.Errorf("failed to update refresh token in db: %v", err)
		return "", "", errors.New("failed to update token")
	}

	// Return both the new access token and the new refresh token.
	return newAccessToken, newRefreshToken, nil
}

// ForgotPassword handles the forgot password flow.
func (uc *UserUsecase) ForgotPassword(ctx context.Context, email string) error {
	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("email not found: %w", err)
	}

	// Generate a password reset token/link
	resetToken, err := uc.randomGenerator.GenerateRandomToken(32)
	if err != nil {
		return fmt.Errorf("failed to create password reset token: %w", err)
	}

	// Hash the token before storing it to match the schema
	hashedResetToken, err := bcrypt.GenerateFromPassword([]byte(resetToken), 7)
	if err != nil {
		return fmt.Errorf("failed to hash reset token: %w", err)
	}
	// generate verifier. it will be used to identify the reset token
	verifier, err := uc.randomGenerator.GenerateRandomToken(16)
	if err != nil {
		return fmt.Errorf("failed to generate verifier: %w", err)
	}

	// Store the token
	tokenEntity := &entity.Token{
		ID:        uc.uuidGenerator.NewUUID(),
		UserID:    user.ID,
		TokenType: entity.TokenTypePasswordReset,
		TokenHash: string(hashedResetToken),
		ExpiresAt: time.Now().Add(uc.config.GetPasswordResetTokenExpiry()),
		CreatedAt: time.Now(),
		Revoke:    false,
	}
	if err := uc.tokenRepo.CreateToken(ctx, tokenEntity); err != nil {
		uc.logger.Errorf("failed to store password reset token for user %s: %v", user.ID, err)
		return errors.New("failed to initiate password reset")
	}

	// The reset link should use the unhashed token
	emailSubject := "Password Reset Request"
	resetLink := fmt.Sprintf("%s/reset-password?verifier=%s&token=%s", uc.config.GetAppBaseURL(), verifier, resetToken)
	emailBody := fmt.Sprintf("Hi %s,\n\nYou have requested to reset your password. Please click the following link to reset your password: %s\n\nIf you did not request this, please ignore this email.\n\nThanks,\nThe Team", user.Username, resetLink)

	if err := uc.mailService.SendEmail(ctx, user.Email, emailSubject, emailBody); err != nil {
		uc.logger.Errorf("failed to send password reset email to %s: %v", user.Email, err)
		return errors.New("failed to send password reset email")
	}

	return nil
}

// ResetPassword handles the password reset flow using a password reset token.
func (uc *UserUsecase) ResetPassword(ctx context.Context, verifier, resetToken, newPassword string) error {
	// check if the token exists using the verifier
	token, err := uc.tokenRepo.GetTokenByVerifier(ctx, verifier)
	if err != nil {
		return fmt.Errorf("invalid verifier and token doesnt exist: %w", err)
	}

	// check if the token is expired
	if time.Now().After(token.ExpiresAt) {
		return fmt.Errorf("invalid token. it is expired")
	}
	// check if it is revoked
	if token.Revoke {
		return fmt.Errorf("invalid token. It is revoked")
	}

	// check if the token hash and plain rest token matchs
	if err = bcrypt.CompareHashAndPassword([]byte(token.TokenHash), []byte(resetToken)); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return fmt.Errorf("token doesnt match: %w", err)
		}
		return fmt.Errorf("failded to match the the hashed and plain token: %w", err)
	}

	// Hash the new password before updating the user.
	hashedPassword, err := uc.hasher.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %v", err)
	}

	// Update the user's password.
	if err = uc.userRepo.UpdateUserPassword(ctx, token.UserID, hashedPassword); err != nil {
		return fmt.Errorf("failed to update password for user %s: %v", token.UserID, err)
	}

	// revoke the password reset token
	if err = uc.tokenRepo.RevokeToken(ctx, token.ID); err != nil {
		return fmt.Errorf("failed to revoke reset password")
	}

	// Return success, confirming the change.
	return nil
}

// Logout handles user logout.
func (uc *UserUsecase) Logout(ctx context.Context, refreshToken string) error {
	// Parse the refresh token to get the user claims, which gives us the UserID.
	claims, err := uc.jwtService.ParseRefreshToken(refreshToken)
	if err != nil {
		uc.logger.Warnf("failed to parse refresh token on logout, assuming it's already invalid: %v", err)
		return nil
	}

	// Retrieve the stored token by UserID to get its database ID.
	storedToken, err := uc.tokenRepo.GetTokenByUserID(ctx, claims.UserID)
	if err != nil {
		if err.Error() == errTokenNotFound {
			uc.logger.Warnf("refresh token for user %s not found during logout, assuming it's already deleted", claims.UserID)
			return nil
		}
		uc.logger.Errorf("failed to retrieve stored refresh token for user %s: %v", claims.UserID, err)
		return errors.New(errInternalServer)
	}

	// Delete the token from the database.
	if err := uc.tokenRepo.RevokeToken(ctx, storedToken.ID); err != nil {
		uc.logger.Errorf("failed to revoke refresh token for user %s: %v", claims.UserID, err)
		return errors.New("failed to revoke token")
	}

	return nil
}

// PromoteUser promotes a user to an Admin role.
func (uc *UserUsecase) PromoteUser(ctx context.Context, userID string) (*entity.User, error) {
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if err.Error() == errUserNotFound {
			return nil, errors.New("user not found")
		}
		uc.logger.Errorf("failed to retrieve user for promotion: %v", err)
		return nil, errors.New(errInternalServer)
	}

	if user.Role == entity.UserRoleAdmin {
		return user, errors.New("user is already an admin")
	}

	user.Role = entity.UserRoleAdmin

	_, err = uc.userRepo.UpdateUser(ctx, user)
	if err != nil {
		uc.logger.Errorf("failed to promote user %s: %v", userID, err)
		return nil, errors.New("failed to promote user")
	}

	return user, nil
}

// DemoteUser demotes an Admin back to a regular user (member).
func (uc *UserUsecase) DemoteUser(ctx context.Context, userID string) (*entity.User, error) {
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if err.Error() == errUserNotFound {
			return nil, errors.New("user not found")
		}
		uc.logger.Errorf("failed to retrieve user for demotion: %v", err)
		return nil, errors.New(errInternalServer)
	}

	if user.Role == entity.UserRoleUser {
		return user, errors.New("user is already a regular member")
	}

	user.Role = entity.UserRoleUser

	_, err = uc.userRepo.UpdateUser(ctx, user)
	if err != nil {
		uc.logger.Errorf("failed to demote user %s: %v", userID, err)
		return nil, errors.New("failed to demote user")
	}

	return user, nil
}

// UpdateProfile allows a registered user to update their profile details.
func (uc *UserUsecase) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) (*entity.User, error) {
	uc.logger.Infof("UpdateProfile called for user %s with updates: %+v", userID, updates)

	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if err.Error() == errUserNotFound {
			return nil, errors.New("user not found")
		}
		uc.logger.Errorf("failed to retrieve user for profile update: %v", err)
		return nil, errors.New(errInternalServer)
	}

	uc.logger.Infof("Current user before update: %+v", user)

	// Check for username uniqueness if username is being updated
	if val, ok := updates["username"]; ok {
		if username, isString := val.(string); isString {
			existingUserByUsername, err := uc.userRepo.GetUserByUsername(ctx, username)
			if err != nil && err.Error() != errUserNotFound {
				uc.logger.Errorf("failed to check for existing username during update: %v", err)
				return nil, errors.New(errInternalServer)
			}
			if existingUserByUsername != nil && existingUserByUsername.ID != userID {
				return nil, fmt.Errorf("username %s already taken", username)
			}
		}
	}

	uc.logger.Infof("About to update user %s with updates: %+v", userID, updates)

	// Apply updates to user struct
	for k, v := range updates {
		switch k {
		case "username":
			if username, ok := v.(string); ok {
				user.Username = username
			}
		case "first_name":
			if firstName, ok := v.(string); ok {
				user.FirstName = &firstName
			}
		case "last_name":
			if lastName, ok := v.(string); ok {
				user.LastName = &lastName
			}
		case "avatar_url":
			if avatarURL, ok := v.(string); ok {
				user.AvatarURL = &avatarURL
			}
		case "is_active":
			if isActive, ok := v.(bool); ok {
				user.IsActive = isActive
			}
		}
	}
	user.UpdatedAt = time.Now()
	_, err = uc.userRepo.UpdateUser(ctx, user)
	if err != nil {
		uc.logger.Errorf("failed to update profile for user %s: %v", userID, err)
		return nil, errors.New("failed to update profile")
	}

	uc.logger.Infof("User %s updated successfully", userID)

	// Retrieve and return the updated user
	updatedUser, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		uc.logger.Errorf("failed to retrieve updated user: %v", err)
		return nil, errors.New("failed to retrieve updated user")
	}

	return updatedUser, nil
}

// login with OAuth2
func (uc *UserUsecase) LoginWithOAuth(ctx context.Context, firstName, lastName, email string) (string, string, error) {
	// Check if user with the given email already exists
	user, err := uc.userRepo.GetUserByEmail(ctx, email)
	if err != nil && err.Error() != errUserNotFound {
		uc.logger.Errorf("failed to check for existing user by email: %v", err)
		return "", "", errors.New(errInternalServer)
	}

	// If user does not exist, create a new one
	if user == nil {
		// Create a new user entity
		var pFirstName *string
		if firstName != "" {
			pFirstName = &firstName
		}
		var pLastName *string
		if lastName != "" {
			pLastName = &lastName
		}

		newUser := &entity.User{
			ID:           uc.uuidGenerator.NewUUID(),
			Username:     email, // Or generate a unique username
			Email:        email,
			PasswordHash: "", // No password for OAuth users
			Role:         entity.UserRoleUser,
			IsVerified:   true,
			IsActive:     true, // OAuth users are active by default
			AvatarURL:    nil,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			FirstName:    pFirstName,
			LastName:     pLastName,
		}

		// Save the new user to the database
		if err := uc.userRepo.CreateUser(ctx, newUser); err != nil {
			uc.logger.Errorf("failed to create user from OAuth: %v", err)
			return "", "", fmt.Errorf("failed to register user")
		}
		user = newUser
	}

	// At this point, we have a user (either existing or newly created)
	// Generate access and refresh tokens
	accessToken, err := uc.jwtService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		uc.logger.Errorf("failed to generate access token for OAuth user: %v", err)
		return "", "", errors.New("failed to generate token")
	}

	refreshToken, err := uc.jwtService.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		uc.logger.Errorf("failed to generate refresh token for OAuth user: %v", err)
		return "", "", errors.New("failed to generate token")
	}

	refreshTokenExpiry := uc.config.GetRefreshTokenExpiry()
	if refreshTokenExpiry <= 0 {
		uc.logger.Errorf("invalid refresh token expiry configuration: %v", refreshTokenExpiry)
		return "", "", errors.New("invalid refresh token expiry configuration")
	}

	// Create token entity
	tokenEntity := &entity.Token{
		ID:        uc.uuidGenerator.NewUUID(),
		UserID:    user.ID,
		TokenType: entity.TokenTypeRefresh,
		TokenHash: uc.hasher.HashString(refreshToken),
		ExpiresAt: time.Now().Add(refreshTokenExpiry),
		CreatedAt: time.Now(),
		Revoke:    false,
	}
	if err := uc.tokenRepo.CreateToken(ctx, tokenEntity); err != nil {
		uc.logger.Errorf("failed to store refresh token for OAuth user %s: %v", user.ID, err)
		return "", "", errors.New("failed to store token")
	}

	return accessToken, refreshToken, nil
}

func (uc *UserUsecase) GetUserByID(ctx context.Context, userID string) (*entity.User, error) {
	user, err := uc.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		if err.Error() == errUserNotFound {
			return nil, errors.New("user not found")
		}

		uc.logger.Errorf("failed to retrieve user by ID: %v", err)
		return nil, errors.New(errInternalServer)
	}

	return user, nil
}
