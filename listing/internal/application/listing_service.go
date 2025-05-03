package application

import (
	"errors"
	"time"

	"listing/internal/domain"
)

// ListingService handles the business logic for listings
type ListingService struct {
	repo domain.ListingRepository
}

// NewListingService creates a new instance of ListingService
func NewListingService(repo domain.ListingRepository) *ListingService {
	return &ListingService{
		repo: repo,
	}
}

// GetListing retrieves a listing by ID
func (s *ListingService) GetListing(id string) (*domain.Listing, error) {
	if id == "" {
		return nil, errors.New("listing ID is required")
	}
	return s.repo.GetByID(id)
}

// CreateListing creates a new listing
func (s *ListingService) CreateListing(listing *domain.Listing) error {
	if listing.Title == "" {
		return errors.New("listing title is required")
	}
	if listing.Price < 0 {
		return errors.New("listing price must be positive")
	}

	now := time.Now()
	listing.CreatedAt = now
	listing.UpdatedAt = now

	return s.repo.Create(listing)
}

// UpdateListing updates an existing listing
func (s *ListingService) UpdateListing(listing *domain.Listing) error {
	if listing.ID == "" {
		return errors.New("listing ID is required")
	}
	if listing.Title == "" {
		return errors.New("listing title is required")
	}
	if listing.Price < 0 {
		return errors.New("listing price must be positive")
	}

	listing.UpdatedAt = time.Now()
	return s.repo.Update(listing)
}

// DeleteListing deletes a listing by ID
func (s *ListingService) DeleteListing(id string) error {
	if id == "" {
		return errors.New("listing ID is required")
	}
	return s.repo.Delete(id)
}

// ListListings retrieves all listings
func (s *ListingService) ListListings() ([]*domain.Listing, error) {
	return s.repo.List()
}
