package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	externalservices "watcher/external-services"
	"watcher/logger"

	"go.mongodb.org/mongo-driver/bson"
)

// BackupFlusherContract defines the interface for backup flushing operations
type BackupFlusherContract interface {
	Flush(events []externalservices.MongoChangeEvent)
}

// BackupFlusher handles dumping events to a backup queue
type BackupFlusher struct {
	primaryRedisService  externalservices.RedisServiceContract
	backupRedisService   externalservices.RedisServiceContract
	fallbackMongoService externalservices.MongoServiceContract
}

// Ensure BackupFlusher implements BackupFlusherContract
var _ BackupFlusherContract = (*BackupFlusher)(nil)

// NewBackupFlusher creates a new BackupFlusher instance
func NewBackupFlusher(
	primaryRedisService externalservices.RedisServiceContract,
	backupRedisService externalservices.RedisServiceContract,
	fallbackMongoService externalservices.MongoServiceContract,
) *BackupFlusher {
	return &BackupFlusher{
		primaryRedisService:  primaryRedisService,
		backupRedisService:   backupRedisService,
		fallbackMongoService: fallbackMongoService,
	}
}

// Flush stores events in the backup queue
func (b *BackupFlusher) Flush(events []externalservices.MongoChangeEvent) {
	logger.Info().
		Int("event_count", len(events)).
		Msg("Starting backup flush operation")

	var wg sync.WaitGroup
	wg.Add(2)

	// Store in Redis concurrently
	go func() {
		defer wg.Done()
		b.FlushToRedis(events)
	}()

	// Store in MongoDB concurrently
	go func() {
		defer wg.Done()
		if err := b.storeInFallback(context.Background(), events...); err != nil {
			logger.Error().
				Err(err).
				Int("event_count", len(events)).
				Msg("Failed to store events in fallback MongoDB")
		}
	}()

	wg.Wait()
	logger.Info().
		Int("event_count", len(events)).
		Msg("Completed backup flush operation")
}

func (b *BackupFlusher) FlushToRedis(events []externalservices.MongoChangeEvent) {
	eventsJSON, err := json.Marshal(events)
	if err != nil {
		logger.Error().
			Err(err).
			Int("event_count", len(events)).
			Msg("Failed to marshal events for Redis")
		return
	}

	ctx := context.Background()

	// Push to primary Redis
	err = retryOperation(ctx, func() error {
		return b.primaryRedisService.RPush(ctx, eventsJSON)
	}, 3, time.Second)

	if err != nil {
		logger.Error().
			Err(err).
			Int("event_count", len(events)).
			Msg("Failed to store events in primary Redis")
	} else {
		logger.Debug().
			Int("event_count", len(events)).
			Msg("Successfully stored events in primary Redis")
	}

	// Push to backup Redis
	err = retryOperation(ctx, func() error {
		return b.backupRedisService.RPush(ctx, eventsJSON)
	}, 3, time.Second)

	if err != nil {
		logger.Error().
			Err(err).
			Int("event_count", len(events)).
			Msg("Failed to store events in backup Redis")
	} else {
		logger.Debug().
			Int("event_count", len(events)).
			Msg("Successfully stored events in backup Redis")
	}
}

func (b *BackupFlusher) storeInFallback(ctx context.Context, events ...externalservices.MongoChangeEvent) error {
	logger.Debug().
		Int("event_count", len(events)).
		Msg("Connecting to fallback MongoDB")

	if err := b.fallbackMongoService.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to MongoDB for fallback: %v", err)
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

	collection := b.fallbackMongoService.GetCollection()
	_, err := collection.InsertMany(ctx, documents)
	if err != nil {
		return err
	}

	logger.Debug().
		Int("event_count", len(events)).
		Msg("Successfully stored events in fallback MongoDB")
	return nil
}
