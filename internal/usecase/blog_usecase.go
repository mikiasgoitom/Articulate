package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
	"github.com/mikiasgoitom/Articulate/internal/domain/entity"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/metrics"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
	"github.com/mikiasgoitom/Articulate/internal/utils"
)

// IBlogUseCase defines blog-related business logic
type IBlogUseCase interface {
	CreateBlog(ctx context.Context, title, content string, authorID string, slug string, status entity.BlogStatus, featuredImageID *string, tags []string) (*entity.Blog, error)
	GetBlogs(ctx context.Context, page, pageSize int, sortBy string, sortOrder string, dateFrom *time.Time, dateTo *time.Time) (blogs []entity.Blog, totalCount int, currentPage int, totalPages int, err error)
	GetBlogDetail(cnt context.Context, slug string) (blog entity.Blog, err error)
	UpdateBlog(ctx context.Context, blogID, authorID string, title *string, content *string, status *entity.BlogStatus, featuredImageID *string) (*entity.Blog, error)
	DeleteBlog(ctx context.Context, blogID, userID string, isAdmin bool) (bool, error)
	SearchAndFilterBlogs(ctx context.Context, query string, tags []string, dateFrom *time.Time, dateTo *time.Time, minViews *int, maxViews *int, minLikes *int, maxLikes *int, authorID *string, page int, pageSize int) ([]entity.Blog, int, int, int, error)
	TrackBlogView(ctx context.Context, blogID, userID, ipAddress, userAgent string) error
	GetPopularBlogs(ctx context.Context, page, pageSize int) ([]entity.Blog, int, int, int, error)
}

// BlogStatus is defined in entity.BlogStatus

// BlogUseCaseImpl implements the BlogUseCase interface
type BlogUseCaseImpl struct {
	blogRepo  contract.IBlogRepository
	uuidgen   contract.IUUIDGenerator
	logger    usecasecontract.IAppLogger
	aiUC      usecasecontract.IAIUseCase
	blogCache contract.IBlogCache
	// simple metrics
	detailHits uint64
	detailMiss uint64
	listHits   uint64
	listMiss   uint64
}

// NewBlogUseCase creates a new instance of BlogUseCase
func NewBlogUseCase(blogRepo contract.IBlogRepository, uuidgenrator contract.IUUIDGenerator, logger usecasecontract.IAppLogger, aiUC usecasecontract.IAIUseCase) *BlogUseCaseImpl {
	return &BlogUseCaseImpl{
		blogRepo: blogRepo,
		logger:   logger,
		uuidgen:  uuidgenrator,
		aiUC:     aiUC,
	}
}

// check if BlogUseCaseImpl implements the IBlogUseCase
var _ IBlogUseCase = (*BlogUseCaseImpl)(nil)

// separate blog instance for blogCache injection
func (uc *BlogUseCaseImpl) SetBlogCache(cache contract.IBlogCache) {
	uc.blogCache = cache
}

// buildBlogsListCacheKey builds a stable key for list endpoint caching
func buildBlogsListCacheKey(page, pageSize int, sortBy string, sortOrder string, dateFrom, dateTo *time.Time) string {
	df := ""
	dt := ""
	if dateFrom != nil {
		df = dateFrom.UTC().Format(time.RFC3339)
	}
	if dateTo != nil {
		dt = dateTo.UTC().Format(time.RFC3339)
	}
	return fmt.Sprintf("blogs:list:p=%d:s=%d:sb=%s:so=%s:df=%s:dt=%s", page, pageSize, sortBy, sortOrder, df, dt)
}

