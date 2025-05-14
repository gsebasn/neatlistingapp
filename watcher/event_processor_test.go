package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"
	"syscall"
	"testing"
	"time"
	externalservices "watcher/external-services"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestTypesenseService is a test-specific implementation of TypesenseClient
type TestTypesenseService struct {
	upsertErr error
	deleteErr error
}

func NewTestTypesenseService() externalservices.TypesenseServiceContract {
	return &TestTypesenseService{}
}

func (t *TestTypesenseService) UpsertDocument(ctx context.Context, document interface{}) error {
	if document == nil || (reflect.ValueOf(document).Kind() == reflect.Map && reflect.ValueOf(document).IsNil()) {
		return errors.New("document is nil")
	}
	return t.upsertErr
}

func (t *TestTypesenseService) DeleteDocument(ctx context.Context, documentID string) error {
	if documentID == "" {
		return errors.New("documentID is empty")
	}
	return t.deleteErr
}

func (t *TestTypesenseService) ImportDocuments(ctx context.Context, documents []interface{}, action string) error {
	return nil
}

// TestRedisService is a test-specific implementation of RedisClient
type TestRedisService struct {
	rpushErr       error
	lpopErr        error
	llenErr        error
	lastRPushValue interface{}
}

func NewTestRedisService() externalservices.RedisServiceContract {
	return &TestRedisService{}
}

func (t *TestRedisService) RPush(ctx context.Context, value interface{}) error {
	t.lastRPushValue = value
	return t.rpushErr
}

func (t *TestRedisService) LPop(ctx context.Context) (string, error) {
	return "", t.lpopErr
}

func (t *TestRedisService) LLen(ctx context.Context) (int64, error) {
	return 0, t.llenErr
}

func (t *TestRedisService) ConfigSet(ctx context.Context, parameter, value string) error {
	return nil
}

func (t *TestRedisService) Ping(ctx context.Context) error {
	return nil
}

// TestMongoService is a test-specific implementation of MongoDBClient
type TestMongoService struct {
	collection *mongo.Collection
	client     *mongo.Client
}

func NewTestMongoService() externalservices.MongoServiceContract {
	// Create a mock client and collection for testing
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		// For testing purposes, we'll just return a service with nil client/collection
		// The test will handle the error appropriately
		return &TestMongoService{}
	}

	collection := client.Database("test").Collection("test")
	return &TestMongoService{
		client:     client,
		collection: collection,
	}
}

func (t *TestMongoService) Connect(ctx context.Context) error {
	if t.client == nil {
		return fmt.Errorf("mock client not initialized")
	}
	return nil
}

func (t *TestMongoService) Disconnect(ctx context.Context) error {
	if t.client != nil {
		return t.client.Disconnect(ctx)
	}
	return nil
}

func (t *TestMongoService) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (externalservices.ChangeStream, error) {
	return nil, nil
}

func (t *TestMongoService) GetCollection() *mongo.Collection {
	return t.collection
}

// TestBackupFlusher is a mock implementation of BackupFlusher for testing
type TestBackupFlusher struct {
	lastFlushedEvents []externalservices.MongoChangeEvent
	flushCalled       bool
}

// Ensure TestBackupFlusher implements BackupFlusherContract
var _ BackupFlusherContract = (*TestBackupFlusher)(nil)

func NewTestBackupFlusher() *TestBackupFlusher {
	return &TestBackupFlusher{}
}

func (b *TestBackupFlusher) Flush(events []externalservices.MongoChangeEvent) {
	b.lastFlushedEvents = make([]externalservices.MongoChangeEvent, len(events))
	copy(b.lastFlushedEvents, events)
	b.flushCalled = true
}

// TestEventProcessor is a test-specific version of EventProcessor that allows method overrides
type TestEventProcessor struct {
	*EventProcessor
	processBatchFunc func()
}

