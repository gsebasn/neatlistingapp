package main

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"watcher/interfaces"
)

// MockTypesenseClient implements interfaces.TypesenseClient for testing
type MockTypesenseClient struct {
	upsertErr error
	deleteErr error
}

func (m *MockTypesenseClient) UpsertDocument(ctx context.Context, collection string, document interface{}) error {
	if document == nil || (reflect.ValueOf(document).Kind() == reflect.Map && reflect.ValueOf(document).IsNil()) {
		return errors.New("document is nil")
	}
	return nil
}

func (m *MockTypesenseClient) DeleteDocument(ctx context.Context, collection string, documentID string) error {
	if documentID == "" {
		return errors.New("documentID is empty")
	}
	return nil
}

func (m *MockTypesenseClient) ImportDocuments(ctx context.Context, collection string, documents []interface{}, action string) error {
	return nil
}

// MockRedisClient implements interfaces.RedisClient for testing
type MockRedisClient struct {
	rpushErr error
	lpopErr  error
	llenErr  error
}

func (m *MockRedisClient) RPush(ctx context.Context, key string, value interface{}) error {
	return m.rpushErr
}

func (m *MockRedisClient) LPop(ctx context.Context, key string) (string, error) {
	return "", m.lpopErr
}

func (m *MockRedisClient) LLen(ctx context.Context, key string) (int64, error) {
	return 0, m.llenErr
}

func (m *MockRedisClient) ConfigSet(ctx context.Context, parameter, value string) error {
	return nil
}

func (m *MockRedisClient) Ping(ctx context.Context) error {
	return nil
}
func TestProcessBatchBufferManagement(t *testing.T) {
	config := &Config{
		Typesense: TypesenseConfig{
			CollectionName: "test",
			MaxRetries:     1,
			RetryBackoff:   time.Millisecond * 100,
		},
	}

	// Create initial events
	initialEvents := []interfaces.Event{
		{OperationType: "insert", DocumentID: "1", Document: map[string]interface{}{"id": "1"}},
		{OperationType: "update", DocumentID: "2", Document: map[string]interface{}{"id": "2"}},
	}

	// Create events that will be added to accumulating buffer during processing
	accumulatingEvents := []interfaces.Event{
		{OperationType: "insert", DocumentID: "3", Document: map[string]interface{}{"id": "3"}},
		{OperationType: "update", DocumentID: "4", Document: map[string]interface{}{"id": "4"}},
	}

	processor := &EventProcessorImpl{
		activeBuffer:       make([]interfaces.Event, len(initialEvents)),
		accumulatingBuffer: make([]interfaces.Event, 0),
		remainingBuffer:    make([]interfaces.Event, 0),
		config:             config,
		typesenseClient:    &MockTypesenseClient{},
		redisClient:        &MockRedisClient{},
		isProcessing:       true,
	}

	// Copy initial events to active buffer
	copy(processor.activeBuffer, initialEvents)

	// Create a channel to signal when processing has started
	processingStarted := make(chan struct{})
	// Create a channel to signal when we've added accumulating events
	accumulatingAdded := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)

	// Start processing in a goroutine
	go func() {
		defer wg.Done()

		// Signal that processing is about to start
		close(processingStarted)

		// Wait for accumulating events to be added
		<-accumulatingAdded

		processor.processBatch()
	}()

	// Wait for processing to start
	<-processingStarted

	// Add events to accumulating buffer
	processor.mu.Lock()
	processor.accumulatingBuffer = append(processor.accumulatingBuffer, accumulatingEvents...)
	processor.mu.Unlock()

	// Signal that accumulating events have been added
	close(accumulatingAdded)

	// Wait for processing to complete
	wg.Wait()

	// Verify the final state of buffers
	if len(processor.accumulatingBuffer) != 0 {
		t.Errorf("expected empty accumulating buffer, got %d events", len(processor.accumulatingBuffer))
	}

	if len(processor.remainingBuffer) != 0 {
		t.Errorf("expected empty remaining buffer, got %d events", len(processor.remainingBuffer))
	}

	// Verify that active buffer contains all events in correct order
	expectedTotalEvents := len(accumulatingEvents)
	if len(processor.activeBuffer) != expectedTotalEvents {
		t.Errorf("expected %d events in active buffer, got %d", expectedTotalEvents, len(processor.activeBuffer))
	}

	// Verify isProcessing flag is reset
	if processor.isProcessing {
		t.Error("expected isProcessing to be false after processing")
	}

	// Verify the order of events in the active buffer and verify that accumulating events has been added to the active buffer.
	// They are the new active events.
	for i, event := range processor.activeBuffer {
		if event.DocumentID != accumulatingEvents[i].DocumentID {
			t.Errorf("expected active event %s at position %d, got %s",
				accumulatingEvents[i].DocumentID, i+1, event.DocumentID)
		}
	}
}

