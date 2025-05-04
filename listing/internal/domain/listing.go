package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PropertyType represents the type of property
type PropertyType string

const (
	House      PropertyType = "house"
	Apartment  PropertyType = "apartment"
	Condo      PropertyType = "condo"
	Townhouse  PropertyType = "townhouse"
	Land       PropertyType = "land"
	Commercial PropertyType = "commercial"
)

// ListingStatus represents the status of a listing
type ListingStatus string

const (
	Active  ListingStatus = "active"
	Pending ListingStatus = "pending"
	Sold    ListingStatus = "sold"
)

// Coordinates represents geographical coordinates
type Coordinates struct {
	Lat float64 `bson:"lat" json:"lat"`
	Lng float64 `bson:"lng" json:"lng"`
}

// Address represents a property address
type Address struct {
	Number           string      `bson:"number" json:"number"`
	Street           string      `bson:"street" json:"street"`
	NormalizedStreet string      `bson:"normalizedStreet" json:"normalizedStreet"`
	AltStreetNames   []string    `bson:"altStreetNames" json:"altStreetNames"`
	Apartment        string      `bson:"apartment,omitempty" json:"apartment,omitempty"`
	Floor            int         `bson:"floor,omitempty" json:"floor,omitempty"`
	Unit             string      `bson:"unit,omitempty" json:"unit,omitempty"`
	UnitLoc          string      `bson:"unit_loc,omitempty" json:"unit_loc,omitempty"`
	City             string      `bson:"city" json:"city"`
	State            string      `bson:"state" json:"state"`
	Zipcode          string      `bson:"zipcode" json:"zipcode"`
	Coordinates      Coordinates `bson:"coordinates" json:"coordinates"`
	FullStreet       string      `bson:"fullStreet" json:"fullStreet"`
}

// Location represents a GeoJSON Point
type Location struct {
	Type        string    `bson:"type" json:"type"`
	Coordinates []float64 `bson:"coordinates" json:"coordinates"`
}

// ClimateRisk represents climate-related risk factors
type ClimateRisk struct {
	Flood int `bson:"flood" json:"flood"`
	Fire  int `bson:"fire" json:"fire"`
	Heat  int `bson:"heat" json:"heat"`
}

// ListingAgent represents a real estate agent
type ListingAgent struct {
	Name           string                 `bson:"name" json:"name"`
	Email          string                 `bson:"email" json:"email"`
	Phone          string                 `bson:"phone" json:"phone"`
	CustomMetadata map[string]interface{} `bson:"customMetadata" json:"customMetadata"`
}

// ExtraFeatures represents additional property features
type ExtraFeatures struct {
	SolarPanels   bool    `bson:"solarPanels" json:"solarPanels"`
	LastRenovated int     `bson:"lastRenovated" json:"lastRenovated"`
	ViewScore     float64 `bson:"viewScore" json:"viewScore"`
}

// Listing represents a real estate listing.
// @Description A real estate listing object.
type Listing struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id" example:"60c72b2f9b1e8b6f1f8e4b1a"`
	Address        Address            `bson:"address" json:"address"`
	Location       Location           `bson:"location" json:"location"`
	Price          int                `bson:"price" json:"price" example:"1200000"`
	Bedrooms       int                `bson:"bedrooms" json:"bedrooms" example:"3"`
	Bathrooms      int                `bson:"bathrooms" json:"bathrooms" example:"2"`
	Sqft           int                `bson:"sqft" json:"sqft" example:"1500"`
	YearBuilt      int                `bson:"yearBuilt" json:"yearBuilt" example:"1990"`
	PropertyType   PropertyType       `bson:"propertyType" json:"propertyType" example:"house"`
	Status         ListingStatus      `bson:"status" json:"status" example:"active"`
	ListingUrl     string             `bson:"listingUrl" json:"listingUrl" example:"https://example.com/listings/123"`
	Images         []string           `bson:"images" json:"images" example:"[\"https://example.com/image1.jpg\"]"`
	Description    string             `bson:"description" json:"description" example:"A beautiful family home."`
	WalkScore      int                `bson:"walkScore" json:"walkScore" example:"80"`
	SchoolRating   int                `bson:"schoolRating" json:"schoolRating" example:"9"`
	CrimeScore     float64            `bson:"crimeScore" json:"crimeScore" example:"2.5"`
	ClimateRisk    ClimateRisk        `bson:"climateRisk" json:"climateRisk"`
	GreenScore     int                `bson:"greenScore" json:"greenScore" example:"7"`
	Tags           []string           `bson:"tags" json:"tags" example:"[\"garden\",\"garage\"]"`
	HoaFees        int                `bson:"hoaFees" json:"hoaFees" example:"200"`
	TaxRate        float64            `bson:"taxRate" json:"taxRate" example:"1.2"`
	EstMonthlyCost int                `bson:"estMonthlyCost" json:"estMonthlyCost" example:"2500"`
	ListingAgent   ListingAgent       `bson:"listingAgent" json:"listingAgent"`
	MetaSummary    string             `bson:"metaSummary" json:"metaSummary" example:"Spacious, well-lit, close to schools."`
	ExtraFeatures  ExtraFeatures      `bson:"extraFeatures" json:"extraFeatures"`
	UpdatedAt      string             `bson:"updatedAt" json:"updatedAt" example:"2023-01-02T00:00:00Z"`
}

// ListingRepository defines the interface for listing data operations
type ListingRepository interface {
	GetByID(id string) (*Listing, error)
	Create(listing *Listing) error
	Update(listing *Listing) error
	Delete(id string) error
	List() ([]*Listing, error)
}
