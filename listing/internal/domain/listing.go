package domain

import "time"

// Listing represents the core domain entity for a listing
type Listing struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListingRepository defines the interface for listing data operations
type ListingRepository interface {
	GetByID(id string) (*Listing, error)
	Create(listing *Listing) error
	Update(listing *Listing) error
	Delete(id string) error
	List() ([]*Listing, error)
}
