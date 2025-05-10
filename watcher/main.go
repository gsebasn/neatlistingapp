package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	externalservices "watcher/external-services"
	"watcher/interfaces"

	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Redis client
	redisClient := externalservices.NewRedisService(
		config.PrimaryRedis.Host,
		config.PrimaryRedis.Port,
		config.PrimaryRedis.Password,
		config.PrimaryRedis.DB,
	)

	// Initialize Typesense client
	typesenseClient := externalservices.NewTypesenseService(
		config.Typesense.APIKey,
		config.Typesense.Host,
		config.Typesense.Port,
		config.Typesense.Protocol,
	)

	// Initialize event processor
	eventProcessor, err := NewEventProcessor(config, redisClient, typesenseClient)
	if err != nil {
		log.Fatalf("Failed to initialize event processor: %v", err)
	}

	// Initialize MongoDB service
	mongoService := externalservices.NewMongoService(
		config.MongoURI,
		config.MongoDatabase,
		config.MongoCollection,
	)

	// Connect to MongoDB
	if err := mongoService.Connect(context.Background()); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoService.Disconnect(context.Background())

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	// Start watching MongoDB changes
	log.Println("Starting MongoDB change stream watcher...")
	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)
	stream, err := mongoService.Watch(ctx, nil, opts)
	if err != nil {
		log.Fatalf("Error creating change stream: %v", err)
	}
	defer stream.Close(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !stream.Next(ctx) {
				if err := stream.Err(); err != nil {
					log.Printf("Change stream error: %v", err)
				}
				continue
			}

			var changeDoc struct {
				OperationType string                 `bson:"operationType"`
				FullDocument  map[string]interface{} `bson:"fullDocument"`
				DocumentKey   map[string]interface{} `bson:"documentKey"`
				ClusterTime   time.Time              `bson:"clusterTime"`
			}

			if err := stream.Decode(&changeDoc); err != nil {
				log.Printf("Failed to decode change document: %v", err)
				continue
			}

			event := interfaces.Event{
				OperationType: changeDoc.OperationType,
				Document:      changeDoc.FullDocument,
				DocumentID:    changeDoc.DocumentKey["_id"].(string),
				Timestamp:     changeDoc.ClusterTime,
			}

			if err := eventProcessor.Enqueue(ctx, event); err != nil {
				log.Printf("Failed to enqueue event: %v", err)
			}
		}
	}
}
