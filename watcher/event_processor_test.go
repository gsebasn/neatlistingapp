package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"watcher/interfaces"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTestEnv() {
	os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	os.Setenv("MONGODB_DATABASE", "testdb")
	os.Setenv("MONGODB_COLLECTION", "testcoll")
	os.Setenv("MAX_BUFFER_SIZE", "1000")
	os.Setenv("BATCH_SIZE", "100")
	os.Setenv("FLUSH_INTERVAL", "5")
	os.Setenv("PRIMARY_REDIS_QUEUE_KEY", "primary-queue")
	os.Setenv("BACKUP_REDIS_QUEUE_KEY", "backup-queue")
}

func teardownTestEnv() {
	os.Unsetenv("MONGODB_URI")
	os.Unsetenv("MONGODB_DATABASE")
	os.Unsetenv("MONGODB_COLLECTION")
	os.Unsetenv("MAX_BUFFER_SIZE")
	os.Unsetenv("BATCH_SIZE")
	os.Unsetenv("FLUSH_INTERVAL")
	os.Unsetenv("PRIMARY_REDIS_QUEUE_KEY")
	os.Unsetenv("BACKUP_REDIS_QUEUE_KEY")
}

// MockRedisClient implements interfaces.RedisClient for testing
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) RPush(ctx context.Context, key string, value interface{}) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockRedisClient) LPop(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) LLen(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisClient) ConfigSet(ctx context.Context, parameter, value string) error {
	args := m.Called(ctx, parameter, value)
	return args.Error(0)
}