func (t *TestEventProcessor) processBatch() {
	if t.processBatchFunc != nil {
		t.processBatchFunc()
	} else {
		t.EventProcessor.processBatch()
	}
}

func (t *TestEventProcessor) startProcessingTimer() {
	for {
		select {
		case <-t.ticker.C:
			t.mu.Lock()
			// Only process if we have events
			if len(t.activeBuffer) > 0 {
				go t.processBatch() // This will use our overridden method
			}
			t.mu.Unlock()
		case <-t.stopChan:
			t.ticker.Stop()
			return
		case <-t.done:
			t.ticker.Stop()
			return
		}
	}
}

func TestProcessBatchBufferManagement(t *testing.T) {
	// Create initial events
	initialEvents := []externalservices.MongoChangeEvent{
		{OperationType: "insert", DocumentID: "1", Document: map[string]interface{}{"id": "1"}},
		{OperationType: "update", DocumentID: "2", Document: map[string]interface{}{"id": "2"}},
	}

	mockRedis := NewTestRedisService()
	processor := &EventProcessor{
		activeBuffer:           make([]externalservices.MongoChangeEvent, len(initialEvents)),
		remainingBuffer:        make([]externalservices.MongoChangeEvent, 0),
		typesenseService:       NewTestTypesenseService(),
		primaryRedis:           mockRedis,
		backupRedis:            NewTestRedisService(),
		watchingMongoDBService: &TestMongoService{},
		fallbackMongoService:   &TestMongoService{},
		processBusy:            make(chan bool, 1), // Initialize the channel
	}

	// Copy initial events to active buffer
	copy(processor.activeBuffer, initialEvents)

	// Get the process busy channel
	isProcessBusy := processor.GetProcessBusyState()

	// Start processing in a goroutine
	go processor.processBatch()

	// Wait for processing to start
	processing := <-isProcessBusy
	assert.True(t, processing, "Expected processing to start")

	// Wait for processing to finish
	processing = <-isProcessBusy
	assert.False(t, processing, "Expected processing to finish")

	// Verify buffers are empty after processing
	assert.Equal(t, 0, len(processor.activeBuffer), "Expected empty active buffer")
	assert.Equal(t, 0, len(processor.remainingBuffer), "Expected empty remaining buffer")
}

func TestProcessBatchBufferManagementWhenAllActiveEventsFailedToBeInserted(t *testing.T) {
	// Create initial events
	initialEvents := []externalservices.MongoChangeEvent{
		{OperationType: "insert", DocumentID: "1", Document: nil},
		{OperationType: "update", DocumentID: "2", Document: nil},
	}

	processor := &EventProcessor{
		activeBuffer:           make([]externalservices.MongoChangeEvent, len(initialEvents)),
		remainingBuffer:        make([]externalservices.MongoChangeEvent, 0),
		typesenseService:       NewTestTypesenseService(),
		primaryRedis:           NewTestRedisService(),
		backupRedis:            NewTestRedisService(),
		watchingMongoDBService: &TestMongoService{},
		fallbackMongoService:   &TestMongoService{},
		processBusy:            make(chan bool, 1), // Initialize the channel
	}

	// Copy initial events to active buffer
	copy(processor.activeBuffer, initialEvents)

	// Get the process busy channel
	isProcessBusy := processor.GetProcessBusyState()

	// Start processing in a goroutine
	go processor.processBatch()

	// Wait for processing to start
	processing := <-isProcessBusy
	assert.True(t, processing, "Expected processing to start")

	// Wait for processing to finish
	processing = <-isProcessBusy
	assert.False(t, processing, "Expected processing to finish")

	// Verify buffer states
	assert.Equal(t, 0, len(processor.remainingBuffer), "Expected empty remaining buffer")
	assert.Equal(t, len(initialEvents), len(processor.activeBuffer), "Expected all events in active buffer")

	// Verify the order of events in the active buffer
	for i := 1; i < len(processor.activeBuffer); i++ {
		assert.LessOrEqual(t, processor.activeBuffer[i-1].DocumentID, processor.activeBuffer[i].DocumentID,
			"activeBuffer should be sorted by DocumentID in ascending order")
	}
}

