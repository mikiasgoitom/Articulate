package contract

import (
	"context"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type PaginationMeta struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrevious bool  `json:"has_previous"`
}

var MaxCommentDepth = 5

type ICommentRepository interface {
	// Core CRUD operations
	Create(ctx context.Context, comment *entity.Comment) error
	GetByID(ctx context.Context, id string) (*entity.Comment, error)
	Update(ctx context.Context, comment *entity.Comment) error
	Delete(ctx context.Context, id string) error

	// Listing operations
	GetTopLevelComments(ctx context.Context, blogID string, pagination Pagination) ([]*entity.Comment, int64, error)
	GetCommentThread(ctx context.Context, parentID string) (*entity.CommentThread, error)
	GetCommentsByUser(ctx context.Context, userID string, pagination Pagination) ([]*entity.Comment, int64, error)

	// Status and moderation
	UpdateStatus(ctx context.Context, id, status string) error
	GetCommentCount(ctx context.Context, blogID string) (int64, error)

	// Like system
	LikeComment(ctx context.Context, commentID, userID string) error
	UnlikeComment(ctx context.Context, commentID, userID string) error
	IsCommentLikedByUser(ctx context.Context, commentID, userID string) (bool, error)
	GetCommentLikeCount(ctx context.Context, commentID string) (int64, error)

	// Reporting system
	ReportComment(ctx context.Context, report *entity.CommentReport) error
	GetCommentReports(ctx context.Context, pagination Pagination) ([]*entity.CommentReport, int64, error)
	UpdateReportStatus(ctx context.Context, reportID string, status string, reviewerID string) error
}
