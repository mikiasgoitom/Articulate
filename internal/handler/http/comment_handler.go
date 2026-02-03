package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mikiasgoitom/Articulate/internal/dto"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

type CommentHandler struct {
	commentUC usecasecontract.ICommentUseCase
}

func NewCommentHandler(commentUC usecasecontract.ICommentUseCase) *CommentHandler {
	return &CommentHandler{
		commentUC: commentUC,
	}
}

// Core CRUD Operations
func (h *CommentHandler) CreateComment(c *gin.Context) {
	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}
	blogID := c.Param("blogID")
	if blogID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Blog ID is required"})
		return
	}
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := userIDStr.(string)

	// parent_id and target_id are handled in req (DTO)
	comment, err := h.commentUC.CreateComment(c.Request.Context(), req, userID, blogID)
	if err != nil {
		if err.Error() == "blog not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, comment)
}

func (h *CommentHandler) GetComment(c *gin.Context) {
	commentIDStr := c.Param("commentID")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID format"})
		return
	}

	// Get user ID if authenticated (optional for viewing)
	var userID *string
	if userIDStr, exists := c.Get("user_id"); exists {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			uidStr := uid.String()
			userID = &uidStr
		}
	}

	comment, err := h.commentUC.GetComment(c.Request.Context(), commentID.String(), userID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": comment})
}

func (h *CommentHandler) UpdateComment(c *gin.Context) {
	var req dto.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	commentIDStr := c.Param("commentID")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID format"})
		return
	}

	// Get user ID from auth middleware
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	comment, err := h.commentUC.UpdateComment(c.Request.Context(), commentID.String(), userID.String(), req)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: can only edit your own comments" ||
			err.Error() == "comment edit time window has expired" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": comment})
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	commentIDStr := c.Param("commentID")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID format"})
		return
	}

	// Get user ID from auth middleware
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.commentUC.DeleteComment(c.Request.Context(), commentID.String(), userID.String())
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "unauthorized: can only delete your own comments" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

// Listing Operations
func (h *CommentHandler) GetBlogComments(c *gin.Context) {
	blogID := c.Param("blogID")
	if blogID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Blog ID is required"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// Get user ID if authenticated (optional)
	var userID *string
	if userIDStr, exists := c.Get("user_id"); exists {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			uidStr := uid.String()
			userID = &uidStr
		}
	}

	comments, err := h.commentUC.GetBlogComments(c.Request.Context(), blogID, page, pageSize, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": comments})
}

func (h *CommentHandler) GetCommentThread(c *gin.Context) {
	commentIDStr := c.Param("commentID")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID format"})
		return
	}

	// Get user ID if authenticated (optional)
	var userID *string
	if userIDStr, exists := c.Get("userID"); exists {
		if uid, err := uuid.Parse(userIDStr.(string)); err == nil {
			uidStr := uid.String()
			userID = &uidStr
		}
	}

	thread, err := h.commentUC.GetCommentThread(c.Request.Context(), commentID.String(), userID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": thread})
}

