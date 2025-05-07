package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RedisConfig struct {
	Host            string
	Port            string
	Password        string
	DB              int
	QueueKey        string
	PoolSize        int
	MinIdleConns    int
	MaxRetries      int
	RetryBackoff    time.Duration
	ConnMaxLifetime time.Duration
	// Persistence settings
	SaveInterval   time.Duration
	AppendOnly     bool
	AppendFilename string
	RDBFilename    string
}

type TypesenseConfig struct {
	APIKey            string
	Host              string
	Port              string
	Protocol          string
	CollectionName    string
	ConnectionTimeout time.Duration
	KeepAliveTimeout  time.Duration
	MaxRetries        int
	RetryBackoff      time.Duration
}

type Config struct {
	MongoURI        string
	MongoDatabase   string
	MongoCollection string
	Typesense       TypesenseConfig
	MaxBufferSize   int
	BatchSize       int
	FlushInterval   int
	PrimaryRedis    RedisConfig
	BackupRedis     RedisConfig
}

func loadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	maxBufferSize, err := strconv.Atoi(os.Getenv("MAX_BUFFER_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_BUFFER_SIZE: %v", err)
	}

	batchSize, err := strconv.Atoi(os.Getenv("BATCH_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid BATCH_SIZE: %v", err)
	}

	flushInterval, err := strconv.Atoi(os.Getenv("FLUSH_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid FLUSH_INTERVAL: %v", err)
	}

	primaryRedisDB, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_DB"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_DB: %v", err)
	}

	backupRedisDB, err := strconv.Atoi(os.Getenv("BACKUP_REDIS_DB"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_DB: %v", err)
	}

	primaryPoolSize, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_POOL_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_POOL_SIZE: %v", err)
	}

	backupPoolSize, err := strconv.Atoi(os.Getenv("BACKUP_REDIS_POOL_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_POOL_SIZE: %v", err)
	}

	primaryMinIdleConns, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_MIN_IDLE_CONNS"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_MIN_IDLE_CONNS: %v", err)
	}

	backupMinIdleConns, err := strconv.Atoi(os.Getenv("BACKUP_REDIS_MIN_IDLE_CONNS"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_MIN_IDLE_CONNS: %v", err)
	}

	primaryMaxRetries, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_MAX_RETRIES"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_MAX_RETRIES: %v", err)
	}

	backupMaxRetries, err := strconv.Atoi(os.Getenv("BACKUP_REDIS_MAX_RETRIES"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_MAX_RETRIES: %v", err)
	}

	primaryRetryBackoff, err := time.ParseDuration(os.Getenv("PRIMARY_REDIS_RETRY_BACKOFF"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_RETRY_BACKOFF: %v", err)
	}

	backupRetryBackoff, err := time.ParseDuration(os.Getenv("BACKUP_REDIS_RETRY_BACKOFF"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_RETRY_BACKOFF: %v", err)
	}

	primaryConnMaxLifetime, err := time.ParseDuration(os.Getenv("PRIMARY_REDIS_CONN_MAX_LIFETIME"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_CONN_MAX_LIFETIME: %v", err)
	}

	backupConnMaxLifetime, err := time.ParseDuration(os.Getenv("BACKUP_REDIS_CONN_MAX_LIFETIME"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_CONN_MAX_LIFETIME: %v", err)
	}

	typesenseConnectionTimeout, err := time.ParseDuration(os.Getenv("TYPESENSE_CONNECTION_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid TYPESENSE_CONNECTION_TIMEOUT: %v", err)
	}

	typesenseKeepAliveTimeout, err := time.ParseDuration(os.Getenv("TYPESENSE_KEEPALIVE_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid TYPESENSE_KEEPALIVE_TIMEOUT: %v", err)
	}

	typesenseMaxRetries, err := strconv.Atoi(os.Getenv("TYPESENSE_MAX_RETRIES"))
	if err != nil {
		return nil, fmt.Errorf("invalid TYPESENSE_MAX_RETRIES: %v", err)
	}

	typesenseRetryBackoff, err := time.ParseDuration(os.Getenv("TYPESENSE_RETRY_BACKOFF"))
	if err != nil {
		return nil, fmt.Errorf("invalid TYPESENSE_RETRY_BACKOFF: %v", err)
	}

	// Load Redis persistence settings
	primarySaveInterval, err := time.ParseDuration(os.Getenv("PRIMARY_REDIS_SAVE_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_SAVE_INTERVAL: %v", err)
	}

	backupSaveInterval, err := time.ParseDuration(os.Getenv("BACKUP_REDIS_SAVE_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_SAVE_INTERVAL: %v", err)
	}

	primaryAppendOnly, err := strconv.ParseBool(os.Getenv("PRIMARY_REDIS_APPEND_ONLY"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_APPEND_ONLY: %v", err)
	}

	backupAppendOnly, err := strconv.ParseBool(os.Getenv("BACKUP_REDIS_APPEND_ONLY"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_APPEND_ONLY: %v", err)
	}

	return &Config{
		MongoURI:        os.Getenv("MONGODB_URI"),
		MongoDatabase:   os.Getenv("MONGODB_DATABASE"),
		MongoCollection: os.Getenv("MONGODB_COLLECTION"),
		Typesense: TypesenseConfig{
			APIKey:            os.Getenv("TYPESENSE_API_KEY"),
			Host:              os.Getenv("TYPESENSE_HOST"),
			Port:              os.Getenv("TYPESENSE_PORT"),
			Protocol:          os.Getenv("TYPESENSE_PROTOCOL"),
			CollectionName:    os.Getenv("TYPESENSE_COLLECTION_NAME"),
			ConnectionTimeout: typesenseConnectionTimeout,
			KeepAliveTimeout:  typesenseKeepAliveTimeout,
			MaxRetries:        typesenseMaxRetries,
			RetryBackoff:      typesenseRetryBackoff,
		},
		MaxBufferSize: maxBufferSize,
		BatchSize:     batchSize,
		FlushInterval: flushInterval,
		PrimaryRedis: RedisConfig{
			Host:            os.Getenv("PRIMARY_REDIS_HOST"),
			Port:            os.Getenv("PRIMARY_REDIS_PORT"),
			Password:        os.Getenv("PRIMARY_REDIS_PASSWORD"),
			DB:              primaryRedisDB,
			QueueKey:        os.Getenv("PRIMARY_REDIS_QUEUE_KEY"),
			PoolSize:        primaryPoolSize,
			MinIdleConns:    primaryMinIdleConns,
			MaxRetries:      primaryMaxRetries,
			RetryBackoff:    primaryRetryBackoff,
			ConnMaxLifetime: primaryConnMaxLifetime,
			SaveInterval:    primarySaveInterval,
			AppendOnly:      primaryAppendOnly,
			AppendFilename:  os.Getenv("PRIMARY_REDIS_APPEND_FILENAME"),
			RDBFilename:     os.Getenv("PRIMARY_REDIS_RDB_FILENAME"),
		},
		BackupRedis: RedisConfig{
			Host:            os.Getenv("BACKUP_REDIS_HOST"),
			Port:            os.Getenv("BACKUP_REDIS_PORT"),
			Password:        os.Getenv("BACKUP_REDIS_PASSWORD"),
			DB:              backupRedisDB,
			QueueKey:        os.Getenv("BACKUP_REDIS_QUEUE_KEY"),
			PoolSize:        backupPoolSize,
			MinIdleConns:    backupMinIdleConns,
			MaxRetries:      backupMaxRetries,
			RetryBackoff:    backupRetryBackoff,
			ConnMaxLifetime: backupConnMaxLifetime,
			SaveInterval:    backupSaveInterval,
			AppendOnly:      backupAppendOnly,
			AppendFilename:  os.Getenv("BACKUP_REDIS_APPEND_FILENAME"),
			RDBFilename:     os.Getenv("BACKUP_REDIS_RDB_FILENAME"),
		},
	}, nil
}

type Event struct {
	OperationType string                 `json:"operation_type"`
	Document      map[string]interface{} `json:"document,omitempty"`
	DocumentID    string                 `json:"document_id"`
	Timestamp     time.Time              `json:"timestamp"`
}

type QueueHealth struct {
	PrimaryQueueLength int64
	BackupQueueLength  int64
	LastCheck          time.Time
	PrimaryQueueError  error
	BackupQueueError   error
}

type FallbackStore struct {
	client     *mongo.Client
	database   string
	collection string
}

type EventBuffer struct {
	mu            sync.Mutex
	events        []Event
	lastFlush     time.Time
	config        *Config
	primaryRedis  *redis.Client
	backupRedis   *redis.Client
	health        QueueHealth
	fallbackStore *FallbackStore
}

func NewEventBuffer(config *Config) (*EventBuffer, error) {
	// Initialize primary Redis client with connection pool and persistence
	primaryRedis := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%s", config.PrimaryRedis.Host, config.PrimaryRedis.Port),
		Password:        config.PrimaryRedis.Password,
		DB:              config.PrimaryRedis.DB,
		PoolSize:        config.PrimaryRedis.PoolSize,
		MinIdleConns:    config.PrimaryRedis.MinIdleConns,
		MaxRetries:      config.PrimaryRedis.MaxRetries,
		ConnMaxLifetime: config.PrimaryRedis.ConnMaxLifetime,
	})

	// Initialize backup Redis client with connection pool and persistence
	backupRedis := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%s", config.BackupRedis.Host, config.BackupRedis.Port),
		Password:        config.BackupRedis.Password,
		DB:              config.BackupRedis.DB,
		PoolSize:        config.BackupRedis.PoolSize,
		MinIdleConns:    config.BackupRedis.MinIdleConns,
		MaxRetries:      config.BackupRedis.MaxRetries,
		ConnMaxLifetime: config.BackupRedis.ConnMaxLifetime,
	})

	// Configure Redis persistence settings
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Configure primary Redis persistence
	if err := configureRedisPersistence(ctx, primaryRedis, config.PrimaryRedis); err != nil {
		return nil, fmt.Errorf("failed to configure primary Redis persistence: %v", err)
	}

	// Configure backup Redis persistence
	if err := configureRedisPersistence(ctx, backupRedis, config.BackupRedis); err != nil {
		return nil, fmt.Errorf("failed to configure backup Redis persistence: %v", err)
	}

	// Initialize MongoDB fallback store
	mongoClient, err := connectToMongoDB(config.MongoURI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB fallback store: %v", err)
	}

	fallbackStore := &FallbackStore{
		client:     mongoClient,
		database:   config.MongoDatabase,
		collection: "typesense_sync_fallback",
	}

	// Test Redis connections with retry
	if err := retryOperation(ctx, func() error {
		return primaryRedis.Ping(ctx).Err()
	}, config.PrimaryRedis.MaxRetries, config.PrimaryRedis.RetryBackoff); err != nil {
		return nil, fmt.Errorf("failed to connect to primary Redis: %v", err)
	}

	if err := retryOperation(ctx, func() error {
		return backupRedis.Ping(ctx).Err()
	}, config.BackupRedis.MaxRetries, config.BackupRedis.RetryBackoff); err != nil {
		return nil, fmt.Errorf("failed to connect to backup Redis: %v", err)
	}

	return &EventBuffer{
		events:        make([]Event, 0, config.MaxBufferSize),
		lastFlush:     time.Now(),
		config:        config,
		primaryRedis:  primaryRedis,
		backupRedis:   backupRedis,
		health:        QueueHealth{},
		fallbackStore: fallbackStore,
	}, nil
}

func configureRedisPersistence(ctx context.Context, client *redis.Client, config RedisConfig) error {
	// Configure RDB persistence
	if config.SaveInterval > 0 {
		// Set save interval (e.g., "900 1" means save if 1 change in 900 seconds)
		cmd := client.ConfigSet(ctx, "save", fmt.Sprintf("%d 1", int(config.SaveInterval.Seconds())))
		if err := cmd.Err(); err != nil {
			return fmt.Errorf("failed to set RDB save interval: %v", err)
		}

		// Set RDB filename if specified
		if config.RDBFilename != "" {
			cmd := client.ConfigSet(ctx, "dbfilename", config.RDBFilename)
			if err := cmd.Err(); err != nil {
				return fmt.Errorf("failed to set RDB filename: %v", err)
			}
		}
	}

	// Configure AOF persistence if enabled
	if config.AppendOnly {
		// Enable AOF
		cmd := client.ConfigSet(ctx, "appendonly", "yes")
		if err := cmd.Err(); err != nil {
			return fmt.Errorf("failed to enable AOF: %v", err)
		}

		// Set append filename if specified
		if config.AppendFilename != "" {
			cmd := client.ConfigSet(ctx, "appendfilename", config.AppendFilename)
			if err := cmd.Err(); err != nil {
				return fmt.Errorf("failed to set AOF filename: %v", err)
			}
		}

		// Set AOF fsync policy to "everysec" for a good balance of performance and durability
		cmd = client.ConfigSet(ctx, "appendfsync", "everysec")
		if err := cmd.Err(); err != nil {
			return fmt.Errorf("failed to set AOF fsync policy: %v", err)
		}
	}

	// Save the configuration
	cmd := client.ConfigRewrite(ctx)
	if err := cmd.Err(); err != nil {
		return fmt.Errorf("failed to rewrite Redis configuration: %v", err)
	}

	return nil
}

// retryOperation implements exponential backoff retry logic
func retryOperation(ctx context.Context, operation func() error, maxRetries int, backoff time.Duration) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err = operation(); err == nil {
				return nil
			}
			if i < maxRetries-1 {
				time.Sleep(backoff * time.Duration(1<<uint(i)))
			}
		}
	}
	return err
}

