package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"watcher/interfaces"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type EventProcessorImpl struct {
	mu                 sync.Mutex
	activeBuffer       []interfaces.Event // Buffer currently being processed
	accumulatingBuffer []interfaces.Event // Buffer receiving new events during processing
	remainingBuffer    []interfaces.Event // Buffer of remaining events after a failed operation
	lastFlush          time.Time
	config             *Config
	redisClient        interfaces.RedisClient
	typesenseClient    interfaces.TypesenseClient
	fallbackStore      *FallbackStoreImpl
	health             QueueHealth
	ticker             *time.Ticker
	done               chan struct{}
	eventChan          chan interfaces.Event
	isProcessing       bool
}

func NewEventProcessor(config *Config, redisClient interfaces.RedisClient, typesenseClient interfaces.TypesenseClient) (*EventProcessorImpl, error) {
	fallbackStore := &FallbackStoreImpl{
		client:     nil, // Will be initialized when needed
		database:   config.MongoDatabase,
		collection: config.MongoCollection,
	}

	processor := &EventProcessorImpl{
		activeBuffer:       make([]interfaces.Event, 0),
		accumulatingBuffer: make([]interfaces.Event, 0),
		remainingBuffer:    make([]interfaces.Event, 0),
		lastFlush:          time.Now(),
		config:             config,
		redisClient:        redisClient,
		typesenseClient:    typesenseClient,
		fallbackStore:      fallbackStore,
		health:             QueueHealth{},
		done:               make(chan struct{}),
		eventChan:          make(chan interfaces.Event, config.MaxBufferSize),
		isProcessing:       false,
	}

	// Start the ticker for periodic updates
	processor.ticker = time.NewTicker(time.Duration(config.FlushInterval) * time.Second)
	go processor.processEvents()
	go processor.accumulateEvents()

	return processor, nil
}

func (w *EventProcessorImpl) accumulateEvents() {
	for {
		select {
		case event := <-w.eventChan:
			w.mu.Lock()
			if w.isProcessing {
				// If processing is active, add to accumulating buffer
				w.accumulatingBuffer = append(w.accumulatingBuffer, event)
			} else {
				// If not processing, add to active buffer
				w.activeBuffer = append(w.activeBuffer, event)
				if len(w.activeBuffer) >= w.config.MaxBufferSize {
					// Start processing if buffer is full
					w.isProcessing = true
					go w.processBatch()
				}
			}
			w.mu.Unlock()
		case <-w.done:
			return
		}
	}
}

func (w *EventProcessorImpl) processEvents() {
	for {
		select {
		case <-w.ticker.C:
			w.mu.Lock()
			if len(w.activeBuffer) > 0 && !w.isProcessing {
				w.isProcessing = true
				go w.processBatch()
			}
			w.mu.Unlock()
		case <-w.done:
			w.ticker.Stop()
			return
		}
	}
}

func (w *EventProcessorImpl) setRemainingBuffer(batch []interfaces.Event, i int) {
	w.remainingBuffer = make([]interfaces.Event, len(batch[i:]))
	w.remainingBuffer = batch[i:]
}

func (w *EventProcessorImpl) processBatch() {
	defer func() {
		w.mu.Lock()
		// After processing, move accumulating buffer to active buffer
		w.activeBuffer = append(w.remainingBuffer, w.accumulatingBuffer...)
		w.accumulatingBuffer = make([]interfaces.Event, 0)
		w.isProcessing = false
		w.mu.Unlock()
	}()

	w.mu.Lock()
	// Create a copy of the current buffer for processing
	batch := make([]interfaces.Event, len(w.activeBuffer))
	copy(batch, w.activeBuffer)
	w.activeBuffer = w.activeBuffer[:0] // Clear the active buffer
	w.mu.Unlock()

	// Process events in order
	for i, event := range batch {
		var err error
		switch event.OperationType {
		case "insert", "update":
			err = retryOperation(context.Background(), func() error {
				return w.typesenseClient.UpsertDocument(context.Background(), w.config.Typesense.CollectionName, event.Document)
			}, w.config.Typesense.MaxRetries, w.config.Typesense.RetryBackoff)

			if err != nil {
				log.Printf("Failed to upsert document in Typesense after %d retries (ID: %s): %v",
					w.config.Typesense.MaxRetries, event.DocumentID, err)

				// Get all remaining events including the failed one
				w.setRemainingBuffer(batch, i)
				return
			}
		case "delete":
			err = retryOperation(context.Background(), func() error {
				return w.typesenseClient.DeleteDocument(context.Background(), w.config.Typesense.CollectionName, event.DocumentID)
			}, w.config.Typesense.MaxRetries, w.config.Typesense.RetryBackoff)

			if err != nil {
				log.Printf("Failed to delete document from Typesense after %d retries (ID: %s): %v",
					w.config.Typesense.MaxRetries, event.DocumentID, err)

				// Get all remaining events including the failed one
				w.setRemainingBuffer(batch, i)
				return
			}
		}
	}

	w.mu.Lock()
	w.lastFlush = time.Now()
	w.mu.Unlock()
}

