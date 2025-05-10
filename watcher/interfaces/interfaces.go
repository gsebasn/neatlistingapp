package interfaces

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"
)

// Event represents a MongoDB change event
type Event struct {
	OperationType string                 `json:"operationType"`
	Document      map[string]interface{} `json:"document"`
	DocumentID    string                 `json:"documentId"`
	Timestamp     time.Time              `json:"timestamp"`
}

// RedisClient defines the interface for Redis operations
type RedisClient interface {
	RPush(ctx context.Context, key string, value interface{}) error
	LPop(ctx context.Context, key string) (string, error)
	LLen(ctx context.Context, key string) (int64, error)
	ConfigSet(ctx context.Context, parameter, value string) error
	Ping(ctx context.Context) error
}

// TypesenseClient defines the interface for Typesense operations
type TypesenseClient interface {
	UpsertDocument(ctx context.Context, collection string, document interface{}) error
	DeleteDocument(ctx context.Context, collection string, documentID string) error
	ImportDocuments(ctx context.Context, collection string, documents []interface{}, action string) error
}

// MongoDBClient defines the interface for MongoDB operations
type MongoDBClient interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (ChangeStream, error)
}

// ChangeStream represents a MongoDB change stream
type ChangeStream interface {
	Next(ctx context.Context) bool
	Decode(val interface{}) error
	Close(ctx context.Context) error
	Err() error
}

// MongoWatcher defines the interface for MongoDB change stream operations
type MongoWatcher interface {
	Watch(ctx context.Context, processor EventProcessor) error
	Stop() error
}

// EventProcessor defines the interface for processing events
type EventProcessor interface {
	Enqueue(ctx context.Context, event Event) error
}