// CreateBlog creates a new blog post
func (uc *BlogUseCaseImpl) CreateBlog(ctx context.Context, title, content string, authorID string, slug string, status entity.BlogStatus, featuredImageID *string, tags []string) (*entity.Blog, error) {
	if title == "" {
		return nil, errors.New("title is required")
	}
	if content == "" {
		return nil, errors.New("content is required")
	}
	if authorID == "" {
		return nil, errors.New("author ID is required")
	}

	// If slug is not provided, generate it from the title
	if slug == "" {
		slug = strings.ReplaceAll(strings.ToLower(title), " ", "-")
	}

	blog := &entity.Blog{
		ID:              uc.uuidgen.NewUUID(),
		Title:           title,
		Content:         content,
		AuthorID:        authorID,
		Slug:            slug + "-" + uc.uuidgen.NewUUID(), // A UUID is always appended to ensure the final slug is unique
		Status:          entity.BlogStatus(status),
		Tags:            tags,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		ViewCount:       0,
		LikeCount:       0,
		DislikeCount:    0,
		CommentCount:    0,
		Popularity:      utils.CalculatePopularity(0, 0, 0, 0),
		FeaturedImageID: featuredImageID,
		IsDeleted:       false,
	}

	if status == entity.BlogStatusPublished {
		now := time.Now()
		blog.PublishedAt = &now
	}
	// Check for profanity in the content using AI. If AI check fails (e.g., not configured or service error), proceed but log a warning.
	if uc.aiUC != nil {
		feedback, err := uc.aiUC.CensorAndCheckBlog(ctx, content)
		if err != nil {
			if uc.logger != nil {
				uc.logger.Warningf("AI moderation unavailable, proceeding without block: %v", err)
			}
		} else {
			// Normalize AI feedback and block only on an explicit "no"
			norm := strings.TrimSpace(strings.ToLower(feedback))
			if norm == "no" {
				return nil, errors.New("content contains inappropriate material")
			}
		}
	}

	if err := uc.blogRepo.CreateBlog(ctx, blog); err != nil {
		uc.logger.Errorf("failed to create blog: %v", err)
		return nil, fmt.Errorf("failed to create blog: %w", err)
	}
	// Add tags to blog if provided
	if len(tags) > 0 {
		err := uc.blogRepo.AddTagsToBlog(ctx, blog.ID, tags)
		if err != nil {
			uc.logger.Errorf("Failed to add tags to blog: %v", err)
			// Not returning error here to allow blog creation to succeed even if tag association fails
		}
	}

	// Invalidate list caches after creating a blog
	if uc.blogCache != nil {
		_ = uc.blogCache.InvalidateBlogLists(ctx)
	}
	return blog, nil
}

