package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mikiasgoitom/Articulate/internal/handler/http/dto"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

// UserHandlerInterface defines the methods for user handler to allow interface-based dependency injection (for testing/mocking)
type UserHandlerInterface interface {
	CreateUser(*gin.Context)
	Login(*gin.Context)
	GetUser(*gin.Context)
	GetCurrentUser(*gin.Context)
	UpdateUser(*gin.Context)
	ForgotPassword(*gin.Context)
	ResetPassword(*gin.Context)
	RefreshToken(*gin.Context)
	Logout(*gin.Context)
}

// Ensure UserHandler implements UserHandlerInterface
var _ UserHandlerInterface = (*UserHandler)(nil)

type UserHandler struct {
	userUsecase usecasecontract.IUserUseCase
}

func NewUserHandler(userUsecase usecasecontract.IUserUseCase) *UserHandler {
	return &UserHandler{
		userUsecase: userUsecase,
	}
}

// CreateUser handles user registration (signup)
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := BindAndValidate(c, &req); err != nil {
		return
	}

	_, err := h.userUsecase.Register(c.Request.Context(), req.Username, req.Email, req.Password, req.FirstName, req.LastName)
	if err != nil {
		ErrorHandler(c, http.StatusConflict, err.Error())
		return
	}

	MessageHandler(c, http.StatusCreated, "User created successfully. Please check your email to verify your account.")
}

// Login handles user authentication
func (h *UserHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := BindAndValidate(c, &req); err != nil {
		ErrorHandler(c, http.StatusBadRequest, "Bad Request credentials or unverified email")
		return
	}

	user, accessToken, refreshToken, err := h.userUsecase.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		ErrorHandler(c, http.StatusUnauthorized, "Invalid credentials or unverified email")
		return
	}

	response := dto.LoginResponse{
		User:         dto.ToUserResponse(*user),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	SuccessHandler(c, http.StatusOK, response)
}

// GetUser handles retrieving user by ID
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	user, err := h.userUsecase.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		ErrorHandler(c, http.StatusNotFound, "User not found")
		return
	}
	SuccessHandler(c, http.StatusOK, dto.ToUserResponse(*user))
}

// GetCurrentUser handles retrieving the current authenticated user
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		ErrorHandler(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	user, err := h.userUsecase.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		ErrorHandler(c, http.StatusNotFound, "User not found")
		return
	}
	SuccessHandler(c, http.StatusOK, dto.ToUserResponse(*user))
}

// UpdateUser handles updating user profile
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		ErrorHandler(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req dto.UpdateUserRequest
	if err := BindAndValidate(c, &req); err != nil {
		ErrorHandler(c, http.StatusBadRequest, "Invalid or Bad request")
		return
	}

	fmt.Printf("Request received: %+v\n", req)
	updates := updateUserRequestToMap(req)
	updatedUser, err := h.userUsecase.UpdateProfile(c.Request.Context(), userID.(string), updates)
	if err != nil {
		ErrorHandler(c, http.StatusBadRequest, err.Error())
		return
	}
	SuccessHandler(c, http.StatusOK, dto.ToUserResponse(*updatedUser))
}

// ForgotPassword handles password reset request
func (h *UserHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := BindAndValidate(c, &req); err != nil {
		ErrorHandler(c, http.StatusBadRequest, "Invalid or Bad request")

		return
	}

	err := h.userUsecase.ForgotPassword(c.Request.Context(), req.Email)
	if err != nil {
		// Don't reveal if email exists or not for security
		MessageHandler(c, http.StatusOK, "If an account with that email exists, a password reset link has been sent")
		return
	}

	MessageHandler(c, http.StatusOK, "If an account with that email exists, a password reset link has been sent")
}

// ResetPassword handles password reset with token
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := BindAndValidate(c, &req); err != nil {
		ErrorHandler(c, http.StatusBadRequest, "Invalid or Bad request")
		return
	}
	if req.Token == "" || req.Password == "" || req.Verifier == "" {
		ErrorHandler(c, http.StatusBadRequest, "Invalid or missing token/password/verifier")
		return
	}
	if len(req.Password) < 8 {
		ErrorHandler(c, http.StatusBadRequest, "Password must be at least 8 characters long")
		return
	}

	err := h.userUsecase.ResetPassword(c.Request.Context(), req.Verifier, req.Token, req.Password)
	if err != nil {
		ErrorHandler(c, http.StatusBadRequest, "Invalid or expired reset token")
		return
	}

	MessageHandler(c, http.StatusOK, "Password reset successfully")
}

// RefreshToken handles token refresh
func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorHandler(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.RefreshToken == "" {
		ErrorHandler(c, http.StatusBadRequest, "Refresh token required")
		return
	}

	newAccessToken, newRefreshToken, err := h.userUsecase.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		ErrorHandler(c, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	response := gin.H{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
	}

	SuccessHandler(c, http.StatusOK, response)
}

// Logout handles user logout
func (h *UserHandler) Logout(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorHandler(c, http.StatusBadRequest, "Invalid or missing refresh token")
		return
	}

	err := h.userUsecase.Logout(c.Request.Context(), req.RefreshToken)
	if err != nil {
		ErrorHandler(c, http.StatusInternalServerError, "Failed to logout")
		return
	}

	MessageHandler(c, http.StatusOK, "Logged out successfully")
}

func updateUserRequestToMap(req dto.UpdateUserRequest) map[string]interface{} {
	updates := make(map[string]interface{})

	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.FirstName != nil {
		updates["firstname"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["lastname"] = *req.LastName
	}
	if req.AvatarURL != nil {
		updates["avatarURL"] = *req.AvatarURL
	}

	return updates
}
