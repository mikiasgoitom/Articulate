package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"github.com/mikiasgoitom/Articulate/internal/handler/http/dto"
	"github.com/mikiasgoitom/Articulate/internal/usecase"
)

// BlogHandlerInterface defines the methods for Blog handler to allow interface-based dependency injection (for testing/mocking)
type BlogHandlerInterface interface {
	CreateBlogHandler(*gin.Context)
	GetBlogsHandler(*gin.Context)
	GetBlogDetailHandler(*gin.Context)
	UpdateBlogHandler(*gin.Context)
	DeleteBlogHandler(*gin.Context)
	TrackBlogViewHandler(*gin.Context)
	SearchAndFilterBlogsHandler(*gin.Context)
	GetPopularBlogsHandler(*gin.Context)
}

// Ensure BlogHandler implements BlogHandlerInterface
var _ BlogHandlerInterface = (*BlogHandler)(nil)

type BlogHandler struct {
	blogUsecase usecase.IBlogUseCase
}

func NewBlogHandler(blogUsecase usecase.IBlogUseCase) *BlogHandler {
	return &BlogHandler{
		blogUsecase: blogUsecase,
	}
}

// CreateBlogHandler
func (h *BlogHandler) CreateBlogHandler(cxt *gin.Context) {
	var req dto.CreateBlogRequest
	if err := BindAndValidate(cxt, &req); err != nil {
		ErrorHandler(cxt, http.StatusBadRequest, err.Error())
		return
	}

	// get the author id from the request body as user id which will be of any type
	authorIDAny, exists := cxt.Get("userID")

	if !exists {
		ErrorHandler(cxt, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// assert the type of the user id
	authorID, ok := authorIDAny.(string)
	if !ok {
		ErrorHandler(cxt, http.StatusBadRequest, "Invalid user ID format in token")
		return
	}

	_, err := h.blogUsecase.CreateBlog(cxt.Request.Context(), req.Title, req.Content, authorID, req.Slug, entity.BlogStatus(req.Status), req.FeaturedImageID, req.Tags)

	if err != nil {
		// Map known validation/moderation errors to 400
		if strings.Contains(strings.ToLower(err.Error()), "inappropriate") {
			ErrorHandler(cxt, http.StatusBadRequest, "Content contains inappropriate material")
			return
		}
		ErrorHandler(cxt, http.StatusInternalServerError, "Failed to create blog")
		return
	}

	SuccessHandler(cxt, http.StatusCreated, "Blog created successfully")
}

// GetBlogsHandler
func (h *BlogHandler) GetBlogsHandler(cxt *gin.Context) {
	// 1. get the page size and page number
	pageStr := cxt.DefaultQuery("page", "1")
	pageSizeStr := cxt.DefaultQuery("pageSize", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		ErrorHandler(cxt, http.StatusBadRequest, "Invalid page number")
		return
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		ErrorHandler(cxt, http.StatusBadRequest, "Invalid page size")
		return
	}

	// 2. get sorting parameters
	sortBy := cxt.DefaultQuery("sortBy", "created_at")
	sortOrder := cxt.DefaultQuery("sortOrder", "desc")
	if sortOrder != "asc" && sortOrder != "desc" {
		ErrorHandler(cxt, http.StatusBadRequest, "Invalid sort order. Use 'asc' or 'desc' ")
		return
	}

	// 3. get date filtering parameters
	var dateFrom, dateTo *time.Time
	dateFromStr := cxt.Query("dateFrom")
	if dateFromStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, dateFromStr)
		if err != nil {
			ErrorHandler(cxt, http.StatusBadRequest, "Invalid dateFrom format. Use RFC3339 (e.g., 2025-08-06T15:04:05Z)")
			return
		}
		dateFrom = &parsedTime
	}

	dateToStr := cxt.Query("dateTo")
	if dateToStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			ErrorHandler(cxt, http.StatusBadRequest, "Invalid dateTo format. Use RFC3339 (e.g., 2025-08-06T15:04:05Z)")
			return
		}
		dateTo = &parsedTime
	}

	// call the usecase
	blogs, totalCount, currentPage, totalPages, err := h.blogUsecase.GetBlogs(cxt.Request.Context(), page, pageSize, sortBy, sortOrder, dateFrom, dateTo)
	if err != nil {
		ErrorHandler(cxt, http.StatusInternalServerError, "Failed to get blog posts")
		return
	}

	var blogResponses []dto.BlogResponse
	for _, blog := range blogs {
		blogResponses = append(blogResponses, dto.ToBlogResponse(&blog))
	}

	responses := dto.PaginatedBlogResponse{
		Blogs:       blogResponses,
		TotalCount:  totalCount,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
	}

	SuccessHandler(cxt, http.StatusOK, responses)
}

// GetBlogDetailHandler
func (h *BlogHandler) GetBlogDetailHandler(cxt *gin.Context) {
	slug := cxt.Param("slug")
	blog, err := h.blogUsecase.GetBlogDetail(cxt.Request.Context(), slug)
	if err != nil {
		ErrorHandler(cxt, http.StatusNotFound, "Blog not found")
		return
	}

	SuccessHandler(cxt, http.StatusOK, dto.ToBlogResponse(&blog))
}

