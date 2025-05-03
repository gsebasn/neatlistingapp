package main

import (
	"log"

	_ "listing/docs"
	"listing/internal/application"
	"listing/internal/domain"
	"listing/internal/infrastructure/repository"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Listing API
// @version         1.0
// @description     A simple listing service API
// @host            localhost:8080
// @BasePath        /api/v1

func main() {
	// Initialize dependencies
	repo := repository.NewMemoryRepository()
	service := application.NewListingService(repo)

	// Initialize router
	r := gin.Default()

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// Health check endpoint
		v1.GET("/health", HealthCheck)

		// Listing endpoints
		v1.GET("/listings", ListListings(service))
		v1.GET("/listings/:id", GetListing(service))
		v1.POST("/listings", CreateListing(service))
		v1.PUT("/listings/:id", UpdateListing(service))
		v1.DELETE("/listings/:id", DeleteListing(service))
	}

	// Swagger documentation endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start the server
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

// HealthCheck godoc
// @Summary      Health check endpoint
// @Description  Get the health status of the API
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
	})
}

// ListListings godoc
// @Summary      List all listings
// @Description  Get all listings
// @Tags         listings
// @Produce      json
// @Success      200  {array}   domain.Listing
// @Router       /listings [get]
func ListListings(service *application.ListingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		listings, err := service.ListListings()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, listings)
	}
}

// GetListing godoc
// @Summary      Get a listing by ID
// @Description  Get a listing by its ID
// @Tags         listings
// @Produce      json
// @Param        id   path      string  true  "Listing ID"
// @Success      200  {object}  domain.Listing
// @Failure      404  {object}  map[string]string
// @Router       /listings/{id} [get]
func GetListing(service *application.ListingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		listing, err := service.GetListing(id)
		if err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, listing)
	}
}

// CreateListing godoc
// @Summary      Create a new listing
// @Description  Create a new listing
// @Tags         listings
// @Accept       json
// @Produce      json
// @Param        listing  body      domain.Listing  true  "Listing object"
// @Success      201     {object}  domain.Listing
// @Failure      400     {object}  map[string]string
// @Router       /listings [post]
func CreateListing(service *application.ListingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var listing domain.Listing
		if err := c.ShouldBindJSON(&listing); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		if err := service.CreateListing(&listing); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(201, listing)
	}
}

// UpdateListing godoc
// @Summary      Update a listing
// @Description  Update an existing listing
// @Tags         listings
// @Accept       json
// @Produce      json
// @Param        id       path      string         true  "Listing ID"
// @Param        listing  body      domain.Listing  true  "Listing object"
// @Success      200     {object}  domain.Listing
// @Failure      400     {object}  map[string]string
// @Router       /listings/{id} [put]
func UpdateListing(service *application.ListingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var listing domain.Listing
		if err := c.ShouldBindJSON(&listing); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		listing.ID = id
		if err := service.UpdateListing(&listing); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, listing)
	}
}

// DeleteListing godoc
// @Summary      Delete a listing
// @Description  Delete a listing by its ID
// @Tags         listings
// @Produce      json
// @Param        id   path      string  true  "Listing ID"
// @Success      204  "No Content"
// @Failure      404  {object}  map[string]string
// @Router       /listings/{id} [delete]
func DeleteListing(service *application.ListingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := service.DeleteListing(id); err != nil {
			c.JSON(404, gin.H{"error": err.Error()})
			return
		}
		c.Status(204)
	}
}
