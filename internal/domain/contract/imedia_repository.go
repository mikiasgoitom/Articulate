package contract

import (
	"context"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

// MediaFilterOptions holds database-agnostic parameters for filtering, sorting, and pagination.
type MediaFilterOptions struct {
	UploadedByUserID *string
	MimeType         *string
	Page             int64
	Limit            int64
	SortBy           string // e.g., "created_at", "file_name"
	SortOrder        string // "asc" or "desc"
}

// IMediaRepository defines the interface for media data persistence.
type IMediaRepository interface {
	CreateMedia(ctx context.Context, media *entity.Media) error
	GetMediaByID(ctx context.Context, mediaID string) (*entity.Media, error)
	GetMedia(ctx context.Context, opts *MediaFilterOptions) ([]*entity.Media, error)
	UpdateMedia(ctx context.Context, mediaID string, updates map[string]interface{}) error
	DeleteMedia(ctx context.Context, mediaID string) error
	AssociateMediaWithBlog(ctx context.Context, mediaID, blogID string) error
	RemoveMediaFromBlog(ctx context.Context, mediaID string) error
	GetMediaByBlogID(ctx context.Context, blogID string) ([]*entity.Media, error)
}
