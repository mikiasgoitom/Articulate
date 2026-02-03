package dto

import (
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

// UserResponse is the DTO for a user.
type UserResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	Role      string  `json:"role"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	AvatarURL *string `json:"avatar_url"`
	CreatedAt string  `json:"created_at"`
}

// LoginResponse is the DTO for a successful login.
type LoginResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

// converts an entity.User to a UserResponse DTO.
func ToUserResponse(user entity.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      string(user.Role),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	}
}

// MessageResponse is a generic response for success/error messages.
type MessageResponse struct {
	Message string `json:"message"`
}

// ErrorResponse is a response for errors.
type ErrorResponse struct {
	Error string `json:"error"`
}
