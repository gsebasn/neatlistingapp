package handlers

import (
	"net/http"
	"time"

	"listing/internal/application"
	"listing/internal/domain"
	"listing/internal/domain/listing"
	"listing/internal/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

// ListingHandler handles HTTP requests for listings
type ListingHandler struct {
	service *application.ListingService
}

// NewListingHandler creates a new ListingHandler
func NewListingHandler(service *application.ListingService) *ListingHandler {
	return &ListingHandler{
		service: service,
	}
}

// CreateListing handles the creation of a new listing
func (h *ListingHandler) CreateListing(c *gin.Context) {
	// Get the validated request from context
	req := middleware.GetValidatedModel(c).(*listing.CreateListingRequest)

	// Convert request to domain model
	listing := &domain.Listing{
		Description: req.Description,
		Price:       int(req.Price),
		Location: domain.Location{
			Type:        "Point",
			Coordinates: []float64{req.Location.Longitude, req.Location.Latitude},
		},
		Address: domain.Address{
			Street:  req.Location.Address,
			City:    req.Location.City,
			State:   req.Location.State,
			Zipcode: req.Location.PostalCode,
			Coordinates: domain.Coordinates{
				Lat: req.Location.Latitude,
				Lng: req.Location.Longitude,
			},
		},
		Images:       req.Images,
		Status:       domain.ListingStatus(req.Status),
		PropertyType: domain.PropertyType(req.PropertyType),
		UpdatedAt:    req.UpdatedAt.Format(time.RFC3339),
		ListingAgent: domain.ListingAgent{
			Name:  req.Contact.Name,
			Email: req.Contact.Email,
			Phone: req.Contact.Phone,
		},
	}

	// Create listing
	if err := h.service.CreateListing(listing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listing"})
		return
	}

	c.JSON(http.StatusCreated, listing)
}

// UpdateListing handles the update of an existing listing
func (h *ListingHandler) UpdateListing(c *gin.Context) {
	// Get the validated path parameters
	pathParams := middleware.GetValidatedPath(c).(*listing.ListingIDRequest)
	id := pathParams.ID

	// Get the validated request body
	req := middleware.GetValidatedModel(c).(*listing.UpdateListingRequest)

	// Get existing listing
	existing, err := h.service.GetListing(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}

	// Update fields if provided
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Price != 0 {
		existing.Price = int(req.Price)
	}
	if req.Location != nil {
		existing.Location = domain.Location{
			Type:        "Point",
			Coordinates: []float64{req.Location.Longitude, req.Location.Latitude},
		}
		existing.Address = domain.Address{
			Street:  req.Location.Address,
			City:    req.Location.City,
			State:   req.Location.State,
			Zipcode: req.Location.PostalCode,
			Coordinates: domain.Coordinates{
				Lat: req.Location.Latitude,
				Lng: req.Location.Longitude,
			},
		}
	}
	if req.Images != nil {
		existing.Images = req.Images
	}
	if req.Contact != nil {
		existing.ListingAgent = domain.ListingAgent{
			Name:  req.Contact.Name,
			Email: req.Contact.Email,
			Phone: req.Contact.Phone,
		}
	}
	if req.Status != "" {
		existing.Status = domain.ListingStatus(req.Status)
	}
	if req.PropertyType != "" {
		existing.PropertyType = domain.PropertyType(req.PropertyType)
	}
	existing.UpdatedAt = req.UpdatedAt.Format(time.RFC3339)

	// Update listing
	if err := h.service.UpdateListing(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update listing"})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// GetListing handles retrieving a single listing
func (h *ListingHandler) GetListing(c *gin.Context) {
	// Get the validated path parameters
	pathParams := middleware.GetValidatedPath(c).(*listing.ListingIDRequest)
	id := pathParams.ID

	listing, err := h.service.GetListing(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
		return
	}
	c.JSON(http.StatusOK, listing)
}

// ListListings handles retrieving multiple listings
func (h *ListingHandler) ListListings(c *gin.Context) {
	listings, err := h.service.ListListings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve listings"})
		return
	}
	c.JSON(http.StatusOK, listings)
}

// DeleteListing handles the deletion of a listing
func (h *ListingHandler) DeleteListing(c *gin.Context) {
	// Get the validated path parameters
	pathParams := middleware.GetValidatedPath(c).(*listing.ListingIDRequest)
	id := pathParams.ID

	if err := h.service.DeleteListing(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete listing"})
		return
	}
	c.Status(http.StatusNoContent)
}