func TestEventProcessor_Enqueue(t *testing.T) {
	typesenseService := NewTestTypesenseService()
	primaryRedis := NewTestRedisService()
	backupRedis := NewTestRedisService()
	watchingMongo := &TestMongoService{}
	fallbackMongo := &TestMongoService{}

	processor, err := NewEventProcessor(
		typesenseService,
		primaryRedis,
		backupRedis,
		watchingMongo,
		fallbackMongo,
		1000,
		time.Second*5,
		time.Second*10,
	)
	assert.NoError(t, err)
	assert.NotNil(t, processor)
	defer processor.Stop()

	ctx := context.Background()
	event := externalservices.MongoChangeEvent{
		OperationType: "insert",
		Document:      map[string]interface{}{"test": "value"},
		DocumentID:    "123",
		Timestamp:     time.Now().Unix(),
	}

	err = processor.Enqueue(ctx, event)
	assert.NoError(t, err)
}
func TestEventProcessor_WithTypesenseErrors(t *testing.T) {
	typesenseService := &TestTypesenseService{upsertErr: errors.New("typesense error")}
	primaryRedis := NewTestRedisService()
	backupRedis := NewTestRedisService()
	watchingMongo := &TestMongoService{}
	fallbackMongo := &TestMongoService{}

	processor, err := NewEventProcessor(
		typesenseService,
		primaryRedis,
		backupRedis,
		watchingMongo,
		fallbackMongo,
		1000,
		time.Second*5,
		time.Second*10,
	)
	assert.NoError(t, err)
	assert.NotNil(t, processor)
	defer processor.Stop()

	ctx := context.Background()
	event := externalservices.MongoChangeEvent{
		OperationType: "insert",
		Document:      map[string]interface{}{"test": "value"},
		DocumentID:    "123",
		Timestamp:     time.Now().Unix(),
	}

	err = processor.Enqueue(ctx, event)
	assert.NoError(t, err) // Should not error as it's just queued
}

func TestEventProcessor_FlushAndClose(t *testing.T) {
	// Create initial events
	initialEvents := []externalservices.MongoChangeEvent{
		{OperationType: "insert", DocumentID: "1", Document: map[string]interface{}{"id": "1"}},
		{OperationType: "update", DocumentID: "2", Document: map[string]interface{}{"id": "2"}},
	}

	mockRedis := NewTestRedisService().(*TestRedisService) // Type assert to TestRedisService
	mockBackupFlusher := NewTestBackupFlusher()
	processor := &EventProcessor{
		activeBuffer:           make([]externalservices.MongoChangeEvent, len(initialEvents)),
		remainingBuffer:        make([]externalservices.MongoChangeEvent, 0),
		typesenseService:       NewTestTypesenseService(),
		primaryRedis:           mockRedis,
		backupRedis:            NewTestRedisService(),
		watchingMongoDBService: &TestMongoService{},
		fallbackMongoService:   &TestMongoService{},
		processBusy:            make(chan bool, 1),
		done:                   make(chan struct{}),
		eventChan:              make(chan externalservices.MongoChangeEvent, 100), // Initialize eventChan
		backupFlusher:          mockBackupFlusher,
		maxBufferSize:          100,
		processInterval:        time.Second * 5,
		stopChan:               make(chan struct{}),
		isNearLimit:            make(chan bool, 1),
	}

	// Start the accumulateEvents goroutine
	go processor.accumulateEvents()

	// Copy initial events to active buffer
	copy(processor.activeBuffer, initialEvents)

	// Get the process busy channel
	isProcessBusy := processor.GetProcessBusyState()

	// Start FlushAndClose in a goroutine since it blocks on processBusy
	go processor.FlushAndClose()

	// Wait for processing to start
	processing := <-isProcessBusy
	assert.True(t, processing, "Expected processing to start")

	// Wait for processing to finish
	processing = <-isProcessBusy
	assert.False(t, processing, "Expected processing to finish")

	// Give a small amount of time for the accumulateEvents goroutine to process the done signal
	time.Sleep(10 * time.Millisecond)

	// Verify done channel is closed
	select {
	case <-processor.done:
		// Channel is closed as expected
	default:
		t.Error("Expected done channel to be closed")
	}

	// Verify eventChan is closed
	select {
	case _, ok := <-processor.eventChan:
		if ok {
			t.Error("Expected eventChan to be closed")
		}
	default:
		t.Error("Expected eventChan to be closed")
	}

	// Verify backupFlusher was called with the correct events
	assert.True(t, mockBackupFlusher.flushCalled, "Expected backupFlusher.Flush to be called")
	assert.Equal(t, initialEvents, mockBackupFlusher.lastFlushedEvents, "Expected backupFlusher to be called with initial events")

	// Verify active buffer is cleared after flushing
	assert.Equal(t, 0, len(processor.activeBuffer), "Expected active buffer to be cleared after flushing")
}

