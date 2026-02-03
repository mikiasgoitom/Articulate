package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	// "time"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"github.com/mikiasgoitom/Articulate/internal/dto"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

type commentUseCase struct {
	commentRepo contract.ICommentRepository
	blogRepo    contract.IBlogRepository
	userRepo    contract.IUserRepository
}

func NewCommentUseCase(
	commentRepo contract.ICommentRepository,
	blogRepo contract.IBlogRepository,
	userRepo contract.IUserRepository,
) usecasecontract.ICommentUseCase {
	return &commentUseCase{
		commentRepo: commentRepo,
		blogRepo:    blogRepo,
		userRepo:    userRepo,
	}
}

// Core Operations
func (uc *commentUseCase) CreateComment(ctx context.Context, req dto.CreateCommentRequest, userID, blogID string) (*dto.CommentResponse, error) {
	// Validate blog exists
	_, err := uc.blogRepo.GetBlogByID(ctx, blogID)
	if err != nil {
		return nil, fmt.Errorf("blog not found: %w", err)
	}

	// Validate content
	if err := uc.validateContent(req.Content); err != nil {
		return nil, err
	}

	commentType := req.Type
	if commentType == "" {
		if req.ParentID != nil && *req.ParentID != "" {
			commentType = "reply"
		} else {
			commentType = "comment"
		}
	}

	var targetUserName string
	if req.TargetID != nil && *req.TargetID != "" {
		target, err := uc.commentRepo.GetByID(ctx, *req.TargetID)
		if err == nil {
			targetUserName = target.AuthorName
		}
	}

	replyCount := 0
	if req.ParentID != nil && *req.ParentID != "" {
		parent, err := uc.commentRepo.GetByID(ctx, *req.ParentID)
		if err != nil {
			return nil, fmt.Errorf("invalid parent/target relationship: parent comment not found: %w", err)
		}
		replyCount = parent.ReplyCount + 1
		parent.ReplyCount = replyCount
		_ = uc.commentRepo.Update(ctx, parent)

		// If no explicit target provided, default target to the parent comment's author
		if (req.TargetID == nil || *req.TargetID == "") && targetUserName == "" {
			targetUserName = parent.AuthorName
		}
	}

	// Fetch author name from userRepo
	authorName := ""
	if uc.userRepo != nil {
		user, err := uc.userRepo.GetUserByID(ctx, userID)
		if err == nil {
			authorName = user.Username
		}
	}

	comment := &entity.Comment{
		BlogID:         blogID,
		AuthorID:       userID,
		AuthorName:     authorName,
		Content:        strings.TrimSpace(req.Content),
		ParentID:       req.ParentID,
		TargetID:       req.TargetID,
		Type:           commentType,
		TargetUserName: targetUserName,
		Status:         "approved",
		ReplyCount:     0,
	}

	// Create comment
	if err := uc.commentRepo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	// Update blog popularity after comment creation
	if blogID != "" && uc.blogRepo != nil {
		if updater, ok := uc.blogRepo.(interface {
			UpdateBlogPopularity(context.Context, string) error
		}); ok {
			_ = updater.UpdateBlogPopularity(ctx, blogID)
		}
	}

	// Return response
	return uc.toCommentResponse(ctx, comment, &userID)
}

func (uc *commentUseCase) GetComment(ctx context.Context, commentID string, userID *string) (*dto.CommentResponse, error) {
	comment, err := uc.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	return uc.toCommentResponse(ctx, comment, userID)
}

func (uc *commentUseCase) UpdateComment(ctx context.Context, commentID, userID string, req dto.UpdateCommentRequest) (*dto.CommentResponse, error) {
	// Get existing comment
	comment, err := uc.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if comment.AuthorID != userID {
		return nil, errors.New("unauthorized: can only edit your own comments")
	}

	// Validate content
	if err := uc.validateContent(req.Content); err != nil {
		return nil, err
	}

	// Update comment
	comment.Content = strings.TrimSpace(req.Content)
	if err := uc.commentRepo.Update(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to update comment: %w", err)
	}

	return uc.toCommentResponse(ctx, comment, &userID)
}

