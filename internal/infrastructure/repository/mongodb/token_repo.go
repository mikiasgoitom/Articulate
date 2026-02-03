package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ---------- DTO layer ------------------
type tokenDTO struct {
	ID        string    `bson:"_id"`
	UserID    string    `bson:"user_id"`
	TokenType string    `bson:"token_type"`
	TokenHash string    `bson:"token_hash"`
	Verifier  string    `bson:"verifier"`
	CreatedAt time.Time `bson:"created_at"`
	ExpiresAt time.Time `bson:"expires_at"`
	Revoke    bool      `bson:"revoke"`
}

func (t *tokenDTO) ToEntity() *entity.Token {
	return &entity.Token{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenType: entity.TokenType(t.TokenType),
		Verifier:  t.Verifier,
		TokenHash: t.TokenHash,
		CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt,
		Revoke:    t.Revoke,
	}
}

func FromTokenEntityToDTO(t *entity.Token) *tokenDTO {
	return &tokenDTO{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenType: string(t.TokenType),
		Verifier:  t.Verifier,
		TokenHash: t.TokenHash,
		CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt,
		Revoke:    t.Revoke,
	}
}

// ---------------------------------------

type TokenRepository struct {
	Collection *mongo.Collection
}

// check in compile time if TokenRepository implements ITokenRepository
var _ contract.ITokenRepository = (*TokenRepository)(nil)

func NewTokenRepository(colln *mongo.Collection) *TokenRepository {
	return &TokenRepository{
		Collection: colln,
	}
}
func (r *TokenRepository) CreateToken(ctx context.Context, token *entity.Token) error {
	dto := FromTokenEntityToDTO(token)
	_, err := r.Collection.InsertOne(ctx, dto)
	if err != nil {
		return err
	}

	return nil
}

func (r *TokenRepository) GetTokenByID(ctx context.Context, id string) (*entity.Token, error) {
	filter := bson.M{"_id": id}
	var dto tokenDTO
	err := r.Collection.FindOne(ctx, filter).Decode(&dto)
	if err != nil {
		return nil, err
	}
	token := dto.ToEntity()

	return token, nil
}

// get user by user id
func (r *TokenRepository) GetTokenByUserID(ctx context.Context, userID string) (*entity.Token, error) {
	filter := bson.M{"user_id": userID}
	var dto tokenDTO
	err := r.Collection.FindOne(ctx, filter).Decode(&dto)
	if err != nil {
		return nil, err
	}
	token := dto.ToEntity()

	return token, nil
}

// UpdateToken updates the token hash and expiry
func (r *TokenRepository) UpdateToken(ctx context.Context, tokenID string, tokenHash string, expiry time.Time) error {
	filter := bson.M{"_id": tokenID}
	update := bson.M{"$set": bson.M{"token_hash": tokenHash, "expires_at": expiry}}
	_, err := r.Collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *TokenRepository) GetTokenByVerifier(ctx context.Context, verifier string) (*entity.Token, error) {
	filter := bson.M{"verifier": verifier}
	var dto *tokenDTO

	err := r.Collection.FindOne(ctx, filter).Decode(&dto)
	if err != nil {
		return nil, err
	}

	token := dto.ToEntity()
	return token, nil
}

// Revoke marks a token as revoked by its ID
func (r *TokenRepository) RevokeToken(ctx context.Context, id string) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"revoke": true}}
	result, err := r.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("failed to revoke token with: %v", id)
	}

	return nil
}

// GetTokenByUserID retrieves a token by user ID (string).
func (r *TokenRepository) RevokeAllTokensForUser(ctx context.Context, userID string, tokenType entity.TokenType) error {
	filter := bson.D{
		{Key: "user_id", Value: userID},
		{Key: "token_type", Value: string(tokenType)},
		{Key: "revoke", Value: false},
	}
	update := bson.D{
		{Key: "$set", Value: bson.M{"revoke": true}},
	}

	_, err := r.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return fmt.Errorf("token not found")
		}
		return err
	}

	return nil
}