func (w *EventProcessorImpl) Enqueue(ctx context.Context, event interfaces.Event) error {
	select {
	case w.eventChan <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *EventProcessorImpl) storeInRedis(events []interfaces.Event) {
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		log.Printf("Failed to marshal events: %v", err)
		return
	}

	ctx := context.Background()
	err = retryOperation(ctx, func() error {
		return w.redisClient.RPush(ctx, w.config.PrimaryRedis.QueueKey, eventsJSON)
	}, w.config.PrimaryRedis.MaxRetries, w.config.PrimaryRedis.RetryBackoff)

	if err != nil {
		// Try backup Redis
		err = retryOperation(ctx, func() error {
			return w.redisClient.RPush(ctx, w.config.BackupRedis.QueueKey, eventsJSON)
		}, w.config.BackupRedis.MaxRetries, w.config.BackupRedis.RetryBackoff)

		if err != nil {
			// Store in fallback if both Redis instances fail
			if err := w.storeInFallback(ctx, events...); err != nil {
				log.Printf("Failed to store events in fallback: %v", err)
			}
		}
	}
}

func (w *EventProcessorImpl) storeFailedEvents(events []interfaces.Event) {
	ctx := context.Background()
	if err := w.storeInFallback(ctx, events...); err != nil {
		log.Printf("Failed to store failed events in fallback: %v", err)
	}
}

func (w *EventProcessorImpl) storeInFallback(ctx context.Context, events ...interfaces.Event) error {
	if w.fallbackStore.client == nil {
		client, err := ConnectToMongoDB(w.config.MongoURI)
		if err != nil {
			return fmt.Errorf("failed to connect to MongoDB for fallback: %v", err)
		}
		w.fallbackStore.client = client
	}

	documents := make([]interface{}, len(events))
	for i, event := range events {
		documents[i] = bson.M{
			"operation_type": event.OperationType,
			"document":       event.Document,
			"document_id":    event.DocumentID,
			"timestamp":      event.Timestamp,
		}
	}

	_, err := w.fallbackStore.client.Database(w.fallbackStore.database).
		Collection(w.fallbackStore.collection).
		InsertMany(ctx, documents)

	return err
}

func (w *EventProcessorImpl) RecoverFromRedis() error {
	ctx := context.Background()

	events, err := w.recoverFromRedis(ctx, w.redisClient, w.config.PrimaryRedis.QueueKey)
	if err != nil {
		events, err = w.recoverFromRedis(ctx, w.redisClient, w.config.BackupRedis.QueueKey)
		if err != nil {
			return fmt.Errorf("failed to recover from both Redis instances: %v", err)
		}
	}

	for _, event := range events {
		if err := w.Enqueue(ctx, event); err != nil {
			log.Printf("Failed to enqueue recovered event: %v", err)
		}
	}

	return nil
}

func (w *EventProcessorImpl) recoverFromRedis(ctx context.Context, client interfaces.RedisClient, queueKey string) ([]interfaces.Event, error) {
	data, err := client.LPop(ctx, queueKey)
	if err != nil {
		return nil, err
	}

	var events []interfaces.Event
	if err := json.Unmarshal([]byte(data), &events); err != nil {
		return nil, err
	}

	return events, nil
}

func (w *EventProcessorImpl) checkQueueHealth() {
	ctx := context.Background()

	primaryLength, err := w.redisClient.LLen(ctx, w.config.PrimaryRedis.QueueKey)
	if err != nil {
		w.health.PrimaryQueueError = err
	} else {
		w.health.PrimaryQueueLength = primaryLength
		w.health.PrimaryQueueError = nil
	}

	backupLength, err := w.redisClient.LLen(ctx, w.config.BackupRedis.QueueKey)
	if err != nil {
		w.health.BackupQueueError = err
	} else {
		w.health.BackupQueueLength = backupLength
		w.health.BackupQueueError = nil
	}

	w.health.LastCheck = time.Now()
}

func (w *EventProcessorImpl) GetQueueHealth() QueueHealth {
	return w.health
}

func (w *EventProcessorImpl) Close() {
	close(w.done)
	w.mu.Lock()
	if len(w.activeBuffer) > 0 {
		w.processBatch()
	}
	w.mu.Unlock()
}

func retryOperation(ctx context.Context, operation func() error, maxRetries int, backoff time.Duration) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		if err = operation(); err == nil {
			return nil
		}
		time.Sleep(backoff)
	}
	return err
}

func ConnectToMongoDB(uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}
	return client, nil
}