func (uc *commentUseCase) DeleteComment(ctx context.Context, commentID, userID string) error {
	// Get existing comment
	comment, err := uc.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	// Check ownership (or admin role - would need to check user role)
	if comment.AuthorID != userID {
		return errors.New("unauthorized: can only delete your own comments")
	}

	err = uc.commentRepo.Delete(ctx, commentID)
	if err != nil {
		return err
	}

	// Update blog popularity after comment deletion
	if comment.BlogID != "" && uc.blogRepo != nil {
		if updater, ok := uc.blogRepo.(interface {
			UpdateBlogPopularity(context.Context, string) error
		}); ok {
			_ = updater.UpdateBlogPopularity(ctx, comment.BlogID)
		}
	}
	return nil
}

// Listing Operations
func (uc *commentUseCase) GetBlogComments(ctx context.Context, blogID string, page, pageSize int, userID *string) (*dto.CommentsResponse, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	pagination := contract.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	comments, total, err := uc.commentRepo.GetTopLevelComments(ctx, blogID, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get blog comments: %w", err)
	}

	// Convert to response DTOs
	commentResponses := make([]*dto.CommentResponse, len(comments))
	for i, comment := range comments {
		commentResponses[i], err = uc.toCommentResponse(ctx, comment, userID)
		if err != nil {
			return nil, err
		}
	}

	// Create pagination meta
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	paginationMeta := dto.PaginationMeta{
		CurrentPage: page,
		PageSize:    pageSize,
		TotalItems:  total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	return &dto.CommentsResponse{
		Comments:   commentResponses,
		Pagination: paginationMeta,
	}, nil
}

func (uc *commentUseCase) GetCommentThread(ctx context.Context, commentID string, userID *string) (*dto.CommentThreadResponse, error) {
	thread, err := uc.commentRepo.GetCommentThread(ctx, commentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment thread: %w", err)
	}

	return uc.toCommentThreadResponse(ctx, thread, userID)
}

func (uc *commentUseCase) GetUserComments(ctx context.Context, userID string, page, pageSize int) (*dto.CommentsResponse, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	pagination := contract.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	comments, total, err := uc.commentRepo.GetCommentsByUser(ctx, userID, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get user comments: %w", err)
	}

	// Convert to response DTOs
	commentResponses := make([]*dto.CommentResponse, len(comments))
	for i, comment := range comments {
		commentResponses[i], err = uc.toCommentResponse(ctx, comment, &userID)
		if err != nil {
			return nil, err
		}
	}

	// Create pagination meta
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	paginationMeta := dto.PaginationMeta{
		CurrentPage: page,
		PageSize:    pageSize,
		TotalItems:  total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	return &dto.CommentsResponse{
		Comments:   commentResponses,
		Pagination: paginationMeta,
	}, nil
}

// Moderation
func (uc *commentUseCase) UpdateCommentStatus(ctx context.Context, commentID, moderatorID string, req dto.UpdateCommentStatusRequest) error {
	// Here you would check if moderatorID has admin/moderator role
	// For now, we'll assume they do

	return uc.commentRepo.UpdateStatus(ctx, commentID, req.Status)
}

// Engagement
func (uc *commentUseCase) LikeComment(ctx context.Context, commentID, userID string) error {
	// Check if comment exists
	_, err := uc.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	return uc.commentRepo.LikeComment(ctx, commentID, userID)
}

func (uc *commentUseCase) UnlikeComment(ctx context.Context, commentID, userID string) error {
	// Check if comment exists
	_, err := uc.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	return uc.commentRepo.UnlikeComment(ctx, commentID, userID)
}

// Reporting
func (uc *commentUseCase) ReportComment(ctx context.Context, commentID, userID string, req dto.ReportCommentRequest) error {
	// Check if comment exists
	_, err := uc.commentRepo.GetByID(ctx, commentID)
	if err != nil {
		return err
	}

	report := &entity.CommentReport{
		CommentID:  commentID,
		ReporterID: userID,
		Reason:     req.Reason,
		Details:    req.Details,
	}

	return uc.commentRepo.ReportComment(ctx, report)
}

func (uc *commentUseCase) GetCommentReports(ctx context.Context, page, pageSize int) (*dto.ReportsResponse, error) {
	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	pagination := contract.Pagination{
		Page:     page,
		PageSize: pageSize,
	}

	reports, total, err := uc.commentRepo.GetCommentReports(ctx, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment reports: %w", err)
	}

	// Convert to response DTOs
	reportResponses := make([]*dto.CommentReportResponse, len(reports))
	for i, report := range reports {
		reportResponses[i] = &dto.CommentReportResponse{
			ID:         report.ID,
			CommentID:  report.CommentID,
			ReporterID: report.ReporterID,
			Reason:     report.Reason,
			Details:    report.Details,
			Status:     report.Status,
			CreatedAt:  report.CreatedAt,
			ReviewedAt: report.ReviewedAt,
			ReviewedBy: report.ReviewedBy,
		}
	}

	// Create pagination meta
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	paginationMeta := dto.PaginationMeta{
		CurrentPage: page,
		PageSize:    pageSize,
		TotalItems:  total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	return &dto.ReportsResponse{
		Reports:    reportResponses,
		Pagination: paginationMeta,
	}, nil
}

func (uc *commentUseCase) UpdateReportStatus(ctx context.Context, reportID, reviewerID string, status string) error {
	return uc.commentRepo.UpdateReportStatus(ctx, reportID, status, reviewerID)
}

// Helper Methods
func (uc *commentUseCase) validateContent(content string) error {
	content = strings.TrimSpace(content)

	if len(content) == 0 {
		return errors.New("comment content cannot be empty")
	}

	if len(content) > 1000 {
		return errors.New("comment content too long (max 1000 characters)")
	}

	// Add profanity filter, spam detection, etc.
	if uc.containsProfanity(content) {
		return errors.New("comment contains inappropriate language")
	}

	return nil
}

func (uc *commentUseCase) containsProfanity(content string) bool {
	// Implement profanity detection logic
	// For now, return false
	if strings.Contains(strings.ToLower(content), "badword") {
		return true
	}
	return false
}

func (uc *commentUseCase) toCommentResponse(ctx context.Context, comment *entity.Comment, userID *string) (*dto.CommentResponse, error) {
	// Get author name
	author, err := uc.userRepo.GetUserByID(ctx, comment.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment author: %w", err)
	}

	// Check if liked by current user
	var isLiked bool
	if userID != nil {
		isLiked, _ = uc.commentRepo.IsCommentLikedByUser(ctx, comment.ID, *userID)
	}

	// Use stored reply count for now (could be recalculated if needed)
	replyCount := comment.ReplyCount

	return &dto.CommentResponse{
		ID:             comment.ID,
		BlogID:         comment.BlogID,
		Type:           comment.Type,
		ParentID:       comment.ParentID,
		TargetID:       comment.TargetID,
		AuthorID:       comment.AuthorID,
		AuthorName:     author.Username,
		TargetUserName: comment.TargetUserName,
		Content:        comment.Content,
		Status:         comment.Status,
		LikeCount:      comment.LikeCount,
		IsLiked:        isLiked,
		CreatedAt:      comment.CreatedAt,
		UpdatedAt:      comment.UpdatedAt,
		ReplyCount:     replyCount,
	}, nil
}

func (uc *commentUseCase) toCommentThreadResponse(ctx context.Context, thread *entity.CommentThread, userID *string) (*dto.CommentThreadResponse, error) {
	commentResponse, err := uc.toCommentResponse(ctx, thread.Comment, userID)
	if err != nil {
		return nil, err
	}

	response := &dto.CommentThreadResponse{
		Comment: commentResponse,
		Depth:   thread.Depth,
		Replies: make([]*dto.CommentThreadResponse, len(thread.Replies)),
	}

	for i, reply := range thread.Replies {
		response.Replies[i], err = uc.toCommentThreadResponse(ctx, reply, userID)
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}

func (uc *commentUseCase) GetBlogCommentsCount(ctx context.Context, blogID string) (int64, error) {
	count, err := uc.commentRepo.GetCommentCount(ctx, blogID)
	if err != nil {
		return 0, fmt.Errorf("failed to get blog comments count: %w", err)
	}
	return count, nil
}