func (b *EventBuffer) Add(event Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Add timestamp to event
	event.Timestamp = time.Now()

	// Add to memory buffer
	b.events = append(b.events, event)

	// Also persist to both Redis queues with retry
	ctx := context.Background()
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event for Redis: %v", err)
		return
	}

	// Try primary Redis queue first with retry
	err = retryOperation(ctx, func() error {
		return b.primaryRedis.LPush(ctx, b.config.PrimaryRedis.QueueKey, eventJSON).Err()
	}, b.config.PrimaryRedis.MaxRetries, b.config.PrimaryRedis.RetryBackoff)

	if err != nil {
		log.Printf("Error pushing event to primary Redis queue after retries: %v", err)
		// If primary fails, try backup queue with retry
		err = retryOperation(ctx, func() error {
			return b.backupRedis.LPush(ctx, b.config.BackupRedis.QueueKey, eventJSON).Err()
		}, b.config.BackupRedis.MaxRetries, b.config.BackupRedis.RetryBackoff)

		if err != nil {
			log.Printf("Error pushing event to backup Redis queue after retries: %v", err)
			// If both Redis queues fail, store in MongoDB fallback
			if err := b.storeInFallback(ctx, event); err != nil {
				log.Printf("Error storing event in fallback store: %v", err)
			} else {
				log.Printf("Successfully stored event in fallback store")
			}
		} else {
			log.Printf("Successfully pushed event to backup Redis queue")
		}
	}
}

