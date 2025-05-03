package routes

import (
	"listing/internal/domain/listing"
	"listing/internal/infrastructure/api/handlers"
	"listing/internal/infrastructure/middleware"

	"reflect"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all routes for the application
func SetupRoutes(router *gin.Engine, listingHandler *handlers.ListingHandler) {
	// Create validator middleware
	validator := middleware.NewValidator()

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check endpoint
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
			})
		})

		// Listing routes
		listings := v1.Group("/listings")
		{
			listings.POST("",
				validator.ValidateRequest(reflect.TypeOf(listing.CreateListingRequest{})),
				listingHandler.CreateListing,
			)
			listings.PUT("/:id",
				validator.ValidatePath(reflect.TypeOf(listing.ListingIDRequest{})),
				validator.ValidateRequest(reflect.TypeOf(listing.UpdateListingRequest{})),
				listingHandler.UpdateListing,
			)
			listings.GET("/:id",
				validator.ValidatePath(reflect.TypeOf(listing.ListingIDRequest{})),
				listingHandler.GetListing,
			)
			listings.GET("",
				validator.ValidateQuery(reflect.TypeOf(listing.ListListingsRequest{})),
				listingHandler.ListListings,
			)
			listings.DELETE("/:id",
				validator.ValidatePath(reflect.TypeOf(listing.ListingIDRequest{})),
				listingHandler.DeleteListing,
			)
		}
	}
}
