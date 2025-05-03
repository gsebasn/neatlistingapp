package repository

import (
	"errors"
	"sync"

	"listing/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MemoryRepository is an in-memory implementation of the repository interface
type MemoryRepository struct {
	mu       sync.RWMutex
	listings map[primitive.ObjectID]*domain.Listing
}

// NewMemoryRepository creates a new instance of MemoryRepository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		listings: make(map[primitive.ObjectID]*domain.Listing),
	}
}

// GetByID retrieves a listing by ID
func (r *MemoryRepository) GetByID(id string) (*domain.Listing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid ID format")
	}

	listing, exists := r.listings[objectID]
	if !exists {
		return nil, errors.New("listing not found")
	}
	return listing, nil
}

// Create stores a new listing
func (r *MemoryRepository) Create(listing *domain.Listing) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.listings[listing.ID]; exists {
		return errors.New("listing already exists")
	}

	r.listings[listing.ID] = listing
	return nil
}

// Update modifies an existing listing
func (r *MemoryRepository) Update(listing *domain.Listing) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.listings[listing.ID]; !exists {
		return errors.New("listing not found")
	}

	r.listings[listing.ID] = listing
	return nil
}

// Delete removes a listing by ID
func (r *MemoryRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid ID format")
	}

	if _, exists := r.listings[objectID]; !exists {
		return errors.New("listing not found")
	}

	delete(r.listings, objectID)
	return nil
}

// List returns all listings
func (r *MemoryRepository) List() ([]*domain.Listing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	listings := make([]*domain.Listing, 0, len(r.listings))
	for _, listing := range r.listings {
		listings = append(listings, listing)
	}
	return listings, nil
}