func (b *EventBuffer) storeInFallback(ctx context.Context, event Event) error {
	// Convert event to BSON document
	doc := bson.M{
		"operation_type": event.OperationType,
		"document":       event.Document,
		"document_id":    event.DocumentID,
		"timestamp":      event.Timestamp,
		"status":         "pending",
	}

	// Insert into MongoDB fallback collection
	_, err := b.fallbackStore.client.Database(b.fallbackStore.database).
		Collection(b.fallbackStore.collection).
		InsertOne(ctx, doc)

	return err
}

func (b *EventBuffer) GetBatch() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.events) == 0 {
		return nil
	}

	batchSize := min(len(b.events), b.config.BatchSize)
	batch := b.events[:batchSize]
	b.events = b.events[batchSize:]
	b.lastFlush = time.Now()

	return batch
}

func (b *EventBuffer) ShouldFlush() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return len(b.events) >= b.config.BatchSize ||
		(len(b.events) > 0 && time.Since(b.lastFlush) >= time.Duration(b.config.FlushInterval)*time.Second)
}

func (b *EventBuffer) RecoverFromRedis() error {
	ctx := context.Background()

	// Try to recover from primary queue first
	events, err := b.primaryRedis.LRange(ctx, b.config.PrimaryRedis.QueueKey, 0, -1).Result()
	if err != nil {
		log.Printf("Error getting events from primary Redis queue: %v", err)
		// If primary fails, try backup queue
		events, err = b.backupRedis.LRange(ctx, b.config.BackupRedis.QueueKey, 0, -1).Result()
		if err != nil {
			log.Printf("Error getting events from backup Redis queue: %v", err)
			// If both Redis queues fail, try to recover from MongoDB fallback
			return b.recoverFromFallback(ctx)
		}
		log.Printf("Recovered events from backup Redis queue")
	}

	// Clear both Redis queues
	if err := b.primaryRedis.Del(ctx, b.config.PrimaryRedis.QueueKey).Err(); err != nil {
		log.Printf("Warning: Failed to clear primary Redis queue: %v", err)
	}
	if err := b.backupRedis.Del(ctx, b.config.BackupRedis.QueueKey).Err(); err != nil {
		log.Printf("Warning: Failed to clear backup Redis queue: %v", err)
	}

	// Unmarshal and add events to memory buffer
	for _, eventJSON := range events {
		var event Event
		if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
			log.Printf("Error unmarshaling event from Redis: %v", err)
			continue
		}
		b.events = append(b.events, event)
	}

	return nil
}

