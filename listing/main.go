package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "listing/docs"
	"listing/internal/api/routes"
	"listing/internal/domain"
	"listing/internal/infrastructure/config"
	"listing/internal/infrastructure/logging"
	"listing/internal/infrastructure/repository"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// @title           Listing API
// @version         1.0
// @description     A real estate listing service API that provides endpoints for managing property listings, including creation, retrieval, updates, and deletion of listings. The API supports filtering listings by various criteria such as price range, property type, number of bedrooms, and listing status.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host            localhost:8080
// @BasePath        /api/v1

// @tag.name        Health
// @tag.description Health check endpoints for monitoring service status and database connectivity

// @tag.name        Listings
// @tag.description Listing management endpoints for creating, reading, updating, and deleting property listings. Supports filtering by status, property type, price range, and number of bedrooms.

// @tag.name        Swagger
// @tag.description Swagger documentation endpoints

// @schemes         http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	logger, err := logging.New(cfg)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("Starting application",
		zap.String("environment", string(cfg.Env)),
		zap.String("port", cfg.Server.Port),
	)

	// Create context with timeout for MongoDB connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MongoDB.Timeout)
	defer cancel()

	// Initialize MongoDB repository
	listingRepo, err := repository.NewMongoRepository[domain.Listing](ctx, cfg.MongoDB.URI, cfg.MongoDB.Database, "listings")
	if err != nil {
		logger.Fatal("Failed to initialize MongoDB repository", zap.Error(err))
	}
	fmt.Printf("Connected to MongoDB database: %s, collection: listings\n", cfg.MongoDB.Database)

	// Verify MongoDB connection
	count, err := listingRepo.Count()
	if err != nil {
		logger.Fatal("Failed to count documents", zap.Error(err))
	}
	fmt.Printf("Found %d documents in listings collection\n", count)

	// Initialize router with all routes
	router := routes.SetupRouter(listingRepo, cfg)

	// Add Swagger documentation endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Create a server with configured timeouts
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting",
			zap.String("port", cfg.Server.Port),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server",
				zap.Error(err),
			)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown",
			zap.Error(err),
		)
	}

	logger.Info("Server exited")
}