func TestEventProcessor_StartProcessingTime_WhenThereAreEvents(t *testing.T) {
	// Create initial events
	initialEvents := []externalservices.MongoChangeEvent{
		{OperationType: "insert", DocumentID: "1", Document: map[string]interface{}{"id": "1"}},
		{OperationType: "update", DocumentID: "2", Document: map[string]interface{}{"id": "2"}},
	}

	// Create a ticker with a very short interval for testing
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	// Channel to track when processBatch is called
	processBatchCalled := make(chan struct{})

	baseProcessor := &EventProcessor{
		activeBuffer:           make([]externalservices.MongoChangeEvent, len(initialEvents)),
		remainingBuffer:        make([]externalservices.MongoChangeEvent, 0),
		typesenseService:       NewTestTypesenseService(),
		primaryRedis:           NewTestRedisService(),
		backupRedis:            NewTestRedisService(),
		watchingMongoDBService: &TestMongoService{},
		fallbackMongoService:   &TestMongoService{},
		processBusy:            make(chan bool, 1),
		done:                   make(chan struct{}),
		eventChan:              make(chan externalservices.MongoChangeEvent, 100),
		backupFlusher:          NewTestBackupFlusher(),
		maxBufferSize:          100,
		processInterval:        time.Second * 5,
		stopChan:               make(chan struct{}),
		ticker:                 ticker,
		isNearLimit:            make(chan bool, 1),
	}

	processor := &TestEventProcessor{
		EventProcessor: baseProcessor,
		processBatchFunc: func() {
			close(processBatchCalled)
			baseProcessor.processBatch()
		},
	}

	// Copy initial events to active buffer
	copy(processor.activeBuffer, initialEvents)

	// Start the processing timer in a goroutine
	go processor.startProcessingTimer()

	// Wait for processBatch to be called
	select {
	case <-processBatchCalled:
		// processBatch was called as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected processBatch to be called within 100ms")
	}

	// Stop the processor
	close(processor.stopChan)
}