func (b *EventBuffer) recoverFromFallback(ctx context.Context) error {
	// Find all pending events in fallback store
	cursor, err := b.fallbackStore.client.Database(b.fallbackStore.database).
		Collection(b.fallbackStore.collection).
		Find(ctx, bson.M{"status": "pending"})
	if err != nil {
		return fmt.Errorf("error querying fallback store: %v", err)
	}
	defer cursor.Close(ctx)

	// Process each event
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("Error decoding fallback document: %v", err)
			continue
		}

		// Convert BSON to Event
		event := Event{
			OperationType: doc["operation_type"].(string),
			Document:      doc["document"].(map[string]interface{}),
			DocumentID:    doc["document_id"].(string),
			Timestamp:     doc["timestamp"].(time.Time),
		}

		// Add to memory buffer
		b.events = append(b.events, event)

		// Mark as processed in fallback store
		_, err := b.fallbackStore.client.Database(b.fallbackStore.database).
			Collection(b.fallbackStore.collection).
			UpdateOne(
				ctx,
				bson.M{"_id": doc["_id"]},
				bson.M{"$set": bson.M{"status": "processed"}},
			)
		if err != nil {
			log.Printf("Error updating fallback document status: %v", err)
		}
	}

	return cursor.Err()
}

func (b *EventBuffer) checkQueueHealth() {
	b.mu.Lock()
	defer b.mu.Unlock()

	ctx := context.Background()
	now := time.Now()

	// Check primary queue with retry
	var primaryLength int64
	err := retryOperation(ctx, func() error {
		var err error
		primaryLength, err = b.primaryRedis.LLen(ctx, b.config.PrimaryRedis.QueueKey).Result()
		return err
	}, b.config.PrimaryRedis.MaxRetries, b.config.PrimaryRedis.RetryBackoff)

	if err != nil {
		b.health.PrimaryQueueError = fmt.Errorf("primary queue health check failed after retries: %v", err)
		log.Printf("Primary queue health check failed: %v", err)
	} else {
		b.health.PrimaryQueueError = nil
		b.health.PrimaryQueueLength = primaryLength
	}

	// Check backup queue with retry
	var backupLength int64
	err = retryOperation(ctx, func() error {
		var err error
		backupLength, err = b.backupRedis.LLen(ctx, b.config.BackupRedis.QueueKey).Result()
		return err
	}, b.config.BackupRedis.MaxRetries, b.config.BackupRedis.RetryBackoff)

	if err != nil {
		b.health.BackupQueueError = fmt.Errorf("backup queue health check failed after retries: %v", err)
		log.Printf("Backup queue health check failed: %v", err)
	} else {
		b.health.BackupQueueError = nil
		b.health.BackupQueueLength = backupLength
	}

	b.health.LastCheck = now

	// Log queue status
	log.Printf("Queue Health Status - Primary: %d items, Backup: %d items, Last Check: %s",
		b.health.PrimaryQueueLength,
		b.health.BackupQueueLength,
		b.health.LastCheck.Format(time.RFC3339))
}