func (m *MockRedisClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockTypesenseClient implements interfaces.TypesenseClient for testing
type MockTypesenseClient struct {
	mock.Mock
}

func (m *MockTypesenseClient) UpsertDocument(ctx context.Context, collection string, document interface{}) error {
	args := m.Called(ctx, collection, document)
	return args.Error(0)
}

func (m *MockTypesenseClient) DeleteDocument(ctx context.Context, collection string, documentID string) error {
	args := m.Called(ctx, collection, documentID)
	return args.Error(0)
}

func TestNewEventProcessor(t *testing.T) {
	setupTestEnv()
	defer teardownTestEnv()

	config := &Config{
		MongoDatabase:   "testdb",
		MongoCollection: "testcoll",
		MaxBufferSize:   100,
		BatchSize:       10,
		FlushInterval:   5,
		PrimaryRedis: RedisConfig{
			QueueKey: "primary-queue",
		},
		BackupRedis: RedisConfig{
			QueueKey: "backup-queue",
		},
	}

	mockRedis := new(MockRedisClient)
	mockTypesense := new(MockTypesenseClient)

	processor, err := NewEventProcessor(config, mockRedis, mockTypesense)
	assert.NoError(t, err)
	assert.NotNil(t, processor)
	assert.Equal(t, config, processor.config)
	assert.Equal(t, mockRedis, processor.redisClient)
	assert.Equal(t, mockTypesense, processor.typesenseClient)
	assert.NotNil(t, processor.fallbackStore)
	assert.Equal(t, "testdb", processor.fallbackStore.database)
	assert.Equal(t, "testcoll", processor.fallbackStore.collection)
}

func TestEventProcessor_Add(t *testing.T) {
	setupTestEnv()
	defer teardownTestEnv()

	config := &Config{
		MaxBufferSize: 2,
		BatchSize:     2,
		FlushInterval: 5,
		PrimaryRedis: RedisConfig{
			QueueKey: "primary-queue",
		},
	}

	mockRedis := new(MockRedisClient)
	mockTypesense := new(MockTypesenseClient)

	processor, _ := NewEventProcessor(config, mockRedis, mockTypesense)

	// Test single event
	event1 := interfaces.Event{
		OperationType: "insert",
		Document:      map[string]interface{}{"id": "1"},
		DocumentID:    "1",
		Timestamp:     time.Now(),
	}

	processor.Add(event1)
	assert.Equal(t, 1, len(processor.events))

	// Test auto-flush when buffer is full
	event2 := interfaces.Event{
		OperationType: "insert",
		Document:      map[string]interface{}{"id": "2"},
		DocumentID:    "2",
		Timestamp:     time.Now(),
	}

	eventsJSON, _ := json.Marshal([]interfaces.Event{event1, event2})
	mockRedis.On("RPush", mock.Anything, "primary-queue", eventsJSON).Return(nil)

	processor.Add(event2)
	assert.Equal(t, 0, len(processor.events)) // Buffer should be empty after flush
	mockRedis.AssertExpectations(t)
}

func TestEventProcessor_GetBatch(t *testing.T) {
	setupTestEnv()
	defer teardownTestEnv()

	config := &Config{
		MaxBufferSize: 10,
		BatchSize:     5,
		FlushInterval: 5,
	}

	mockRedis := new(MockRedisClient)
	mockTypesense := new(MockTypesenseClient)

	processor, _ := NewEventProcessor(config, mockRedis, mockTypesense)

	// Add some events
	events := []interfaces.Event{
		{
			OperationType: "insert",
			Document:      map[string]interface{}{"id": "1"},
			DocumentID:    "1",
			Timestamp:     time.Now(),
		},
		{
			OperationType: "update",
			Document:      map[string]interface{}{"id": "2"},
			DocumentID:    "2",
			Timestamp:     time.Now(),
		},
	}

	for _, event := range events {
		processor.Add(event)
	}

	// Get batch
	batch := processor.GetBatch()
	assert.Equal(t, len(events), len(batch))
	assert.Equal(t, 0, len(processor.events)) // Buffer should be empty
	assert.Equal(t, events[0].DocumentID, batch[0].DocumentID)
	assert.Equal(t, events[1].DocumentID, batch[1].DocumentID)
}

func TestEventProcessor_ShouldFlush(t *testing.T) {
	setupTestEnv()
	defer teardownTestEnv()

	config := &Config{
		MaxBufferSize: 10,
		BatchSize:     2,
		FlushInterval: 1, // 1 second
	}

	mockRedis := new(MockRedisClient)
	mockTypesense := new(MockTypesenseClient)

	processor, _ := NewEventProcessor(config, mockRedis, mockTypesense)

	// Test batch size flush
	processor.Add(interfaces.Event{DocumentID: "1"})
	assert.False(t, processor.ShouldFlush())

	processor.Add(interfaces.Event{DocumentID: "2"})
	assert.True(t, processor.ShouldFlush())

	// Test time-based flush
	processor.GetBatch() // Clear the buffer
	processor.Add(interfaces.Event{DocumentID: "3"})
	time.Sleep(2 * time.Second)
	assert.True(t, processor.ShouldFlush())
}

func TestEventProcessor_RecoverFromRedis(t *testing.T) {
	setupTestEnv()
	defer teardownTestEnv()

	config := &Config{
		PrimaryRedis: RedisConfig{
			QueueKey: "primary-queue",
		},
		BackupRedis: RedisConfig{
			QueueKey: "backup-queue",
		},
	}

	mockRedis := new(MockRedisClient)
	mockTypesense := new(MockTypesenseClient)

	processor, _ := NewEventProcessor(config, mockRedis, mockTypesense)

	// Test recovery from primary Redis
	events := []interfaces.Event{
		{
			OperationType: "insert",
			Document:      map[string]interface{}{"id": "1"},
			DocumentID:    "1",
			Timestamp:     time.Now(),
		},
	}

	eventsJSON, _ := json.Marshal(events)
	mockRedis.On("LPop", mock.Anything, "primary-queue").Return(string(eventsJSON), nil)

	err := processor.RecoverFromRedis()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(processor.events))
	mockRedis.AssertExpectations(t)

	// Test fallback to backup Redis
	processor.GetBatch() // Clear the buffer
	mockRedis.On("LPop", mock.Anything, "primary-queue").Return("", assert.AnError)
	mockRedis.On("LPop", mock.Anything, "backup-queue").Return(string(eventsJSON), nil)

	err = processor.RecoverFromRedis()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(processor.events))
	mockRedis.AssertExpectations(t)
}

func TestEventProcessor_ProcessBatch(t *testing.T) {
	setupTestEnv()
	defer teardownTestEnv()

	config := &Config{
		MaxBufferSize: 10,
		BatchSize:     5,
		FlushInterval: 5,
		PrimaryRedis: RedisConfig{
			QueueKey: "primary-queue",
		},
	}

	mockRedis := new(MockRedisClient)
	mockTypesense := new(MockTypesenseClient)

	processor, _ := NewEventProcessor(config, mockRedis, mockTypesense)

	events := []interfaces.Event{
		{
			OperationType: "insert",
			Document:      map[string]interface{}{"id": "1"},
			DocumentID:    "1",
			Timestamp:     time.Now(),
		},
		{
			OperationType: "update",
			Document:      map[string]interface{}{"id": "2"},
			DocumentID:    "2",
			Timestamp:     time.Now(),
		},
	}

	eventsJSON, _ := json.Marshal(events)
	mockRedis.On("RPush", mock.Anything, "primary-queue", eventsJSON).Return(nil)

	err := processor.ProcessBatch(context.Background(), events)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(processor.events))
	mockRedis.AssertExpectations(t)
}
