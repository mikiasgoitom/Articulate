package entity

import (
	"time"
)

// User represents a registered user in the system
type User struct {
	ID           string    `bson:"_id,omitempty" json:"id"`
	Username     string    `bson:"username" json:"username"`
	Email        string    `bson:"email" json:"email"`
	PasswordHash string    `bson:"password_hash" json:"-"`
	Role         UserRole  `bson:"role" json:"role"`
	IsActive     bool      `bson:"is_active" json:"is_active"`
	IsVerified   bool      `bson:"is_verified" json:"is_verified"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
	FirstName    *string   `bson:"firstname,omitempty" json:"firstname,omitempty"`
	LastName     *string   `bson:"lastname,omitempty" json:"lastname,omitempty"`
	AvatarURL    *string   `bson:"avatar_url,omitempty" json:"avatar_url,omitempty"`
}

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

func DefaultRole() UserRole {
	return UserRoleUser
}