// GetBlogs retrieves paginated list of blogs
func (uc *BlogUseCaseImpl) GetBlogs(ctx context.Context, page, pageSize int, sortBy string, sortOrder string, dateFrom *time.Time, dateTo *time.Time) ([]entity.Blog, int, int, int, error) {

	// Try cache first
	if uc.blogCache != nil {
		key := buildBlogsListCacheKey(page, pageSize, sortBy, sortOrder, dateFrom, dateTo)
		t0 := time.Now()
		cached, found, err := uc.blogCache.GetBlogsPage(ctx, key)
		elapsed := time.Since(t0)
		if err == nil && found && cached != nil {
			atomic.AddUint64(&uc.listHits, 1)
			go metrics.IncListHit()
			go metrics.AddHitDuration(elapsed.Seconds())
			if uc.logger != nil {
				uc.logger.Infof("cache hit: blogs list key=%s took=%s", key, elapsed)
			}
			total := cached.Total
			totalPages := 0
			if pageSize > 0 {
				totalPages = (total + pageSize - 1) / pageSize
			}
			return cached.Blogs, total, page, totalPages, nil
		} else if err == nil && !found {
			atomic.AddUint64(&uc.listMiss, 1)
			go metrics.IncListMiss()
			go metrics.AddMissDuration(elapsed.Seconds())
			if uc.logger != nil {
				uc.logger.Infof("cache miss: blogs list key=%s took=%s", key, elapsed)
			}
		} else if err != nil && uc.logger != nil {
			uc.logger.Warningf("cache error: blogs list key=%s err=%v took=%s", key, err, elapsed)
		}
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	filterOptions := &contract.BlogFilterOptions{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: string(sortOrder),
		DateFrom:  dateFrom,
		DateTo:    dateTo,
	}

	// Only return published or archived blogs (not drafts)
	dbStart := time.Now()
	blogs, totalCount, err := uc.blogRepo.GetBlogs(ctx, filterOptions)
	if err != nil {
		uc.logger.Errorf("failed to get blogs: %v", err)
		return nil, 0, 0, 0, fmt.Errorf("failed to get blogs: %w", err)
	}
	if uc.logger != nil {
		uc.logger.Infof("db fetch: blogs list page=%d size=%d took=%s", page, pageSize, time.Since(dbStart))
	}

	var filteredBlogs []entity.Blog
	for _, blog := range blogs {
		if blog.Status == entity.BlogStatusPublished || blog.Status == entity.BlogStatusArchived {
			filteredBlogs = append(filteredBlogs, *blog)
		}
	}

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	// If there is a cache miss before retuning save the results to the cache
	if uc.blogCache != nil {
		key := buildBlogsListCacheKey(page, pageSize, sortBy, sortOrder, dateFrom, dateTo)
		_ = uc.blogCache.SetBlogsPage(ctx, key, &contract.CachedBlogsPage{Blogs: filteredBlogs, Total: int(totalCount)})
		if uc.logger != nil {
			uc.logger.Infof("cache set: blogs list key=%s size=%d ttl=%s", key, len(filteredBlogs), 5*time.Minute)
		}
	}

	return filteredBlogs, int(totalCount), page, totalPages, nil
}

// GetBlogDetail retrieves a blog by its slug
func (uc *BlogUseCaseImpl) GetBlogDetail(ctx context.Context, slug string) (entity.Blog, error) {
	if slug == "" {
		return entity.Blog{}, errors.New("slug is required")
	}

	// Cache first
	if uc.blogCache != nil {
		t0 := time.Now()
		cached, found, err := uc.blogCache.GetBlogBySlug(ctx, slug)
		elapsed := time.Since(t0)
		if err == nil && found && cached != nil {
			atomic.AddUint64(&uc.detailHits, 1)
			go metrics.IncDetailHit()
			go metrics.AddHitDuration(elapsed.Seconds())
			if uc.logger != nil {
				uc.logger.Infof("cache hit: blog detail slug=%s took=%s", slug, elapsed)
			}
			if cached.Status == entity.BlogStatusPublished || cached.Status == entity.BlogStatusArchived {
				return *cached, nil
			}
		} else if err == nil && !found {
			atomic.AddUint64(&uc.detailMiss, 1)
			go metrics.IncDetailMiss()
			go metrics.AddMissDuration(elapsed.Seconds())
			if uc.logger != nil {
				uc.logger.Infof("cache miss: blog detail slug=%s took=%s", slug, elapsed)
			}
		} else if err != nil && uc.logger != nil {
			uc.logger.Warningf("cache error: blog detail slug=%s err=%v took=%s", slug, err, elapsed)
		}
	}

	dbStart := time.Now()
	blog, err := uc.blogRepo.GetBlogBySlug(ctx, slug)
	if err != nil {
		uc.logger.Errorf("failed to get blog by slug: %v", err)
		return entity.Blog{}, fmt.Errorf("failed to get blog: %w", err)
	}
	if uc.logger != nil {
		uc.logger.Infof("db fetch: blog detail slug=%s took=%s", slug, time.Since(dbStart))
	}
	if blog == nil || blog.IsDeleted {
		return entity.Blog{}, errors.New("blog not found")
	}
	// Only allow published or archived blogs to be fetched by slug
	if blog.Status != entity.BlogStatusPublished && blog.Status != entity.BlogStatusArchived {
		return entity.Blog{}, errors.New("blog not found")
	}

	// Set cache on successful DB fetch
	if uc.blogCache != nil {
		_ = uc.blogCache.SetBlogBySlug(ctx, slug, blog)
	}
	return *blog, nil
}

// UpdateBlog updates an existing blog post
func (uc *BlogUseCaseImpl) UpdateBlog(ctx context.Context, blogID, authorID string, title *string, content *string, status *entity.BlogStatus, featuredImageID *string) (*entity.Blog, error) {
	if blogID == "" {
		return nil, errors.New("blog ID is required")
	}
	if authorID == "" {
		return nil, errors.New("author ID is required")
	}

	// Get existing blog
	blog, err := uc.blogRepo.GetBlogByID(ctx, blogID)
	if err != nil {
		uc.logger.Errorf("failed to get blog: %v", err)
		return nil, fmt.Errorf("failed to get blog: %w", err)
	}
	if blog == nil {
		return nil, errors.New("blog not found")
	}

	// Check if user is the author
	if blog.AuthorID != authorID {
		return nil, errors.New("unauthorized: only the author can update this blog")
	}

	updates := make(map[string]interface{})
	oldSlug := blog.Slug

	if title != nil {
		updates["title"] = *title
		// Generate a new slug from the new title
		newSlug := strings.ReplaceAll(strings.ToLower(*title), " ", "-")
		updates["slug"] = newSlug + "-" + uc.uuidgen.NewUUID()
	}
	if content != nil {
		updates["content"] = *content
		// if content is edited check for profanity
		feedback, err := uc.aiUC.CensorAndCheckBlog(ctx, *content)
		if err != nil {
			return nil, fmt.Errorf("failed to check content: %w", err)
		}
		if feedback == "no" {
			return nil, errors.New("content contains inappropriate material")
		}
	}

	if status != nil {
		updates["status"] = *status
		if *status == entity.BlogStatusPublished && blog.PublishedAt == nil {
			now := time.Now()
			updates["published_at"] = &now
		}
	}

	if featuredImageID != nil {
		updates["featured_image_id"] = *featuredImageID
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := uc.blogRepo.UpdateBlog(ctx, blogID, updates); err != nil {
			uc.logger.Errorf("failed to update blog: %v", err)
			return nil, fmt.Errorf("failed to update blog: %w", err)
		}
	}

	// Return updated blog
	updatedBlog, err := uc.blogRepo.GetBlogByID(ctx, blogID)
	if err != nil {
		uc.logger.Errorf("failed to get updated blog: %v", err)
		return nil, fmt.Errorf("failed to get updated blog: %w", err)
	}

	// Invalidate caches after update
	if uc.blogCache != nil {
		_ = uc.blogCache.InvalidateBlogLists(ctx)
		if updatedBlog != nil && updatedBlog.Slug != "" {
			_ = uc.blogCache.InvalidateBlogBySlug(ctx, updatedBlog.Slug)
		}
		// If slug changed, invalidate the old slug key as well
		if oldSlug != "" && updatedBlog != nil && updatedBlog.Slug != oldSlug {
			_ = uc.blogCache.InvalidateBlogBySlug(ctx, oldSlug)
		}
	}

	return updatedBlog, nil
}

// DeleteBlog deletes a blog post
func (uc *BlogUseCaseImpl) DeleteBlog(ctx context.Context, blogID, userID string, isAdmin bool) (bool, error) {
	if blogID == "" {
		return false, errors.New("blog ID is required")
	}
	if userID == "" {
		return false, errors.New("user ID is required")
	}

	blog, err := uc.blogRepo.GetBlogByID(ctx, blogID)
	if err != nil {
		uc.logger.Errorf("failed to get blog: %v", err)
		return false, fmt.Errorf("failed to get blog: %w", err)
	}
	if blog == nil {
		return false, errors.New("blog not found")
	}

	// Check authorization
	if !isAdmin && blog.AuthorID != userID {
		return false, errors.New("unauthorized: only the author or admin can delete this blog")
	}

	if err := uc.blogRepo.DeleteBlog(ctx, blogID); err != nil {
		uc.logger.Errorf("failed to delete blog: %v", err)
		return false, fmt.Errorf("failed to delete blog: %w", err)
	}

	// Invalidate caches after delete
	if uc.blogCache != nil {
		_ = uc.blogCache.InvalidateBlogLists(ctx)
		if blog.Slug != "" {
			_ = uc.blogCache.InvalidateBlogBySlug(ctx, blog.Slug)
		}
	}

	return true, nil
}

// TrackBlogView tracks a view on a blog post, ensuring it's authentic by checking user ID, IP address, and User-Agent.

// isBot returns true if the User-Agent string matches common bot patterns.
func isBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	botSignatures := []string{"bot", "spider", "crawl", "slurp", "curl", "wget", "python-requests", "httpclient", "feedfetcher", "mediapartners-google"}
	for _, sig := range botSignatures {
		if strings.Contains(ua, sig) {
			return true
		}
	}
	return false
}
func (uc *BlogUseCaseImpl) TrackBlogView(ctx context.Context, blogID, userID, ipAddress, userAgent string) error {
	if blogID == "" {
		return errors.New("blog ID is required")
	}

	// For a view to be considered unique, either the userID (if logged in) or the IP address must be provided.
	if userID == "" && ipAddress == "" {
		return errors.New("unable to track view without user ID or IP address")
	}

	// 1. Basic Bot Detection
	if isBot(userAgent) {
		uc.logger.Infof("Bot detected, view not counted for blog %s. User-Agent: %s", blogID, userAgent)
		return nil
	}

	// 2. Check for recent view from this user/IP for this specific blog post
	hasViewed, err := uc.blogRepo.HasViewedRecently(ctx, blogID, userID, ipAddress)
	if err != nil {
		uc.logger.Errorf("failed to check for recent blog view: %v", err)
		return fmt.Errorf("failed to check for recent blog view: %w", err)
	}
	if hasViewed {
		// Already viewed recently: return sentinel error for handler
		uc.logger.Infof("User %s or IP %s already viewed blog %s recently", userID, ipAddress, blogID)
		return errors.New("already viewed recently")
	}

	// 3. Advanced Velocity & Rotation Checks (using Redis cache)
	const (
		maxIpVelocity     = 10      // max 10 views from one IP in 5 mins
		ipVelocityTTL     = 5 * 60  // 5 minutes in seconds
		maxUserIPs        = 5       // max 5 different IPs for one user in 1 hour
		userIPRotationTTL = 60 * 60 // 60 minutes in seconds
	)
	if uc.blogCache != nil {
		// IP velocity check: Has this IP viewed too many different blogs in the last 5 minutes?
		// Add this view to the IP's recent views set
		_ = uc.blogCache.AddRecentViewByIP(ctx, ipAddress, blogID, int64(ipVelocityTTL))
		ipViewCount, err := uc.blogCache.GetRecentViewCountByIP(ctx, ipAddress)
		if err == nil {
			if ipViewCount > int64(maxIpVelocity) {
				uc.logger.Warningf("High IP velocity detected for %s. Views: %d", ipAddress, ipViewCount)
				return fmt.Errorf("exceeded view velocity limit: too many views from this IP recently")
			}
		} else {
			// Redis failed, fallback to DB
			shortWindow := time.Now().Add(-5 * time.Minute)
			ipViews, dbErr := uc.blogRepo.GetRecentViewsByIP(ctx, ipAddress, shortWindow)
			if dbErr == nil && len(ipViews) > maxIpVelocity {
				uc.logger.Warningf("[DB Fallback] High IP velocity detected for %s. Views: %d", ipAddress, len(ipViews))
				return fmt.Errorf("exceeded view velocity limit: too many views from this IP recently")
			}
		}

		// User-IP rotation check: Has this user account used too many IPs in the last 1 hour?
		// Add this IP to the user's recent IPs set
		if userID != "" {
			_ = uc.blogCache.AddRecentViewByUser(ctx, userID, ipAddress, int64(userIPRotationTTL))
			userIPCount, err := uc.blogCache.GetRecentIPCountByUser(ctx, userID)
			if err == nil {
				if userIPCount > int64(maxUserIPs) {
					uc.logger.Warningf("High IP rotation detected for user %s. IPs used: %d", userID, userIPCount)
					return fmt.Errorf("exceeded IP rotation limit: too many IPs used by this user recently")
				}
			} else {
				// Redis failed, fallback to DB
				mediumWindow := time.Now().Add(-60 * time.Minute)
				userViews, dbErr := uc.blogRepo.GetRecentViewsByUser(ctx, userID, mediumWindow)
				if dbErr == nil {
					ipSet := make(map[string]struct{})
					for _, view := range userViews {
						ipSet[view.IPAddress] = struct{}{}
					}
					if len(ipSet) > maxUserIPs {
						uc.logger.Warningf("[DB Fallback] High IP rotation detected for user %s. IPs used: %d", userID, len(ipSet))
						return fmt.Errorf("exceeded IP rotation limit: too many IPs used by this user recently")
					}
				}
			}
		}
	}

	// If all checks pass, increment the view count and record the view on the DB
	if err := uc.blogRepo.IncrementViewCount(ctx, blogID); err != nil {
		uc.logger.Errorf("failed to increment view count: %v", err)
		return fmt.Errorf("failed to increment view count: %w", err)
	}

	if err := uc.blogRepo.RecordView(ctx, blogID, userID, ipAddress, userAgent); err != nil {
		uc.logger.Errorf("failed to record user view: %v", err)
		return fmt.Errorf("failed to record user view: %w", err)
	}

	// Update popularity after view
	if err := uc.UpdateBlogPopularity(ctx, blogID); err != nil {
		uc.logger.Errorf("failed to update blog popularity after view: %v", err)
	}
	return nil
}