func (b *EventBuffer) GetQueueHealth() QueueHealth {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.health
}

func processBatch(ctx context.Context, typesenseClient *typesense.Client, typesenseCollection string, events []Event, buffer *EventBuffer) error {
	// Group events by operation type
	upserts := make([]interface{}, 0)
	deletes := make([]string, 0)

	for _, event := range events {
		switch event.OperationType {
		case "insert", "update":
			upserts = append(upserts, event.Document)
		case "delete":
			deletes = append(deletes, event.DocumentID)
		}
	}

	// Process upserts in batch with retry
	if len(upserts) > 0 {
		action := "upsert"
		batchSize := len(upserts)
		err := retryOperation(ctx, func() error {
			_, err := typesenseClient.Collection(typesenseCollection).Documents().Import(ctx, upserts, &api.ImportDocumentsParams{
				Action:    &action,
				BatchSize: &batchSize,
			})
			return err
		}, buffer.config.Typesense.MaxRetries, buffer.config.Typesense.RetryBackoff)

		if err != nil {
			// If Typesense fails, store in fallback
			for _, event := range events {
				if event.OperationType == "insert" || event.OperationType == "update" {
					if err := buffer.storeInFallback(ctx, event); err != nil {
						log.Printf("Error storing event in fallback store: %v", err)
					}
				}
			}
			return fmt.Errorf("error batch upserting to Typesense after retries: %v", err)
		}
		log.Printf("Successfully batch upserted %d documents to Typesense", len(upserts))
	}

	// Process deletes in batch with retry
	if len(deletes) > 0 {
		for _, docID := range deletes {
			err := retryOperation(ctx, func() error {
				_, err := typesenseClient.Collection(typesenseCollection).Document(docID).Delete(ctx)
				return err
			}, buffer.config.Typesense.MaxRetries, buffer.config.Typesense.RetryBackoff)

			if err != nil {
				// If Typesense fails, store in fallback
				for _, event := range events {
					if event.OperationType == "delete" && event.DocumentID == docID {
						if err := buffer.storeInFallback(ctx, event); err != nil {
							log.Printf("Error storing event in fallback store: %v", err)
						}
					}
				}
				log.Printf("Error deleting document from Typesense after retries: %v", err)
			}
		}
		log.Printf("Successfully deleted %d documents from Typesense", len(deletes))
	}

	// Remove processed events from both Redis queues
	for _, event := range events {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error marshaling event for Redis removal: %v", err)
			continue
		}

		// Remove from primary queue
		if err := buffer.primaryRedis.LRem(ctx, buffer.config.PrimaryRedis.QueueKey, 1, eventJSON).Err(); err != nil {
			log.Printf("Error removing event from primary Redis queue: %v", err)
		}

		// Remove from backup queue
		if err := buffer.backupRedis.LRem(ctx, buffer.config.BackupRedis.QueueKey, 1, eventJSON).Err(); err != nil {
			log.Printf("Error removing event from backup Redis queue: %v", err)
		}
	}

	return nil
}