func TestEventProcessor_StartProcessingTime_WhenNoEvents(t *testing.T) {
	// No initial events
	initialEvents := []externalservices.MongoChangeEvent{}

	// Create a ticker with a very short interval for testing
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	// Channel to track if processBatch is called
	processBatchCalled := make(chan struct{}, 1)

	baseProcessor := &EventProcessor{
		activeBuffer:           make([]externalservices.MongoChangeEvent, len(initialEvents)),
		remainingBuffer:        make([]externalservices.MongoChangeEvent, 0),
		typesenseService:       NewTestTypesenseService(),
		primaryRedis:           NewTestRedisService(),
		backupRedis:            NewTestRedisService(),
		watchingMongoDBService: &TestMongoService{},
		fallbackMongoService:   &TestMongoService{},
		processBusy:            make(chan bool, 1),
		done:                   make(chan struct{}),
		eventChan:              make(chan externalservices.MongoChangeEvent, 100),
		backupFlusher:          NewTestBackupFlusher(),
		maxBufferSize:          100,
		processInterval:        time.Second * 5,
		stopChan:               make(chan struct{}),
		ticker:                 ticker,
		isNearLimit:            make(chan bool, 1),
	}

	processor := &TestEventProcessor{
		EventProcessor: baseProcessor,
		processBatchFunc: func() {
			processBatchCalled <- struct{}{}
			baseProcessor.processBatch()
		},
	}

	// Copy initial events to active buffer (empty)
	copy(processor.activeBuffer, initialEvents)

	// Start the processing timer in a goroutine
	go processor.startProcessingTimer()

	// Wait for a bit longer than the ticker interval
	select {
	case <-processBatchCalled:
		t.Error("processBatch should NOT be called when there are no events")
	case <-time.After(50 * time.Millisecond):
		// Success: processBatch was not called
	}

	// Stop the processor
	close(processor.stopChan)
}

