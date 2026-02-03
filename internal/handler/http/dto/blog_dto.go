package dto

import (
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
)

// Request DTOs for Blog Handlers

// CreateBlogRequest defines the structure for creating a new blog
type CreateBlogRequest struct {
	Title           string   `json:"title" binding:"required"`
	Content         string   `json:"content" binding:"required"`
	Slug            string   `json:"slug" binding:"required"`
	Status          string   `json:"status" binding:"required,oneof=draft published archived"`
	FeaturedImageID *string  `json:"featured_image_id"`
	Tags            []string `json:"tags"`
}

// UpdateBlogRequest defines the structure for updating an existing blog
type UpdateBlogRequest struct {
	Title           *string  `json:"title"`
	Content         *string  `json:"content"`
	Slug            *string  `json:"slug"`
	Status          *string  `json:"status" binding:"omitempty,oneof=draft published archived"`
	FeaturedImageID *string  `json:"featured_image_id"`
	Tags            []string `json:"tags"`
}

// Response DTOs

// BlogResponse defines the standard JSON response for a single blog
type BlogResponse struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Content         string     `json:"content"`
	AuthorID        string     `json:"author_id"`
	Slug            string     `json:"slug"`
	Status          string     `json:"status"`
	ViewCount       int        `json:"view_count"`
	LikeCount       int        `json:"like_count"`
	CommentCount    int        `json:"comment_count"`
	Popularity      float64    `json:"popularity"`
	FeaturedImageID *string    `json:"featured_image_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
}

// PaginatedBlogResponse defines the structure for a paginated list of blogs.
type PaginatedBlogResponse struct {
	Blogs       []BlogResponse `json:"blogs"`
	TotalCount  int            `json:"total_count"`
	CurrentPage int            `json:"current_page"`
	TotalPages  int            `json:"total_pages"`
}

// DTO Mapper
// a mapper function to convert *entity.Blog to a BlogResponse

func ToBlogResponse(blog *entity.Blog) BlogResponse {
	return BlogResponse{
		ID:              blog.ID,
		Title:           blog.Title,
		Content:         blog.Content,
		AuthorID:        blog.AuthorID,
		Slug:            blog.Slug,
		Status:          string(blog.Status),
		ViewCount:       blog.ViewCount,
		LikeCount:       blog.LikeCount,
		CommentCount:    blog.CommentCount,
		Popularity:      blog.Popularity,
		FeaturedImageID: blog.FeaturedImageID,
		CreatedAt:       blog.CreatedAt,
		UpdatedAt:       blog.UpdatedAt,
		PublishedAt:     blog.PublishedAt,
	}
}
