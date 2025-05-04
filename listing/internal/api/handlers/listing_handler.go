package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"listing/internal/domain"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ListingHandler handles HTTP requests for listings
type ListingHandler struct {
	repo domain.ListingRepository
}

// NewListingHandler creates a new instance of ListingHandler
func NewListingHandler(repo domain.ListingRepository) *ListingHandler {
	return &ListingHandler{
		repo: repo,
	}
}

// GetListing handles GET /api/v1/listings/:id
// @Summary      Get a listing by ID
// @Description  Get details of a specific listing by its ID.
// @Tags         listings
// @Produce      json
// @Param        id   path      string  true  "Listing ID"
// @Success      200  {object}  domain.Listing
// @Failure      404  {object}  map[string]string
// @Router       /listings/{id} [get]
func (h *ListingHandler) GetListing(c *gin.Context) {
	id := c.Param("id")
	listing, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}
	c.JSON(http.StatusOK, listing)
}

// CreateListing handles POST /api/v1/listings
// @Summary      Create a new listing
// @Description  Add a new property listing.
// @Tags         listings
// @Accept       json
// @Produce      json
// @Param        listing  body      domain.Listing  true  "Listing to create"
// @Success      201      {object}  domain.Listing
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /listings [post]
func (h *ListingHandler) CreateListing(c *gin.Context) {
	var listing domain.Listing
	if err := c.ShouldBindJSON(&listing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate a new ObjectID
	listing.ID = primitive.NewObjectID()

	if err := h.repo.Create(&listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listing"})
		return
	}

	c.JSON(http.StatusCreated, listing)
}

// UpdateListing handles PUT /api/v1/listings/:id
// @Summary      Update a listing
// @Description  Update an existing property listing by ID.
// @Tags         listings
// @Accept       json
// @Produce      json
// @Param        id      path      string         true  "Listing ID"
// @Param        listing body      domain.Listing true  "Listing to update"
// @Success      200     {object}  domain.Listing
// @Failure      400     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /listings/{id} [put]
func (h *ListingHandler) UpdateListing(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var listing domain.Listing
	if err := c.ShouldBindJSON(&listing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listing.ID = objectID
	if err := h.repo.Update(&listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update listing"})
		return
	}

	c.JSON(http.StatusOK, listing)
}

// DeleteListing handles DELETE /api/v1/listings/:id
// @Summary      Delete a listing
// @Description  Delete a property listing by ID.
// @Tags         listings
// @Param        id   path      string  true  "Listing ID"
// @Success      204  "No Content"
// @Failure      500  {object}  map[string]string
// @Router       /listings/{id} [delete]
func (h *ListingHandler) DeleteListing(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete listing"})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListListings handles GET /api/v1/listings
// @Summary      List all listings
// @Description  Get a list of all property listings, with optional filters.
// @Tags         listings
// @Produce      json
// @Param        status        query     string  false  "Listing status"
// @Param        propertyType  query     string  false  "Property type"
// @Param        minPrice      query     int     false  "Minimum price"
// @Param        maxPrice      query     int     false  "Maximum price"
// @Param        minBedrooms   query     int     false  "Minimum bedrooms"
// @Param        maxBedrooms   query     int     false  "Maximum bedrooms"
// @Success      200  {array}   domain.Listing
// @Failure      500  {object}  map[string]string
// @Router       /listings [get]
func (h *ListingHandler) ListListings(c *gin.Context) {
	// Get query parameters
	status := c.Query("status")
	propertyType := c.Query("propertyType")
	minPrice := c.Query("minPrice")
	maxPrice := c.Query("maxPrice")
	minBedrooms := c.Query("minBedrooms")
	maxBedrooms := c.Query("maxBedrooms")

	// Get all listings
	listings, err := h.repo.List()
	if err != nil {
		fmt.Printf("Error fetching listings: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch listings: %v", err)})
		return
	}

	// Initialize empty slice if no listings found
	if listings == nil {
		listings = make([]*domain.Listing, 0)
	}

	fmt.Printf("Retrieved %d listings from repository\n", len(listings))

	// Apply filters if provided
	filteredListings := listings
	if status != "" {
		filteredListings = filterByStatus(filteredListings, domain.ListingStatus(status))
	}
	if propertyType != "" {
		filteredListings = filterByPropertyType(filteredListings, domain.PropertyType(propertyType))
	}
	if minPrice != "" {
		if min, err := strconv.Atoi(minPrice); err == nil {
			filteredListings = filterByMinPrice(filteredListings, min)
		}
	}
	if maxPrice != "" {
		if max, err := strconv.Atoi(maxPrice); err == nil {
			filteredListings = filterByMaxPrice(filteredListings, max)
		}
	}
	if minBedrooms != "" {
		if min, err := strconv.Atoi(minBedrooms); err == nil {
			filteredListings = filterByMinBedrooms(filteredListings, min)
		}
	}
	if maxBedrooms != "" {
		if max, err := strconv.Atoi(maxBedrooms); err == nil {
			filteredListings = filterByMaxBedrooms(filteredListings, max)
		}
	}

	// Initialize empty slice if no filtered listings found
	if filteredListings == nil {
		filteredListings = make([]*domain.Listing, 0)
	}

	fmt.Printf("Returning %d filtered listings\n", len(filteredListings))
	c.JSON(http.StatusOK, filteredListings)
}

// Helper filter functions
func filterByStatus(listings []*domain.Listing, status domain.ListingStatus) []*domain.Listing {
	var filtered []*domain.Listing
	for _, l := range listings {
		if l.Status == status {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

func filterByPropertyType(listings []*domain.Listing, propertyType domain.PropertyType) []*domain.Listing {
	var filtered []*domain.Listing
	for _, l := range listings {
		if l.PropertyType == propertyType {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

func filterByMinPrice(listings []*domain.Listing, minPrice int) []*domain.Listing {
	var filtered []*domain.Listing
	for _, l := range listings {
		if l.Price >= minPrice {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

func filterByMaxPrice(listings []*domain.Listing, maxPrice int) []*domain.Listing {
	var filtered []*domain.Listing
	for _, l := range listings {
		if l.Price <= maxPrice {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

func filterByMinBedrooms(listings []*domain.Listing, minBedrooms int) []*domain.Listing {
	var filtered []*domain.Listing
	for _, l := range listings {
		if l.Bedrooms >= minBedrooms {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

func filterByMaxBedrooms(listings []*domain.Listing, maxBedrooms int) []*domain.Listing {
	var filtered []*domain.Listing
	for _, l := range listings {
		if l.Bedrooms <= maxBedrooms {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

// HealthCheck handles GET /api/v1/listings/health
// @Summary      Health check endpoint
// @Description  Check if the service is healthy and database connection is working. This endpoint verifies both the API service availability and the MongoDB database connectivity.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]string{status=string,message=string}  "Service is healthy"
// @Success      503  {object}  map[string]string{status=string,error=string}    "Service is unhealthy"
// @Example      {json} 200 Response
//
//	{
//	  "status": "healthy",
//	  "message": "Service is up and running"
//	}
//
// @Example      {json} 503 Response
//
//	{
//	  "status": "unhealthy",
//	  "error": "Database connection failed: connection refused"
//	}
//
// @Router       /listings/health [get]
func (h *ListingHandler) HealthCheck(c *gin.Context) {
	// Check database connection by attempting to list listings
	err := h.repo.Ping()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  fmt.Sprintf("Database connection failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "Service is up and running",
	})
}
