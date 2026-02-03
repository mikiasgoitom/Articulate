package usecasecontract

import (
	"context"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

// SortOrder defines sorting direction for list queries
type SortOrder string

const (
	SortOrderASC  SortOrder = "asc"
	SortOrderDESC SortOrder = "desc"
)

type IBlogUseCase interface {
	CreateBlog(ctx context.Context, title, content string, authorID string, slug string, status entity.BlogStatus, featuredImageID *string, tags []string) (*entity.Blog, error)
	GetBlogs(ctx context.Context, page, pageSize int, sortBy string, sortOrder SortOrder, dateFrom *time.Time, dateTo *time.Time) ([]entity.Blog, int, int, int, error)
	GetBlogDetail(ctx context.Context, slug string) (entity.Blog, error)
	UpdateBlog(ctx context.Context, blogID, authorID string, title *string, content *string, status *entity.BlogStatus, featuredImageID *string) (*entity.Blog, error)
	DeleteBlog(ctx context.Context, blogID, userID string, isAdmin bool) (bool, error)
	TrackBlogView(ctx context.Context, blogID, userID, ipAddress, userAgent string) error
	GetPopularBlogs(ctx context.Context, page, pageSize int) ([]entity.Blog, int, int, int, error)
	SearchAndFilterBlogs(
		ctx context.Context,
		query string,
		tags []string,
		dateFrom *time.Time,
		dateTo *time.Time,
		minViews *int,
		maxViews *int,
		minLikes *int,
		maxLikes *int,
		authorID *string,
		page int,
		pageSize int,
	) ([]entity.Blog, int, int, int, error)
	UpdateBlogPopularity(ctx context.Context, blogID string) error
}