func TestProcessBatchBufferManagementAllActiveEventsFailedToBeInserted(t *testing.T) {
	config := &Config{
		Typesense: TypesenseConfig{
			CollectionName: "test",
			MaxRetries:     1,
			RetryBackoff:   time.Millisecond * 100,
		},
	}

	// Create initial events
	initialEvents := []interfaces.Event{
		{OperationType: "insert", DocumentID: "1", Document: nil},
		{OperationType: "update", DocumentID: "2", Document: nil},
	}

	// Create events that will be added to accumulating buffer during processing
	accumulatingEvents := []interfaces.Event{
		{OperationType: "insert", DocumentID: "3", Document: map[string]interface{}{"id": "3"}},
		{OperationType: "update", DocumentID: "4", Document: map[string]interface{}{"id": "4"}},
	}

	processor := &EventProcessorImpl{
		activeBuffer:       make([]interfaces.Event, len(initialEvents)),
		accumulatingBuffer: make([]interfaces.Event, 0),
		remainingBuffer:    make([]interfaces.Event, 0),
		config:             config,
		typesenseClient:    &MockTypesenseClient{},
		redisClient:        &MockRedisClient{},
		isProcessing:       true,
	}

	// Copy initial events to active buffer
	copy(processor.activeBuffer, initialEvents)

	// Create a channel to signal when processing has started
	processingStarted := make(chan struct{})
	// Create a channel to signal when we've added accumulating events
	accumulatingAdded := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)

	// Start processing in a goroutine
	go func() {
		defer wg.Done()

		// Signal that processing is about to start
		close(processingStarted)

		// Wait for accumulating events to be added
		<-accumulatingAdded

		processor.processBatch()
	}()

	// Wait for processing to start
	<-processingStarted

	// Add events to accumulating buffer
	processor.mu.Lock()
	processor.accumulatingBuffer = append(processor.accumulatingBuffer, accumulatingEvents...)
	processor.mu.Unlock()

	// Signal that accumulating events have been added
	close(accumulatingAdded)

	// Wait for processing to complete
	wg.Wait()

	// Verify the final state of buffers
	if len(processor.accumulatingBuffer) != 0 {
		t.Errorf("expected empty accumulating buffer, got %d events", len(processor.accumulatingBuffer))
	}

	if len(processor.remainingBuffer) != 0 {
		t.Errorf("expected empty remaining buffer, got %d events", len(processor.remainingBuffer))
	}

	// Verify that active buffer contains all events in correct order
	expectedTotalEvents := len(initialEvents) + len(accumulatingEvents)
	if len(processor.activeBuffer) != expectedTotalEvents {
		t.Errorf("expected %d events in active buffer, got %d", expectedTotalEvents, len(processor.activeBuffer))
	}

	// Verify isProcessing flag is reset
	if processor.isProcessing {
		t.Error("expected isProcessing to be false after processing")
	}

	// Verify the order of events in the active buffer
	for i := 1; i < len(processor.activeBuffer); i++ {
		if processor.activeBuffer[i-1].DocumentID > processor.activeBuffer[i].DocumentID {
			t.Errorf("activeBuffer is not sorted by DocumentID in ascending order at position %d", i)
		}
	}
}