// GetPopularBlogs returns blogs sorted by view count (descending), paginated.
func (uc *BlogUseCaseImpl) GetPopularBlogs(ctx context.Context, page, pageSize int) ([]entity.Blog, int, int, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	filterOptions := &contract.BlogFilterOptions{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "popularity",
		SortOrder: "desc",
	}

	blogs, totalCount, err := uc.blogRepo.GetBlogs(ctx, filterOptions)
	if err != nil {
		uc.logger.Errorf("failed to get popular blogs: %v", err)
		return nil, 0, 0, 0, fmt.Errorf("failed to get popular blogs: %w", err)
	}

	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}

	var blogEntities []entity.Blog
	for _, blog := range blogs {
		blogEntities = append(blogEntities, *blog)
	}

	return blogEntities, int(totalCount), page, totalPages, nil
}

// SearchAndFilterBlogs implements advanced search and filtering for blogs.
func (uc *BlogUseCaseImpl) SearchAndFilterBlogs(
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
) ([]entity.Blog, int, int, int, error) {
	filterOptions := &contract.BlogFilterOptions{
		Page:     page,
		PageSize: pageSize,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		MinViews: minViews,
		MaxViews: maxViews,
		MinLikes: minLikes,
		MaxLikes: maxLikes,
		AuthorID: authorID,
		TagIDs:   tags,
	}
	var blogs []*entity.Blog
	var totalCount int64
	var err error
	if query != "" {
		blogs, totalCount, err = uc.blogRepo.SearchBlogs(ctx, query, filterOptions)
	} else {
		blogs, totalCount, err = uc.blogRepo.GetBlogs(ctx, filterOptions)
	}
	if err != nil {
		uc.logger.Errorf("failed to search/filter blogs: %v", err)
		return nil, 0, 0, 0, fmt.Errorf("failed to search/filter blogs: %w", err)
	}
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize != 0 {
		totalPages++
	}
	var blogEntities []entity.Blog
	for _, blog := range blogs {
		blogEntities = append(blogEntities, *blog)
	}
	return blogEntities, int(totalCount), page, totalPages, nil
}

// UpdateBlogPopularity fetches counts and updates the popularity field in the DB
func (uc *BlogUseCaseImpl) UpdateBlogPopularity(ctx context.Context, blogID string) error {
	views, likes, dislikes, comments, err := uc.blogRepo.GetBlogCounts(ctx, blogID)
	if err != nil {
		return err
	}
	popularity := utils.CalculatePopularity(views, likes, dislikes, comments)
	updates := map[string]interface{}{"popularity": popularity}
	return uc.blogRepo.UpdateBlog(ctx, blogID, updates)
}