// UpdateBlogHandler
func (h *BlogHandler) UpdateBlogHandler(cxt *gin.Context) {
	userIDAny, exists := cxt.Get("userID")
	if !exists {
		ErrorHandler(cxt, http.StatusUnauthorized, "User not authenticated")
		return
	}

	userID, ok := userIDAny.(string)
	if !ok {
		ErrorHandler(cxt, http.StatusBadRequest, "Invalid user ID in token")
	}

	blogID := cxt.Param("blogID")

	var req dto.UpdateBlogRequest
	if err := BindAndValidate(cxt, &req); err != nil {
		ErrorHandler(cxt, http.StatusBadRequest, "Bad request")
		return
	}

	var statusPtr *entity.BlogStatus
	if req.Status != nil {
		s := entity.BlogStatus(*req.Status)
		statusPtr = &s
	}
	blog, err := h.blogUsecase.UpdateBlog(cxt.Request.Context(), blogID, userID, req.Title, req.Content, statusPtr, req.FeaturedImageID)

	if err != nil {
		ErrorHandler(cxt, http.StatusInternalServerError, "Failed to update blog")
		return
	}

	SuccessHandler(cxt, http.StatusOK, dto.ToBlogResponse(blog))

}

// DeleteBlogHandler
func (h *BlogHandler) DeleteBlogHandler(cxt *gin.Context) {
	blogID := cxt.Param("blogID")
	userID, exists := cxt.Get("userID")
	if !exists {
		ErrorHandler(cxt, http.StatusUnauthorized, "User Unauthorized")
		return
	}

	var isAdmin bool
	userRole, exists := cxt.Get("userRole")
	if !exists {
		ErrorHandler(cxt, http.StatusUnauthorized, "User Unauthorized")
		return
	}
	// userRole is likely entity.UserRole, compare as string
	if role, ok := userRole.(string); ok {
		if role == "admin" {
			isAdmin = true
		}
	} else if roleEnum, ok := userRole.(entity.UserRole); ok {
		if string(roleEnum) == "admin" {
			isAdmin = true
		}
	} else if roleEnum, ok := userRole.(interface{ String() string }); ok {
		if roleEnum.String() == "admin" {
			isAdmin = true
		}
	}

	ok, err := h.blogUsecase.DeleteBlog(cxt.Request.Context(), blogID, userID.(string), isAdmin)

	if !ok || err != nil {
		ErrorHandler(cxt, http.StatusInternalServerError, "Failed to delete blog")
		return
	}

	SuccessHandler(cxt, http.StatusNoContent, "Deleted successfully")
}

func (h *BlogHandler) TrackBlogViewHandler(c *gin.Context) {
	blogID := c.Param("blogID")
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// User can be anonymous, so we don't fail if userID is not present.
	userIDAny, _ := c.Get("userID")
	userID, _ := userIDAny.(string)

	err := h.blogUsecase.TrackBlogView(c.Request.Context(), blogID, userID, ipAddress, userAgent)
	if err != nil {
		errMsg := err.Error()
		switch {
		case errMsg == "already viewed recently":
			SuccessHandler(c, http.StatusOK, "Already viewed recently")
			return
		case errMsg == "exceeded view velocity limit: too many views from this IP recently":
			ErrorHandler(c, 429, "Exceeded view velocity limit")
			return
		case errMsg == "exceeded IP rotation limit: too many IPs used by this user recently":
			ErrorHandler(c, 429, "Exceeded IP rotation limit")
			return
		default:
			ErrorHandler(c, http.StatusInternalServerError, "Failed to process blog view")
			return
		}
	}

	SuccessHandler(c, http.StatusOK, "view tracked successfully")
}

// SearchAndFilterBlogsHandler handles searching and filtering blogs
func (h *BlogHandler) SearchAndFilterBlogsHandler(c *gin.Context) {
	// Query and filter params
	query := c.Query("q")
	tags := c.QueryArray("tags")
	var dateFrom, dateTo *time.Time
	if v := c.Query("dateFrom"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			dateFrom = &t
		}
	}
	if v := c.Query("dateTo"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			dateTo = &t
		}
	}
	// Numeric filters
	var minViews, maxViews, minLikes, maxLikes *int
	if v := c.Query("minViews"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			minViews = &n
		}
	}
	if v := c.Query("maxViews"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxViews = &n
		}
	}
	if v := c.Query("minLikes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			minLikes = &n
		}
	}
	if v := c.Query("maxLikes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxLikes = &n
		}
	}
	// Author filter
	var authorID *string
	if v := c.Query("authorID"); v != "" {
		authorID = &v
	}
	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	// Call usecase
	blogs, total, current, pages, err := h.blogUsecase.SearchAndFilterBlogs(c.Request.Context(), query, tags, dateFrom, dateTo, minViews, maxViews, minLikes, maxLikes, authorID, page, pageSize)
	if err != nil {
		ErrorHandler(c, http.StatusInternalServerError, "Failed to search and filter blogs")
		return
	}
	// Map to response
	var resp []dto.BlogResponse
	for _, b := range blogs {
		resp = append(resp, dto.ToBlogResponse(&b))
	}
	result := dto.PaginatedBlogResponse{Blogs: resp, TotalCount: total, CurrentPage: current, TotalPages: pages}
	SuccessHandler(c, http.StatusOK, result)
}

// GetPopularBlogsHandler handles retrieval of popular blogs
func (h *BlogHandler) GetPopularBlogsHandler(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	blogs, total, current, pages, err := h.blogUsecase.GetPopularBlogs(c.Request.Context(), page, pageSize)
	if err != nil {
		ErrorHandler(c, http.StatusInternalServerError, "Failed to get popular blogs")
		return
	}
	var resp []dto.BlogResponse
	for _, b := range blogs {
		resp = append(resp, dto.ToBlogResponse(&b))
	}
	result := dto.PaginatedBlogResponse{Blogs: resp, TotalCount: total, CurrentPage: current, TotalPages: pages}
	SuccessHandler(c, http.StatusOK, result)
}

// SearchAndFilterBlogsHandler

// GetRecommendedBlogsHandler
