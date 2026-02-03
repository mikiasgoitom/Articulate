package mocks

import (
	"context"
	"errors"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

// MockUserUsecase is a mock implementation of the UserUsecase interface
type MockUserUsecase struct {
	// Control mock behavior
	ShouldFailCreateUser     bool
	ShouldFailVerifyEmail    bool
	ShouldFailLogin          bool
	ShouldFailGetByID        bool
	ShouldFailUpdateUser     bool
	ShouldFailForgotPassword bool
	ShouldFailResetPassword  bool
	ShouldFailRefreshToken   bool
	ShouldFailLogout         bool
	ShouldFailAuthenticate   bool
	ShouldFailPromoteUser    bool
	ShouldFailDemoteUser     bool
	ShouldFailLoginWithOAuth bool

	// Return values
	MockUser         entity.User
	MockAccessToken  string
	MockRefreshToken string
}

// Ensure MockUserUsecase implements the correct interface for handler.NewUserHandler
var _ usecasecontract.IUserUseCase = (*MockUserUsecase)(nil)

func NewMockUserUsecase() *MockUserUsecase {
	return &MockUserUsecase{
		MockUser: entity.User{
			ID:       "mock-user-id",
			Username: "testuser",
			Email:    "test@example.com",
			Role:     entity.UserRoleUser,
		},
		MockAccessToken:  "mock_access_token",
		MockRefreshToken: "mock_refresh_token",
	}
}

func (m *MockUserUsecase) Register(ctx context.Context, username, email, password, firstName, lastName string) (*entity.User, error) {
	if m.ShouldFailCreateUser {
		return nil, errors.New("user creation failed")
	}
	return &m.MockUser, nil
}

func (m *MockUserUsecase) VerifyEmail(ctx context.Context, token string) error {
	if m.ShouldFailVerifyEmail {
		return errors.New("email verification failed")
	}
	return nil
}

func (m *MockUserUsecase) Login(ctx context.Context, email, password string) (*entity.User, string, string, error) {
	if m.ShouldFailLogin {
		return nil, "", "", errors.New("login failed")
	}
	return &m.MockUser, m.MockAccessToken, m.MockRefreshToken, nil
}

func (m *MockUserUsecase) GetUserByID(ctx context.Context, userID string) (*entity.User, error) {
	if m.ShouldFailGetByID {
		return nil, errors.New("user not found")
	}
	return &m.MockUser, nil
}

func (m *MockUserUsecase) UpdateProfile(ctx context.Context, userID string, updates map[string]interface{}) (*entity.User, error) {
	if m.ShouldFailUpdateUser {
		return nil, errors.New("update user failed")
	}
	return &m.MockUser, nil
}

func (m *MockUserUsecase) ForgotPassword(ctx context.Context, email string) error {
	if m.ShouldFailForgotPassword {
		return errors.New("forgot password failed")
	}
	return nil
}

func (m *MockUserUsecase) ResetPassword(ctx context.Context, verifier, token, password string) error {
	if m.ShouldFailResetPassword {
		return errors.New("reset password failed")
	}
	return nil
}

func (m *MockUserUsecase) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	if m.ShouldFailRefreshToken {
		return "", "", errors.New("refresh token failed")
	}
	return m.MockAccessToken, m.MockRefreshToken, nil
}

func (m *MockUserUsecase) Logout(ctx context.Context, refreshToken string) error {
	if m.ShouldFailLogout {
		return errors.New("logout failed")
	}
	return nil
}

func (m *MockUserUsecase) Authenticate(ctx context.Context, accessToken string) (*entity.User, error) {
	if m.ShouldFailAuthenticate {
		return nil, errors.New("authentication failed")
	}
	return &m.MockUser, nil
}

func (m *MockUserUsecase) PromoteUser(ctx context.Context, userID string) (*entity.User, error) {
	if m.ShouldFailPromoteUser {
		return nil, errors.New("promotion failed")
	}
	user := m.MockUser
	user.Role = entity.UserRoleAdmin
	return &user, nil
}

func (m *MockUserUsecase) DemoteUser(ctx context.Context, userID string) (*entity.User, error) {
	if m.ShouldFailDemoteUser {
		return nil, errors.New("demotion failed")
	}
	user := m.MockUser
	user.Role = entity.UserRoleUser
	return &user, nil
}

func (m *MockUserUsecase) LoginWithOAuth(ctx context.Context, firstName, lastName, email string) (string, string, error) {
	if m.ShouldFailLoginWithOAuth {
		return "", "", errors.New("login with OAuth failed")
	}
	return m.MockAccessToken, m.MockRefreshToken, nil
}
