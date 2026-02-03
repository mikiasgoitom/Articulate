package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	usecase "github.com/mikiasgoitom/Articulate/internal/usecase"
)

type InteractionHandler struct {
	likeUsecase *usecase.LikeUsecase
}

func NewInteractionHandler(likeUsecase *usecase.LikeUsecase) *InteractionHandler {
	return &InteractionHandler{
		likeUsecase: likeUsecase,
	}
}

func (h *InteractionHandler) LikeBlogHandler(c *gin.Context) {
	blogID := c.Param("blogID")
	userID, exists := c.Get("userID")
	if !exists {
		ErrorHandler(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	userIDStr, ok := userID.(string)
	if !ok {
		ErrorHandler(c, http.StatusBadRequest, "Invalid user ID format in token")
		return
	}
	err := h.likeUsecase.ToggleLike(c.Request.Context(), userIDStr, blogID, entity.TargetTypeBlog)
	if err != nil {
		ErrorHandler(c, http.StatusInternalServerError, err.Error())
		return
	}
	// Determine the new state by checking if the user has liked the blog
	reaction, _ := h.likeUsecase.GetUserReaction(c.Request.Context(), userIDStr, blogID)
	if reaction != nil && reaction.Type == entity.LIKE_TYPE_LIKE {
		SuccessHandler(c, http.StatusOK, "Blog liked successfully")
	} else {
		SuccessHandler(c, http.StatusOK, "Blog unliked successfully")
	}
}

func (h *InteractionHandler) DislikeBlogHandler(c *gin.Context) {

	blogID := c.Param("blogID")
	userID, exists := c.Get("userID")
	if !exists {
		ErrorHandler(c, http.StatusUnauthorized, "User not authenticated")
		return
	}
	userIDStr, ok := userID.(string)
	if !ok {
		ErrorHandler(c, http.StatusBadRequest, "Invalid user ID format in token")
		return
	}

	// Validate blogID format (UUID)
	if len(blogID) != 36 {
		ErrorHandler(c, http.StatusBadRequest, "Invalid blog ID format")
		return
	}

	// Check if blog exists using LikeUsecase.ExistsBlog
	if !h.likeUsecase.ExistsBlog(c.Request.Context(), blogID) {
		ErrorHandler(c, http.StatusNotFound, "Blog not found")
		return
	}

	err := h.likeUsecase.ToggleDislike(c.Request.Context(), userIDStr, blogID, entity.TargetTypeBlog)
	if err != nil {
		ErrorHandler(c, http.StatusInternalServerError, err.Error())
		return
	}
	// Determine the new state by checking if the user has disliked the blog
	reaction, _ := h.likeUsecase.GetUserReaction(c.Request.Context(), userIDStr, blogID)
	if reaction != nil && reaction.Type == entity.LIKE_TYPE_DISLIKE {
		SuccessHandler(c, http.StatusOK, "Blog disliked successfully")
	} else {
		SuccessHandler(c, http.StatusOK, "Blog undisliked successfully")
	}
}
