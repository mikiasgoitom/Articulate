package usecasecontract

import (
	"context"

	"github.com/mikiasgoitom/Articulate/internal/dto"
)

type ICommentUseCase interface {
	// Core operations
	CreateComment(ctx context.Context, req dto.CreateCommentRequest, userID, blogID string) (*dto.CommentResponse, error)
	GetComment(ctx context.Context, commentID string, userID *string) (*dto.CommentResponse, error)
	UpdateComment(ctx context.Context, commentID, userID string, req dto.UpdateCommentRequest) (*dto.CommentResponse, error)
	DeleteComment(ctx context.Context, commentID, userID string) error

	// Listing operations
	GetBlogComments(ctx context.Context, blogID string, page, pageSize int, userID *string) (*dto.CommentsResponse, error)
	GetCommentThread(ctx context.Context, commentID string, userID *string) (*dto.CommentThreadResponse, error)
	GetUserComments(ctx context.Context, userID string, page, pageSize int) (*dto.CommentsResponse, error)
	GetBlogCommentsCount(ctx context.Context, blogID string) (int64, error)

	// Moderation
	UpdateCommentStatus(ctx context.Context, commentID, moderatorID string, req dto.UpdateCommentStatusRequest) error
	// Engagement
	LikeComment(ctx context.Context, commentID, userID string) error
	UnlikeComment(ctx context.Context, commentID, userID string) error

	// Reporting
	ReportComment(ctx context.Context, commentID, userID string, req dto.ReportCommentRequest) error
	GetCommentReports(ctx context.Context, page, pageSize int) (*dto.ReportsResponse, error)
	UpdateReportStatus(ctx context.Context, reportID, reviewerID string, status string) error
}
