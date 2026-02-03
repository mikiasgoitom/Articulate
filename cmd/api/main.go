package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	handlerHttp "github.com/mikiasgoitom/Articulate/internal/handler/http"
	redisclient "github.com/mikiasgoitom/Articulate/internal/infrastructure/cache"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/config"
	database "github.com/mikiasgoitom/Articulate/internal/infrastructure/database"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/external_services"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/jwt"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/logger"
	passwordservice "github.com/mikiasgoitom/Articulate/internal/infrastructure/password_service"
	randomgenerator "github.com/mikiasgoitom/Articulate/internal/infrastructure/random_generator"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/repository/mongodb"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/store"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/uuidgen"
	"github.com/mikiasgoitom/Articulate/internal/infrastructure/validator"
	"github.com/mikiasgoitom/Articulate/internal/usecase"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get MongoDB URI and DB name from environment
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		log.Fatal("MONGODB_URI environment variable not set")
	}
	dbName := os.Getenv("MONGODB_DB_NAME")
	if dbName == "" {
		log.Fatal("MONGODB_DB_NAME environment variable not set")
	}

	// Establish MongoDB connection
	mongoClient, err := database.NewMongoDBClient(mongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect()

	// Initialize email service
	smtpHost := os.Getenv("EMAIL_HOST")
	smtpPort := os.Getenv("EMAIL_PORT")
	smtpUsername := os.Getenv("EMAIL_USERNAME")
	smtpPassword := os.Getenv("EMAIL_APP_PASSWORD")
	smtpFrom := os.Getenv("EMAIL_FROM")

	// Register custom validators
	validator.RegisterCustomValidators()

	// Initialize Gin router
	router := gin.Default()

	// Dependency Injection: Repositories
	userCollection := mongoClient.Client.Database(dbName).Collection("users")
	userRepo := mongodb.NewMongoUserRepository(userCollection)
	tokenRepo := mongodb.NewTokenRepository(mongoClient.Client.Database(dbName).Collection("tokens"))
	blogRepo := mongodb.NewBlogRepository(mongoClient.Client.Database(dbName), userCollection)
	likeRepo := mongodb.NewLikeRepository(mongoClient.Client.Database(dbName))
	commentRepo := mongodb.NewCommentRepository(mongoClient.Client.Database(dbName))

	// Dependency Injection: Services
	hasher := passwordservice.NewHasher()
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable not set")
	}
	jwtManager := jwt.NewJWTManager(jwtSecret)
	jwtService := jwt.NewJWTService(jwtManager)
	appLogger := logger.NewStdLogger()
	mailService := external_services.NewEmailService(smtpHost, smtpPort, smtpUsername, smtpPassword, smtpFrom)
	randomGenerator := randomgenerator.NewRandomGenerator()
	appValidator := validator.NewValidator()
	uuidGenerator := uuidgen.NewGenerator()
	appConfig := config.NewConfig()
	aiService := external_services.NewGeminiAIService(appConfig.GetAIServiceAPIKey())
	// config
	baseURL := appConfig.GetAppBaseURL()
	// Dependency Injection: Usecases
	aiUsecase := usecase.NewAIUseCase(aiService)
	emailUsecase := usecase.NewEmailVerificationUseCase(tokenRepo, userRepo, mailService, randomGenerator, uuidGenerator, baseURL)
	userUsecase := usecase.NewUserUsecase(userRepo, tokenRepo, emailUsecase, hasher, jwtService, mailService, appLogger, appConfig, appValidator, uuidGenerator, randomGenerator)

	blogUsecase := usecase.NewBlogUseCase(blogRepo, uuidGenerator, appLogger, aiUsecase)

	// Pass Prometheus metrics to handlers or usecases as needed (import from metrics package)

	// Optional Dependency Injection: Redis cache
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		rdb := redisclient.NewRedisFromURL(context.Background(), redisURL)
		defer redisclient.Close(rdb)
		blogCache := store.NewBlogCacheStore(rdb)
		blogUsecase.SetBlogCache(blogCache)
	}

	// Create like usecase
	likeUsecase := usecase.NewLikeUsecase(likeRepo, blogRepo)

	// Setup API routes
	appRouter := handlerHttp.NewRouter(
		userUsecase, blogUsecase, likeUsecase, emailUsecase,
		userRepo, tokenRepo, hasher, jwtService, mailService,
		appLogger, appConfig, appValidator, uuidGenerator, randomGenerator,
		commentRepo, blogRepo, aiUsecase,
	)
	appRouter.SetupRoutes(router)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
