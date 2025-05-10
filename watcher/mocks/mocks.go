package mocks

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// MockRedisClient implements the RedisClient interface
type MockRedisClient struct {
	ShouldError bool
}

func (m *MockRedisClient) RPush(ctx context.Context, key string, value interface{}) error {
	if m.ShouldError {
		return redis.ErrClosed
	}
	return nil
}

func (m *MockRedisClient) LPop(ctx context.Context, key string) (string, error) {
	if m.ShouldError {
		return "", redis.ErrClosed
	}
	return "test_value", nil
}

func (m *MockRedisClient) LLen(ctx context.Context, key string) (int64, error) {
	if m.ShouldError {
		return 0, redis.ErrClosed
	}
	return 1, nil
}

func (m *MockRedisClient) ConfigSet(ctx context.Context, parameter, value string) error {
	if m.ShouldError {
		return redis.ErrClosed
	}
	return nil
}

func (m *MockRedisClient) Ping(ctx context.Context) error {
	if m.ShouldError {
		return redis.ErrClosed
	}
	return nil
}

// MockTypesenseClient implements the TypesenseClient interface
type MockTypesenseClient struct {
	ShouldError bool
}

func (m *MockTypesenseClient) UpsertDocument(ctx context.Context, collection string, document map[string]interface{}) error {
	if m.ShouldError {
		return redis.ErrClosed
	}
	return nil
}

func (m *MockTypesenseClient) DeleteDocument(ctx context.Context, collection string, documentID string) error {
	if m.ShouldError {
		return redis.ErrClosed
	}
	return nil
}

// MockEventProcessor implements the EventProcessor interface
type MockEventProcessor struct {
	ShouldError bool
}

func (m *MockEventProcessor) ProcessBatch(ctx context.Context, events []Event) error {
	if m.ShouldError {
		return redis.ErrClosed
	}
	return nil
}

// MockMongoWatcher implements the MongoWatcher interface
type MockMongoWatcher struct {
	ShouldError bool
}

func (m *MockMongoWatcher) Watch(ctx context.Context) error {
	if m.ShouldError {
		return redis.ErrClosed
	}
	return nil
}

func (m *MockMongoWatcher) Stop() error {
	if m.ShouldError {
		return redis.ErrClosed
	}
	return nil
}

// Event represents a MongoDB change event
type Event struct {
	OperationType string                 `json:"operation_type"`
	Document      map[string]interface{} `json:"document,omitempty"`
	DocumentID    string                 `json:"document_id"`
	Timestamp     time.Time              `json:"timestamp"`
}
