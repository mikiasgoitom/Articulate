package http

import (
	"time"

	"github.com/didip/tollbooth/v7"
	"github.com/didip/tollbooth/v7/limiter"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/mikiasgoitom/Articulate/internal/domain/contract"
	"github.com/mikiasgoitom/Articulate/internal/handler/http/middleware"
	"github.com/mikiasgoitom/Articulate/internal/usecase"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Router struct {
	userHandler        *UserHandler
	blogHandler        *BlogHandler
	aiHandler          *AIHandler
	emailHandler       *EmailHandler
	interactionHandler *InteractionHandler
	userUsecase        *usecase.UserUsecase
	jwtService         usecase.JWTService
	authHandler        *AuthHandler
	commentHandler     *CommentHandler
}

func NewRouter(userUsecase usecasecontract.IUserUseCase, blogUsecase usecase.IBlogUseCase, likeUsecase *usecase.LikeUsecase, emailVerUC usecasecontract.IEmailVerificationUC, userRepo contract.IUserRepository, tokenRepo contract.ITokenRepository, hasher contract.IHasher, jwtService usecase.JWTService, mailService contract.IEmailService, logger usecasecontract.IAppLogger, config usecasecontract.IConfigProvider, validator usecasecontract.IValidator, uuidGen contract.IUUIDGenerator, randomGen contract.IRandomGenerator, commentRepo contract.ICommentRepository, blogRepo contract.IBlogRepository, aiUsecase usecasecontract.IAIUseCase) *Router {
	baseURL := config.GetAppBaseURL()
	commentUC := usecase.NewCommentUseCase(commentRepo, blogRepo, userRepo)
	return &Router{
		userHandler:        NewUserHandler(userUsecase),
		blogHandler:        NewBlogHandler(blogUsecase),
		aiHandler:          NewAIHandler(aiUsecase),
		emailHandler:       NewEmailHandler(emailVerUC, userRepo),
		interactionHandler: NewInteractionHandler(likeUsecase),
		userUsecase:        usecase.NewUserUsecase(userRepo, tokenRepo, emailVerUC, hasher, jwtService, mailService, logger, config, validator, uuidGen, randomGen),
		jwtService:         jwtService,
		authHandler:        NewAuthHandler(userUsecase, baseURL),
		commentHandler:     NewCommentHandler(commentUC),
	}
}

func (r *Router) SetupRoutes(router *gin.Engine) {
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// rate limiter configuration
	lmt := tollbooth.NewLimiter(10, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})
	lmt.SetIPLookups([]string{"RemoteAddr", "X-Forwarded-For", "X-Real-IP"})
	lmt.SetMessage("Too many requests, please try again later.")
	router.Use(middleware.RateLimiter(lmt))

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/api/v1/metrics", gin.WrapH(promhttp.Handler()))
	// API v1 routes
	v1 := router.Group("/api/v1")

	// Public routes (no authentication required)
	auth := v1.Group("/auth")
	{
		auth.POST("/register", r.userHandler.CreateUser)
		auth.POST("/login", r.userHandler.Login)
		auth.GET("/verify-email", r.emailHandler.HandleVerifyEmailToken)
		auth.POST("/forgot-password", r.userHandler.ForgotPassword)
		auth.POST("/reset-password", r.userHandler.ResetPassword)
		auth.POST("/refresh-token", r.userHandler.RefreshToken)

		auth.POST("/request-verification-email", r.emailHandler.HandleRequestEmailVerification)

		// Google OAuth endpoints
		auth.GET("/google/login", r.authHandler.HandleGoogleLogin)
		auth.GET("/google/callback", r.authHandler.HandleGoogleCallback)
	}

	// Public user routes
	users := v1.Group("/users")
	{
		users.GET("/profile/:id", r.userHandler.GetUser)
	}

	// Public blog routes
	blogs := v1.Group("/blogs")
	{
		blogs.GET("", r.blogHandler.GetBlogsHandler)
		blogs.GET("/search", r.blogHandler.SearchAndFilterBlogsHandler)
		blogs.GET("/popular", r.blogHandler.GetPopularBlogsHandler)
		blogs.GET("/slug/:slug", r.blogHandler.GetBlogDetailHandler)
	}

	// Protected routes (authentication required)
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleWare(r.jwtService, r.userUsecase))
	{
		// Current user routes
		protected.GET("/me", r.userHandler.GetCurrentUser)
		protected.PUT("/me", r.userHandler.UpdateUser)

		// Blog routes
		protected.POST("/blogs", r.blogHandler.CreateBlogHandler)
		protected.POST("/blogs/generateBlog", r.aiHandler.HandleBlogContentGeneration)
		protected.POST("/blogs/suggestModificationByAI", r.aiHandler.HandleSuggestAndModifyContent)
		protected.PUT("/blogs/:blogID", r.blogHandler.UpdateBlogHandler)
		protected.DELETE("/blogs/:blogID", r.blogHandler.DeleteBlogHandler)

		// Interaction routes
		protected.POST("/blogs/:blogID/like", r.interactionHandler.LikeBlogHandler)
		protected.POST("/blogs/:blogID/dislike", r.interactionHandler.DislikeBlogHandler)
		protected.POST("/blogs/:blogID/view", r.blogHandler.TrackBlogViewHandler)

		// Comment CRUD routes
		protected.POST("/blogs/:blogID/comment", r.commentHandler.CreateComment)
		protected.POST("/comments/:commentID/reply", r.commentHandler.CreateReply) // Create a reply to a comment
		protected.GET("/blogs/:blogID/comments", r.commentHandler.GetBlogComments)
		protected.GET("/blogs/:blogID/comments/count", r.commentHandler.GetBlogCommentsCount) // Total comments in a blog
		protected.GET("/comments/:commentID", r.commentHandler.GetComment)                    // Single comment by ID
		protected.GET("/comments/:commentID/replies", r.commentHandler.GetCommentReplies)     // Fetch all replies (nested) for a comment
		protected.GET("/comments/:commentID/count", r.commentHandler.GetCommentStatistics)    // Fetch comment by ID with total reply count
		protected.GET("/comments/:commentID/depth", r.commentHandler.GetCommentDepth)         // Depth of a comment thread
		protected.PUT("/comments/:commentID", r.commentHandler.UpdateComment)
		protected.DELETE("/comments/:commentID", r.commentHandler.DeleteComment)
		protected.GET("/comments/:commentID/thread", r.commentHandler.GetCommentThread) // Fetch comment thread (all nested replies)

		// Comment engagement & moderation
		protected.POST("/comments/:commentID/like", r.commentHandler.LikeComment)
		protected.POST("/comments/:commentID/unlike", r.commentHandler.UnlikeComment)
		protected.POST("/comments/:commentID/report", r.commentHandler.ReportComment)
		protected.PUT("/comments/:commentID/status", r.commentHandler.UpdateCommentStatus)
		protected.GET("/users/:userId/comments", r.commentHandler.GetUserComments)
	}

	// Logout route (no authentication required just accept the refresh token from the request body and invalidate the user session)
	v1.POST("/logout", r.userHandler.Logout)
}
