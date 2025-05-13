package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	externalservices "watcher/external-services"

	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create Redis services
	primaryRedis := externalservices.NewRedisService(
		config.PrimaryRedis.Host,
		config.PrimaryRedis.Port,
		config.PrimaryRedis.Password,
		config.PrimaryRedis.DB,
		config.PrimaryRedis.QueueKey,
	)

	backupRedis := externalservices.NewRedisService(
		config.SecondaryRedis.Host,
		config.SecondaryRedis.Port,
		config.SecondaryRedis.Password,
		config.SecondaryRedis.DB,
		config.SecondaryRedis.QueueKey,
	)

	// Create MongoDB services
	watchingMongoDBService := externalservices.NewMongoService(
		config.PrimaryMongoURI,
		config.PrimaryMongoDatabase,
		config.PrimaryMongoCollection,
	)

	fallbackMongoService := externalservices.NewMongoService(
		config.FallbackMongoURI,
		config.FallbackMongoDatabase,
		config.FallbackMongoCollection,
	)

	// Create Typesense service
	typesenseService := externalservices.NewTypesenseService(
		config.Typesense.APIKey,
		config.Typesense.Host,
		config.Typesense.Port,
		config.Typesense.Protocol,
		config.Typesense.CollectionName,
	)

	eventProcessor, err := NewEventProcessor(
		typesenseService,
		primaryRedis,
		backupRedis,
		watchingMongoDBService,
		fallbackMongoService,
		config.MaxBufferSize,
		time.Duration(config.FlushInterval)*time.Second,
	)
	if err != nil {
		log.Fatalf("Failed to initialize event processor: %v", err)
	}

	// Connect to MongoDB services
	if err := watchingMongoDBService.Connect(context.Background()); err != nil {
		log.Fatalf("Failed to connect to watching MongoDB: %v", err)
	}
	defer watchingMongoDBService.Disconnect(context.Background())

	if err := fallbackMongoService.Connect(context.Background()); err != nil {
		log.Fatalf("Failed to connect to fallback MongoDB: %v", err)
	}
	defer fallbackMongoService.Disconnect(context.Background())

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
	stream, err := watchingMongoDBService.Watch(ctx, nil, opts)
	if err != nil {
		log.Fatalf("Error creating change stream: %v", err)
	}
	defer stream.Close(ctx)

	// Get the processing state channel
	processingState := eventProcessor.GetProcessBusyState()
	isProcessing := false

	for {
		select {
		case <-ctx.Done():
			return
		case processing := <-processingState:
			isProcessing = processing
			if !isProcessing {
				log.Println("Processing finished, resuming MongoDB event reading")
			} else {
				log.Println("Processing started, pausing MongoDB event reading")
			}
		default:
			// Only read from MongoDB if we're not processing
			if isProcessing {
				// Sleep briefly to prevent tight loop
				time.Sleep(100 * time.Millisecond)
				continue
			}

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

			event := externalservices.MongoChangeEvent{
				OperationType: changeDoc.OperationType,
				Document:      changeDoc.FullDocument,
				DocumentID:    changeDoc.DocumentKey["_id"].(string),
				Timestamp:     changeDoc.ClusterTime.Unix(),
			}

			if err := eventProcessor.Enqueue(ctx, event); err != nil {
				log.Printf("Failed to enqueue event: %v", err)
			}
		}
	}
}