func (h *CommentHandler) GetUserComments(c *gin.Context) {
	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	comments, err := h.commentUC.GetUserComments(c.Request.Context(), userID.String(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": comments})
}

// Moderation
func (h *CommentHandler) UpdateCommentStatus(c *gin.Context) {
	var req dto.UpdateCommentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	commentIDStr := c.Param("commentID")

	moderatorIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	moderatorID, err := uuid.Parse(moderatorIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid moderator ID"})
		return
	}

	err = h.commentUC.UpdateCommentStatus(c.Request.Context(), commentIDStr, moderatorID.String(), req)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment status updated successfully"})
}

// Engagement
func (h *CommentHandler) LikeComment(c *gin.Context) {
	commentIDStr := c.Param("commentID")

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.commentUC.LikeComment(c.Request.Context(), commentIDStr, userID.String())
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "comment already liked by user" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment liked successfully"})
}

func (h *CommentHandler) UnlikeComment(c *gin.Context) {
	commentIDStr := c.Param("commentID")

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.commentUC.UnlikeComment(c.Request.Context(), commentIDStr, userID.String())
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "comment not liked by user" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment unliked successfully"})
}

// Reporting
func (h *CommentHandler) ReportComment(c *gin.Context) {
	var req dto.ReportCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	commentIDStr := c.Param("commentID")
	userIDStr, exists := c.Get("user_id")

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	err = h.commentUC.ReportComment(c.Request.Context(), commentIDStr, userID.String(), req)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment reported successfully"})
}

func (h *CommentHandler) GetCommentReports(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	reports, err := h.commentUC.GetCommentReports(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

// Additional handler methods for the new comment endpoints

// CreateReply creates a reply to a comment
func (h *CommentHandler) CreateReply(c *gin.Context) {
	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get parent comment ID from URL
	parentcommentID := c.Param("commentID")
	if parentcommentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment ID is required"})
		return
	}

	// Get user ID from auth middleware
	userIDStr, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userID := userIDStr.(string)

	// Set the parent comment ID in the request
	req.ParentID = &parentcommentID

	// We need to get the blog ID from the parent comment
	parentComment, err := h.commentUC.GetComment(c.Request.Context(), parentcommentID, &userID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Parent comment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create the reply using the parent comment's blog ID
	comment, err := h.commentUC.CreateComment(c.Request.Context(), req, userID, parentComment.BlogID)
	if err != nil {
		if err.Error() == "blog not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "parent comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}

// GetCommentReplies gets replies to a specific comment with pagination
func (h *CommentHandler) GetCommentReplies(c *gin.Context) {
	commentID := c.Param("commentID")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment ID is required"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	depth, _ := strconv.Atoi(c.DefaultQuery("depth", "3"))

	// Validate page and pageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	if depth < 1 || depth > 10 {
		depth = 3
	}

	// Get optional user ID for personalized data
	var userID *string
	if userIDStr, exists := c.Get("user_id"); exists {
		uid := userIDStr.(string)
		userID = &uid
	} else if userIDStr, exists := c.Get("userID"); exists {
		uid := userIDStr.(string)
		userID = &uid
	}

	// Use the existing GetCommentThread to fetch the full nested tree
	thread, err := h.commentUC.GetCommentThread(c.Request.Context(), commentID, userID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Flatten all nested replies into a single list
	flat := make([]*dto.CommentThreadResponse, 0)
	var flatten func(nodes []*dto.CommentThreadResponse)
	flatten = func(nodes []*dto.CommentThreadResponse) {
		for _, n := range nodes {
			// Shallow copy without children to keep payload lean
			copy := &dto.CommentThreadResponse{
				Comment: n.Comment,
				Depth:   n.Depth,
				Replies: nil,
			}
			flat = append(flat, copy)
			if len(n.Replies) > 0 {
				flatten(n.Replies)
			}
		}
	}
	flatten(thread.Replies)

	c.JSON(http.StatusOK, gin.H{
		"replies": flat,
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
			"total":     len(flat),
			"has_more":  false,
		},
	})
}

// GetCommentDepth gets the depth of a comment thread
func (h *CommentHandler) GetCommentDepth(c *gin.Context) {
	commentID := c.Param("commentID")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment ID is required"})
		return
	}

	// Get optional user ID
	var userID *string
	if userIDStr, exists := c.Get("user_id"); exists {
		uid := userIDStr.(string)
		userID = &uid
	} else if userIDStr, exists := c.Get("userID"); exists {
		uid := userIDStr.(string)
		userID = &uid
	}

	// Get the comment thread to calculate depth
	thread, err := h.commentUC.GetCommentThread(c.Request.Context(), commentID, userID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate depth recursively (simplified implementation)
	depth := h.calculateThreadDepth(thread, 1)

	c.JSON(http.StatusOK, gin.H{
		"comment_id": commentID,
		"depth":      depth,
		"max_depth":  depth,
	})
}

// GetBlogCommentsCount gets the total count of comments for a blog
func (h *CommentHandler) GetBlogCommentsCount(c *gin.Context) {
	blogID := c.Param("blogId")
	if blogID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Blog ID is required"})
		return
	}

	// Use a dedicated count method to get the total number of comments
	commentCount, err := h.commentUC.GetBlogCommentsCount(c.Request.Context(), blogID)
	if err != nil {
		if err.Error() == "blog not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"blog_id":       blogID,
		"comment_count": commentCount,
	})
}

// Helper function to calculate thread depth recursively
func (h *CommentHandler) calculateThreadDepth(thread *dto.CommentThreadResponse, currentDepth int) int {
	if len(thread.Replies) == 0 {
		return currentDepth
	}

	maxDepth := currentDepth
	for range thread.Replies {
		// In a full implementation, you'd need to recursively check replies
		// For now, we'll assume each reply adds one level
		replyDepth := currentDepth + 1
		if replyDepth > maxDepth {
			maxDepth = replyDepth
		}
	}

	return maxDepth
}

// Additional Advanced Comment Endpoints

// GetCommentsByUser gets all comments by a specific user with pagination
func (h *CommentHandler) GetCommentsByUser(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	comments, err := h.commentUC.GetUserComments(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

// LikeCommentToggle toggles like status on a comment
func (h *CommentHandler) LikeCommentToggle(c *gin.Context) {
	commentID := c.Param("commentID")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment ID is required"})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		userIDStr, exists = c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}
	}

	userID := userIDStr.(string)

	// Check if user has already liked the comment
	comment, err := h.commentUC.GetComment(c.Request.Context(), commentID, &userID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Toggle like/unlike
	if comment.IsLiked {
		err = h.commentUC.UnlikeComment(c.Request.Context(), commentID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Comment unliked successfully",
			"liked":   false,
		})
	} else {
		err = h.commentUC.LikeComment(c.Request.Context(), commentID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "Comment liked successfully",
			"liked":   true,
		})
	}
}

// GetCommentStatistics gets comprehensive statistics for a comment
func (h *CommentHandler) GetCommentStatistics(c *gin.Context) {
	commentID := c.Param("commentID")
	if commentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment ID is required"})
		return
	}

	// Get optional user ID for personalized stats
	var userID *string
	if userIDStr, exists := c.Get("user_id"); exists {
		uid := userIDStr.(string)
		userID = &uid
	} else if userIDStr, exists := c.Get("userID"); exists {
		uid := userIDStr.(string)
		userID = &uid
	}

	// Get comment details
	comment, err := h.commentUC.GetComment(c.Request.Context(), commentID, userID)
	if err != nil {
		if err.Error() == "comment not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get thread to calculate depth and reply count
	thread, err := h.commentUC.GetCommentThread(c.Request.Context(), commentID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	depth := h.calculateThreadDepth(thread, 1)

	c.JSON(http.StatusOK, gin.H{
		"comment_id":   commentID,
		"like_count":   comment.LikeCount,
		"reply_count":  comment.ReplyCount,
		"thread_depth": depth,
		"is_liked":     comment.IsLiked,
		"author_id":    comment.AuthorID,
		"created_at":   comment.CreatedAt,
		"updated_at":   comment.UpdatedAt,
		"status":       comment.Status,
	})
}

// BulkDeleteComments allows admins to delete multiple comments
func (h *CommentHandler) BulkDeleteComments(c *gin.Context) {
	// Check if user is admin
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	var req struct {
		CommentIDs []string `json:"comment_ids" validate:"required,min=1,max=100"`
		Reason     string   `json:"reason" validate:"max=500"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID := userIDStr.(string)

	deletedCount := 0
	errors := make([]string, 0)

	for _, commentID := range req.CommentIDs {
		err := h.commentUC.DeleteComment(c.Request.Context(), commentID, userID)
		if err != nil {
			errors = append(errors, commentID+": "+err.Error())
		} else {
			deletedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"deleted_count":   deletedCount,
		"total_requested": len(req.CommentIDs),
		"errors":          errors,
		"reason":          req.Reason,
	})
}

// SearchComments searches comments by content or author
func (h *CommentHandler) SearchComments(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	// Parse pagination and filters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	blogID := c.Query("blog_id")
	authorID := c.Query("author_id")
	status := c.DefaultQuery("status", "approved")

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// For now, we'll return a simple response
	// In a real implementation, you'd implement search in the usecase
	c.JSON(http.StatusOK, gin.H{
		"query": query,
		"filters": gin.H{
			"blog_id":   blogID,
			"author_id": authorID,
			"status":    status,
		},
		"pagination": gin.H{
			"page":      page,
			"page_size": pageSize,
		},
		"comments": []interface{}{},
		"message":  "Search functionality not fully implemented yet",
	})
}
