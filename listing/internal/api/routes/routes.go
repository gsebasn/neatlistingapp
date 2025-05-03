package routes

import (
	"listing/internal/api/handlers"
	"listing/internal/domain"
	"listing/internal/infrastructure/config"
	"listing/internal/infrastructure/logging"
	"listing/internal/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter configures all routes for the application
func SetupRouter(repo domain.ListingRepository, cfg *config.Config) *gin.Engine {
	// Create logger
	logger, err := logging.New(cfg)
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	// Create router
	router := gin.New()

	// Add middleware
	router.Use(gin.Recovery())
	router.Use(middleware.CorrelationMiddleware())
	router.Use(middleware.LoggingMiddleware(logger, cfg))

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Create handlers
	listingHandler := handlers.NewListingHandler(repo)

	// API routes
	api := router.Group("/api/v1")
	{
		// Health check route
		api.GET("/listing/health", listingHandler.HealthCheck)

		// Listing routes
		listings := api.Group("/listings")
		{
			listings.GET("", listingHandler.ListListings)
			listings.GET("/:id", listingHandler.GetListing)
			listings.POST("", listingHandler.CreateListing)
			listings.PUT("/:id", listingHandler.UpdateListing)
			listings.DELETE("/:id", listingHandler.DeleteListing)
		}
	}

	return router
}
