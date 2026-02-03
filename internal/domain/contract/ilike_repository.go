package contract

import (
	"context"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

// ILikeRepository defines the interface for reaction data persistence.
type ILikeRepository interface {
	CreateReaction(ctx context.Context, like *entity.Like) error
	DeleteReaction(ctx context.Context, reactionID string) error                                                                         // Changed from uuid.UUID to string
	GetReactionByUserIDAndTargetID(ctx context.Context, userID, targetID string) (*entity.Like, error)                                   // Changed from uuid.UUID to string
	GetReactionByUserIDTargetIDAndType(ctx context.Context, userID, targetID string, reactionType entity.LikeType) (*entity.Like, error) // Changed from uuid.UUID to string
	CountLikesByTargetID(ctx context.Context, targetID string) (int64, error)                                                            // Changed from uuid.UUID to string
	CountDislikesByTargetID(ctx context.Context, targetID string) (int64, error)                                                         // Changed from uuid.UUID to string
}
