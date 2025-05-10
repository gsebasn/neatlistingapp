package main

import (
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

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

type FallbackStoreImpl struct {
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
	fallbackStore *FallbackStoreImpl
}