func TestEventProcessor_ShutdownScenarios(t *testing.T) {
	t.Run("Graceful Shutdown", func(t *testing.T) {
		// Create initial events
		initialEvents := []externalservices.MongoChangeEvent{
			{OperationType: "insert", DocumentID: "1", Document: map[string]interface{}{"id": "1"}},
			{OperationType: "update", DocumentID: "2", Document: map[string]interface{}{"id": "2"}},
		}

		mockBackupFlusher := NewTestBackupFlusher()
		processor := &EventProcessor{
			activeBuffer:           make([]externalservices.MongoChangeEvent, len(initialEvents)),
			remainingBuffer:        make([]externalservices.MongoChangeEvent, 0),
			typesenseService:       NewTestTypesenseService(),
			primaryRedis:           NewTestRedisService(),
			backupRedis:            NewTestRedisService(),
			watchingMongoDBService: &TestMongoService{},
			fallbackMongoService:   &TestMongoService{},
			processBusy:            make(chan bool, 2), // Increased buffer size
			done:                   make(chan struct{}),
			eventChan:              make(chan externalservices.MongoChangeEvent, 100),
			backupFlusher:          mockBackupFlusher,
			maxBufferSize:          100,
			processInterval:        time.Second * 5,
			stopChan:               make(chan struct{}),
			isNearLimit:            make(chan bool, 1),
		}

		// Copy initial events to active buffer
		copy(processor.activeBuffer, initialEvents)

		// Start the accumulateEvents goroutine with a WaitGroup
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			processor.accumulateEvents()
		}()

		// Create a channel to simulate shutdown signal
		sigChan := make(chan os.Signal, 1)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// Channel to signal when shutdown is complete
		shutdownComplete := make(chan struct{})

		// Start a goroutine to simulate signal handling
		go func() {
			select {
			case <-sigChan:
				log.Println("Simulating shutdown signal")
				processor.FlushAndClose()
				close(shutdownComplete)
			case <-ctx.Done():
				return
			}
		}()

		// Simulate receiving a shutdown signal
		sigChan <- syscall.SIGTERM

		// Wait for shutdown to complete with timeout
		select {
		case <-shutdownComplete:
			// Shutdown completed successfully
			// Give a small amount of time for accumulateEvents to exit
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			select {
			case <-done:
				// accumulateEvents exited successfully
			case <-time.After(500 * time.Millisecond):
				t.Fatal("accumulateEvents did not exit in time")
			}
		case <-time.After(time.Second):
			t.Fatal("Shutdown did not complete in time")
		}

		// Verify backupFlusher was called with the correct events
		assert.True(t, mockBackupFlusher.flushCalled, "Expected backupFlusher.Flush to be called during shutdown")
		assert.Equal(t, initialEvents, mockBackupFlusher.lastFlushedEvents, "Expected backupFlusher to be called with initial events")

		// Verify done channel is closed
		select {
		case <-processor.done:
			// Channel is closed as expected
		default:
			t.Error("Expected done channel to be closed during shutdown")
		}

		// Verify eventChan is closed
		select {
		case _, ok := <-processor.eventChan:
			if ok {
				t.Error("Expected eventChan to be closed during shutdown")
			}
		default:
			t.Error("Expected eventChan to be closed during shutdown")
		}

		// Verify active buffer is cleared after flushing
		assert.Equal(t, 0, len(processor.activeBuffer), "Expected active buffer to be cleared after shutdown flush")
	})

	t.Run("Unexpected Shutdown", func(t *testing.T) {
		// Create initial events
		initialEvents := []externalservices.MongoChangeEvent{
			{OperationType: "insert", DocumentID: "1", Document: map[string]interface{}{"id": "1"}},
			{OperationType: "update", DocumentID: "2", Document: map[string]interface{}{"id": "2"}},
		}

		mockBackupFlusher := NewTestBackupFlusher()
		processor := &EventProcessor{
			activeBuffer:           make([]externalservices.MongoChangeEvent, len(initialEvents)),
			remainingBuffer:        make([]externalservices.MongoChangeEvent, 0),
			typesenseService:       NewTestTypesenseService(),
			primaryRedis:           NewTestRedisService(),
			backupRedis:            NewTestRedisService(),
			watchingMongoDBService: &TestMongoService{},
			fallbackMongoService:   &TestMongoService{},
			processBusy:            make(chan bool, 2), // Increased buffer size
			done:                   make(chan struct{}),
			eventChan:              make(chan externalservices.MongoChangeEvent, 100),
			backupFlusher:          mockBackupFlusher,
			maxBufferSize:          100,
			processInterval:        time.Second * 5,
			stopChan:               make(chan struct{}),
			isNearLimit:            make(chan bool, 1),
		}

		// Copy initial events to active buffer
		copy(processor.activeBuffer, initialEvents)

		// Start the accumulateEvents goroutine with a WaitGroup
		var wg2 sync.WaitGroup
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			processor.accumulateEvents()
		}()

		// Channel to signal when shutdown is complete
		shutdownComplete := make(chan struct{})

		// Simulate an unexpected shutdown in a goroutine
		go func() {
			processor.FlushAndClose()
			close(shutdownComplete)
		}()

		// Wait for shutdown to complete with timeout
		select {
		case <-shutdownComplete:
			// Shutdown completed successfully
			// Give a small amount of time for accumulateEvents to exit
			done := make(chan struct{})
			go func() {
				wg2.Wait()
				close(done)
			}()
			select {
			case <-done:
				// accumulateEvents exited successfully
			case <-time.After(500 * time.Millisecond):
				t.Fatal("accumulateEvents did not exit in time")
			}
		case <-time.After(time.Second):
			t.Fatal("Shutdown did not complete in time")
		}

		// Verify backupFlusher was called with the correct events
		assert.True(t, mockBackupFlusher.flushCalled, "Expected backupFlusher.Flush to be called during unexpected shutdown")
		assert.Equal(t, initialEvents, mockBackupFlusher.lastFlushedEvents, "Expected backupFlusher to be called with initial events")

		// Verify done channel is closed
		select {
		case <-processor.done:
			// Channel is closed as expected
		default:
			t.Error("Expected done channel to be closed during unexpected shutdown")
		}

		// Verify eventChan is closed
		select {
		case _, ok := <-processor.eventChan:
			if ok {
				t.Error("Expected eventChan to be closed during unexpected shutdown")
			}
		default:
			t.Error("Expected eventChan to be closed during unexpected shutdown")
		}

		// Verify active buffer is cleared after flushing
		assert.Equal(t, 0, len(processor.activeBuffer), "Expected active buffer to be cleared after unexpected shutdown flush")
	})
}
