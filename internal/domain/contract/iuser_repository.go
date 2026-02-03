package contract

import (
	"context"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

type IUserRepository interface {
	CreateUser(ctx context.Context, user *entity.User) error
	GetUserByID(ctx context.Context, id string) (*entity.User, error)
	// GetUserByUsername retrieves a user by username.
	GetUserByUsername(ctx context.Context, username string) (*entity.User, error)
	// GetUserByEmail retrieves a user by email.
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	// UpdateUser updates an existing user and returns the updated user.
	UpdateUser(ctx context.Context, user *entity.User) (*entity.User, error)
	// UpdateUserPassword updates user's password by ID with the provided hashed password.
	UpdateUserPassword(ctx context.Context, id string, hashedPassword string) error
	// DeleteUser removes a user by ID.
	DeleteUser(ctx context.Context, id string) error
}
