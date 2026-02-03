package contract

import (
	"context"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

// ITagRepository defines the interface for tag data persistence.
type ITagRepository interface {
	CreateTag(ctx context.Context, tag *entity.Tag) error
	GetTagByID(ctx context.Context, tagID string) (*entity.Tag, error)
	GetTagByName(ctx context.Context, name string) (*entity.Tag, error)
	GetAllTags(ctx context.Context) ([]*entity.Tag, error)
	UpdateTag(ctx context.Context, tagID string, updates map[string]interface{}) error
	DeleteTag(ctx context.Context, tagID string) error
}