func watchMongoChanges(client *mongo.Client, database, collection string, typesenseClient *typesense.Client, typesenseCollection string, config *Config) error {
	ctx := context.Background()
	coll := client.Database(database).Collection(collection)

	buffer, err := NewEventBuffer(config)
	if err != nil {
		return fmt.Errorf("failed to create event buffer: %v", err)
	}

	// Recover any pending events from Redis
	if err := buffer.RecoverFromRedis(); err != nil {
		log.Printf("Warning: Failed to recover events from Redis: %v", err)
	}

	// Create a change stream
	changeStream, err := coll.Watch(ctx, mongo.Pipeline{})
	if err != nil {
		return fmt.Errorf("failed to create change stream: %v", err)
	}
	defer changeStream.Close(ctx)

	fmt.Printf("Watching for changes in %s.%s...\n", database, collection)

	// Start a goroutine to periodically flush the buffer
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if buffer.ShouldFlush() {
				if batch := buffer.GetBatch(); batch != nil {
					if err := processBatch(ctx, typesenseClient, typesenseCollection, batch, buffer); err != nil {
						log.Printf("Error processing batch: %v", err)
					}
				}
			}
		}
	}()

	// Start a goroutine to periodically check queue health
	go func() {
		healthTicker := time.NewTicker(30 * time.Second)
		defer healthTicker.Stop()

		for range healthTicker.C {
			buffer.checkQueueHealth()
		}
	}()

	for changeStream.Next(ctx) {
		var changeDoc bson.M
		if err := changeStream.Decode(&changeDoc); err != nil {
			log.Printf("Error decoding change document: %v", err)
			continue
		}

		operationType := changeDoc["operationType"]
		fullDocument := changeDoc["fullDocument"]

		switch operationType {
		case "insert", "update":
			// Convert the document to a map for Typesense
			doc := make(map[string]interface{})
			bsonBytes, err := bson.Marshal(fullDocument)
			if err != nil {
				log.Printf("Error marshaling document: %v", err)
				continue
			}
			if err := bson.Unmarshal(bsonBytes, &doc); err != nil {
				log.Printf("Error unmarshaling document: %v", err)
				continue
			}

			// Add to buffer
			buffer.Add(Event{
				OperationType: operationType.(string),
				Document:      doc,
				DocumentID:    fmt.Sprintf("%v", doc["_id"]),
			})

		case "delete":
			// Get the document ID from the change document
			docKey := changeDoc["documentKey"].(bson.M)["_id"]
			docKeyStr := fmt.Sprintf("%v", docKey)

			// Add to buffer
			buffer.Add(Event{
				OperationType: "delete",
				DocumentID:    docKeyStr,
			})
		}

		// Check if we should process the buffer
		if buffer.ShouldFlush() {
			if batch := buffer.GetBatch(); batch != nil {
				if err := processBatch(ctx, typesenseClient, typesenseCollection, batch, buffer); err != nil {
					log.Printf("Error processing batch: %v", err)
				}
			}
		}
	}

	if err := changeStream.Err(); err != nil {
		return fmt.Errorf("change stream error: %v", err)
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func connectToMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	return client, nil
}

