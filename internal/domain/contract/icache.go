package contract

import (
	"context"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

// CachedBlogsPage is the cached payload for list endpoints.
type CachedBlogsPage struct {
	Blogs []entity.Blog `json:"blogs"`
	Total int           `json:"total"`
}

// IBlogCache defines caching operations for blogs.
type IBlogCache interface {
	// Detail (by slug)
	GetBlogBySlug(ctx context.Context, slug string) (*entity.Blog, bool, error)
	SetBlogBySlug(ctx context.Context, slug string, blog *entity.Blog) error
	InvalidateBlogBySlug(ctx context.Context, slug string) error

	// List pages (key built by usecase)
	GetBlogsPage(ctx context.Context, key string) (*CachedBlogsPage, bool, error)
	SetBlogsPage(ctx context.Context, key string, page *CachedBlogsPage) error
	InvalidateBlogLists(ctx context.Context) error

	// Fraud detection cache helpers
	AddRecentViewByIP(ctx context.Context, ip, blogID string, ttlSeconds int64) error
	GetRecentViewCountByIP(ctx context.Context, ip string) (int64, error)
	AddRecentViewByUser(ctx context.Context, userID, ip string, ttlSeconds int64) error
	GetRecentIPCountByUser(ctx context.Context, userID string) (int64, error)
}
