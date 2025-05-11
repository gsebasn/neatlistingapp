package main

import (
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

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
