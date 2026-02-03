package contract

import (
	"context"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

type ITokenRepository interface {
	CreateToken(ctx context.Context, token *entity.Token) error
	GetTokenByID(ctx context.Context, id string) (*entity.Token, error)
	GetTokenByUserID(ctx context.Context, userID string) (*entity.Token, error)
	UpdateToken(ctx context.Context, tokenID string, tokenHash string, expiry time.Time) error
	GetTokenByVerifier(ctx context.Context, verifier string) (*entity.Token, error)
	RevokeToken(ctx context.Context, id string) error
	RevokeAllTokensForUser(ctx context.Context, userID string, tokenType entity.TokenType) error
}