func connectToTypesense(config *Config) (*typesense.Client, error) {
	// Create Typesense client with standard configuration
	client := typesense.NewClient(
		typesense.WithServer(fmt.Sprintf("%s://%s:%s", config.Typesense.Protocol, config.Typesense.Host, config.Typesense.Port)),
		typesense.WithAPIKey(config.Typesense.APIKey),
		typesense.WithConnectionTimeout(config.Typesense.ConnectionTimeout),
	)

	// Test connection with retry
	ctx, cancel := context.WithTimeout(context.Background(), config.Typesense.ConnectionTimeout)
	defer cancel()

	err := retryOperation(ctx, func() error {
		// Try to get collection info as a connection test
		_, err := client.Collection(config.Typesense.CollectionName).Retrieve(ctx)
		return err
	}, config.Typesense.MaxRetries, config.Typesense.RetryBackoff)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to Typesense after retries: %v", err)
	}

	return client, nil
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to MongoDB
	mongoClient, err := connectToMongoDB(config.MongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	// Connect to Typesense
	typesenseClient, err := connectToTypesense(config)
	if err != nil {
		log.Fatalf("Failed to connect to Typesense: %v", err)
	}

	// Start watching for changes
	err = watchMongoChanges(mongoClient, config.MongoDatabase, config.MongoCollection, typesenseClient, config.Typesense.CollectionName, config)
	if err != nil {
		log.Fatalf("Error watching MongoDB changes: %v", err)
	}
}
